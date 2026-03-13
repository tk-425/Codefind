from __future__ import annotations

import re
from typing import Any

from qdrant_client import AsyncQdrantClient, models

from .base import SearchResult, StoredPoint, VectorPoint, VectorStore

TEXT_QUERY_TOKEN_PATTERN = re.compile(r"[a-z0-9_]+")
TEXT_INDEX_FIELDS = ("content", "symbol_name", "path")
TEXT_INDEX_PARAMS = models.TextIndexParams(
    type=models.TextIndexType.TEXT,
    tokenizer=models.TokenizerType.WORD,
    lowercase=True,
)


class QdrantAdapter(VectorStore):
    def __init__(self, url: str) -> None:
        self._client = AsyncQdrantClient(url=url)
        self._text_indexes_ensured: set[str] = set()

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

    async def ensure_collection(self, collection: str, vector_size: int) -> None:
        if await self._client.collection_exists(collection):
            await self._ensure_text_indexes(collection)
            return
        await self._client.create_collection(
            collection_name=collection,
            vectors_config=models.VectorParams(
                size=vector_size,
                distance=models.Distance.COSINE,
            ),
        )
        await self._ensure_text_indexes(collection)

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

    async def query_lexical(
        self,
        collection: str,
        query_text: str,
        filters: dict[str, object],
        top_k: int,
    ) -> list[SearchResult]:
        if not query_text.strip():
            return []

        await self._ensure_text_indexes(collection)

        results_by_id: dict[str, SearchResult] = {}
        per_field_limit = max(top_k * 3, 20)
        for field in TEXT_INDEX_FIELDS:
            points, _ = await self._client.scroll(
                collection_name=collection,
                scroll_filter=self._build_filter(filters, text_match=(field, query_text)),
                limit=per_field_limit,
                with_payload=True,
                with_vectors=False,
            )
            for point in points:
                payload = point.payload or {}
                score = self._lexical_score(query_text=query_text, payload=payload, matched_field=field)
                if score <= 0:
                    continue
                point_id = str(point.id)
                existing = results_by_id.get(point_id)
                if existing is None or score > existing.score:
                    results_by_id[point_id] = SearchResult(id=point_id, score=score, payload=payload)

        return sorted(results_by_id.values(), key=lambda item: item.score, reverse=True)[:top_k]

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

    async def _ensure_text_indexes(self, collection: str) -> None:
        if collection in self._text_indexes_ensured:
            return
        for field_name in TEXT_INDEX_FIELDS:
            await self._client.create_payload_index(
                collection_name=collection,
                field_name=field_name,
                field_schema=TEXT_INDEX_PARAMS,
                wait=True,
            )
        self._text_indexes_ensured.add(collection)

    def _build_filter(
        self,
        filters: dict[str, Any],
        text_match: tuple[str, str] | None = None,
    ) -> models.Filter | None:
        if not filters and text_match is None:
            return None
        conditions = [
            models.FieldCondition(
                key=key,
                match=models.MatchValue(value=value),
            )
            for key, value in filters.items()
        ]
        if text_match is not None:
            field_name, query_text = text_match
            conditions.append(
                models.FieldCondition(
                    key=field_name,
                    match=models.MatchText(text=query_text),
                )
            )
        return models.Filter(must=conditions)

    def _lexical_score(
        self,
        *,
        query_text: str,
        payload: dict[str, Any],
        matched_field: str,
    ) -> float:
        query_lower = query_text.lower()
        query_tokens = self._text_tokens(query_text)
        if not query_tokens:
            return 0.0

        field_text = payload.get(matched_field)
        if not isinstance(field_text, str) or not field_text.strip():
            return 0.0
        field_lower = field_text.lower()
        field_tokens = self._text_tokens(field_text)

        overlap = len(query_tokens & field_tokens)
        if overlap == 0 and query_lower not in field_lower:
            return 0.0

        score = 0.2
        if query_lower == field_lower:
            score += 0.45
        elif query_lower in field_lower:
            score += 0.28

        score += min(overlap, 4) * 0.08

        symbol_name = payload.get("symbol_name")
        if isinstance(symbol_name, str):
            symbol_lower = symbol_name.lower()
            symbol_tokens = self._text_tokens(symbol_name)
            if query_lower == symbol_lower:
                score += 0.18
            elif query_tokens & symbol_tokens:
                score += 0.1

        if matched_field == "symbol_name":
            score += 0.08
        elif matched_field == "path":
            score += 0.04

        return min(score, 0.99)

    def _text_tokens(self, text: str) -> set[str]:
        return {match.group(0) for match in TEXT_QUERY_TOKEN_PATTERN.finditer(text.lower())}
