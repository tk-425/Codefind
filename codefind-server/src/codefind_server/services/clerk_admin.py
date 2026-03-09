from __future__ import annotations

from dataclasses import dataclass
from typing import Any

import httpx

from ..config import Settings
from ..middleware.auth import normalize_org_role


class ClerkAdminError(RuntimeError):
    """Raised when Clerk admin operations fail."""


@dataclass(slots=True, frozen=True)
class ClerkPaginatedResponse:
    data: list[dict[str, Any]]
    total_count: int


class ClerkAdminService:
    def __init__(
        self,
        *,
        settings: Settings,
        client: httpx.AsyncClient | None = None,
    ) -> None:
        self._settings = settings
        self._client = client

    async def list_user_orgs(self, *, user_id: str) -> ClerkPaginatedResponse:
        payload = await self._request(
            "GET",
            f"/users/{user_id}/organization_memberships",
        )
        items = payload.get("data", [])
        if not isinstance(items, list):
            items = []
        return ClerkPaginatedResponse(data=[self._normalize_membership(item) for item in items], total_count=_total_count(payload, items))

    async def list_org_members(self, *, organization_id: str) -> ClerkPaginatedResponse:
        payload = await self._request(
            "GET",
            f"/organizations/{organization_id}/memberships",
        )
        items = payload.get("data", [])
        if not isinstance(items, list):
            items = []
        return ClerkPaginatedResponse(data=[self._normalize_membership(item) for item in items], total_count=_total_count(payload, items))

    async def list_org_invitations(self, *, organization_id: str) -> ClerkPaginatedResponse:
        payload = await self._request(
            "GET",
            f"/organizations/{organization_id}/invitations",
            params={"status": "pending"},
        )
        items = payload.get("data", [])
        if not isinstance(items, list):
            items = []
        return ClerkPaginatedResponse(data=[self._normalize_invitation(item) for item in items], total_count=_total_count(payload, items))

    async def create_org_invitation(
        self,
        *,
        organization_id: str,
        inviter_user_id: str,
        email_address: str,
        role: str,
    ) -> dict[str, Any]:
        payload = await self._request(
            "POST",
            f"/organizations/{organization_id}/invitations",
            json={
                "inviter_user_id": inviter_user_id,
                "email_address": email_address,
                "role": role,
                "redirect_url": f"{self._settings.web_app_url.rstrip('/')}/accept-invitation",
            },
        )
        return self._normalize_invitation(payload)

    async def revoke_org_invitation(
        self,
        *,
        organization_id: str,
        invitation_id: str,
        requesting_user_id: str,
    ) -> dict[str, Any]:
        payload = await self._request(
            "POST",
            f"/organizations/{organization_id}/invitations/{invitation_id}/revoke",
            json={"requesting_user_id": requesting_user_id},
        )
        return self._normalize_invitation(payload)

    async def remove_org_member(
        self,
        *,
        organization_id: str,
        user_id: str,
    ) -> dict[str, Any]:
        payload = await self._request(
            "DELETE",
            f"/organizations/{organization_id}/memberships/{user_id}",
        )
        return self._normalize_membership(payload)

    async def _request(
        self,
        method: str,
        path: str,
        *,
        params: dict[str, Any] | None = None,
        json: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        headers = {
            "Authorization": f"Bearer {self._settings.clerk_secret_key}",
            "Accept": "application/json",
        }
        if self._client is not None:
            response = await self._client.request(
                method,
                f"https://api.clerk.com/v1{path}",
                params=params,
                json=json,
                headers=headers,
            )
            return self._handle_response(response)

        async with httpx.AsyncClient(timeout=5.0) as client:
            response = await client.request(
                method,
                f"https://api.clerk.com/v1{path}",
                params=params,
                json=json,
                headers=headers,
            )
            return self._handle_response(response)

    @staticmethod
    def _handle_response(response: httpx.Response) -> dict[str, Any]:
        try:
            response.raise_for_status()
        except httpx.HTTPStatusError as error:
            detail = _extract_error_detail(error.response)
            raise ClerkAdminError(detail) from error
        except httpx.HTTPError as error:
            raise ClerkAdminError("Failed to reach Clerk Backend API.") from error

        data = response.json()
        if not isinstance(data, dict):
            raise ClerkAdminError("Unexpected Clerk response shape.")
        return data

    @staticmethod
    def _normalize_membership(item: dict[str, Any]) -> dict[str, Any]:
        public_user_data = item.get("public_user_data")
        if not isinstance(public_user_data, dict):
            public_user_data = item.get("publicUserData")
        email_addresses = None
        if isinstance(public_user_data, dict):
            email_addresses = public_user_data.get("identifier") or public_user_data.get("email_address")
        organization = item.get("organization")
        if not isinstance(organization, dict):
            organization = None
        return {
            "membership_id": item.get("id"),
            "organization_id": (organization.get("id") if organization else None) or item.get("organization_id"),
            "organization_name": organization.get("name") if organization else None,
            "organization_slug": organization.get("slug") if organization else None,
            "user_id": item.get("user_id")
            or (public_user_data.get("user_id") if isinstance(public_user_data, dict) else None)
            or (public_user_data.get("userId") if isinstance(public_user_data, dict) else None),
            "role": normalize_org_role(item.get("role")) or item.get("role"),
            "first_name": (
                public_user_data.get("first_name")
                if isinstance(public_user_data, dict)
                else None
            )
            or (
                public_user_data.get("firstName")
                if isinstance(public_user_data, dict)
                else None
            ),
            "last_name": (
                public_user_data.get("last_name")
                if isinstance(public_user_data, dict)
                else None
            )
            or (
                public_user_data.get("lastName")
                if isinstance(public_user_data, dict)
                else None
            ),
            "email_address": email_addresses,
            "profile_image_url": (
                public_user_data.get("image_url")
                if isinstance(public_user_data, dict)
                else None
            )
            or (
                public_user_data.get("profileImageUrl")
                if isinstance(public_user_data, dict)
                else None
            ),
        }

    @staticmethod
    def _normalize_invitation(item: dict[str, Any]) -> dict[str, Any]:
        return {
            "id": item.get("id"),
            "email_address": item.get("email_address"),
            "role": normalize_org_role(item.get("role")) or item.get("role"),
            "status": item.get("status"),
            "organization_id": item.get("org_id") or item.get("organization_id") or item.get("organizationId"),
            "created_at": item.get("created_at"),
            "updated_at": item.get("updated_at"),
            "expires_at": item.get("expires_at"),
            "inviter_user_id": item.get("inviter_user_id"),
        }


def _extract_error_detail(response: httpx.Response) -> str:
    try:
        payload = response.json()
    except ValueError:
        return f"Clerk request failed: {response.status_code}"
    errors = payload.get("errors")
    if isinstance(errors, list) and errors:
        first_error = errors[0]
        if isinstance(first_error, dict) and isinstance(first_error.get("message"), str):
            return first_error["message"]
    return f"Clerk request failed: {response.status_code}"


def _total_count(payload: dict[str, Any], items: list[dict[str, Any]]) -> int:
    total_count = payload.get("total_count", payload.get("totalCount"))
    if isinstance(total_count, int):
        return total_count
    return len(items)
