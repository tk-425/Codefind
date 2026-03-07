from __future__ import annotations

from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.testclient import TestClient

from codefind_server.routes.health import router


class DummyVectorStore:
    async def healthcheck(self) -> bool:
        return True


@asynccontextmanager
async def lifespan_override(app: FastAPI):
    class Settings:
        ollama_url = "http://localhost:11434"

    app.state.settings = Settings()
    app.state.vector_store = DummyVectorStore()
    yield


def test_health_route_is_unauthenticated(monkeypatch):
    async def always_ok(_base_url: str) -> bool:
        return True

    monkeypatch.setattr("codefind_server.routes.health._check_ollama", always_ok)
    app = FastAPI(lifespan=lifespan_override)
    app.include_router(router)

    with TestClient(app) as client:
        response = client.get("/health")

    assert response.status_code == 200
    assert response.json()["status"] == "ok"
