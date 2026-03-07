from __future__ import annotations

from fastapi import FastAPI
from fastapi.testclient import TestClient

from codefind_server.config import Settings
from codefind_server.middleware.rate_limit import RateLimitIdentity, RateLimitMiddleware


async def _identity_resolver(_request) -> RateLimitIdentity:
    return RateLimitIdentity(org_id="org_123", user_id="user_123")


def _settings() -> Settings:
    return Settings(
        environment="test",
        vector_store="qdrant",
        qdrant_url="http://localhost:6333",
        ollama_url="http://localhost:11434",
        clerk_iss="https://clerk.example.com",
        clerk_azp="http://localhost:3000",
        clerk_jwks_url="https://clerk.example.com/jwks",
        clerk_secret_key="secret",
        rate_limit_window_seconds=60,
        rate_limit_auth_per_window=1,
        rate_limit_admin_per_window=1,
        rate_limit_query_per_window=1,
        rate_limit_default_per_window=2,
    )


def test_admin_rate_limit_uses_ip_bucket():
    app = FastAPI()
    app.add_middleware(RateLimitMiddleware, settings=_settings())

    @app.get("/admin/test")
    async def admin():
        return {"ok": True}

    with TestClient(app) as client:
        first = client.get("/admin/test")
        second = client.get("/admin/test")

    assert first.status_code == 200
    assert second.status_code == 429
    assert second.headers["Retry-After"] == "59"


def test_query_rate_limit_uses_org_and_user_bucket():
    app = FastAPI()
    app.add_middleware(
        RateLimitMiddleware,
        settings=_settings(),
        identity_resolver=_identity_resolver,
    )

    @app.get("/query/test")
    async def query():
        return {"ok": True}

    with TestClient(app) as client:
        first = client.get("/query/test")
        second = client.get("/query/test")

    assert first.status_code == 200
    assert second.status_code == 429
