from __future__ import annotations

from ..adapters.base import HybridQuery, SearchResult, SparseVectorData, VectorStore


MAX_SEMANTIC_CANDIDATES = 100
MAX_SPARSE_CANDIDATES = 60


def semantic_candidate_limit(*, page_size: int, top_k: int) -> int:
    requested = max(page_size * 5, top_k * 3, 30)
    return min(requested, MAX_SEMANTIC_CANDIDATES)


def sparse_candidate_limit(*, page_size: int, top_k: int) -> int:
    requested = max(page_size * 3, top_k * 2, 20)
    return min(requested, MAX_SPARSE_CANDIDATES)


async def retrieve_candidates(
    *,
    vector_store: VectorStore,
    collections: list[str],
    dense_vector: list[float],
    sparse_vector: SparseVectorData,
    filters: dict[str, object],
    page_size: int,
    top_k: int,
) -> list[tuple[str, SearchResult]]:
    dense_limit = semantic_candidate_limit(page_size=page_size, top_k=top_k)
    sparse_limit = sparse_candidate_limit(page_size=page_size, top_k=top_k)
    candidate_limit = max(dense_limit, sparse_limit)

    combined: list[tuple[str, SearchResult]] = []
    for collection_name in collections:
        results = await vector_store.query(
            collection=collection_name,
            query=HybridQuery(
                dense_vector=dense_vector,
                sparse_vector=sparse_vector,
                filters=filters,
                top_k=candidate_limit,
                dense_top_k=dense_limit,
                sparse_top_k=sparse_limit,
            ),
        )
        combined.extend((collection_name, result) for result in results)

    return combined
