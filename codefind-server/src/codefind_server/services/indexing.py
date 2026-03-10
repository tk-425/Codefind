from __future__ import annotations

from datetime import UTC, datetime
from collections import OrderedDict

from ..adapters.base import VectorPoint, VectorStore
from ..models.requests import ChunkPurgeRequest, ChunkStatusUpdateRequest, IndexRequest
from ..models.responses import (
    ChunkPurgeResponse,
    ChunkStatusUpdateResponse,
    IndexResponse,
    RepoClearResponse,
    TombstonedChunkListResponse,
    TombstonedChunkSummaryResponse,
)
from .collection_scope import collection_name_for
from .ollama import OllamaService


class IndexingService:
    def __init__(self, *, vector_store: VectorStore, ollama: OllamaService) -> None:
        self._vector_store = vector_store
        self._ollama = ollama

    async def index_chunks(self, *, org_id: str, request: IndexRequest) -> IndexResponse:
        collection = collection_name_for(org_id, request.repo_id)
        points: list[VectorPoint] = []

        for chunk in request.chunks:
            embedding = await self._ollama.embed(chunk.content)
            await self._vector_store.ensure_collection(collection, len(embedding.embedding))
            payload = {
                "repo_id": request.repo_id,
                "path": chunk.metadata.path,
                "language": chunk.metadata.language,
                "start_line": chunk.metadata.start_line,
                "end_line": chunk.metadata.end_line,
                "content_hash": chunk.metadata.content_hash,
                "status": chunk.metadata.status,
                "symbol_name": chunk.metadata.symbol_name,
                "symbol_kind": chunk.metadata.symbol_kind,
                "parent_name": chunk.metadata.parent_name,
                "indexed_at": chunk.metadata.indexed_at
                or datetime.now(UTC).isoformat(),
                "chunking_method": chunk.metadata.chunking_method,
                "fallback_reason": chunk.metadata.fallback_reason,
                "snippet": chunk.content,
                "content": chunk.content,
            }
            points.append(
                VectorPoint(
                    id=chunk.id,
                    vector=embedding.embedding,
                    payload={k: v for k, v in payload.items() if v is not None and v != ""},
                )
            )

        await self._vector_store.upsert(collection, points)
        return IndexResponse(
            status="ok",
            repo_id=request.repo_id,
            indexed_count=len(points),
            accepted=True,
        )

    async def update_chunk_status(
        self,
        *,
        org_id: str,
        request: ChunkStatusUpdateRequest,
    ) -> ChunkStatusUpdateResponse:
        collection = collection_name_for(org_id, request.repo_id)
        payload: dict[str, object] = {"status": request.status}
        if request.status == "tombstoned":
            payload["tombstoned_at"] = datetime.now(UTC).isoformat()

        await self._vector_store.update_payload(collection, request.chunk_ids, payload)
        return ChunkStatusUpdateResponse(
            status="ok",
            repo_id=request.repo_id,
            updated_count=len(request.chunk_ids),
        )

    async def list_tombstoned_chunks(
        self,
        *,
        org_id: str,
        repo_id: str,
    ) -> TombstonedChunkListResponse:
        collection = collection_name_for(org_id, repo_id)
        if collection not in await self._vector_store.list_collections():
            return TombstonedChunkListResponse(
                status="ok",
                repo_id=repo_id,
                found_count=0,
                files=[],
            )
        points = await self._vector_store.scroll(
            collection,
            {"status": "tombstoned", "repo_id": repo_id},
        )
        files = self._summarize_tombstoned_points(points)
        return TombstonedChunkListResponse(
            status="ok",
            repo_id=repo_id,
            found_count=len(points),
            files=files,
        )

    async def purge_tombstoned_chunks(
        self,
        *,
        org_id: str,
        request: ChunkPurgeRequest,
    ) -> ChunkPurgeResponse:
        collection = collection_name_for(org_id, request.repo_id)
        if collection not in await self._vector_store.list_collections():
            return ChunkPurgeResponse(
                status="ok",
                repo_id=request.repo_id,
                found_count=0,
                purged_count=0,
                files=[],
            )
        points = await self._vector_store.scroll(
            collection,
            {"status": "tombstoned", "repo_id": request.repo_id},
        )

        cutoff = datetime.now(UTC).timestamp() - (request.older_than_days * 86400)
        matching_points = []
        for point in points:
            tombstoned_at = point.payload.get("tombstoned_at")
            if not isinstance(tombstoned_at, str):
                continue
            try:
                tombstoned_ts = datetime.fromisoformat(tombstoned_at.replace("Z", "+00:00")).timestamp()
            except ValueError:
                continue
            if tombstoned_ts <= cutoff:
                matching_points.append(point)

        files = self._summarize_tombstoned_points(matching_points)
        if matching_points:
            await self._vector_store.delete(collection, [point.id for point in matching_points])

        return ChunkPurgeResponse(
            status="ok",
            repo_id=request.repo_id,
            found_count=len(matching_points),
            purged_count=len(matching_points),
            files=files,
        )

    async def clear_repo_index(self, *, org_id: str, repo_id: str) -> RepoClearResponse:
        collection = collection_name_for(org_id, repo_id)
        if collection in await self._vector_store.list_collections():
            await self._vector_store.delete_collection(collection)
            return RepoClearResponse(status="ok", repo_id=repo_id, cleared=True)
        return RepoClearResponse(status="ok", repo_id=repo_id, cleared=False)

    def _summarize_tombstoned_points(self, points) -> list[TombstonedChunkSummaryResponse]:
        by_path: OrderedDict[str, tuple[int, str | None]] = OrderedDict()
        for point in points:
            path = point.payload.get("path")
            if not isinstance(path, str) or not path:
                path = "unknown"

            tombstoned_at = point.payload.get("tombstoned_at")
            tombstoned_at_value = tombstoned_at if isinstance(tombstoned_at, str) else None

            chunk_count, existing_tombstoned_at = by_path.get(path, (0, tombstoned_at_value))
            by_path[path] = (chunk_count + 1, existing_tombstoned_at or tombstoned_at_value)

        return [
            TombstonedChunkSummaryResponse(
                path=path,
                chunk_count=chunk_count,
                tombstoned_at=tombstoned_at,
            )
            for path, (chunk_count, tombstoned_at) in by_path.items()
        ]
