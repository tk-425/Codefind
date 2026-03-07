from __future__ import annotations

from typing import Any

from qdrant_client import AsyncQdrantClient, models

from .base import SearchResult, VectorPoint, VectorStore


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
        await self._client.upsert(
            collection_name=collection,
            points=[
                models.PointStruct(
                    id=point.id,
                    vector=point.vector,
                    payload=point.payload,
                )
                for point in points
            ],
        )

    async def query(
        self,
        collection: str,
        vector: list[float],
        filters: dict[str, object],
        top_k: int,
    ) -> list[SearchResult]:
        response = await self._client.query_points(
            collection_name=collection,
            query=vector,
            query_filter=self._build_filter(filters),
            limit=top_k,
        )
        return [
            SearchResult(
                id=str(point.id),
                score=point.score,
                payload=point.payload or {},
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

    async def close(self) -> None:
        await self._client.close()

    def _build_filter(
        self,
        filters: dict[str, Any],
    ) -> models.Filter | None:
        if not filters:
            return None
        conditions = [
            models.FieldCondition(
                key=key,
                match=models.MatchValue(value=value),
            )
            for key, value in filters.items()
        ]
        return models.Filter(must=conditions)
