from __future__ import annotations

import re

from ..adapters.base import SearchResult


IMPLEMENTATION_INTENT = "implementation"
REFERENCE_INTENT = "reference"
TEST_INTENT = "test"
CONFIG_INTENT = "config"

QUERY_TOKEN_PATTERN = re.compile(r"[a-z0-9_]+")
QUERY_IDENTIFIER_PATTERN = re.compile(r"\b[A-Za-z_][A-Za-z0-9_]*\b")
CAMEL_CASE_BOUNDARY_PATTERN = re.compile(r"([a-z0-9])([A-Z])")
SHORT_REFERENCE_PATTERN = re.compile(r"^\s*[A-Za-z_][A-Za-z0-9_]*\s*=\s*[A-Za-z_][A-Za-z0-9_.]*\s*$")
COMMAND_BUILDER_SYMBOL_PATTERN = re.compile(r"^new[A-Z0-9].*Command$")
SYMBOL_PART_SUFFIX_PATTERN = re.compile(r"\s+\(part\s+\d+\)$", re.IGNORECASE)
TEST_SNIPPET_PATTERN = re.compile(r"\b(assert|expect|pytest|t\.parallel|testing\.t|unittest|mock)\b", re.IGNORECASE)
EVAL_TEST_SNIPPET_PATTERN = re.compile(r"\b(rerank_results|combined=\[|query_text=|assert)\b", re.IGNORECASE)
DECLARATION_PATTERNS = (
    re.compile(r"^\s*func\s+[A-Za-z0-9_]+"),
    re.compile(r"^\s*def\s+[A-Za-z0-9_]+"),
    re.compile(r"^\s*async\s+def\s+[A-Za-z0-9_]+"),
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
ERRORLIKE_SYMBOL_KINDS = {"class", "exception", "error"}
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
OPERATIONAL_FAMILIES = {
    "maintenance": {"cleanup", "purge", "delete", "remove", "clear", "prune", "tombstone"},
}
RANKING_TOKENS = {"rank", "ranking", "rerank", "reranker"}


def _query_tokens(text: str) -> set[str]:
    return {match.group(0) for match in QUERY_TOKEN_PATTERN.finditer(text.lower())}


def _query_identifiers(text: str) -> set[str]:
    identifiers: set[str] = set()
    for match in QUERY_IDENTIFIER_PATTERN.finditer(text):
        value = match.group(0)
        lowered = value.lower()
        if lowered in QUERY_STOPWORDS:
            continue
        if "_" in value or any(char.isupper() for char in value[1:]) or len(value) >= 10:
            identifiers.add(value)
    return identifiers


def _scoring_tokens(text: str) -> set[str]:
    normalized = CAMEL_CASE_BOUNDARY_PATTERN.sub(r"\1 \2", text)
    raw_tokens = _query_tokens(normalized)
    expanded_tokens: set[str] = set()
    for token in raw_tokens:
        expanded_tokens.add(token)
        if token.endswith("s") and len(token) > 4:
            expanded_tokens.add(token[:-1])
        if "_" in token:
            parts = {part for part in token.split("_") if part}
            expanded_tokens.update(parts)
            expanded_tokens.update(part[:-1] for part in parts if part.endswith("s") and len(part) > 4)
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


def _is_test_like(payload: dict[str, object]) -> bool:
    path = _payload_text(payload, "path")
    if _is_test_path(path):
        return True
    snippet = _payload_text(payload, "snippet") or _payload_text(payload, "content")
    symbol_name = _payload_text(payload, "symbol_name").lower()
    return bool(TEST_SNIPPET_PATTERN.search(snippet)) or symbol_name.startswith("test")


def _is_eval_like_test(payload: dict[str, object]) -> bool:
    if not _is_test_like(payload):
        return False
    snippet = _payload_text(payload, "snippet") or _payload_text(payload, "content")
    path = _payload_text(payload, "path").lower()
    return bool(EVAL_TEST_SNIPPET_PATTERN.search(snippet)) or "ranking" in path or "query" in path


def _exact_symbol_match_score(query_tokens: set[str], payload: dict[str, object]) -> float:
    symbol_name = _normalized_symbol_name(_payload_text(payload, "symbol_name"))
    if not symbol_name or not query_tokens:
        return 0.0
    symbol_tokens = _scoring_tokens(symbol_name)
    if not symbol_tokens:
        return 0.0
    overlap = len(query_tokens & symbol_tokens)
    if overlap == 0:
        return 0.0
    return min(overlap, 3) * 0.05


def _is_definition_like(payload: dict[str, object]) -> bool:
    snippet = _payload_text(payload, "snippet")
    content = _payload_text(payload, "content")
    text = snippet or content
    if not text:
        return False
    candidate_lines = [line.strip() for line in text.splitlines() if line.strip()]
    for line in candidate_lines[:3]:
        if line.startswith("@"):
            continue
        if any(pattern.search(line) for pattern in DECLARATION_PATTERNS):
            return True
    return False


def _symbol_kind(payload: dict[str, object]) -> str:
    return _payload_text(payload, "symbol_kind").lower()


def _normalized_symbol_name(symbol_name: str) -> str:
    return SYMBOL_PART_SUFFIX_PATTERN.sub("", symbol_name).strip()


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


def _operational_family_hits(tokens: set[str]) -> dict[str, set[str]]:
    return {
        family: tokens & variants
        for family, variants in OPERATIONAL_FAMILIES.items()
        if tokens & variants
    }


def _candidate_tokens(payload: dict[str, object]) -> set[str]:
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
    return _scoring_tokens(haystack_text)


def _is_ranking_internal(payload: dict[str, object]) -> bool:
    path = _payload_text(payload, "path").lower()
    symbol_name = _payload_text(payload, "symbol_name").lower()
    snippet = (_payload_text(payload, "snippet") or _payload_text(payload, "content")).lower()
    return (
        symbol_name.startswith("_")
        and ("ranking" in path or "rerank" in symbol_name or "rerank_results" in snippet)
    )


def _is_symbol_chunk_part(payload: dict[str, object]) -> bool:
    symbol_name = _payload_text(payload, "symbol_name")
    return bool(SYMBOL_PART_SUFFIX_PATTERN.search(symbol_name))


def _is_command_builder(payload: dict[str, object]) -> bool:
    symbol_name = _payload_text(payload, "symbol_name")
    snippet = _payload_text(payload, "snippet")
    return bool(COMMAND_BUILDER_SYMBOL_PATTERN.match(symbol_name)) or "cobra.command" in snippet.lower()


def _looks_like_wrapper(payload: dict[str, object]) -> bool:
    if _is_command_builder(payload):
        return True
    snippet = (_payload_text(payload, "snippet") or _payload_text(payload, "content")).strip().lower()
    if not snippet:
        return False
    if "@router." in snippet or "depends(" in snippet:
        return True
    if "return await " in snippet or "return " in snippet:
        return snippet.count("(") <= 3 and snippet.count("\n") <= 3
    return False


def _is_route_wrapper(payload: dict[str, object]) -> bool:
    snippet = (_payload_text(payload, "snippet") or _payload_text(payload, "content")).lower()
    return "@router." in snippet or "depends(" in snippet


def _is_pass_through_wrapper(payload: dict[str, object]) -> bool:
    snippet = (_payload_text(payload, "snippet") or _payload_text(payload, "content")).lower()
    if not snippet:
        return False
    return (
        "return await " in snippet
        or "\n    return response" in snippet
        or ".purge_" in snippet
        or ".delete_" in snippet
        or ".remove_" in snippet
    )


def _wrapper_penalty(payload: dict[str, object]) -> float:
    if not _looks_like_wrapper(payload):
        return 0.0
    if _is_command_builder(payload):
        return -0.18
    if _is_route_wrapper(payload):
        return -0.16
    if _is_pass_through_wrapper(payload):
        return -0.16
    return -0.08


def _implementation_depth_boost(payload: dict[str, object]) -> float:
    if _symbol_kind(payload) not in FUNCTION_SYMBOL_KINDS:
        return 0.0
    snippet = _payload_text(payload, "snippet") or _payload_text(payload, "content")
    if not snippet:
        return 0.0
    line_count = len([line for line in snippet.splitlines() if line.strip()])
    if line_count < 6:
        return 0.0
    if _looks_like_wrapper(payload):
        return 0.0
    boost = 0.10
    if any(token in snippet.lower() for token in ("for ", "while ", "if ", "await ", "append(", "delete(", "return ")):
        boost += 0.06
    return boost


def _implementation_chunk_body_boost(query_tokens: set[str], payload: dict[str, object]) -> float:
    if _symbol_kind(payload) not in FUNCTION_SYMBOL_KINDS:
        return 0.0
    if not _is_symbol_chunk_part(payload):
        return 0.0
    if _looks_like_wrapper(payload):
        return 0.0

    snippet = _payload_text(payload, "snippet") or _payload_text(payload, "content")
    if not snippet:
        return 0.0
    line_count = len([line for line in snippet.splitlines() if line.strip()])
    if line_count < 5:
        return 0.0

    candidate_tokens = _candidate_tokens(payload)
    overlap = len(query_tokens & candidate_tokens)
    boost = 0.08
    if overlap > 0:
        boost += min(overlap, 2) * 0.04
    if any(token in snippet.lower() for token in ("for ", "while ", "if ", "await ", "append(", "delete(", "continue", "matching_")):
        boost += 0.06
    return boost


def _implementation_test_penalty(query_tokens: set[str], payload: dict[str, object]) -> float:
    if not _is_test_like(payload):
        return 0.0
    snippet_tokens = _scoring_tokens(_payload_text(payload, "snippet"))
    overlap = len(query_tokens & snippet_tokens)
    penalty = -0.18
    if overlap > 0:
        penalty -= min(overlap, 2) * 0.06
    if _is_eval_like_test(payload):
        penalty -= 0.10
    snippet = (_payload_text(payload, "snippet") or _payload_text(payload, "content")).lower()
    if "self." in snippet and ("append(" in snippet or "return {" in snippet or "return {" in snippet):
        penalty -= 0.14
    return penalty


def _operational_implementation_boost(query_tokens: set[str], payload: dict[str, object]) -> float:
    if _symbol_kind(payload) not in FUNCTION_SYMBOL_KINDS or not _is_definition_like(payload):
        return 0.0

    query_families = _operational_family_hits(query_tokens)
    if not query_families:
        return 0.0

    candidate_tokens = _candidate_tokens(payload)
    candidate_families = _operational_family_hits(candidate_tokens)
    shared_families = set(query_families) & set(candidate_families)
    if not shared_families:
        return 0.0

    family_terms = set().union(*query_families.values(), *candidate_families.values())
    concrete_query_terms = query_tokens - family_terms
    concrete_candidate_terms = candidate_tokens - family_terms
    concrete_overlap = concrete_query_terms & concrete_candidate_terms
    if not concrete_overlap:
        return 0.0

    boost = 0.24
    penalty = _wrapper_penalty(payload)
    if penalty < 0:
        boost += penalty
    return boost


def _command_builder_implementation_penalty(query_tokens: set[str], payload: dict[str, object]) -> float:
    if not _is_command_builder(payload) or not _is_definition_like(payload):
        return 0.0

    query_families = _operational_family_hits(query_tokens)
    if not query_families:
        return 0.0

    candidate_tokens = _candidate_tokens(payload)
    candidate_families = _operational_family_hits(candidate_tokens)
    shared_families = set(query_families) & set(candidate_families)
    if not shared_families:
        return 0.0

    family_terms = set().union(*query_families.values(), *candidate_families.values())
    concrete_query_terms = query_tokens - family_terms
    concrete_candidate_terms = candidate_tokens - family_terms
    if concrete_query_terms & concrete_candidate_terms:
        return 0.0

    return -0.26


def _usage_site_boost(query_tokens: set[str], payload: dict[str, object]) -> float:
    if not query_tokens:
        return 0.0
    snippet_tokens = _scoring_tokens(_payload_text(payload, "snippet"))
    if not snippet_tokens:
        return 0.0
    overlap = len(query_tokens & snippet_tokens)
    if overlap == 0:
        return 0.0
    boost = min(overlap, 3) * 0.08
    if not _is_definition_like(payload):
        boost += 0.08
    if not _is_test_like(payload):
        boost += 0.04
    return boost


def _config_surface_boost(query_tokens: set[str], payload: dict[str, object]) -> float:
    if not query_tokens:
        return 0.0
    snippet_tokens = _scoring_tokens(
        " ".join(
            value
            for value in (
                _payload_text(payload, "symbol_name"),
                _payload_text(payload, "path"),
                _payload_text(payload, "snippet"),
            )
            if value
        )
    )
    overlap = len(query_tokens & snippet_tokens)
    if overlap == 0:
        return 0.0
    boost = 0.0
    if _is_env_config_like(payload):
        boost += 0.12
    if _is_config_value_like(payload):
        boost += 0.10
    if _is_config_path(_payload_text(payload, "path")):
        boost += 0.08
    if _payload_text(payload, "symbol_name").lower() in {"get_settings", "settings"}:
        boost += 0.06
    if overlap >= 2:
        boost += 0.06
    return boost


def _runtime_config_boost(query_tokens: set[str], payload: dict[str, object]) -> float:
    if not query_tokens:
        return 0.0
    snippet = (_payload_text(payload, "snippet") or _payload_text(payload, "content")).lower()
    symbol_name = _payload_text(payload, "symbol_name").lower()
    haystack_tokens = _scoring_tokens(" ".join((_payload_text(payload, "path"), symbol_name, snippet)))
    overlap = len(query_tokens & haystack_tokens)
    if overlap == 0:
        return 0.0

    boost = 0.0
    config_markers = ("getenv(", "retry", "backoff", "attempt", "timeout", "settings")
    if any(marker in snippet for marker in config_markers):
        boost += 0.10
    if any(marker in symbol_name for marker in ("retry", "backoff", "attempt", "timeout")):
        boost += 0.08
    if _is_env_config_like(payload):
        boost += 0.06
    if overlap >= 2:
        boost += 0.06
    return boost


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


def _errorlike_implementation_penalty(payload: dict[str, object]) -> float:
    kind = _symbol_kind(payload)
    if kind not in ERRORLIKE_SYMBOL_KINDS:
        return 0.0
    symbol_name = _payload_text(payload, "symbol_name").lower()
    snippet = (_payload_text(payload, "snippet") or _payload_text(payload, "content")).lower()
    if "error" in symbol_name or "exception" in symbol_name or "error" in snippet or "exception" in snippet:
        return -0.16
    return 0.0


def _strong_function_overlap_boost(query_tokens: set[str], payload: dict[str, object]) -> float:
    if _symbol_kind(payload) not in FUNCTION_SYMBOL_KINDS:
        return 0.0
    if not _is_definition_like(payload) and not _is_symbol_chunk_part(payload):
        return 0.0
    symbol_tokens = _scoring_tokens(_normalized_symbol_name(_payload_text(payload, "symbol_name")))
    if not symbol_tokens or not query_tokens:
        return 0.0
    overlap = len(query_tokens & symbol_tokens)
    if overlap == 0:
        return 0.0
    return min(overlap, 2) * 0.05


def _definition_alias_boost(
    query_tokens: set[str],
    query_identifiers: set[str],
    payload: dict[str, object],
    *,
    definition_query: bool,
) -> float:
    if not definition_query or not _is_short_reference_like(payload) or _is_test_like(payload):
        return 0.0
    symbol_tokens = _scoring_tokens(_normalized_symbol_name(_payload_text(payload, "symbol_name")))
    snippet_tokens = _scoring_tokens(_payload_text(payload, "snippet"))
    haystack_tokens = symbol_tokens | snippet_tokens
    if not haystack_tokens:
        return 0.0
    overlap = len(query_tokens & haystack_tokens)
    if overlap == 0:
        return 0.0
    boost = 0.16 + min(overlap, 3) * 0.06
    haystack_text = " ".join(
        value
        for value in (
            _normalized_symbol_name(_payload_text(payload, "symbol_name")),
            _payload_text(payload, "snippet"),
        )
        if value
    )
    lowered_haystack = haystack_text.lower()
    if any(identifier.lower() in lowered_haystack for identifier in query_identifiers):
        boost += 0.18
    return boost


def _ranking_internal_penalty(query_tokens: set[str], payload: dict[str, object], *, definition_query: bool) -> float:
    if definition_query:
        return 0.0
    if query_tokens & RANKING_TOKENS:
        return 0.0
    if not _is_ranking_internal(payload):
        return 0.0
    return -0.52


def _rerank_score(
    *,
    query_text: str,
    intent: str,
    payload: dict[str, object],
    base_score: float,
    duplicate_index: int,
) -> float:
    query_tokens = _scoring_tokens(query_text)
    query_identifiers = _query_identifiers(query_text)
    lowered_query = query_text.lower()
    definition_query = "defined" in lowered_query or "definition" in lowered_query
    path = _payload_text(payload, "path")
    score = base_score
    score += _token_overlap_score(query_tokens, payload)
    score += _identifier_overlap_score(query_tokens, payload)
    score += _exact_symbol_match_score(query_tokens, payload)
    score += _symbol_kind_score(payload, intent)

    is_definition = _is_definition_like(payload)
    is_short_reference = _is_short_reference_like(payload)
    is_test = _is_test_like(payload)
    is_config = _is_config_path(path)
    is_env_config = _is_env_config_like(payload)
    is_config_value = _is_config_value_like(payload)

    if intent == IMPLEMENTATION_INTENT:
        implementation_specific_query = "implemented" in lowered_query or bool(_operational_family_hits(query_tokens))
        if is_definition:
            score += 0.18
        if is_test:
            score -= 0.62
            if _is_eval_like_test(payload):
                score -= 0.18
            if implementation_specific_query:
                score += _implementation_test_penalty(query_tokens, payload)
        if is_config:
            score -= 0.05
        if is_short_reference:
            score -= 0.18
            score += _definition_alias_boost(
                query_tokens,
                query_identifiers,
                payload,
                definition_query=definition_query,
            )
        if "function" in query_text.lower():
            kind = _symbol_kind(payload)
            if kind in FUNCTION_SYMBOL_KINDS:
                score += 0.08
            elif kind in CLASSLIKE_SYMBOL_KINDS:
                score -= 0.08
        score += _ranking_internal_penalty(query_tokens, payload, definition_query=definition_query)
        score += _errorlike_implementation_penalty(payload)
        score += _strong_function_overlap_boost(query_tokens, payload)
        score += _operational_implementation_boost(query_tokens, payload)
        score += _implementation_depth_boost(payload)
        score += _implementation_chunk_body_boost(query_tokens, payload)
        score += _command_builder_implementation_penalty(query_tokens, payload)
        if implementation_specific_query:
            score += _wrapper_penalty(payload)
        if _symbol_kind(payload) in CLASSLIKE_SYMBOL_KINDS and "implemented" in query_text.lower():
            score -= 0.08
    elif intent == REFERENCE_INTENT:
        if is_short_reference:
            score += 0.08
        if is_definition:
            score -= 0.16
        if is_test:
            score -= 0.78
            if _is_eval_like_test(payload):
                score -= 0.16
        if not is_test and not is_definition:
            score += 0.06
        score += _usage_site_boost(query_tokens, payload)
        symbol_name = _payload_text(payload, "symbol_name")
        symbol_tokens = _scoring_tokens(symbol_name)
        if query_tokens and symbol_tokens and query_tokens == symbol_tokens and is_definition:
            score -= 0.08
        if not is_test and query_tokens and not query_tokens.issubset(symbol_tokens):
            snippet_tokens = _scoring_tokens(_payload_text(payload, "snippet"))
            if query_tokens & snippet_tokens:
                score += 0.14
    elif intent == TEST_INTENT:
        if is_test:
            score += 0.16
        else:
            score -= 0.2
        if is_definition:
            score -= 0.04
        if is_short_reference:
            score -= 0.12
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
            score -= 0.42
            if _is_eval_like_test(payload):
                score -= 0.34
        score += _config_surface_boost(query_tokens, payload)
        score += _runtime_config_boost(query_tokens, payload)

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
