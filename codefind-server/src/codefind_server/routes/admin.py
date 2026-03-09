from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, Request, status

from ..logging import emit_audit_event
from ..middleware.auth import OrgContext, require_admin
from ..models.requests import CreateOrganizationInvitationRequest
from ..models.responses import (
    OrganizationInvitationListResponse,
    OrganizationInvitationResponse,
    OrganizationMemberListResponse,
    OrganizationMemberResponse,
)
from ..services.clerk_admin import ClerkAdminError, ClerkAdminService


router = APIRouter(prefix="/admin", tags=["admin"])


def get_clerk_admin_service(request: Request) -> ClerkAdminService:
    settings = getattr(request.app.state, "settings", None)
    if settings is None:
        raise RuntimeError("Application settings were not initialized.")
    return ClerkAdminService(settings=settings)


@router.get("/members", response_model=OrganizationMemberListResponse)
async def list_members(
    context: OrgContext = Depends(require_admin),
    service: ClerkAdminService = Depends(get_clerk_admin_service),
) -> OrganizationMemberListResponse:
    try:
        payload = await service.list_org_members(organization_id=context.org_id)
    except ClerkAdminError as error:
        raise HTTPException(status_code=status.HTTP_502_BAD_GATEWAY, detail=str(error)) from error
    return OrganizationMemberListResponse(data=payload.data, total_count=payload.total_count)


@router.get("/invitations", response_model=OrganizationInvitationListResponse)
async def list_invitations(
    context: OrgContext = Depends(require_admin),
    service: ClerkAdminService = Depends(get_clerk_admin_service),
) -> OrganizationInvitationListResponse:
    try:
        payload = await service.list_org_invitations(organization_id=context.org_id)
    except ClerkAdminError as error:
        raise HTTPException(status_code=status.HTTP_502_BAD_GATEWAY, detail=str(error)) from error
    return OrganizationInvitationListResponse(data=payload.data, total_count=payload.total_count)


@router.post("/invite", response_model=OrganizationInvitationResponse, status_code=status.HTTP_201_CREATED)
async def invite_member(
    invitation: CreateOrganizationInvitationRequest,
    context: OrgContext = Depends(require_admin),
    service: ClerkAdminService = Depends(get_clerk_admin_service),
) -> OrganizationInvitationResponse:
    try:
        created = await service.create_org_invitation(
            organization_id=context.org_id,
            inviter_user_id=context.user_id,
            email_address=invitation.email_address,
            role=invitation.role,
        )
    except ClerkAdminError as error:
        emit_audit_event(
            event_type="admin.invite",
            result="failure",
            target=invitation.email_address,
            metadata={"reason": str(error)},
        )
        raise HTTPException(status_code=status.HTTP_502_BAD_GATEWAY, detail=str(error)) from error

    emit_audit_event(
        event_type="admin.invite",
        result="success",
        target=invitation.email_address,
        metadata={"role": invitation.role, "invitation_id": created["id"]},
    )
    return OrganizationInvitationResponse.model_validate(created)


@router.post("/invitations/{invitation_id}/revoke", response_model=OrganizationInvitationResponse)
async def revoke_invitation(
    invitation_id: str,
    context: OrgContext = Depends(require_admin),
    service: ClerkAdminService = Depends(get_clerk_admin_service),
) -> OrganizationInvitationResponse:
    try:
        revoked = await service.revoke_org_invitation(
            organization_id=context.org_id,
            invitation_id=invitation_id,
            requesting_user_id=context.user_id,
        )
    except ClerkAdminError as error:
        emit_audit_event(
            event_type="admin.revoke_invite",
            result="failure",
            target=invitation_id,
            metadata={"reason": str(error)},
        )
        raise HTTPException(status_code=status.HTTP_502_BAD_GATEWAY, detail=str(error)) from error

    emit_audit_event(
        event_type="admin.revoke_invite",
        result="success",
        target=revoked.get("email_address") or invitation_id,
        metadata={"invitation_id": invitation_id},
    )
    return OrganizationInvitationResponse.model_validate(revoked)


@router.delete("/members/{user_id}", response_model=OrganizationMemberResponse)
async def remove_member(
    user_id: str,
    context: OrgContext = Depends(require_admin),
    service: ClerkAdminService = Depends(get_clerk_admin_service),
) -> OrganizationMemberResponse:
    if user_id == context.user_id:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Admins cannot remove themselves from the active organization.",
        )
    try:
        removed = await service.remove_org_member(
            organization_id=context.org_id,
            user_id=user_id,
        )
    except ClerkAdminError as error:
        emit_audit_event(
            event_type="admin.remove_member",
            result="failure",
            target=user_id,
            metadata={"reason": str(error)},
        )
        raise HTTPException(status_code=status.HTTP_502_BAD_GATEWAY, detail=str(error)) from error

    emit_audit_event(
        event_type="admin.remove_member",
        result="success",
        target=user_id,
        metadata={"removed_user_id": user_id},
    )
    return OrganizationMemberResponse.model_validate(removed)
