from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, Query, Request, status

from ..adapters.base import VectorStore
from ..middleware.auth import OrgContext, require_auth
from ..models.responses import RepoStatsResponse, StatsResponse
from ..services import collection_name_for, repo_id_from_collection, validate_repo_id
from .collections import get_vector_store


router = APIRouter(prefix="/stats", tags=["stats"])


def _active_chunk_filters() -> dict[str, object]:
    return {"status": "active"}


def _deleted_chunk_filters() -> dict[str, object]:
    return {"status": "tombstoned"}


async def _count_repo_chunks(vector_store: VectorStore, collection_name: str) -> tuple[int, int]:
    active_chunks = await vector_store.count(collection_name, _active_chunk_filters())
    deleted_chunks = await vector_store.count(collection_name, _deleted_chunk_filters())
    return active_chunks, deleted_chunks


def _overhead_percent(active_chunks: int, deleted_chunks: int) -> float:
    if active_chunks <= 0:
        if deleted_chunks <= 0:
            return 0.0
        return 100.0
    return round((deleted_chunks / active_chunks) * 100, 1)


@router.get("", response_model=StatsResponse)
async def get_stats(
    request: Request,
    repo_id: str | None = Query(default=None),
    context: OrgContext = Depends(require_auth),
    vector_store: VectorStore = Depends(get_vector_store),
) -> StatsResponse:
    del request
    if repo_id is not None:
        try:
            repo_id = validate_repo_id(repo_id)
        except ValueError as error:
            raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(error)) from error
        collection_name = collection_name_for(context.org_id, repo_id)
        active_chunks, deleted_chunks = await _count_repo_chunks(vector_store, collection_name)
        return StatsResponse(
            repo_id=repo_id,
            repo_count=1,
            chunk_count=active_chunks,
            active_chunks=active_chunks,
            deleted_chunks=deleted_chunks,
            total_chunks=active_chunks + deleted_chunks,
            overhead_percent=_overhead_percent(active_chunks, deleted_chunks),
            repos=[RepoStatsResponse(repo_id=repo_id, chunk_count=active_chunks)],
        )

    collections = await vector_store.list_collections()
    repo_stats: list[RepoStatsResponse] = []
    total_active_chunks = 0
    total_deleted_chunks = 0
    for collection_name in collections:
        scoped_repo_id = repo_id_from_collection(context.org_id, collection_name)
        if scoped_repo_id is None:
            continue
        active_chunks, deleted_chunks = await _count_repo_chunks(vector_store, collection_name)
        total_active_chunks += active_chunks
        total_deleted_chunks += deleted_chunks
        repo_stats.append(RepoStatsResponse(repo_id=scoped_repo_id, chunk_count=active_chunks))

    repo_stats.sort(key=lambda item: item.repo_id)
    return StatsResponse(
        repo_id=None,
        repo_count=len(repo_stats),
        chunk_count=total_active_chunks,
        active_chunks=total_active_chunks,
        deleted_chunks=total_deleted_chunks,
        total_chunks=total_active_chunks + total_deleted_chunks,
        overhead_percent=_overhead_percent(total_active_chunks, total_deleted_chunks),
        repos=repo_stats,
    )
