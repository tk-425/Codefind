from __future__ import annotations

from fastapi import FastAPI
from fastapi.testclient import TestClient
import pytest

from codefind_server.config import Settings
from codefind_server.routes.auth import router


@pytest.fixture
def settings() -> Settings:
    return Settings(
        environment="test",
        web_app_url="http://localhost:5173",
        vector_store="qdrant",
        qdrant_url="http://localhost:6333",
        ollama_url="http://localhost:11434",
        clerk_iss="https://clerk.example.com",
        clerk_azp="http://localhost:5173",
        clerk_jwks_url="https://clerk.example.com/jwks",
        clerk_secret_key="secret",
        audit_log_path=None,
        sentry_dsn=None,
    )


def _make_app() -> FastAPI:
    app = FastAPI()
    app.include_router(router)
    return app


def test_auth_signin_redirects_to_local_web_app(
    monkeypatch: pytest.MonkeyPatch,
    settings: Settings,
):
    monkeypatch.setattr("codefind_server.routes.auth.get_settings", lambda: settings)
    app = _make_app()

    with TestClient(app) as client:
        response = client.get("/auth/signin", follow_redirects=False)

    assert response.status_code == 307
    assert response.headers["location"] == "http://localhost:5173/signin"
    assert response.headers["cache-control"] == "no-store"


def test_auth_signin_preserves_valid_cli_callback(
    monkeypatch: pytest.MonkeyPatch,
    settings: Settings,
):
    monkeypatch.setattr("codefind_server.routes.auth.get_settings", lambda: settings)
    app = _make_app()

    with TestClient(app) as client:
        response = client.get(
            "/auth/signin",
            params={"redirect_uri": "http://127.0.0.1:49152/callback"},
            follow_redirects=False,
        )

    assert response.status_code == 307
    assert (
        response.headers["location"]
        == "http://localhost:5173/signin?redirect_uri=http%3A%2F%2F127.0.0.1%3A49152%2Fcallback"
    )


@pytest.mark.parametrize(
    "redirect_uri",
    [
        "http://localhost:49152/callback",
        "http://127.0.0.1/callback",
        "http://127.0.0.1:49152/wrong",
        "https://127.0.0.1:49152/callback",
        "http://192.168.0.5:49152/callback",
    ],
)
def test_auth_signin_rejects_non_localhost_callback(
    monkeypatch: pytest.MonkeyPatch,
    settings: Settings,
    redirect_uri: str,
):
    monkeypatch.setattr("codefind_server.routes.auth.get_settings", lambda: settings)
    app = _make_app()

    with TestClient(app) as client:
        response = client.get("/auth/signin", params={"redirect_uri": redirect_uri})

    assert response.status_code == 400
    assert response.json()["detail"] == "redirect_uri must match http://127.0.0.1:<port>/callback"
