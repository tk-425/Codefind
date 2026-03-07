from __future__ import annotations

import pytest

from codefind_server.config import SettingsError, get_settings


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

