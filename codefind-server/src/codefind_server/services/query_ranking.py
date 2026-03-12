from __future__ import annotations

import re

from ..adapters.base import SearchResult


IMPLEMENTATION_INTENT = "implementation"
REFERENCE_INTENT = "reference"
TEST_INTENT = "test"
CONFIG_INTENT = "config"

QUERY_TOKEN_PATTERN = re.compile(r"[a-z0-9_]+")
CAMEL_CASE_BOUNDARY_PATTERN = re.compile(r"([a-z0-9])([A-Z])")
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
NON_IMPLEMENTATION_SYMBOL_KINDS = {"variable", "constant", "property", "field", "key", "string", "number", "boolean"}
FUNCTION_SYMBOL_KINDS = {"function", "method", "constructor"}
CLASSLIKE_SYMBOL_KINDS = {"class", "interface", "module", "namespace"}
QUERY_STOPWORDS = {
    "where",
    "is",
    "the",
    "a",
    "an",
    "who",
    "for",
    "to",
    "in",
    "of",
    "implemented",
    "implementation",
    "defined",
    "configured",
    "config",
    "setup",
    "uses",
    "used",
    "calls",
    "called",
    "referenced",
    "reference",
    "function",
}


def _query_tokens(text: str) -> set[str]:
    return {match.group(0) for match in QUERY_TOKEN_PATTERN.finditer(text.lower())}


def _scoring_tokens(text: str) -> set[str]:
    normalized = CAMEL_CASE_BOUNDARY_PATTERN.sub(r"\1 \2", text)
    raw_tokens = _query_tokens(normalized)
    expanded_tokens: set[str] = set()
    for token in raw_tokens:
        expanded_tokens.add(token)
        if "_" in token:
            expanded_tokens.update(part for part in token.split("_") if part)
    return {token for token in expanded_tokens if token not in QUERY_STOPWORDS}


def classify_intent(query_text: str) -> str:
    lowered = query_text.lower()
    if any(token in lowered for token in ("test", "tests", "example", "examples")):
        return TEST_INTENT
    if any(token in lowered for token in ("config", "setup", "env", "environment", "variable")):
        return CONFIG_INTENT
    if any(token in lowered for token in ("who calls", "callers", "used by", "who uses", " uses ", " used", " referenced", "references", "reference")):
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


def _production_reference_path_boost(path: str) -> float:
    lowered = path.lower()
    if any(lowered.startswith(prefix) for prefix in ("codefind-server/src/", "internal/", "cmd/", "web/src/")):
        return 0.04
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
    haystack_text = " ".join(
        value
        for value in (
            _payload_text(payload, "symbol_name"),
            _payload_text(payload, "parent_name"),
            _payload_text(payload, "path"),
            _payload_text(payload, "snippet"),
        )
        if value
    )
    if not haystack_text:
        return 0.0
    haystack_tokens = _scoring_tokens(haystack_text)
    overlap = len(query_tokens & haystack_tokens)
    return min(overlap, 4) * 0.05


def _identifier_overlap_score(query_tokens: set[str], payload: dict[str, object]) -> float:
    if not query_tokens:
        return 0.0
    haystack_text = " ".join(
        value
        for value in (
            _payload_text(payload, "symbol_name"),
            _payload_text(payload, "parent_name"),
            _payload_text(payload, "path"),
            _payload_text(payload, "snippet"),
        )
        if value
    )
    if not haystack_text:
        return 0.0
    haystack_tokens = _scoring_tokens(haystack_text)
    overlap = sum(1 for token in query_tokens if token in haystack_tokens and (len(token) >= 8 or "_" in token))
    return min(overlap, 2) * 0.08


def _is_env_config_like(payload: dict[str, object]) -> bool:
    snippet = (_payload_text(payload, "snippet") or _payload_text(payload, "content")).lower()
    return "getenv(" in snippet or "import.meta.env" in snippet or ".env" in snippet


def _is_config_value_like(payload: dict[str, object]) -> bool:
    snippet = (_payload_text(payload, "snippet") or _payload_text(payload, "content")).strip()
    return bool(re.match(r"^[A-Z0-9_]+\s*=", snippet)) or _is_env_config_like(payload)


def _symbol_kind_score(payload: dict[str, object], intent: str) -> float:
    kind = _symbol_kind(payload)
    if intent == IMPLEMENTATION_INTENT:
        if kind in FUNCTION_SYMBOL_KINDS:
            return 0.08
        if kind in CLASSLIKE_SYMBOL_KINDS:
            return 0.02
        if kind in NON_IMPLEMENTATION_SYMBOL_KINDS:
            return -0.08
    if intent == REFERENCE_INTENT and kind in FUNCTION_SYMBOL_KINDS:
        return 0.01
    if intent == TEST_INTENT and kind in FUNCTION_SYMBOL_KINDS:
        return -0.03
    return 0.0


def _rerank_score(
    *,
    query_text: str,
    intent: str,
    payload: dict[str, object],
    base_score: float,
    duplicate_index: int,
) -> float:
    query_tokens = _scoring_tokens(query_text)
    path = _payload_text(payload, "path")
    score = base_score
    score += _token_overlap_score(query_tokens, payload)
    score += _identifier_overlap_score(query_tokens, payload)
    score += _implementation_path_boost(path)
    score += _symbol_kind_score(payload, intent)

    is_definition = _is_definition_like(payload)
    is_short_reference = _is_short_reference_like(payload)
    is_test = _is_test_path(path)
    is_config = _is_config_path(path)
    is_env_config = _is_env_config_like(payload)
    is_config_value = _is_config_value_like(payload)

    if intent == IMPLEMENTATION_INTENT:
        if is_definition:
            score += 0.18
        if is_test:
            score -= 0.36
        if is_config:
            score -= 0.05
        if is_short_reference:
            score -= 0.12
        if "function" in query_text.lower():
            kind = _symbol_kind(payload)
            if kind in FUNCTION_SYMBOL_KINDS:
                score += 0.08
            elif kind in CLASSLIKE_SYMBOL_KINDS:
                score -= 0.08
    elif intent == REFERENCE_INTENT:
        if is_short_reference:
            score += 0.08
        score += _production_reference_path_boost(path)
        if is_definition:
            score -= 0.03
        if is_test:
            score -= 0.14
    elif intent == TEST_INTENT:
        if is_test:
            score += 0.16
        else:
            score -= 0.12
        if is_definition:
            score -= 0.04
        if is_short_reference:
            score -= 0.06
    elif intent == CONFIG_INTENT:
        if is_config:
            score += 0.18
        else:
            score -= 0.14
        if is_env_config:
            score += 0.14
        if is_config_value:
            score += 0.12
        if is_test:
            score -= 0.02

    if duplicate_index > 0:
        score -= min(duplicate_index, 3) * 0.01

    return score


def rerank_results(
    *,
    query_text: str,
    combined: list[tuple[str, SearchResult]],
) -> list[tuple[str, SearchResult, float]]:
    intent = classify_intent(query_text)
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
