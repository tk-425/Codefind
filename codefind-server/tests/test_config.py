from __future__ import annotations

from pathlib import Path

import pytest

from codefind_server.config import REPO_ROOT, Settings, SettingsError, get_settings


@pytest.fixture(autouse=True)
def clear_settings_cache():
    get_settings.cache_clear()
    yield
    get_settings.cache_clear()


def test_get_settings_raises_for_missing_required_env(monkeypatch: pytest.MonkeyPatch):
    for name in (
        "VECTOR_STORE",
        "QDRANT_URL",
        "OLLAMA_URL",
        "CLERK_ISS",
        "CLERK_AZP",
        "CLERK_JWKS_URL",
        "CLERK_SECRET_KEY",
    ):
        monkeypatch.delenv(name, raising=False)

    with pytest.raises(SettingsError):
        get_settings()


def test_settings_reject_audit_log_path_inside_repo(tmp_path: Path):
    audit_path = REPO_ROOT / "audit.jsonl"
    settings = Settings(
        environment="test",
        web_app_url="http://localhost:5173",
        vector_store="qdrant",
        qdrant_url="http://localhost:6333",
        ollama_url="http://localhost:11434",
        clerk_iss="https://clerk.example.com",
        clerk_azp="http://localhost:3000",
        clerk_jwks_url="https://clerk.example.com/jwks",
        clerk_secret_key="secret",
        audit_log_path=str(audit_path),
    )

    with pytest.raises(SettingsError, match="outside the repository"):
        settings.validate_required()


def test_settings_reject_non_positive_limits():
    settings = Settings(
        environment="test",
        web_app_url="http://localhost:5173",
        vector_store="qdrant",
        qdrant_url="http://localhost:6333",
        ollama_url="http://localhost:11434",
        clerk_iss="https://clerk.example.com",
        clerk_azp="http://localhost:3000",
        clerk_jwks_url="https://clerk.example.com/jwks",
        clerk_secret_key="secret",
        rate_limit_auth_per_window=0,
    )

    with pytest.raises(SettingsError, match="positive integer"):
        settings.validate_required()


def test_settings_reject_invalid_web_app_url():
    settings = Settings(
        environment="test",
        web_app_url="localhost:5173",
        vector_store="qdrant",
        qdrant_url="http://localhost:6333",
        ollama_url="http://localhost:11434",
        clerk_iss="https://clerk.example.com",
        clerk_azp="http://localhost:3000",
        clerk_jwks_url="https://clerk.example.com/jwks",
        clerk_secret_key="secret",
    )

    with pytest.raises(SettingsError, match="WEB_APP_URL"):
        settings.validate_required()
