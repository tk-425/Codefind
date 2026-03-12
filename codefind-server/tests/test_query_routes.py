from __future__ import annotations

from fastapi import FastAPI
from fastapi.testclient import TestClient

from codefind_server.adapters.base import SearchResult
from codefind_server.middleware.auth import OrgContext, require_auth
from codefind_server.routes.collections import router as collections_router
from codefind_server.routes.query import router as query_router
from codefind_server.routes.stats import router as stats_router
from codefind_server.routes.tokenize import router as tokenize_router


class DummyVectorStore:
    def __init__(self) -> None:
        self.query_calls: list[dict[str, object]] = []
        self.count_calls: list[dict[str, object]] = []

    async def list_collections(self) -> list[str]:
        return ["org_123_repo-a", "org_123_repo-b", "org_other_repo-z"]

    async def query(self, collection: str, vector: list[float], filters: dict[str, object], top_k: int):
        self.query_calls.append(
            {
                "collection": collection,
                "vector": vector,
                "filters": filters,
                "top_k": top_k,
            }
        )
        return [
            SearchResult(
                id=f"{collection}:chunk-1",
                score=0.95 if collection.endswith("repo-a") else 0.75,
                payload={
                    "repo_id": "repo-a" if collection.endswith("repo-a") else "repo-b",
                    "project": "codefind",
                    "language": "go",
                    "path": "cmd/codefind/main.go",
                    "snippet": "func main() {}",
                    "content": "func main() {}",
                    "symbol_name": "main",
                    "chunking_method": "symbol",
                },
            )
        ]

    async def count(self, collection: str, filters: dict[str, object]) -> int:
        self.count_calls.append({"collection": collection, "filters": filters})
        counts = {
            ("org_123_repo-a", "active"): 12,
            ("org_123_repo-a", "tombstoned"): 3,
            ("org_123_repo-b", "active"): 8,
            ("org_123_repo-b", "tombstoned"): 1,
        }
        status = filters.get("status")
        return counts.get((collection, status), 0)


class DummyOllama:
    async def embed(self, text: str):
        assert text
        return type("EmbeddingResponse", (), {"embedding": [0.1, 0.2, 0.3]})()


class DummyTokenizer:
    model_name = "bert-base-uncased"

    def tokenize(self, text: str) -> list[str]:
        return text.split()


async def _require_auth() -> OrgContext:
    return OrgContext(org_id="org_123", org_role="org:member", user_id="user_123")


def _make_app() -> FastAPI:
    app = FastAPI()
    app.include_router(collections_router)
    app.include_router(stats_router)
    app.include_router(query_router)
    app.include_router(tokenize_router)
    app.state.vector_store = DummyVectorStore()
    app.state.ollama = DummyOllama()
    app.state.tokenizer = DummyTokenizer()
    app.dependency_overrides[require_auth] = _require_auth
    return app


def test_list_collections_only_returns_current_org_repos():
    app = _make_app()
    with TestClient(app) as client:
        response = client.get("/collections")

    assert response.status_code == 200
    assert response.json() == {
        "data": [{"repo_id": "repo-a"}, {"repo_id": "repo-b"}],
        "total_count": 2,
    }


def test_stats_are_org_scoped():
    app = _make_app()
    vector_store: DummyVectorStore = app.state.vector_store
    with TestClient(app) as client:
        response = client.get("/stats")

    assert response.status_code == 200
    assert response.json()["repo_count"] == 2
    assert response.json()["chunk_count"] == 20
    assert response.json()["active_chunks"] == 20
    assert response.json()["deleted_chunks"] == 4
    assert response.json()["total_chunks"] == 24
    assert response.json()["overhead_percent"] == 20.0
    assert vector_store.count_calls == [
        {"collection": "org_123_repo-a", "filters": {"status": "active"}},
        {"collection": "org_123_repo-a", "filters": {"status": "tombstoned"}},
        {"collection": "org_123_repo-b", "filters": {"status": "active"}},
        {"collection": "org_123_repo-b", "filters": {"status": "tombstoned"}},
    ]


def test_query_searches_only_current_org_collections_and_clamps_top_k():
    app = _make_app()
    vector_store: DummyVectorStore = app.state.vector_store
    with TestClient(app) as client:
        response = client.post(
            "/query",
            json={
                "query_text": "main function",
                "project": "codefind",
                "language": "go",
                "top_k": 999,
                "page": 1,
                "page_size": 10,
            },
        )

    assert response.status_code == 200
    assert len(vector_store.query_calls) == 2
    assert {call["collection"] for call in vector_store.query_calls} == {
        "org_123_repo-a",
        "org_123_repo-b",
    }
    assert all(call["top_k"] == 100 for call in vector_store.query_calls)
    assert all(
        call["filters"] == {"status": "active", "project": "codefind", "language": "go"}
        for call in vector_store.query_calls
    )
    assert response.json()["total_count"] == 2


def test_query_uses_deeper_candidate_pool_than_page_size():
    app = _make_app()
    vector_store: DummyVectorStore = app.state.vector_store

    with TestClient(app) as client:
        response = client.post(
            "/query",
            json={"query_text": "main function", "repo_id": "repo-a", "top_k": 10, "page": 1, "page_size": 10},
        )

    assert response.status_code == 200
    assert vector_store.query_calls[0]["top_k"] == 50


def test_query_prefers_definition_like_chunks_for_implementation_queries():
    app = _make_app()

    class RankingVectorStore(DummyVectorStore):
        async def query(self, collection: str, vector: list[float], filters: dict[str, object], top_k: int):
            return [
                SearchResult(
                    id="ref-1",
                    score=0.94,
                    payload={
                        "repo_id": "repo-a",
                        "project": "codefind",
                        "language": "go",
                        "path": "cmd/codefind/cli_runtime.go",
                        "snippet": "startCallbackServer = authflow.StartCallbackServer",
                        "content": "startCallbackServer = authflow.StartCallbackServer",
                    },
                ),
                SearchResult(
                    id="test-1",
                    score=0.93,
                    payload={
                        "repo_id": "repo-a",
                        "project": "codefind",
                        "language": "python",
                        "path": "codefind-server/tests/test_auth.py",
                        "snippet": "async def protected(_ctx: OrgContext = Depends(require_auth)): return {'ok': True}",
                        "content": "async def protected(_ctx: OrgContext = Depends(require_auth)): return {'ok': True}",
                    },
                ),
                SearchResult(
                    id="def-1",
                    score=0.89,
                    payload={
                        "repo_id": "repo-a",
                        "project": "codefind",
                        "language": "go",
                        "path": "internal/authflow/login.go",
                        "snippet": "func BuildSignInURL(baseURL string) string {",
                        "content": "func BuildSignInURL(baseURL string) string {",
                        "symbol_name": "BuildSignInURL",
                        "symbol_kind": "function",
                        "chunking_method": "symbol",
                    },
                ),
            ]

    app.state.vector_store = RankingVectorStore()

    with TestClient(app) as client:
        response = client.post(
            "/query",
            json={"query_text": "where is the clerk auth function", "repo_id": "repo-a", "top_k": 10, "page": 1, "page_size": 10},
        )

    assert response.status_code == 200
    data = response.json()["data"]
    assert data[0]["id"] == "def-1"
    assert data[1]["id"] == "ref-1"
    assert data[2]["id"] == "test-1"


def test_query_prefers_reference_like_chunks_for_reference_queries():
    app = _make_app()

    class ReferenceVectorStore(DummyVectorStore):
        async def query(self, collection: str, vector: list[float], filters: dict[str, object], top_k: int):
            return [
                SearchResult(
                    id="def-1",
                    score=0.92,
                    payload={
                        "repo_id": "repo-a",
                        "project": "codefind",
                        "language": "go",
                        "path": "internal/authflow/login.go",
                        "snippet": "func BuildSignInURL(baseURL string) string {",
                        "content": "func BuildSignInURL(baseURL string) string {",
                        "symbol_name": "BuildSignInURL",
                        "symbol_kind": "function",
                        "chunking_method": "symbol",
                    },
                ),
                SearchResult(
                    id="ref-1",
                    score=0.90,
                    payload={
                        "repo_id": "repo-a",
                        "project": "codefind",
                        "language": "go",
                        "path": "cmd/codefind/cli_runtime.go",
                        "snippet": "buildSignInURL = authflow.BuildSignInURL",
                        "content": "buildSignInURL = authflow.BuildSignInURL",
                    },
                ),
            ]

    app.state.vector_store = ReferenceVectorStore()

    with TestClient(app) as client:
        response = client.post(
            "/query",
            json={"query_text": "who calls BuildSignInURL", "repo_id": "repo-a", "top_k": 10, "page": 1, "page_size": 10},
        )

    assert response.status_code == 200
    data = response.json()["data"]
    assert data[0]["id"] == "ref-1"
    assert data[1]["id"] == "def-1"


def test_query_prefers_tests_for_test_queries():
    app = _make_app()

    class TestIntentVectorStore(DummyVectorStore):
        async def query(self, collection: str, vector: list[float], filters: dict[str, object], top_k: int):
            return [
                SearchResult(
                    id="def-1",
                    score=0.93,
                    payload={
                        "repo_id": "repo-a",
                        "project": "codefind",
                        "language": "go",
                        "path": "internal/authflow/login.go",
                        "snippet": "func BuildSignInURL(baseURL string) string {",
                        "content": "func BuildSignInURL(baseURL string) string {",
                        "symbol_name": "BuildSignInURL",
                        "symbol_kind": "function",
                        "chunking_method": "symbol",
                    },
                ),
                SearchResult(
                    id="test-1",
                    score=0.88,
                    payload={
                        "repo_id": "repo-a",
                        "project": "codefind",
                        "language": "go",
                        "path": "internal/authflow/login_test.go",
                        "snippet": "func TestBuildSignInURL(t *testing.T) {",
                        "content": "func TestBuildSignInURL(t *testing.T) {",
                    },
                ),
            ]

    app.state.vector_store = TestIntentVectorStore()

    with TestClient(app) as client:
        response = client.post(
            "/query",
            json={"query_text": "test for BuildSignInURL", "repo_id": "repo-a", "top_k": 10, "page": 1, "page_size": 10},
        )

    assert response.status_code == 200
    data = response.json()["data"]
    assert data[0]["id"] == "test-1"
    assert data[1]["id"] == "def-1"


def test_query_rejects_invalid_repo_id():
    app = _make_app()
    with TestClient(app) as client:
        response = client.post(
            "/query",
            json={
                "query_text": "main function",
                "repo_id": "../bad",
                "page": 1,
                "page_size": 10,
                "top_k": 10,
            },
        )

    assert response.status_code == 400


def test_tokenize_returns_token_count():
    app = _make_app()
    with TestClient(app) as client:
        response = client.post("/tokenize", json={"text": "alpha beta gamma"})

    assert response.status_code == 200
    assert response.json() == {
        "model": "bert-base-uncased",
        "tokens": ["alpha", "beta", "gamma"],
        "token_count": 3,
    }
