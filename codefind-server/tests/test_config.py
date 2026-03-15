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


def test_settings_reject_non_positive_ollama_embed_batch_size():
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
        ollama_embed_batch_size=0,
    )

    with pytest.raises(SettingsError, match="OLLAMA_EMBED_BATCH_SIZE"):
        settings.validate_required()


def test_settings_reject_non_positive_ollama_embed_timeout_seconds():
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
        ollama_embed_timeout_seconds=0,
    )

    with pytest.raises(SettingsError, match="OLLAMA_EMBED_TIMEOUT_SECONDS"):
        settings.validate_required()


def test_settings_reject_non_positive_ollama_embed_max_attempts():
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
        ollama_embed_max_attempts=0,
    )

    with pytest.raises(SettingsError, match="OLLAMA_EMBED_MAX_ATTEMPTS"):
        settings.validate_required()


def test_settings_reject_non_positive_ollama_embed_retry_backoff_seconds():
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
        ollama_embed_retry_backoff_seconds=0,
    )

    with pytest.raises(SettingsError, match="OLLAMA_EMBED_RETRY_BACKOFF_SECONDS"):
        settings.validate_required()


def test_get_settings_reads_ollama_embed_batch_size(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setenv("VECTOR_STORE", "qdrant")
    monkeypatch.setenv("QDRANT_URL", "http://localhost:6333")
    monkeypatch.setenv("OLLAMA_URL", "http://localhost:11434")
    monkeypatch.setenv("OLLAMA_EMBED_BATCH_SIZE", "48")
    monkeypatch.setenv("CLERK_ISS", "https://clerk.example.com")
    monkeypatch.setenv("CLERK_AZP", "http://localhost:3000")
    monkeypatch.setenv("CLERK_JWKS_URL", "https://clerk.example.com/jwks")
    monkeypatch.setenv("CLERK_SECRET_KEY", "secret")

    settings = get_settings()

    assert settings.ollama_embed_batch_size == 48


def test_get_settings_reads_ollama_embed_timeout_seconds(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setenv("VECTOR_STORE", "qdrant")
    monkeypatch.setenv("QDRANT_URL", "http://localhost:6333")
    monkeypatch.setenv("OLLAMA_URL", "http://localhost:11434")
    monkeypatch.setenv("OLLAMA_EMBED_TIMEOUT_SECONDS", "420")
    monkeypatch.setenv("CLERK_ISS", "https://clerk.example.com")
    monkeypatch.setenv("CLERK_AZP", "http://localhost:3000")
    monkeypatch.setenv("CLERK_JWKS_URL", "https://clerk.example.com/jwks")
    monkeypatch.setenv("CLERK_SECRET_KEY", "secret")

    settings = get_settings()

    assert settings.ollama_embed_timeout_seconds == 420


def test_get_settings_reads_ollama_retry_settings(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setenv("VECTOR_STORE", "qdrant")
    monkeypatch.setenv("QDRANT_URL", "http://localhost:6333")
    monkeypatch.setenv("OLLAMA_URL", "http://localhost:11434")
    monkeypatch.setenv("OLLAMA_EMBED_MAX_ATTEMPTS", "5")
    monkeypatch.setenv("OLLAMA_EMBED_RETRY_BACKOFF_SECONDS", "2.5")
    monkeypatch.setenv("CLERK_ISS", "https://clerk.example.com")
    monkeypatch.setenv("CLERK_AZP", "http://localhost:3000")
    monkeypatch.setenv("CLERK_JWKS_URL", "https://clerk.example.com/jwks")
    monkeypatch.setenv("CLERK_SECRET_KEY", "secret")

    settings = get_settings()

    assert settings.ollama_embed_max_attempts == 5
    assert settings.ollama_embed_retry_backoff_seconds == 2.5


def test_get_settings_reads_sparse_retrieval_settings(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setenv("VECTOR_STORE", "qdrant")
    monkeypatch.setenv("QDRANT_URL", "http://localhost:6333")
    monkeypatch.setenv("OLLAMA_URL", "http://localhost:11434")
    monkeypatch.setenv("SPARSE_RETRIEVAL_ENABLED", "false")
    monkeypatch.setenv("SPARSE_EMBED_MODEL", "custom/splade")
    monkeypatch.setenv("SPARSE_EMBED_BATCH_SIZE", "8")
    monkeypatch.setenv("CLERK_ISS", "https://clerk.example.com")
    monkeypatch.setenv("CLERK_AZP", "http://localhost:3000")
    monkeypatch.setenv("CLERK_JWKS_URL", "https://clerk.example.com/jwks")
    monkeypatch.setenv("CLERK_SECRET_KEY", "secret")

    settings = get_settings()

    assert settings.sparse_retrieval_enabled is False
    assert settings.sparse_embed_model == "custom/splade"
    assert settings.sparse_embed_batch_size == 8


def test_get_settings_reads_sparse_cache_dir(monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setenv("VECTOR_STORE", "qdrant")
    monkeypatch.setenv("QDRANT_URL", "http://localhost:6333")
    monkeypatch.setenv("OLLAMA_URL", "http://localhost:11434")
    monkeypatch.setenv("SPARSE_EMBED_CACHE_DIR", "~/.codefind-model-cache/fastembed")
    monkeypatch.setenv("CLERK_ISS", "https://clerk.example.com")
    monkeypatch.setenv("CLERK_AZP", "http://localhost:3000")
    monkeypatch.setenv("CLERK_JWKS_URL", "https://clerk.example.com/jwks")
    monkeypatch.setenv("CLERK_SECRET_KEY", "secret")

    settings = get_settings()

    assert settings.sparse_embed_cache_dir == "~/.codefind-model-cache/fastembed"


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
