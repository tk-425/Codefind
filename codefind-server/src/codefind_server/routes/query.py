from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, Request, status

from ..adapters.base import SearchResult, VectorStore
from ..middleware.auth import OrgContext, require_auth
from ..models.requests import QueryRequest
from ..models.responses import QueryResponse, QueryResultResponse
from ..services import OllamaService, collection_name_for, repo_id_from_collection, validate_repo_id
from ..services.ollama import OllamaError
from ..services.query_retrieval import retrieve_candidates
from ..services.query_ranking import rerank_results
from .collections import get_vector_store


router = APIRouter(prefix="/query", tags=["query"])

ALLOWED_FILTER_KEYS = {"project", "language", "status"}


def get_ollama_service(request: Request) -> OllamaService:
    return request.app.state.ollama


def _build_filters(payload: QueryRequest) -> dict[str, object]:
    filters: dict[str, object] = {"status": "active"}
    if payload.project:
        filters["project"] = payload.project
    if payload.language:
        filters["language"] = payload.language
    return {key: value for key, value in filters.items() if key in ALLOWED_FILTER_KEYS}


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
    combined = await retrieve_candidates(
        vector_store=vector_store,
        collections=collections,
        query_text=payload.query_text,
        semantic_vector=embed_response.embedding,
        filters=filters,
        page_size=payload.page_size,
        top_k=payload.top_k,
    )

    reranked = rerank_results(query_text=payload.query_text, combined=combined)
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
