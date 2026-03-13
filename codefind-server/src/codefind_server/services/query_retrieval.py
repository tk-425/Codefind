from __future__ import annotations

from ..adapters.base import SearchResult, VectorStore


MAX_SEMANTIC_CANDIDATES = 100
MAX_LEXICAL_CANDIDATES = 60


def semantic_candidate_limit(*, page_size: int, top_k: int) -> int:
    requested = max(page_size * 5, top_k * 3, 30)
    return min(requested, MAX_SEMANTIC_CANDIDATES)


def lexical_candidate_limit(*, page_size: int, top_k: int) -> int:
    requested = max(page_size * 3, top_k * 2, 20)
    return min(requested, MAX_LEXICAL_CANDIDATES)


async def retrieve_candidates(
    *,
    vector_store: VectorStore,
    collections: list[str],
    query_text: str,
    semantic_vector: list[float],
    filters: dict[str, object],
    page_size: int,
    top_k: int,
) -> list[tuple[str, SearchResult]]:
    semantic_limit = semantic_candidate_limit(page_size=page_size, top_k=top_k)
    lexical_limit = lexical_candidate_limit(page_size=page_size, top_k=top_k)

    combined: dict[tuple[str, str], tuple[str, SearchResult]] = {}
    for collection_name in collections:
        semantic_results = await vector_store.query(
            collection=collection_name,
            vector=semantic_vector,
            filters=filters,
            top_k=semantic_limit,
        )
        lexical_results = await vector_store.query_lexical(
            collection=collection_name,
            query_text=query_text,
            filters=filters,
            top_k=lexical_limit,
        )
        tagged_results = [(result, "semantic") for result in semantic_results]
        tagged_results.extend((result, "lexical") for result in lexical_results)
        for result, source in tagged_results:
            key = (collection_name, result.id)
            payload = dict(result.payload)
            existing = combined.get(key)
            if existing is None:
                payload["_retrieval_sources"] = [source]
                combined[key] = (
                    collection_name,
                    SearchResult(id=result.id, score=result.score, payload=payload),
                )
                continue

            _, current = existing
            merged_payload = dict(current.payload)
            merged_sources = merged_payload.get("_retrieval_sources")
            if not isinstance(merged_sources, list):
                merged_sources = []
            if source not in merged_sources:
                merged_sources.append(source)
            merged_payload["_retrieval_sources"] = merged_sources

            for field, value in payload.items():
                if field not in merged_payload or merged_payload[field] in (None, ""):
                    merged_payload[field] = value

            merged_score = max(current.score, result.score)
            combined[key] = (
                collection_name,
                SearchResult(id=current.id, score=merged_score, payload=merged_payload),
            )

    return list(combined.values())
