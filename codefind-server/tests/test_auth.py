from __future__ import annotations

from dataclasses import dataclass
from datetime import UTC, datetime, timedelta

import jwt
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from fastapi import Depends, FastAPI
from fastapi.testclient import TestClient
import pytest

from codefind_server.config import Settings
from codefind_server.middleware.auth import OrgContext, require_admin, require_auth


@dataclass
class DummySigningKey:
    key: object


class DummyJwkClient:
    def __init__(self, key: object) -> None:
        self._key = key

    def get_signing_key_from_jwt(self, _token: str) -> DummySigningKey:
        return DummySigningKey(key=self._key)


@pytest.fixture
def settings() -> Settings:
    return Settings(
        environment="test",
        web_app_url="http://localhost:5173",
        vector_store="qdrant",
        qdrant_url="http://localhost:6333",
        ollama_url="http://localhost:11434",
        clerk_iss="https://clerk.example.com",
        clerk_azp="http://localhost:3000",
        clerk_jwks_url="https://clerk.example.com/jwks",
        clerk_secret_key="secret",
        audit_log_path=None,
        sentry_dsn=None,
    )


@pytest.fixture
def rsa_keys() -> tuple[bytes, object]:
    private_key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
    private_pem = private_key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.PKCS8,
        encryption_algorithm=serialization.NoEncryption(),
    )
    return private_pem, private_key.public_key()


def _build_token(private_key: bytes, settings: Settings, **claims: str) -> str:
    payload = {
        "sub": "user_123",
        "iss": settings.clerk_iss,
        "azp": settings.clerk_azp,
        "org_id": "org_123",
        "org_role": "org:admin",
        "exp": datetime.now(UTC) + timedelta(minutes=5),
    }
    payload.update(claims)
    return jwt.encode(payload, private_key, algorithm="RS256")


def _build_v2_token(private_key: bytes, settings: Settings, *, org_id: str = "org_123", org_role: str = "admin") -> str:
    payload = {
        "sub": "user_123",
        "iss": settings.clerk_iss,
        "azp": settings.clerk_azp,
        "exp": datetime.now(UTC) + timedelta(minutes=5),
        "o": {
            "id": org_id,
            "rol": org_role,
        },
    }
    return jwt.encode(payload, private_key, algorithm="RS256")


def _make_app() -> FastAPI:
    app = FastAPI()

    @app.get("/protected")
    async def protected(_ctx: OrgContext = Depends(require_auth)):
        return {"ok": True}

    @app.get("/admin")
    async def admin(_ctx: OrgContext = Depends(require_admin)):
        return {"ok": True}

    return app


def test_require_auth_accepts_valid_token(
    monkeypatch: pytest.MonkeyPatch,
    settings: Settings,
    rsa_keys: tuple[bytes, object],
):
    private_key, public_key = rsa_keys
    token = _build_token(private_key, settings)

    monkeypatch.setattr(
        "codefind_server.middleware.auth.get_settings",
        lambda: settings,
    )
    monkeypatch.setattr(
        "codefind_server.middleware.auth.get_jwk_client",
        lambda _url: DummyJwkClient(public_key),
    )

    app = _make_app()
    with TestClient(app) as client:
        response = client.get("/protected", headers={"Authorization": f"Bearer {token}"})

    assert response.status_code == 200
    assert response.json() == {"ok": True}


def test_require_auth_rejects_invalid_token(
    monkeypatch: pytest.MonkeyPatch,
    settings: Settings,
    rsa_keys: tuple[bytes, object],
):
    private_key, public_key = rsa_keys
    token = _build_token(private_key, settings, azp="https://wrong.example.com")

    monkeypatch.setattr(
        "codefind_server.middleware.auth.get_settings",
        lambda: settings,
    )
    monkeypatch.setattr(
        "codefind_server.middleware.auth.get_jwk_client",
        lambda _url: DummyJwkClient(public_key),
    )

    app = _make_app()
    with TestClient(app) as client:
        response = client.get("/protected", headers={"Authorization": f"Bearer {token}"})

    assert response.status_code == 401


def test_require_auth_rejects_missing_org(
    monkeypatch: pytest.MonkeyPatch,
    settings: Settings,
    rsa_keys: tuple[bytes, object],
):
    private_key, public_key = rsa_keys
    token = _build_token(private_key, settings, org_id=None)  # type: ignore[arg-type]

    monkeypatch.setattr(
        "codefind_server.middleware.auth.get_settings",
        lambda: settings,
    )
    monkeypatch.setattr(
        "codefind_server.middleware.auth.get_jwk_client",
        lambda _url: DummyJwkClient(public_key),
    )

    app = _make_app()
    with TestClient(app) as client:
        response = client.get("/protected", headers={"Authorization": f"Bearer {token}"})

    assert response.status_code == 403


def test_require_admin_rejects_member_role(
    monkeypatch: pytest.MonkeyPatch,
    settings: Settings,
    rsa_keys: tuple[bytes, object],
):
    private_key, public_key = rsa_keys
    token = _build_token(private_key, settings, org_role="org:member")

    monkeypatch.setattr(
        "codefind_server.middleware.auth.get_settings",
        lambda: settings,
    )
    monkeypatch.setattr(
        "codefind_server.middleware.auth.get_jwk_client",
        lambda _url: DummyJwkClient(public_key),
    )

    app = _make_app()
    with TestClient(app) as client:
        response = client.get("/admin", headers={"Authorization": f"Bearer {token}"})

    assert response.status_code == 403


def test_require_auth_accepts_v2_nested_org_claims(
    monkeypatch: pytest.MonkeyPatch,
    settings: Settings,
    rsa_keys: tuple[bytes, object],
):
    private_key, public_key = rsa_keys
    token = _build_v2_token(private_key, settings)

    monkeypatch.setattr(
        "codefind_server.middleware.auth.get_settings",
        lambda: settings,
    )
    monkeypatch.setattr(
        "codefind_server.middleware.auth.get_jwk_client",
        lambda _url: DummyJwkClient(public_key),
    )

    app = _make_app()
    with TestClient(app) as client:
        response = client.get("/protected", headers={"Authorization": f"Bearer {token}"})

    assert response.status_code == 200
    assert response.json() == {"ok": True}
