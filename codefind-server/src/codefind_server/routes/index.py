from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, Request, status

from ..adapters.base import VectorStore
from ..middleware.auth import OrgContext, require_admin
from ..models.requests import ChunkPurgeRequest, ChunkStatusUpdateRequest, IndexRequest
from ..models.responses import ChunkPurgeResponse, ChunkStatusUpdateResponse, IndexResponse
from ..services import IndexingService, OllamaService


router = APIRouter(tags=["index"])


def get_vector_store(request: Request) -> VectorStore:
    return request.app.state.vector_store


def get_ollama_service(request: Request) -> OllamaService:
    return request.app.state.ollama


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
) -> IndexResponse:
    return await indexing_service.index_chunks(org_id=context.org_id, request=request)


@router.patch("/chunks/status", response_model=ChunkStatusUpdateResponse)
async def update_chunk_status(
    request: ChunkStatusUpdateRequest,
    context: OrgContext = Depends(require_admin),
    indexing_service: IndexingService = Depends(get_indexing_service),
) -> ChunkStatusUpdateResponse:
    return await indexing_service.update_chunk_status(org_id=context.org_id, request=request)


@router.delete("/chunks/purge", response_model=ChunkPurgeResponse)
async def purge_chunks(
    request: ChunkPurgeRequest,
    context: OrgContext = Depends(require_admin),
) -> ChunkPurgeResponse:
    raise HTTPException(
        status_code=status.HTTP_501_NOT_IMPLEMENTED,
        detail=(
            "Phase 8 Slice 1 only defines the /chunks/purge contract. "
            "Chunk purge logic is not implemented yet."
        ),
    )
