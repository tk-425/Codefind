from __future__ import annotations

from dataclasses import dataclass

from fastapi import HTTPException, status


@dataclass(slots=True, frozen=True)
class OrgContext:
    org_id: str
    org_role: str
    user_id: str


async def require_auth() -> OrgContext:
    raise HTTPException(
        status_code=status.HTTP_501_NOT_IMPLEMENTED,
        detail="Phase 2: Clerk JWT auth not implemented yet",
    )


async def require_admin() -> OrgContext:
    raise HTTPException(
        status_code=status.HTTP_501_NOT_IMPLEMENTED,
        detail="Phase 2: Clerk admin auth not implemented yet",
    )
