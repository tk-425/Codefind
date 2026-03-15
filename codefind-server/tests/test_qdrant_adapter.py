from __future__ import annotations

import asyncio

import pytest
from qdrant_client import models

from codefind_server.adapters.base import HybridQuery, SparseVectorData, VectorPoint
from codefind_server.adapters.qdrant import (
    CollectionSchemaError,
    DENSE_VECTOR_NAME,
    SPARSE_VECTOR_NAME,
    QdrantAdapter,
)


class FakeQdrantClient:
    def __init__(self) -> None:
        self.exists = False
        self.create_collection_calls: list[dict[str, object]] = []
        self.upsert_calls: list[dict[str, object]] = []
        self.query_points_calls: list[dict[str, object]] = []
        self.collection_info = models.CollectionInfo(
            status=models.CollectionStatus.GREEN,
            optimizer_status=models.OptimizersStatusOneOf.OK,
            vectors_count=None,
            indexed_vectors_count=0,
            points_count=0,
            segments_count=1,
            config=models.CollectionConfig(
                params=models.CollectionParams(
                    vectors={
                        DENSE_VECTOR_NAME: models.VectorParams(
                            size=3,
                            distance=models.Distance.COSINE,
                        )
                    },
                    sparse_vectors={
                        SPARSE_VECTOR_NAME: models.SparseVectorParams()
                    },
                ),
                hnsw_config=models.HnswConfig(
                    m=16,
                    ef_construct=100,
                    full_scan_threshold=10000,
                ),
                optimizer_config=models.OptimizersConfig(
                    deleted_threshold=0.2,
                    vacuum_min_vector_number=1000,
                    default_segment_number=0,
                    flush_interval_sec=5,
                ),
                wal_config=models.WalConfig(
                    wal_capacity_mb=32,
                    wal_segments_ahead=0,
                ),
            ),
            payload_schema={},
        )

    async def get_collections(self):
        return models.CollectionsResponse(collections=[])

    async def collection_exists(self, _collection_name: str) -> bool:
        return self.exists

    async def create_collection(self, **kwargs):
        self.create_collection_calls.append(kwargs)

    async def get_collection(self, _collection_name: str):
        return self.collection_info

    async def upsert(self, **kwargs):
        self.upsert_calls.append(kwargs)

    async def query_points(self, **kwargs):
        self.query_points_calls.append(kwargs)
        return type(
            "QueryPointsResponse",
            (),
            {
                "points": [
                    models.ScoredPoint(
                        id="chunk-1",
                        version=1,
                        score=0.91,
                        payload={"path": "main.go"},
                        vector=None,
                        shard_key=None,
                        order_value=None,
                    )
                ]
            },
        )()

    async def close(self) -> None:
        return None


def _make_adapter(fake_client: FakeQdrantClient) -> QdrantAdapter:
    adapter = QdrantAdapter(url="http://localhost:6333")
    adapter._client = fake_client
    return adapter


def test_ensure_collection_creates_named_dense_and_sparse_vectors():
    fake_client = FakeQdrantClient()
    adapter = _make_adapter(fake_client)

    asyncio.run(adapter.ensure_collection("org_123_repo-a", 3))

    assert len(fake_client.create_collection_calls) == 1
    call = fake_client.create_collection_calls[0]
    assert DENSE_VECTOR_NAME in call["vectors_config"]
    assert SPARSE_VECTOR_NAME in call["sparse_vectors_config"]


def test_ensure_collection_rejects_existing_dense_only_collection():
    fake_client = FakeQdrantClient()
    fake_client.exists = True
    fake_client.collection_info.config.params.sparse_vectors = None
    adapter = _make_adapter(fake_client)

    with pytest.raises(CollectionSchemaError, match="missing sparse vector config"):
        asyncio.run(adapter.ensure_collection("org_123_repo-a", 3))


def test_upsert_writes_named_dense_and_sparse_vectors():
    fake_client = FakeQdrantClient()
    adapter = _make_adapter(fake_client)

    asyncio.run(
        adapter.upsert(
            "org_123_repo-a",
            [
                VectorPoint(
                    id="chunk-1",
                    dense_vector=[0.1, 0.2, 0.3],
                    sparse_vector=SparseVectorData(indices=[1, 4], values=[0.8, 0.2]),
                    payload={"path": "main.go"},
                )
            ],
        )
    )

    point = fake_client.upsert_calls[0]["points"][0]
    assert DENSE_VECTOR_NAME in point.vector
    assert SPARSE_VECTOR_NAME in point.vector
    assert point.vector[DENSE_VECTOR_NAME] == [0.1, 0.2, 0.3]
    assert point.vector[SPARSE_VECTOR_NAME].indices == [1, 4]


def test_query_uses_prefetch_and_fusion_query():
    fake_client = FakeQdrantClient()
    adapter = _make_adapter(fake_client)

    results = asyncio.run(
        adapter.query(
            "org_123_repo-a",
            HybridQuery(
                dense_vector=[0.1, 0.2, 0.3],
                sparse_vector=SparseVectorData(indices=[2, 5], values=[0.7, 0.3]),
                filters={"status": "active"},
                top_k=10,
                dense_top_k=50,
                sparse_top_k=30,
            ),
        )
    )

    assert results[0].id == "chunk-1"
    call = fake_client.query_points_calls[0]
    assert isinstance(call["query"], models.FusionQuery)
    assert call["query"].fusion == models.Fusion.RRF
    assert len(call["prefetch"]) == 2
    assert call["prefetch"][0].using == DENSE_VECTOR_NAME
    assert call["prefetch"][0].limit == 50
    assert call["prefetch"][1].using == SPARSE_VECTOR_NAME
    assert call["prefetch"][1].limit == 30
