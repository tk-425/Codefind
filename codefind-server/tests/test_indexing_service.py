from __future__ import annotations

import asyncio

from codefind_server.adapters.base import StoredPoint
from codefind_server.services.indexing import IndexingService


class DummyVectorStore:
    def __init__(self, collections: list[str]) -> None:
        self.collections = list(collections)
        self.deleted_collections: list[str] = []

    async def healthcheck(self) -> bool:
        return True

    async def upsert(self, collection: str, points) -> None:
        raise NotImplementedError

    async def ensure_collection(self, collection: str, vector_size: int) -> None:
        raise NotImplementedError

    async def query(self, collection: str, vector, filters, top_k: int):
        raise NotImplementedError

    async def update_payload(self, collection: str, ids, payload) -> None:
        raise NotImplementedError

    async def delete(self, collection: str, ids) -> None:
        raise NotImplementedError

    async def delete_collection(self, collection: str) -> None:
        self.deleted_collections.append(collection)
        self.collections = [existing for existing in self.collections if existing != collection]

    async def list_collections(self) -> list[str]:
        return list(self.collections)

    async def count(self, collection: str, filters) -> int:
        raise NotImplementedError

    async def scroll(self, collection: str, filters, limit: int = 1000) -> list[StoredPoint]:
        raise NotImplementedError


def test_clear_repo_index_only_deletes_target_repo_collection():
    vector_store = DummyVectorStore(["org_123_repo-a", "org_123_repo-b"])
    service = IndexingService(vector_store=vector_store, ollama=object())

    response = asyncio.run(service.clear_repo_index(org_id="org_123", repo_id="repo-a"))

    assert response.status == "ok"
    assert response.repo_id == "repo-a"
    assert response.cleared is True
    assert vector_store.deleted_collections == ["org_123_repo-a"]
    assert vector_store.collections == ["org_123_repo-b"]


def test_clear_repo_index_returns_not_cleared_when_collection_missing():
    vector_store = DummyVectorStore(["org_123_repo-b"])
    service = IndexingService(vector_store=vector_store, ollama=object())

    response = asyncio.run(service.clear_repo_index(org_id="org_123", repo_id="repo-a"))

    assert response.status == "ok"
    assert response.repo_id == "repo-a"
    assert response.cleared is False
    assert vector_store.deleted_collections == []
