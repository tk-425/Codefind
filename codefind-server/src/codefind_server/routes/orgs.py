from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, status

from ..middleware.auth import OrgContext, require_auth
from ..models.responses import OrgListResponse
from ..routes.admin import get_clerk_admin_service
from ..services.clerk_admin import ClerkAdminError, ClerkAdminService


router = APIRouter(prefix="/orgs", tags=["orgs"])


@router.get("", response_model=OrgListResponse)
async def list_orgs(
    context: OrgContext = Depends(require_auth),
    service: ClerkAdminService = Depends(get_clerk_admin_service),
) -> OrgListResponse:
    try:
        payload = await service.list_user_orgs(user_id=context.user_id)
    except ClerkAdminError as error:
        raise HTTPException(status_code=status.HTTP_502_BAD_GATEWAY, detail=str(error)) from error
    return OrgListResponse(data=payload.data, total_count=payload.total_count)
