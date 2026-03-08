from __future__ import annotations

from fastapi import FastAPI, HTTPException, status
from fastapi.testclient import TestClient

from codefind_server.middleware.auth import OrgContext
from codefind_server.middleware.auth import require_admin, require_auth
from codefind_server.routes.admin import router as admin_router
from codefind_server.routes.admin import get_clerk_admin_service
from codefind_server.routes.orgs import router as orgs_router


class DummyClerkAdminService:
    async def list_user_orgs(self, *, user_id: str):
        assert user_id == "user_admin"
        return type("Payload", (), {"data": [{"organization_id": "org_123", "organization_name": "Acme", "organization_slug": "acme", "role": "org:admin"}], "total_count": 1})()

    async def list_org_members(self, *, organization_id: str):
        assert organization_id == "org_123"
        return type("Payload", (), {"data": [{"membership_id": "orgmem_1", "user_id": "user_member", "role": "org:member", "first_name": "Jane", "last_name": "Member", "email_address": "jane@example.com", "profile_image_url": None}], "total_count": 1})()

    async def list_org_invitations(self, *, organization_id: str):
        assert organization_id == "org_123"
        return type("Payload", (), {"data": [{"id": "orginv_1", "email_address": "new@example.com", "role": "org:member", "status": "pending", "organization_id": "org_123", "created_at": 1, "updated_at": 1, "expires_at": 2, "inviter_user_id": "user_admin"}], "total_count": 1})()

    async def create_org_invitation(self, *, organization_id: str, inviter_user_id: str, email_address: str, role: str):
        assert organization_id == "org_123"
        assert inviter_user_id == "user_admin"
        return {
            "id": "orginv_2",
            "email_address": email_address,
            "role": role,
            "status": "pending",
            "organization_id": organization_id,
            "created_at": 10,
            "updated_at": 10,
            "expires_at": 20,
            "inviter_user_id": inviter_user_id,
        }

    async def remove_org_member(self, *, organization_id: str, user_id: str):
        assert organization_id == "org_123"
        return {
            "membership_id": "orgmem_1",
            "user_id": user_id,
            "role": "org:member",
            "first_name": "Jane",
            "last_name": "Member",
            "email_address": "jane@example.com",
            "profile_image_url": None,
        }


async def _require_admin() -> OrgContext:
    return OrgContext(org_id="org_123", org_role="org:admin", user_id="user_admin")


async def _require_member() -> OrgContext:
    return OrgContext(org_id="org_123", org_role="org:member", user_id="user_member")


async def _forbid_admin() -> OrgContext:
    raise HTTPException(
        status_code=status.HTTP_403_FORBIDDEN,
        detail="Admin role required.",
    )


def _make_app():
    app = FastAPI()
    app.include_router(admin_router)
    app.include_router(orgs_router)
    app.dependency_overrides[get_clerk_admin_service] = lambda: DummyClerkAdminService()
    return app


def test_admin_routes_require_admin():
    app = _make_app()
    app.dependency_overrides[require_admin] = _forbid_admin
    with TestClient(app) as client:
        response = client.get("/admin/members")

    assert response.status_code == 403


def test_list_orgs_returns_memberships():
    app = _make_app()
    app.dependency_overrides[require_auth] = _require_admin

    with TestClient(app) as client:
        response = client.get("/orgs")

    assert response.status_code == 200
    assert response.json()["data"][0]["organization_id"] == "org_123"


def test_list_members_returns_member_data():
    app = _make_app()
    app.dependency_overrides[require_admin] = _require_admin
    with TestClient(app) as client:
        response = client.get("/admin/members")

    assert response.status_code == 200
    assert response.json()["data"][0]["user_id"] == "user_member"


def test_list_invitations_returns_pending_invitations():
    app = _make_app()
    app.dependency_overrides[require_admin] = _require_admin
    with TestClient(app) as client:
        response = client.get("/admin/invitations")

    assert response.status_code == 200
    assert response.json()["data"][0]["id"] == "orginv_1"


def test_invite_member_returns_created_invitation():
    app = _make_app()
    app.dependency_overrides[require_admin] = _require_admin
    with TestClient(app) as client:
        response = client.post(
            "/admin/invite",
            json={"email_address": "new@example.com", "role": "org:member"},
        )

    assert response.status_code == 201
    assert response.json()["email_address"] == "new@example.com"


def test_remove_member_rejects_self_removal():
    app = _make_app()
    app.dependency_overrides[require_admin] = _require_admin
    with TestClient(app) as client:
        response = client.delete("/admin/members/user_admin")

    assert response.status_code == 400
