from __future__ import annotations

from fastapi import FastAPI, Request
from fastapi.testclient import TestClient

from codefind_server.middleware import REQUEST_ID_HEADER, request_context_middleware


def test_request_context_sets_and_returns_request_id():
    app = FastAPI()
    app.middleware("http")(request_context_middleware)

    @app.get("/context")
    async def context(request: Request):
        ctx = request.state.request_context
        return {"request_id": ctx.request_id, "path": ctx.path}

    with TestClient(app) as client:
        response = client.get("/context", headers={REQUEST_ID_HEADER: "req-123"})

    assert response.status_code == 200
    assert response.headers[REQUEST_ID_HEADER] == "req-123"
    assert response.json() == {"request_id": "req-123", "path": "/context"}
