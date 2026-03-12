from __future__ import annotations

from codefind_server.adapters.base import SearchResult
from codefind_server.routes.query import _rerank_results


def _candidate(
    *,
    result_id: str,
    score: float,
    path: str,
    snippet: str,
    language: str = "go",
    symbol_name: str | None = None,
    symbol_kind: str | None = None,
    chunking_method: str | None = None,
):
    return (
        "org_123_repo-a",
        SearchResult(
            id=result_id,
            score=score,
            payload={
                "repo_id": "repo-a",
                "project": "codefind",
                "language": language,
                "path": path,
                "snippet": snippet,
                "content": snippet,
                "symbol_name": symbol_name,
                "symbol_kind": symbol_kind,
                "chunking_method": chunking_method,
            },
        ),
    )


def test_ranking_eval_prefers_definitions_for_implementation_queries():
    reranked = _rerank_results(
        query_text="where is the clerk auth function",
        combined=[
            _candidate(
                result_id="ref-assignment",
                score=0.94,
                path="cmd/codefind/cli_runtime.go",
                snippet="startCallbackServer = authflow.StartCallbackServer",
            ),
            _candidate(
                result_id="definition",
                score=0.89,
                path="cmd/codefind/commands_auth.go",
                snippet='func newAuthLoginCommand(configPath *string) *cobra.Command {',
                symbol_name="newAuthLoginCommand",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="test",
                score=0.92,
                path="codefind-server/tests/test_auth.py",
                language="python",
                snippet="async def protected(_ctx: OrgContext = Depends(require_auth)): return {'ok': True}",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids[:3] == ["definition", "ref-assignment", "test"]


def test_ranking_eval_prefers_references_for_reference_queries():
    reranked = _rerank_results(
        query_text="who calls BuildSignInURL",
        combined=[
            _candidate(
                result_id="definition",
                score=0.92,
                path="internal/authflow/login.go",
                snippet="func BuildSignInURL(baseURL string) string {",
                symbol_name="BuildSignInURL",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="reference",
                score=0.90,
                path="cmd/codefind/cli_runtime.go",
                snippet="buildSignInURL = authflow.BuildSignInURL",
            ),
        ],
    )

    assert [result.id for _, result, _ in reranked][:2] == ["reference", "definition"]


def test_ranking_eval_prefers_tests_for_test_queries():
    reranked = _rerank_results(
        query_text="test for BuildSignInURL",
        combined=[
            _candidate(
                result_id="definition",
                score=0.93,
                path="internal/authflow/login.go",
                snippet="func BuildSignInURL(baseURL string) string {",
                symbol_name="BuildSignInURL",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="test",
                score=0.88,
                path="internal/authflow/login_test.go",
                snippet="func TestBuildSignInURL(t *testing.T) {",
            ),
        ],
    )

    assert [result.id for _, result, _ in reranked][:2] == ["test", "definition"]


def test_ranking_eval_prefers_config_paths_for_config_queries():
    reranked = _rerank_results(
        query_text="where is the clerk env config",
        combined=[
            _candidate(
                result_id="implementation",
                score=0.91,
                path="web/src/lib/auth.ts",
                language="typescript",
                snippet="const CLI_REDIRECT_STORAGE_KEY = 'codefind.cli_redirect_uri'",
            ),
            _candidate(
                result_id="config",
                score=0.86,
                path="codefind-server/.env.example",
                snippet="CLERK_ISS=",
                language="dotenv",
            ),
        ],
    )

    assert [result.id for _, result, _ in reranked][:2] == ["config", "implementation"]


def test_ranking_eval_prefers_real_auth_definition_over_symbol_body_reference():
    reranked = _rerank_results(
        query_text="where is the clerk auth function",
        combined=[
            _candidate(
                result_id="ref-assignment",
                score=0.94,
                path="cmd/codefind/cli_runtime.go",
                snippet="startCallbackServer = authflow.StartCallbackServer",
                symbol_name="runBrowserLogin",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="body-constant",
                score=0.90,
                path="internal/authflow/login.go",
                snippet='callbackResponseBody  = "Authentication received. Return to the Code-Find CLI."',
                symbol_name="StartCallbackServer",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="definition",
                score=0.86,
                path="internal/authflow/login.go",
                snippet="func StartCallbackServer(",
                symbol_name="StartCallbackServer",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="constant",
                score=0.88,
                path="web/src/lib/auth.ts",
                language="typescript",
                snippet="const CLI_REDIRECT_STORAGE_KEY = 'codefind.cli_redirect_uri'",
                symbol_name="CLI_REDIRECT_STORAGE_KEY",
                symbol_kind="constant",
                chunking_method="symbol",
            ),
        ],
    )

    assert [result.id for _, result, _ in reranked][:4] == [
        "definition",
        "body-constant",
        "ref-assignment",
        "constant",
    ]
