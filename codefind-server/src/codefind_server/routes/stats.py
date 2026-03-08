from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, Query, Request, status

from ..adapters.base import VectorStore
from ..middleware.auth import OrgContext, require_auth
from ..models.responses import RepoStatsResponse, StatsResponse
from ..services import collection_name_for, repo_id_from_collection, validate_repo_id
from .collections import get_vector_store


router = APIRouter(prefix="/stats", tags=["stats"])


def _repo_count_filters() -> dict[str, object]:
    return {}


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
        chunk_count = await vector_store.count(collection_name, _repo_count_filters())
        return StatsResponse(
            repo_id=repo_id,
            repo_count=1,
            chunk_count=chunk_count,
            repos=[RepoStatsResponse(repo_id=repo_id, chunk_count=chunk_count)],
        )

    collections = await vector_store.list_collections()
    repo_stats: list[RepoStatsResponse] = []
    total_chunks = 0
    for collection_name in collections:
        scoped_repo_id = repo_id_from_collection(context.org_id, collection_name)
        if scoped_repo_id is None:
            continue
        chunk_count = await vector_store.count(collection_name, _repo_count_filters())
        total_chunks += chunk_count
        repo_stats.append(RepoStatsResponse(repo_id=scoped_repo_id, chunk_count=chunk_count))

    repo_stats.sort(key=lambda item: item.repo_id)
    return StatsResponse(
        repo_id=None,
        repo_count=len(repo_stats),
        chunk_count=total_chunks,
        repos=repo_stats,
    )
