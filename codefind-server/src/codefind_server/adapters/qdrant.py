from __future__ import annotations

from .base import VectorPoint, VectorStore


class QdrantAdapter(VectorStore):
    async def upsert(self, collection: str, points: list[VectorPoint]) -> None:
        raise NotImplementedError("Phase 2: implement Qdrant upsert")

    async def query(
        self,
        collection: str,
        vector: list[float],
        filters: dict[str, object],
        top_k: int,
    ) -> list[dict[str, object]]:
        raise NotImplementedError("Phase 2: implement Qdrant query")

    async def update_payload(
        self,
        collection: str,
        ids: list[str],
        payload: dict[str, object],
    ) -> None:
        raise NotImplementedError("Phase 2: implement Qdrant payload updates")

    async def delete(self, collection: str, ids: list[str]) -> None:
        raise NotImplementedError("Phase 2: implement Qdrant delete")

    async def delete_collection(self, collection: str) -> None:
        raise NotImplementedError("Phase 2: implement Qdrant collection delete")

    async def list_collections(self) -> list[str]:
        raise NotImplementedError("Phase 2: implement Qdrant list collections")

    async def count(self, collection: str, filters: dict[str, object]) -> int:
        raise NotImplementedError("Phase 2: implement Qdrant count")
