from __future__ import annotations

from typing import Any

from qdrant_client import AsyncQdrantClient, models

from .base import HybridQuery, SearchResult, StoredPoint, VectorPoint, VectorStore

DENSE_VECTOR_NAME = "dense"
SPARSE_VECTOR_NAME = "sparse"


class CollectionSchemaError(RuntimeError):
    """Raised when an existing Qdrant collection does not match the expected hybrid schema."""


class QdrantAdapter(VectorStore):
    def __init__(self, url: str) -> None:
        self._client = AsyncQdrantClient(url=url)

    async def healthcheck(self) -> bool:
        try:
            await self._client.get_collections()
        except Exception:
            return False
        return True

    async def upsert(self, collection: str, points: list[VectorPoint]) -> None:
        if not points:
            return
        for point in points:
            if point.sparse_vector is None:
                raise CollectionSchemaError("sparse vector is required for hybrid retrieval upserts")
        await self._client.upsert(
            collection_name=collection,
            points=[
                models.PointStruct(
                    id=point.id,
                    vector={
                        DENSE_VECTOR_NAME: point.dense_vector,
                        SPARSE_VECTOR_NAME: models.SparseVector(
                            indices=point.sparse_vector.indices,
                            values=point.sparse_vector.values,
                        ),
                    },
                    payload=point.payload,
                )
                for point in points
            ],
        )

    async def ensure_collection(self, collection: str, vector_size: int) -> None:
        if await self._client.collection_exists(collection):
            await self._validate_collection_schema(collection=collection, vector_size=vector_size)
            return

        await self._client.create_collection(
            collection_name=collection,
            vectors_config={
                DENSE_VECTOR_NAME: models.VectorParams(
                    size=vector_size,
                    distance=models.Distance.COSINE,
                )
            },
            sparse_vectors_config={
                SPARSE_VECTOR_NAME: models.SparseVectorParams()
            },
        )

    async def query(self, collection: str, query: HybridQuery) -> list[SearchResult]:
        response = await self._client.query_points(
            collection_name=collection,
            prefetch=[
                models.Prefetch(
                    query=query.dense_vector,
                    using=DENSE_VECTOR_NAME,
                    limit=query.dense_top_k,
                    filter=self._build_filter(query.filters),
                ),
                models.Prefetch(
                    query=models.SparseVector(
                        indices=query.sparse_vector.indices,
                        values=query.sparse_vector.values,
                    ),
                    using=SPARSE_VECTOR_NAME,
                    limit=query.sparse_top_k,
                    filter=self._build_filter(query.filters),
                ),
            ],
            query=models.FusionQuery(fusion=models.Fusion.RRF),
            limit=query.top_k,
            with_payload=True,
            with_vectors=False,
        )
        return [
            SearchResult(
                id=str(point.id),
                score=point.score,
                payload={
                    **(point.payload or {}),
                    "_retrieval_sources": [DENSE_VECTOR_NAME, SPARSE_VECTOR_NAME],
                },
            )
            for point in response.points
        ]

    async def update_payload(
        self,
        collection: str,
        ids: list[str],
        payload: dict[str, object],
    ) -> None:
        if not ids:
            return
        await self._client.set_payload(
            collection_name=collection,
            payload=payload,
            points=ids,
        )

    async def delete(self, collection: str, ids: list[str]) -> None:
        if not ids:
            return
        await self._client.delete(
            collection_name=collection,
            points_selector=ids,
        )

    async def delete_collection(self, collection: str) -> None:
        if await self._client.collection_exists(collection):
            await self._client.delete_collection(collection)

    async def list_collections(self) -> list[str]:
        collections = await self._client.get_collections()
        return [collection.name for collection in collections.collections]

    async def count(self, collection: str, filters: dict[str, Any]) -> int:
        response = await self._client.count(
            collection_name=collection,
            count_filter=self._build_filter(filters),
            exact=True,
        )
        return response.count

    async def scroll(
        self,
        collection: str,
        filters: dict[str, Any],
        limit: int = 1000,
    ) -> list[StoredPoint]:
        points, _ = await self._client.scroll(
            collection_name=collection,
            scroll_filter=self._build_filter(filters),
            limit=limit,
            with_payload=True,
            with_vectors=False,
        )
        return [
            StoredPoint(
                id=str(point.id),
                payload=point.payload or {},
            )
            for point in points
        ]

    async def close(self) -> None:
        await self._client.close()

    async def _validate_collection_schema(self, *, collection: str, vector_size: int) -> None:
        info = await self._client.get_collection(collection)
        params = info.config.params
        dense_vectors = params.vectors
        sparse_vectors = params.sparse_vectors

        if not isinstance(dense_vectors, dict) or DENSE_VECTOR_NAME not in dense_vectors:
            raise CollectionSchemaError(
                f"collection '{collection}' must be recreated for native hybrid retrieval: missing dense vector config"
            )
        dense_config = dense_vectors[DENSE_VECTOR_NAME]
        if dense_config.size != vector_size:
            raise CollectionSchemaError(
                f"collection '{collection}' must be recreated for native hybrid retrieval: dense vector size mismatch"
            )
        if not isinstance(sparse_vectors, dict) or SPARSE_VECTOR_NAME not in sparse_vectors:
            raise CollectionSchemaError(
                f"collection '{collection}' must be recreated for native hybrid retrieval: missing sparse vector config"
            )

    def _build_filter(self, filters: dict[str, Any]) -> models.Filter | None:
        if not filters:
            return None
        return models.Filter(
            must=[
                models.FieldCondition(
                    key=key,
                    match=models.MatchValue(value=value),
                )
                for key, value in filters.items()
            ]
        )
