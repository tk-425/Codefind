from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, Request, status

from ..adapters.base import VectorStore
from ..logging import emit_audit_event
from ..middleware.auth import OrgContext, require_admin
from ..models.requests import ChunkPurgeRequest, ChunkStatusUpdateRequest, IndexRequest
from ..models.responses import (
    ChunkPurgeResponse,
    ChunkStatusUpdateResponse,
    IndexResponse,
    TombstonedChunkListResponse,
)
from ..services import IndexJobLockManager, IndexingService, OllamaService


router = APIRouter(tags=["index"])


def _response_value(payload: object, key: str) -> object:
    if isinstance(payload, dict):
        return payload.get(key)
    return getattr(payload, key)


def get_vector_store(request: Request) -> VectorStore:
    return request.app.state.vector_store


def get_ollama_service(request: Request) -> OllamaService:
    return request.app.state.ollama


def get_index_lock_manager(request: Request) -> IndexJobLockManager:
    return request.app.state.index_locks


def get_indexing_service(
    vector_store: VectorStore = Depends(get_vector_store),
    ollama: OllamaService = Depends(get_ollama_service),
) -> IndexingService:
    return IndexingService(vector_store=vector_store, ollama=ollama)


@router.post("/index", response_model=IndexResponse, status_code=status.HTTP_200_OK)
async def index_repo(
    request: IndexRequest,
    context: OrgContext = Depends(require_admin),
    indexing_service: IndexingService = Depends(get_indexing_service),
    lock_manager: IndexJobLockManager = Depends(get_index_lock_manager),
) -> IndexResponse:
    lock_key = f"{context.org_id}:{request.repo_id}"
    acquired = await lock_manager.acquire(lock_key)
    if not acquired:
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail="An indexing job is already active for this repository.",
        )

    emit_audit_event(
        event_type="index.run",
        result="start",
        repo_id=request.repo_id,
        metadata={"chunk_count": len(request.chunks)},
    )
    try:
        response = await indexing_service.index_chunks(org_id=context.org_id, request=request)
    except Exception as error:
        emit_audit_event(
            event_type="index.run",
            result="failure",
            repo_id=request.repo_id,
            metadata={"reason": str(error), "chunk_count": len(request.chunks)},
        )
        raise
    else:
        emit_audit_event(
            event_type="index.run",
            result="success",
            repo_id=request.repo_id,
            metadata={
                "indexed_count": _response_value(response, "indexed_count"),
                "accepted": _response_value(response, "accepted"),
            },
        )
        return response
    finally:
        await lock_manager.release(lock_key)


@router.patch("/chunks/status", response_model=ChunkStatusUpdateResponse)
async def update_chunk_status(
    request: ChunkStatusUpdateRequest,
    context: OrgContext = Depends(require_admin),
    indexing_service: IndexingService = Depends(get_indexing_service),
) -> ChunkStatusUpdateResponse:
    try:
        response = await indexing_service.update_chunk_status(org_id=context.org_id, request=request)
    except Exception as error:
        emit_audit_event(
            event_type="chunks.status",
            result="failure",
            repo_id=request.repo_id,
            metadata={"status": request.status, "chunk_count": len(request.chunk_ids), "reason": str(error)},
        )
        raise

    emit_audit_event(
        event_type="chunks.status",
        result="success",
        repo_id=request.repo_id,
        metadata={"status": request.status, "chunk_count": len(request.chunk_ids)},
    )
    return response


@router.get("/chunks/tombstoned", response_model=TombstonedChunkListResponse)
async def list_tombstoned_chunks(
    repo_id: str,
    context: OrgContext = Depends(require_admin),
    indexing_service: IndexingService = Depends(get_indexing_service),
) -> TombstonedChunkListResponse:
    return await indexing_service.list_tombstoned_chunks(org_id=context.org_id, repo_id=repo_id)


@router.delete("/chunks/purge", response_model=ChunkPurgeResponse)
async def purge_chunks(
    request: ChunkPurgeRequest,
    context: OrgContext = Depends(require_admin),
    indexing_service: IndexingService = Depends(get_indexing_service),
) -> ChunkPurgeResponse:
    try:
        response = await indexing_service.purge_tombstoned_chunks(org_id=context.org_id, request=request)
    except Exception as error:
        emit_audit_event(
            event_type="chunks.purge",
            result="failure",
            repo_id=request.repo_id,
            metadata={"older_than_days": request.older_than_days, "reason": str(error)},
        )
        raise

    emit_audit_event(
        event_type="chunks.purge",
        result="success",
        repo_id=request.repo_id,
        metadata={
            "older_than_days": request.older_than_days,
            "purged_count": _response_value(response, "purged_count"),
        },
    )
    return response
