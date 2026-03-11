from __future__ import annotations

import asyncio

import pytest

from codefind_server.adapters.base import StoredPoint
from codefind_server.models.requests import (
    ChunkMetadataRequest,
    IndexChunkRequest,
    IndexRequest,
)
from codefind_server.services.indexing import IndexingService
from codefind_server.services.ollama import EmbeddingResponse, OllamaError


class DummyVectorStore:
    def __init__(self, collections: list[str]) -> None:
        self.collections = list(collections)
        self.deleted_collections: list[str] = []
        self.ensure_collection_calls: list[tuple[str, int]] = []
        self.upsert_calls: list[tuple[str, list]] = []

    async def healthcheck(self) -> bool:
        return True

    async def upsert(self, collection: str, points) -> None:
        self.upsert_calls.append((collection, list(points)))

    async def ensure_collection(self, collection: str, vector_size: int) -> None:
        self.ensure_collection_calls.append((collection, vector_size))

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


class DummyOllama:
    def __init__(self, responses: list[EmbeddingResponse] | None = None, *, error: Exception | None = None) -> None:
        self.responses = responses or []
        self.error = error
        self.embed_many_calls: list[list[str]] = []
        self.response_index = 0

    async def embed_many(self, texts: list[str]) -> list[EmbeddingResponse]:
        self.embed_many_calls.append(list(texts))
        if self.error is not None:
            raise self.error
        batch_size = len(texts)
        batch = self.responses[self.response_index : self.response_index + batch_size]
        self.response_index += batch_size
        return list(batch)


def _make_index_request() -> IndexRequest:
    return IndexRequest(
        repo_id="repo-a",
        chunks=[
            IndexChunkRequest(
                id="chunk-1",
                content="package main",
                metadata=ChunkMetadataRequest(
                    repo_id="repo-a",
                    path="main.go",
                    language="go",
                    start_line=1,
                    end_line=1,
                    content_hash="hash-1",
                    status="active",
                ),
            ),
            IndexChunkRequest(
                id="chunk-2",
                content="func main() {}",
                metadata=ChunkMetadataRequest(
                    repo_id="repo-a",
                    path="main.go",
                    language="go",
                    start_line=3,
                    end_line=3,
                    content_hash="hash-2",
                    status="active",
                ),
            ),
        ],
    )


def test_index_chunks_batches_embeddings_and_ensures_collection_once(caplog: pytest.LogCaptureFixture):
    caplog.set_level("INFO", logger="codefind")
    vector_store = DummyVectorStore([])
    ollama = DummyOllama(
        responses=[
            EmbeddingResponse(embedding=[0.1, 0.2, 0.3]),
            EmbeddingResponse(embedding=[0.4, 0.5, 0.6]),
        ]
    )
    service = IndexingService(vector_store=vector_store, ollama=ollama)

    response = asyncio.run(service.index_chunks(org_id="org_123", request=_make_index_request()))

    assert response.status == "ok"
    assert response.repo_id == "repo-a"
    assert response.indexed_count == 2
    assert ollama.embed_many_calls == [["package main", "func main() {}"]]
    assert vector_store.ensure_collection_calls == [("org_123_repo-a", 3)]
    assert len(vector_store.upsert_calls) == 1
    collection, points = vector_store.upsert_calls[0]
    assert collection == "org_123_repo-a"
    assert [point.id for point in points] == ["chunk-1", "chunk-2"]
    assert [point.vector for point in points] == [[0.1, 0.2, 0.3], [0.4, 0.5, 0.6]]
    messages = [record.getMessage() for record in caplog.records]
    assert any("[INDEX] plan repo=repo-a" in message for message in messages)
    assert any("[INDEX][EMBED] repo=repo-a batch=1/1 size=2" in message for message in messages)
    assert any("[INDEX][UPSERT] repo=repo-a batch=1/1 points=2 indexed=2" in message for message in messages)
    assert any("[INDEX] complete repo=repo-a indexed=2" in message for message in messages)


def test_index_chunks_splits_large_requests_into_sub_batches():
    embed_batch_size = 32
    chunk_count = embed_batch_size + 1
    request = IndexRequest(
        repo_id="repo-a",
        chunks=[
            IndexChunkRequest(
                id=f"chunk-{index}",
                content=f"content-{index}",
                metadata=ChunkMetadataRequest(
                    repo_id="repo-a",
                    path=f"file-{index}.go",
                    language="go",
                    start_line=1,
                    end_line=1,
                    content_hash=f"hash-{index}",
                    status="active",
                ),
            )
            for index in range(chunk_count)
        ],
    )
    vector_store = DummyVectorStore([])
    ollama = DummyOllama(
        responses=[EmbeddingResponse(embedding=[0.1, 0.2, 0.3]) for _ in range(chunk_count)]
    )
    service = IndexingService(
        vector_store=vector_store,
        ollama=ollama,
        embed_batch_size=embed_batch_size,
    )

    response = asyncio.run(service.index_chunks(org_id="org_123", request=request))

    assert response.indexed_count == chunk_count
    assert ollama.embed_many_calls == [
        [f"content-{index}" for index in range(embed_batch_size)],
        [f"content-{embed_batch_size}"],
    ]
    assert vector_store.ensure_collection_calls == [("org_123_repo-a", 3)]
    assert len(vector_store.upsert_calls) == 2


def test_index_chunks_raises_when_embedding_count_does_not_match_chunks():
    vector_store = DummyVectorStore([])
    ollama = DummyOllama(responses=[EmbeddingResponse(embedding=[0.1, 0.2, 0.3])])
    service = IndexingService(vector_store=vector_store, ollama=ollama)

    with pytest.raises(OllamaError, match="count did not match request count"):
        asyncio.run(service.index_chunks(org_id="org_123", request=_make_index_request()))

    assert vector_store.ensure_collection_calls == []
    assert vector_store.upsert_calls == []


def test_index_chunks_propagates_ollama_errors():
    vector_store = DummyVectorStore([])
    ollama = DummyOllama(error=OllamaError("ollama request failed: timeout"))
    service = IndexingService(vector_store=vector_store, ollama=ollama)

    with pytest.raises(OllamaError, match="timeout"):
        asyncio.run(service.index_chunks(org_id="org_123", request=_make_index_request()))

    assert vector_store.ensure_collection_calls == []
    assert vector_store.upsert_calls == []


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
