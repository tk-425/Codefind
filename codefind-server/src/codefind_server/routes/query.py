from __future__ import annotations

import re
from fastapi import APIRouter, Depends, HTTPException, Request, status

from ..adapters.base import SearchResult, VectorStore
from ..middleware.auth import OrgContext, require_auth
from ..models.requests import QueryRequest
from ..models.responses import QueryResponse, QueryResultResponse
from ..services import OllamaService, collection_name_for, repo_id_from_collection, validate_repo_id
from ..services.ollama import OllamaError
from .collections import get_vector_store


router = APIRouter(prefix="/query", tags=["query"])

ALLOWED_FILTER_KEYS = {"project", "language", "status"}
MAX_TOP_K = 50
MAX_RERANK_CANDIDATES = 100
IMPLEMENTATION_INTENT = "implementation"
REFERENCE_INTENT = "reference"
TEST_INTENT = "test"
CONFIG_INTENT = "config"

QUERY_TOKEN_PATTERN = re.compile(r"[a-z0-9_]+")
SHORT_REFERENCE_PATTERN = re.compile(r"^\s*[A-Za-z_][A-Za-z0-9_]*\s*=\s*[A-Za-z_][A-Za-z0-9_.]*\s*$")
DECLARATION_PATTERNS = (
    re.compile(r"^\s*func\s+[A-Za-z0-9_]+"),
    re.compile(r"^\s*def\s+[A-Za-z0-9_]+"),
    re.compile(r"^\s*class\s+[A-Za-z0-9_]+"),
    re.compile(r"^\s*interface\s+[A-Za-z0-9_]+"),
    re.compile(r"^\s*type\s+[A-Za-z0-9_]+\s+"),
    re.compile(r"^\s*export\s+(async\s+)?function\s+[A-Za-z0-9_]+"),
    re.compile(r"^\s*export\s+class\s+[A-Za-z0-9_]+"),
    re.compile(r"^\s*(async\s+)?function\s+[A-Za-z0-9_]+"),
    re.compile(r"^\s*const\s+[A-Za-z0-9_]+\s*=\s*(async\s*)?\("),
)
IMPLEMENTATION_SYMBOL_KINDS = {"function", "method", "class", "interface", "module", "namespace", "constructor"}
NON_IMPLEMENTATION_SYMBOL_KINDS = {"variable", "constant", "property", "field", "key", "string", "number", "boolean"}


def get_ollama_service(request: Request) -> OllamaService:
    return request.app.state.ollama


def _build_filters(payload: QueryRequest) -> dict[str, object]:
    filters: dict[str, object] = {"status": "active"}
    if payload.project:
        filters["project"] = payload.project
    if payload.language:
        filters["language"] = payload.language
    return {key: value for key, value in filters.items() if key in ALLOWED_FILTER_KEYS}


def _query_tokens(text: str) -> set[str]:
    return {match.group(0) for match in QUERY_TOKEN_PATTERN.finditer(text.lower())}


def _candidate_limit(payload: QueryRequest) -> int:
    requested = max(payload.page_size*5, payload.top_k*3, 30)
    return min(requested, MAX_RERANK_CANDIDATES)


def _classify_intent(query_text: str) -> str:
    lowered = query_text.lower()
    if any(token in lowered for token in ("test", "tests", "example", "examples")):
        return TEST_INTENT
    if any(token in lowered for token in ("config", "setup", "env", "environment", "variable")):
        return CONFIG_INTENT
    if any(token in lowered for token in ("who calls", "callers", "used by", "references", "reference")):
        return REFERENCE_INTENT
    return IMPLEMENTATION_INTENT


def _payload_text(payload: dict[str, object], key: str) -> str:
    value = payload.get(key)
    return value if isinstance(value, str) else ""


def _is_test_path(path: str) -> bool:
    lowered = path.lower()
    return (
        "/tests/" in lowered
        or "tests/" in lowered
        or lowered.startswith("tests/")
        or lowered.endswith("_test.go")
        or lowered.endswith("_test.py")
        or lowered.startswith("test_")
    )


def _is_config_path(path: str) -> bool:
    lowered = path.lower()
    return (
        lowered.endswith(".env")
        or lowered.endswith(".env.example")
        or lowered.endswith(".json")
        or lowered.endswith(".yaml")
        or lowered.endswith(".yml")
        or lowered.endswith(".toml")
        or "/config/" in lowered
        or lowered.endswith("/config.py")
    )


def _implementation_path_boost(path: str) -> float:
    lowered = path.lower()
    if any(lowered.startswith(prefix) for prefix in ("internal/", "cmd/", "web/src/", "codefind-server/src/")):
        return 0.025
    return 0.0


def _is_definition_like(payload: dict[str, object]) -> bool:
    snippet = _payload_text(payload, "snippet")
    content = _payload_text(payload, "content")
    text = snippet or content
    if not text:
        return False
    first_line = text.splitlines()[0]
    return any(pattern.search(first_line) for pattern in DECLARATION_PATTERNS)


def _symbol_kind(payload: dict[str, object]) -> str:
    return _payload_text(payload, "symbol_kind").lower()


def _is_implementation_symbol_kind(payload: dict[str, object]) -> bool:
    return _symbol_kind(payload) in IMPLEMENTATION_SYMBOL_KINDS


def _is_non_implementation_symbol_kind(payload: dict[str, object]) -> bool:
    return _symbol_kind(payload) in NON_IMPLEMENTATION_SYMBOL_KINDS


def _is_short_reference_like(payload: dict[str, object]) -> bool:
    snippet = _payload_text(payload, "snippet")
    content = _payload_text(payload, "content")
    text = (snippet or content).strip()
    if not text:
        return False
    line_count = len(text.splitlines())
    if line_count > 2 or len(text) > 140:
        return False
    return bool(SHORT_REFERENCE_PATTERN.match(text)) and not _is_definition_like(payload)


def _token_overlap_score(query_tokens: set[str], payload: dict[str, object]) -> float:
    if not query_tokens:
        return 0.0
    haystacks = " ".join(
        value
        for value in (
            _payload_text(payload, "symbol_name"),
            _payload_text(payload, "parent_name"),
            _payload_text(payload, "path"),
            _payload_text(payload, "snippet"),
        )
        if value
    ).lower()
    if not haystacks:
        return 0.0
    overlap = sum(1 for token in query_tokens if token in haystacks)
    return min(overlap, 4) * 0.02


def _rerank_score(
    *,
    query_text: str,
    intent: str,
    payload: dict[str, object],
    base_score: float,
    duplicate_index: int,
) -> float:
    query_tokens = _query_tokens(query_text)
    path = _payload_text(payload, "path")
    score = base_score
    score += _token_overlap_score(query_tokens, payload)
    score += _implementation_path_boost(path)

    is_definition = _is_definition_like(payload)
    is_short_reference = _is_short_reference_like(payload)
    is_test = _is_test_path(path)
    is_config = _is_config_path(path)
    is_implementation_symbol = _is_implementation_symbol_kind(payload)
    is_non_implementation_symbol = _is_non_implementation_symbol_kind(payload)

    if intent == IMPLEMENTATION_INTENT:
        if is_definition:
            score += 0.18
        if is_implementation_symbol:
            score += 0.08
        if is_non_implementation_symbol:
            score -= 0.08
        if is_test:
            score -= 0.18
        if is_config:
            score -= 0.05
        if is_short_reference:
            score -= 0.12
    elif intent == REFERENCE_INTENT:
        if is_short_reference:
            score += 0.08
        if is_definition:
            score -= 0.03
        if is_test:
            score -= 0.02
    elif intent == TEST_INTENT:
        if is_test:
            score += 0.12
        elif is_definition:
            score -= 0.04
    elif intent == CONFIG_INTENT:
        if is_config:
            score += 0.12
        if is_test:
            score -= 0.02

    if duplicate_index > 0:
        score -= min(duplicate_index, 3) * 0.03

    return score


def _rerank_results(
    *,
    query_text: str,
    combined: list[tuple[str, SearchResult]],
) -> list[tuple[str, SearchResult, float]]:
    intent = _classify_intent(query_text)
    path_seen: dict[str, int] = {}
    reranked: list[tuple[str, SearchResult, float]] = []
    for collection_name, result in combined:
        path = _payload_text(result.payload, "path")
        duplicate_index = path_seen.get(path, 0)
        reranked.append(
            (
                collection_name,
                result,
                _rerank_score(
                    query_text=query_text,
                    intent=intent,
                    payload=result.payload,
                    base_score=result.score,
                    duplicate_index=duplicate_index,
                ),
            )
        )
        if path:
            path_seen[path] = duplicate_index + 1

    reranked.sort(key=lambda item: (item[2], item[1].score), reverse=True)
    return reranked


def _result_to_response(org_id: str, collection_name: str, result: SearchResult) -> QueryResultResponse:
    payload = result.payload
    repo_id = payload.get("repo_id")
    if not isinstance(repo_id, str) or not repo_id:
        repo_id = repo_id_from_collection(org_id, collection_name) or "unknown"
    page_value = payload.get("page")
    start_line = payload.get("start_line")
    end_line = payload.get("end_line")
    return QueryResultResponse(
        id=result.id,
        score=result.score,
        repo_id=repo_id,
        project=payload.get("project") if isinstance(payload.get("project"), str) else None,
        language=payload.get("language") if isinstance(payload.get("language"), str) else None,
        path=payload.get("path") if isinstance(payload.get("path"), str) else None,
        snippet=payload.get("snippet") if isinstance(payload.get("snippet"), str) else None,
        content=payload.get("content") if isinstance(payload.get("content"), str) else None,
        page=page_value if isinstance(page_value, int) else None,
        start_line=start_line if isinstance(start_line, int) else None,
        end_line=end_line if isinstance(end_line, int) else None,
    )


@router.post("", response_model=QueryResponse)
async def query_collections(
    payload: QueryRequest,
    context: OrgContext = Depends(require_auth),
    vector_store: VectorStore = Depends(get_vector_store),
    ollama: OllamaService = Depends(get_ollama_service),
) -> QueryResponse:
    repo_id = payload.repo_id
    if repo_id is not None:
        try:
            repo_id = validate_repo_id(repo_id)
        except ValueError as error:
            raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(error)) from error

    collections = (
        [collection_name_for(context.org_id, repo_id)]
        if repo_id
        else [
            collection_name
            for collection_name in await vector_store.list_collections()
            if repo_id_from_collection(context.org_id, collection_name) is not None
        ]
    )
    collections.sort()
    if not collections:
        return QueryResponse(
            data=[],
            total_count=0,
            page=payload.page,
            page_size=payload.page_size,
            has_more=False,
        )

    try:
        embed_response = await ollama.embed(payload.query_text)
    except OllamaError as error:
        raise HTTPException(
            status_code=status.HTTP_502_BAD_GATEWAY,
            detail=str(error),
        ) from error
    filters = _build_filters(payload)
    limit = _candidate_limit(payload)

    combined: list[tuple[str, SearchResult]] = []
    for collection_name in collections:
        results = await vector_store.query(
            collection=collection_name,
            vector=embed_response.embedding,
            filters=filters,
            top_k=limit,
        )
        combined.extend((collection_name, result) for result in results)

    reranked = _rerank_results(query_text=payload.query_text, combined=combined)
    offset = (payload.page - 1) * payload.page_size
    page_items = reranked[offset : offset + payload.page_size]
    data = [
        _result_to_response(context.org_id, collection_name, result)
        for collection_name, result, _ in page_items
    ]
    return QueryResponse(
        data=data,
        total_count=len(reranked),
        page=payload.page,
        page_size=payload.page_size,
        has_more=offset + payload.page_size < len(reranked),
    )
