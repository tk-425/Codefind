from __future__ import annotations

from datetime import UTC, datetime

from ..adapters.base import VectorPoint, VectorStore
from ..models.requests import ChunkStatusUpdateRequest, IndexRequest
from ..models.responses import ChunkStatusUpdateResponse, IndexResponse
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
