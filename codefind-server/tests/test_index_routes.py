from __future__ import annotations

from fastapi import FastAPI, HTTPException, status
from fastapi.testclient import TestClient

from codefind_server.middleware.auth import OrgContext, require_admin
from codefind_server.routes.index import get_indexing_service, router as index_router


class DummyIndexingService:
    def __init__(self) -> None:
        self.index_calls: list[dict[str, object]] = []
        self.status_calls: list[dict[str, object]] = []
        self.list_calls: list[dict[str, object]] = []
        self.purge_calls: list[dict[str, object]] = []

    async def index_chunks(self, *, org_id: str, request):
        self.index_calls.append({"org_id": org_id, "request": request})
        return {
            "status": "ok",
            "repo_id": request.repo_id,
            "indexed_count": len(request.chunks),
            "accepted": True,
        }

    async def update_chunk_status(self, *, org_id: str, request):
        self.status_calls.append({"org_id": org_id, "request": request})
        return {
            "status": "ok",
            "repo_id": request.repo_id,
            "updated_count": len(request.chunk_ids),
        }

    async def list_tombstoned_chunks(self, *, org_id: str, repo_id: str):
        self.list_calls.append({"org_id": org_id, "repo_id": repo_id})
        return {
            "status": "ok",
            "repo_id": repo_id,
            "found_count": 2,
            "files": [
                {"path": "main.go", "chunk_count": 2, "tombstoned_at": "2026-03-09T00:00:00Z"}
            ],
        }

    async def purge_tombstoned_chunks(self, *, org_id: str, request):
        self.purge_calls.append({"org_id": org_id, "request": request})
        return {
            "status": "ok",
            "repo_id": request.repo_id,
            "found_count": 1,
            "purged_count": 1,
            "files": [
                {"path": "old.go", "chunk_count": 1, "tombstoned_at": "2026-02-01T00:00:00Z"}
            ],
        }


async def _require_admin() -> OrgContext:
    return OrgContext(org_id="org_123", org_role="org:admin", user_id="user_admin")


async def _forbid_admin() -> OrgContext:
    raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail="Admin role required.")


def _make_app(service: DummyIndexingService) -> FastAPI:
    app = FastAPI()
    app.include_router(index_router)
    app.dependency_overrides[get_indexing_service] = lambda: service
    return app


def test_index_route_requires_admin():
    app = _make_app(DummyIndexingService())
    app.dependency_overrides[require_admin] = _forbid_admin

    with TestClient(app) as client:
        response = client.post(
            "/index",
            json={
                "repo_id": "repo-a",
                "chunks": [
                    {
                        "id": "chunk-1",
                        "content": "func main() {}",
                        "metadata": {
                            "repo_id": "repo-a",
                            "path": "main.go",
                            "language": "go",
                            "start_line": 1,
                            "end_line": 1,
                            "content_hash": "hash-1",
                            "status": "active",
                            "chunking_method": "window",
                        },
                    }
                ],
            },
        )

    assert response.status_code == 403


def test_index_route_passes_org_scoped_chunks_to_service():
    service = DummyIndexingService()
    app = _make_app(service)
    app.dependency_overrides[require_admin] = _require_admin

    with TestClient(app) as client:
        response = client.post(
            "/index",
            json={
                "repo_id": "repo-a",
                "chunks": [
                    {
                        "id": "chunk-1",
                        "content": "func main() {}",
                        "metadata": {
                            "repo_id": "repo-a",
                            "path": "main.go",
                            "language": "go",
                            "start_line": 1,
                            "end_line": 1,
                            "content_hash": "hash-1",
                            "status": "active",
                            "chunking_method": "window",
                        },
                    }
                ],
            },
        )

    assert response.status_code == 200
    assert response.json()["indexed_count"] == 1
    assert len(service.index_calls) == 1
    assert service.index_calls[0]["org_id"] == "org_123"
    assert service.index_calls[0]["request"].repo_id == "repo-a"


def test_chunk_status_route_updates_tombstones():
    service = DummyIndexingService()
    app = _make_app(service)
    app.dependency_overrides[require_admin] = _require_admin

    with TestClient(app) as client:
        response = client.patch(
            "/chunks/status",
            json={
                "repo_id": "repo-a",
                "chunk_ids": ["chunk-1", "chunk-2"],
                "status": "tombstoned",
            },
        )

    assert response.status_code == 200
    assert response.json()["updated_count"] == 2
    assert len(service.status_calls) == 1
    assert service.status_calls[0]["org_id"] == "org_123"
    assert service.status_calls[0]["request"].chunk_ids == ["chunk-1", "chunk-2"]


def test_tombstoned_chunk_list_route_is_admin_only():
    app = _make_app(DummyIndexingService())
    app.dependency_overrides[require_admin] = _forbid_admin

    with TestClient(app) as client:
        response = client.get("/chunks/tombstoned", params={"repo_id": "repo-a"})

    assert response.status_code == 403


def test_tombstoned_chunk_list_route_returns_repo_scoped_summaries():
    service = DummyIndexingService()
    app = _make_app(service)
    app.dependency_overrides[require_admin] = _require_admin

    with TestClient(app) as client:
        response = client.get("/chunks/tombstoned", params={"repo_id": "repo-a"})

    assert response.status_code == 200
    assert response.json()["found_count"] == 2
    assert response.json()["files"][0]["path"] == "main.go"
    assert service.list_calls == [{"org_id": "org_123", "repo_id": "repo-a"}]


def test_purge_route_returns_purge_result():
    service = DummyIndexingService()
    app = _make_app(service)
    app.dependency_overrides[require_admin] = _require_admin

    with TestClient(app) as client:
        response = client.request(
            "DELETE",
            "/chunks/purge",
            json={"repo_id": "repo-a", "older_than_days": 30},
        )

    assert response.status_code == 200
    assert response.json()["purged_count"] == 1
    assert len(service.purge_calls) == 1
    assert service.purge_calls[0]["org_id"] == "org_123"
    assert service.purge_calls[0]["request"].older_than_days == 30
