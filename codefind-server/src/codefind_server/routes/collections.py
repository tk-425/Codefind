from __future__ import annotations

from fastapi import APIRouter, Depends, Request

from ..adapters.base import VectorStore
from ..middleware.auth import OrgContext, require_auth
from ..models.responses import CollectionListResponse, CollectionSummaryResponse
from ..services import repo_id_from_collection


router = APIRouter(prefix="/collections", tags=["collections"])


def get_vector_store(request: Request) -> VectorStore:
    return request.app.state.vector_store


@router.get("", response_model=CollectionListResponse)
async def list_collections(
    context: OrgContext = Depends(require_auth),
    vector_store: VectorStore = Depends(get_vector_store),
) -> CollectionListResponse:
    collections = await vector_store.list_collections()
    repo_ids = sorted(
        repo_id
        for collection_name in collections
        for repo_id in [repo_id_from_collection(context.org_id, collection_name)]
        if repo_id is not None
    )
    return CollectionListResponse(
        data=[CollectionSummaryResponse(repo_id=repo_id) for repo_id in repo_ids],
        total_count=len(repo_ids),
    )
