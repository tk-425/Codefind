from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, status

from ..middleware.auth import OrgContext, require_admin
from ..models.requests import ChunkPurgeRequest, ChunkStatusUpdateRequest, IndexRequest
from ..models.responses import ChunkPurgeResponse, ChunkStatusUpdateResponse, IndexResponse


router = APIRouter(tags=["index"])


@router.post("/index", response_model=IndexResponse, status_code=status.HTTP_202_ACCEPTED)
async def index_repo(
    request: IndexRequest,
    context: OrgContext = Depends(require_admin),
) -> IndexResponse:
    raise HTTPException(
        status_code=status.HTTP_501_NOT_IMPLEMENTED,
        detail=(
            "Phase 8 Slice 1 only defines the /index contract. "
            "The indexing pipeline is not implemented yet."
        ),
    )


@router.patch("/chunks/status", response_model=ChunkStatusUpdateResponse)
async def update_chunk_status(
    request: ChunkStatusUpdateRequest,
    context: OrgContext = Depends(require_admin),
) -> ChunkStatusUpdateResponse:
    raise HTTPException(
        status_code=status.HTTP_501_NOT_IMPLEMENTED,
        detail=(
            "Phase 8 Slice 1 only defines the /chunks/status contract. "
            "Chunk tombstoning/update logic is not implemented yet."
        ),
    )


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
