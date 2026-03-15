from __future__ import annotations

from codefind_server.adapters.base import SearchResult
from codefind_server.services.query_ranking import rerank_results


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
    reranked = rerank_results(
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
    reranked = rerank_results(
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


def test_ranking_eval_keeps_direct_alias_high_for_defined_queries():
    reranked = rerank_results(
        query_text="where is BuildSignInURL defined",
        combined=[
            _candidate(
                result_id="definition",
                score=0.90,
                path="internal/authflow/login.go",
                snippet="func BuildSignInURL(baseURL string) string {",
                symbol_name="BuildSignInURL",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="alias",
                score=0.78,
                path="cmd/codefind/cli_runtime.go",
                snippet="buildSignInURL = authflow.BuildSignInURL",
            ),
            _candidate(
                result_id="semantic-neighbor",
                score=0.80,
                path="codefind-server/src/codefind_server/routes/auth.py",
                language="python",
                snippet="def build_signin_redirect(web_app_url: str, redirect_uri: str | None) -> str:",
                symbol_name="build_signin_redirect",
                symbol_kind="function",
                chunking_method="symbol",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids[0] == "definition"
    assert ids.index("alias") < ids.index("semantic-neighbor")


def test_ranking_eval_prefers_tests_for_test_queries():
    reranked = rerank_results(
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
    reranked = rerank_results(
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
    reranked = rerank_results(
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


def test_ranking_eval_prefers_require_auth_users_over_auth_adjacent_noise():
    reranked = rerank_results(
        query_text="who uses require_auth",
        combined=[
            _candidate(
                result_id="test-user",
                score=0.90,
                path="codefind-server/tests/test_auth.py",
                language="python",
                snippet='@app.get("/protected") async def protected(_ctx: OrgContext = Depends(require_auth)): return {"ok": True}',
                symbol_name="protected",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="route-user",
                score=0.82,
                path="codefind-server/src/codefind_server/routes/tokenize.py",
                language="python",
                snippet="_context=Depends(require_auth)",
                symbol_name="tokenize_text",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="middleware-definition",
                score=0.84,
                path="codefind-server/src/codefind_server/middleware/auth.py",
                language="python",
                snippet="async def require_auth(",
                symbol_name="require_auth",
                symbol_kind="function",
                chunking_method="symbol",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids.index("route-user") < ids.index("test-user")
    assert ids.index("middleware-definition") < ids.index("test-user")


def test_ranking_eval_demotes_ranking_tests_for_usage_queries():
    reranked = rerank_results(
        query_text="who uses require_auth",
        combined=[
            _candidate(
                result_id="ranking-test",
                score=0.94,
                path="codefind-server/tests/test_query_ranking_eval.py",
                language="python",
                snippet='def test_ranking_eval_prefers_require_auth_users_over_auth_adjacent_noise(): reranked = rerank_results(query_text="who uses require_auth", combined=[...])',
                symbol_name="test_ranking_eval_prefers_require_auth_users_over_auth_adjacent_noise",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="route-user",
                score=0.30,
                path="codefind-server/src/codefind_server/routes/query.py",
                language="python",
                snippet='async def query_collections(..., context: OrgContext = Depends(require_auth), ...):',
                symbol_name="query_collections",
                symbol_kind="function",
                chunking_method="symbol",
            ),
        ],
    )

    assert [result.id for _, result, _ in reranked][:2] == ["route-user", "ranking-test"]


def test_ranking_eval_prefers_actual_publishable_key_usage():
    reranked = rerank_results(
        query_text="where is clerk publishable key used",
        combined=[
            _candidate(
                result_id="lsp-noise",
                score=0.90,
                path="internal/lsp/discovery.go",
                snippet='func LSPKeyForLanguage(language string) string { return "typescript/javascript" }',
                symbol_name="LSPKeyForLanguage",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="main-usage",
                score=0.82,
                path="web/src/main.tsx",
                language="typescript",
                snippet="const clerkPublishableKey = import.meta.env.VITE_CLERK_PUBLISHABLE_KEY",
                symbol_name="clerkPublishableKey",
                symbol_kind="constant",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="config-source",
                score=0.80,
                path="codefind-server/src/codefind_server/config.py",
                language="python",
                snippet='clerk_azp=os.getenv("CLERK_AZP", "")',
                symbol_name="get_settings",
                symbol_kind="function",
                chunking_method="symbol",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids[0] == "main-usage"
    assert ids.index("main-usage") < ids.index("lsp-noise")


def test_ranking_eval_prefers_ollama_retry_config_over_unrelated_retry_constants():
    reranked = rerank_results(
        query_text="where is ollama retry configured",
        combined=[
            _candidate(
                result_id="client-noise",
                score=0.91,
                path="internal/client/api_client.go",
                snippet="ollamaEmbedRetryBackoffSeconds = 1",
                symbol_name="ollamaEmbedRetryBackoffSeconds",
                symbol_kind="constant",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="server-config",
                score=0.78,
                path="codefind-server/src/codefind_server/config.py",
                language="python",
                snippet='ollama_embed_retry_backoff_seconds=float(os.getenv("OLLAMA_EMBED_RETRY_BACKOFF_SECONDS", "1.0"))',
                symbol_name="get_settings",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="server-runtime",
                score=0.80,
                path="codefind-server/src/codefind_server/services/ollama.py",
                language="python",
                snippet="OLLAMA_EMBED_RETRY_BACKOFF_SECONDS = 1.0",
                symbol_name="OLLAMA_EMBED_RETRY_BACKOFF_SECONDS",
                symbol_kind="constant",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="lsp-noise",
                score=0.89,
                path="internal/chunker/hybrid.go",
                snippet="const MaxLSPRetries = 3",
                symbol_name="MaxLSPRetries",
                symbol_kind="constant",
                chunking_method="symbol",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids[0] == "server-config"
    assert ids.index("server-runtime") < ids.index("client-noise")
    assert ids.index("server-runtime") < ids.index("lsp-noise")


def test_ranking_eval_demotes_config_tests_below_runtime_settings():
    reranked = rerank_results(
        query_text="where is ollama retry configured",
        combined=[
            _candidate(
                result_id="config-test",
                score=0.90,
                path="codefind-server/tests/test_config.py",
                language="python",
                snippet='def test_get_settings_reads_ollama_retry_settings(...): monkeypatch.setenv("OLLAMA_EMBED_RETRY_BACKOFF_SECONDS", "1.0")',
                symbol_name="test_get_settings_reads_ollama_retry_settings",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="server-config",
                score=0.24,
                path="codefind-server/src/codefind_server/config.py",
                language="python",
                snippet='ollama_embed_retry_backoff_seconds=float(os.getenv("OLLAMA_EMBED_RETRY_BACKOFF_SECONDS", "1.0"))',
                symbol_name="get_settings",
                symbol_kind="function",
                chunking_method="symbol",
            ),
        ],
    )

    assert [result.id for _, result, _ in reranked][:2] == ["server-config", "config-test"]


def test_ranking_eval_demotes_ranking_eval_tests_for_config_queries():
    reranked = rerank_results(
        query_text="where is ollama retry configured",
        combined=[
            _candidate(
                result_id="ranking-test",
                score=0.94,
                path="codefind-server/tests/test_query_ranking_eval.py",
                language="python",
                snippet='def test_ranking_eval_prefers_ollama_retry_config_over_unrelated_retry_constants(): reranked = rerank_results(query_text="where is ollama retry configured", combined=[...]); assert reranked[0][1].id == "server-config"',
                symbol_name="test_ranking_eval_prefers_ollama_retry_config_over_unrelated_retry_constants",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="runtime-config",
                score=0.20,
                path="codefind-server/src/codefind_server/services/ollama.py",
                language="python",
                snippet="OLLAMA_EMBED_RETRY_BACKOFF_SECONDS = 1.0",
                symbol_name="OLLAMA_EMBED_RETRY_BACKOFF_SECONDS",
                symbol_kind="constant",
                chunking_method="symbol",
            ),
        ],
    )

    assert [result.id for _, result, _ in reranked][:2] == ["runtime-config", "ranking-test"]


def test_ranking_eval_prefers_force_reindex_implementation_over_tests():
    reranked = rerank_results(
        query_text="where is force reindex implemented",
        combined=[
            _candidate(
                result_id="test",
                score=0.90,
                path="internal/indexer/index_write_test.go",
                snippet="func TestIndexerIndexForceRebuildTombstonesPreviouslyIndexedFiles(t *testing.T) {",
                symbol_name="TestIndexerIndexForceRebuildTombstonesPreviouslyIndexedFiles",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="implementation",
                score=0.82,
                path="internal/indexer/indexer.go",
                snippet="func runIndexMode(options RunOptions) string { if options.Window { return IndexModeForceWindow } return IndexModeHybrid }",
                symbol_name="runIndexMode",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="command",
                score=0.80,
                path="cmd/codefind/commands_project.go",
                snippet='modeLabel := "hybrid (LSP when available)"',
                symbol_name="newIndexRunCommand",
                symbol_kind="function",
                chunking_method="symbol",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids[0] == "implementation"
    assert ids.index("implementation") < ids.index("test")
    assert ids.index("command") < ids.index("test")


def test_ranking_eval_prefers_auth_entrypoints_over_error_types():
    reranked = rerank_results(
        query_text="where is the clerk auth function",
        combined=[
            _candidate(
                result_id="entrypoint",
                score=0.82,
                path="cmd/codefind/commands_auth.go",
                snippet="func runBrowserLogin(ctx context.Context, stdout io.Writer, configPath string) error {",
                symbol_name="runBrowserLogin",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="command",
                score=0.84,
                path="cmd/codefind/commands_auth.go",
                snippet="func newAuthLoginCommand(configPath *string) *cobra.Command {",
                symbol_name="newAuthLoginCommand",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="error-class",
                score=0.88,
                path="codefind-server/src/codefind_server/middleware/auth.py",
                language="python",
                snippet='class TokenVerificationError(ValueError): """Raised when a Clerk token cannot be verified."""',
                symbol_name="TokenVerificationError",
                symbol_kind="class",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="constant-helper",
                score=0.83,
                path="web/src/lib/auth.ts",
                language="typescript",
                snippet="export function getPostAuthPath(orgId: string | null): string { return orgId ? '/search' : '/no-access' }",
                symbol_name="getPostAuthPath",
                symbol_kind="function",
                chunking_method="symbol",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids[0] == "command"
    assert ids.index("entrypoint") < ids.index("error-class")
    assert ids.index("error-class") > ids.index("command")


def test_ranking_eval_prefers_token_verification_function_over_error_type():
    reranked = rerank_results(
        query_text="where is token verification implemented",
        combined=[
            _candidate(
                result_id="error-class",
                score=0.92,
                path="codefind-server/src/codefind_server/middleware/auth.py",
                language="python",
                snippet='class TokenVerificationError(ValueError): """Raised when a Clerk token cannot be verified."""',
                symbol_name="TokenVerificationError",
                symbol_kind="class",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="implementation",
                score=0.84,
                path="codefind-server/src/codefind_server/middleware/auth.py",
                language="python",
                snippet="def verify_clerk_token(token: str, settings: Settings) -> dict[str, Any]:",
                symbol_name="verify_clerk_token",
                symbol_kind="function",
                chunking_method="symbol",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids[:2] == ["implementation", "error-class"]


def test_ranking_eval_prefers_cleanup_service_over_cli_or_models():
    reranked = rerank_results(
        query_text="where is stale chunk cleanup implemented",
        combined=[
            _candidate(
                result_id="cli-command",
                score=0.88,
                path="cmd/codefind/commands_project.go",
                snippet="func newCleanupCommand(configPath *string) *cobra.Command {",
                symbol_name="newCleanupCommand",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="service-implementation",
                score=0.80,
                path="codefind-server/src/codefind_server/services/indexing.py",
                language="python",
                snippet="async def purge_chunks(self, *, org_id: str, request: ChunkPurgeRequest) -> ChunkPurgeResponse:",
                symbol_name="purge_chunks",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="response-model",
                score=0.86,
                path="codefind-server/src/codefind_server/models/responses.py",
                language="python",
                snippet="class ChunkPurgeResponse(BaseModel):",
                symbol_name="ChunkPurgeResponse",
                symbol_kind="class",
                chunking_method="symbol",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids[0] == "service-implementation"
    assert ids.index("service-implementation") < ids.index("response-model")


def test_ranking_eval_prefers_implementation_over_route_and_command_wrappers():
    reranked = rerank_results(
        query_text="where is stale chunk cleanup implemented",
        combined=[
            _candidate(
                result_id="command-wrapper",
                score=0.55,
                path="cmd/codefind/commands_project.go",
                snippet="func newCleanupCommand(configPath *string) *cobra.Command {",
                symbol_name="newCleanupCommand",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="route-wrapper",
                score=0.51,
                path="codefind-server/src/codefind_server/routes/index.py",
                language="python",
                snippet='@router.delete("/chunks/purge") async def purge_chunks(...): return await indexing_service.purge_chunks(org_id=context.org_id, request=request)',
                symbol_name="purge_chunks",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="implementation",
                score=0.34,
                path="codefind-server/src/codefind_server/services/indexing.py",
                language="python",
                snippet="async def purge_chunks(self, *, org_id: str, request: ChunkPurgeRequest) -> ChunkPurgeResponse:",
                symbol_name="purge_chunks",
                symbol_kind="function",
                chunking_method="symbol",
            ),
        ],
    )

    assert [result.id for _, result, _ in reranked][:3] == ["implementation", "route-wrapper", "command-wrapper"]


def test_ranking_eval_demotes_cleanup_command_and_eval_tests_below_implementation():
    reranked = rerank_results(
        query_text="where is stale chunk cleanup implemented",
        combined=[
            _candidate(
                result_id="command-wrapper",
                score=0.62,
                path="cmd/codefind/commands_project.go",
                snippet="func newCleanupCommand(configPath *string) *cobra.Command {",
                symbol_name="newCleanupCommand",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="ranking-test",
                score=0.70,
                path="codefind-server/tests/test_query_ranking_eval.py",
                language="python",
                snippet='def test_ranking_eval_prefers_cleanup_service_over_cli_or_models(): reranked = rerank_results(query_text="where is stale chunk cleanup implemented", combined=[...]); assert ids[0] == "service-implementation"',
                symbol_name="test_ranking_eval_prefers_cleanup_service_over_cli_or_models",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="implementation",
                score=0.28,
                path="codefind-server/src/codefind_server/services/indexing.py",
                language="python",
                snippet="async def purge_chunks(self, *, org_id: str, request: ChunkPurgeRequest) -> ChunkPurgeResponse:",
                symbol_name="purge_chunks",
                symbol_kind="function",
                chunking_method="symbol",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids[0] == "implementation"
    assert ids.index("implementation") < ids.index("command-wrapper")
    assert ids.index("implementation") < ids.index("ranking-test")


def test_ranking_eval_prefers_cleanup_service_after_method_chunking():
    reranked = rerank_results(
        query_text="where is stale chunk cleanup implemented",
        combined=[
            _candidate(
                result_id="route-wrapper",
                score=0.267,
                path="codefind-server/src/codefind_server/routes/index.py",
                language="python",
                snippet='@router.delete("/chunks/purge", response_model=ChunkPurgeResponse)\nasync def purge_chunks(request: ChunkPurgeRequest, context: OrgContext = Depends(require_admin), indexing_service: IndexingService = Depends(get_indexing_service)) -> ChunkPurgeResponse:\n    response = await indexing_service.purge_tombstoned_chunks(org_id=context.org_id, request=request)\n    return response',
                symbol_name="purge_chunks",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="command-wrapper",
                score=0.243,
                path="cmd/codefind/commands_project.go",
                snippet="func newCleanupCommand(configPath *string) *cobra.Command {",
                symbol_name="newCleanupCommand",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="test-wrapper",
                score=0.500,
                path="codefind-server/tests/test_index_routes.py",
                language="python",
                snippet='async def purge_tombstoned_chunks(self, *, org_id: str, request): self.purge_calls.append({\"org_id\": org_id, \"request\": request}); return {\"status\": \"ok\"}',
                symbol_name="purge_tombstoned_chunks",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="service-implementation",
                score=0.333,
                path="codefind-server/src/codefind_server/services/indexing.py",
                language="python",
                snippet='async def purge_tombstoned_chunks(self, *, org_id: str, request: ChunkPurgeRequest) -> ChunkPurgeResponse:\n    collection = collection_name_for(org_id, request.repo_id)\n    if collection not in await self._vector_store.list_collections():\n        return ChunkPurgeResponse(status=\"ok\", repo_id=request.repo_id, found_count=0, purged_count=0, files=[])\n    points = await self._vector_store.scroll(collection, {\"status\": \"tombstoned\", \"repo_id\": request.repo_id})\n    cutoff = datetime.now(UTC).timestamp() - (request.older_than_days * 86400)\n    matching_points = []\n    for point in points:\n        tombstoned_at = point.payload.get(\"tombstoned_at\")\n        if not isinstance(tombstoned_at, str):\n            continue\n        matching_points.append(point)',
                symbol_name="purge_tombstoned_chunks (part 1)",
                symbol_kind="method",
                chunking_method="symbol",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids[0] == "service-implementation"
    assert ids.index("service-implementation") < ids.index("route-wrapper")
    assert ids.index("service-implementation") < ids.index("command-wrapper")
    assert ids.index("service-implementation") < ids.index("test-wrapper")


def test_ranking_eval_prefers_chunked_method_body_over_route_and_test_wrappers():
    reranked = rerank_results(
        query_text="where is stale chunk cleanup implemented",
        combined=[
            _candidate(
                result_id="route-wrapper",
                score=0.267,
                path="codefind-server/src/codefind_server/routes/index.py",
                language="python",
                snippet='@router.delete("/chunks/purge", response_model=ChunkPurgeResponse)\nasync def purge_chunks(request: ChunkPurgeRequest, context: OrgContext = Depends(require_admin), indexing_service: IndexingService = Depends(get_indexing_service)) -> ChunkPurgeResponse:\n    response = await indexing_service.purge_tombstoned_chunks(org_id=context.org_id, request=request)\n    return response',
                symbol_name="purge_chunks",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="test-wrapper",
                score=0.500,
                path="codefind-server/tests/test_index_routes.py",
                language="python",
                snippet='async def purge_tombstoned_chunks(self, *, org_id: str, request): self.purge_calls.append({\"org_id\": org_id, \"request\": request}); return {\"status\": \"ok\"}',
                symbol_name="purge_tombstoned_chunks",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="service-body-part",
                score=0.333,
                path="codefind-server/src/codefind_server/services/indexing.py",
                language="python",
                snippet='collection, {\"status\": \"tombstoned\", \"repo_id\": request.repo_id})\nfiles = self._summarize_tombstoned_points(points)\ncutoff = datetime.now(UTC).timestamp() - (request.older_than_days * 86400)\nmatching_points = []\nfor point in points:\n    tombstoned_at = point.payload.get(\"tombstoned_at\")\n    if not isinstance(tombstoned_at, str):\n        continue\n    matching_points.append(point)',
                symbol_name="purge_tombstoned_chunks (part 2)",
                symbol_kind="method",
                chunking_method="symbol",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids[0] == "service-body-part"
    assert ids.index("service-body-part") < ids.index("route-wrapper")
    assert ids.index("service-body-part") < ids.index("test-wrapper")


def test_ranking_eval_demotes_ranking_internal_helpers_for_cleanup_queries():
    reranked = rerank_results(
        query_text="where is stale chunk cleanup implemented",
        combined=[
            _candidate(
                result_id="implementation",
                score=0.216,
                path="codefind-server/src/codefind_server/services/indexing.py",
                language="python",
                snippet="async def purge_tombstoned_chunks(self, *, org_id: str, request: ChunkPurgeRequest) -> ChunkPurgeResponse:",
                symbol_name="purge_tombstoned_chunks",
                symbol_kind="method",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="ranking-internal",
                score=0.143,
                path="codefind-server/src/codefind_server/services/query_ranking.py",
                language="python",
                snippet="def _implementation_chunk_body_boost(query_tokens: set[str], payload: dict[str, object]) -> float:",
                symbol_name="_implementation_chunk_body_boost",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="route-wrapper",
                score=0.183,
                path="codefind-server/src/codefind_server/routes/index.py",
                language="python",
                snippet='@router.delete("/chunks/purge", response_model=ChunkPurgeResponse) async def purge_chunks(...): return await indexing_service.purge_tombstoned_chunks(org_id=context.org_id, request=request)',
                symbol_name="purge_chunks",
                symbol_kind="function",
                chunking_method="symbol",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids[0] == "implementation"
    assert ids.index("ranking-internal") > ids.index("route-wrapper")


def test_ranking_eval_prefers_callers_of_load_authenticated_client():
    reranked = rerank_results(
        query_text="who uses loadAuthenticatedClient",
        combined=[
            _candidate(
                result_id="definition",
                score=0.91,
                path="cmd/codefind/cli_runtime.go",
                snippet="func loadAuthenticatedClient(ctx context.Context, stdout io.Writer, path string, quiet bool) (*client.Client, error) {",
                symbol_name="loadAuthenticatedClient",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="caller",
                score=0.84,
                path="cmd/codefind/cli_runtime.go",
                snippet="func requireAdminClient(ctx context.Context, stdout io.Writer, path string) (*client.Client, error) { apiClient, err := loadAuthenticatedClient(ctx, stdout, path, false)",
                symbol_name="requireAdminClient",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="test",
                score=0.86,
                path="cmd/codefind/main_test.go",
                snippet="func TestLoadAuthenticatedClientRequiresStoredToken(t *testing.T) {",
                symbol_name="TestLoadAuthenticatedClientRequiresStoredToken",
                symbol_kind="function",
                chunking_method="symbol",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids[0] == "caller"
    assert ids.index("caller") < ids.index("definition")
    assert ids.index("caller") < ids.index("test")


def test_ranking_eval_prefers_startcallbackserver_reference_over_definition():
    reranked = rerank_results(
        query_text="where is StartCallbackServer referenced",
        combined=[
            _candidate(
                result_id="definition",
                score=0.90,
                path="internal/authflow/login.go",
                snippet="func StartCallbackServer(ctx context.Context, listener net.Listener, allowedOrigin string) (redirectURI string, waitForToken func() (string, error), err error) {",
                symbol_name="StartCallbackServer",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="reference",
                score=0.85,
                path="cmd/codefind/cli_runtime.go",
                snippet="startCallbackServer = authflow.StartCallbackServer",
                symbol_name="startCallbackServer",
                symbol_kind="variable",
                chunking_method="symbol",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids[:2] == ["reference", "definition"]


def test_ranking_eval_prefers_browser_login_tests_over_runtime_aliases():
    reranked = rerank_results(
        query_text="tests for browser login",
        combined=[
            _candidate(
                result_id="test-helper",
                score=0.84,
                path="cmd/codefind/main_test.go",
                snippet="func useBrowserLoginRunner(runner func(context.Context, io.Writer, string) error) func() {",
                symbol_name="useBrowserLoginRunner",
                symbol_kind="function",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="runtime-alias",
                score=0.92,
                path="cmd/codefind/cli_runtime.go",
                snippet="browserLoginRunner = runBrowserLogin",
                symbol_name="browserLoginRunner",
                symbol_kind="variable",
                chunking_method="symbol",
            ),
            _candidate(
                result_id="runtime-implementation",
                score=0.89,
                path="cmd/codefind/commands_auth.go",
                snippet="func runBrowserLogin(ctx context.Context, stdout io.Writer, configPath string) error {",
                symbol_name="runBrowserLogin",
                symbol_kind="function",
                chunking_method="symbol",
            ),
        ],
    )

    ids = [result.id for _, result, _ in reranked]
    assert ids[0] == "test-helper"
    assert ids.index("test-helper") < ids.index("runtime-alias")
