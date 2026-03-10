package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tk-425/Codefind/internal/authflow"
	"github.com/tk-425/Codefind/internal/config"
	"github.com/tk-425/Codefind/internal/keychain"
)

type fakeKeychainProvider struct {
	token string
	err   error
}

func (f *fakeKeychainProvider) Set(service, user, password string) error {
	f.token = password
	return nil
}

func (f *fakeKeychainProvider) Get(service, user string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.token, nil
}

func (f *fakeKeychainProvider) Delete(service, user string) error {
	return nil
}

func useFakeTokenManager(token string, err error) func() {
	previous := defaultTokenManager
	defaultTokenManager = func() *keychain.Manager {
		return keychain.NewManager(&fakeKeychainProvider{token: token, err: err})
	}
	return func() {
		defaultTokenManager = previous
	}
}

func useMutableTokenManager(provider *fakeKeychainProvider) func() {
	previous := defaultTokenManager
	defaultTokenManager = func() *keychain.Manager {
		return keychain.NewManager(provider)
	}
	return func() {
		defaultTokenManager = previous
	}
}

func useBrowserLoginRunner(
	runner func(context.Context, io.Writer, string) error,
) func() {
	previous := browserLoginRunner
	browserLoginRunner = runner
	return func() {
		browserLoginRunner = previous
	}
}

func makeTestToken(expiry time.Time, orgID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(
		`{"org_id":"` + orgID + `","org_role":"org:admin","exp":` + fmt.Sprint(expiry.Unix()) + `}`,
	))
	return header + "." + payload + ".signature"
}

func writeTestConfig(t *testing.T, serverURL string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, config.Config{ServerURL: serverURL, WebAppURL: "http://localhost:5173"}); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}
	return path
}

func writeTestConfigWithOrg(t *testing.T, serverURL, orgID string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, config.Config{
		ServerURL:   serverURL,
		WebAppURL:   "http://localhost:5173",
		ActiveOrgID: orgID,
	}); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}
	return path
}

func executeCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newRootCommand()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return output.String(), err
}

func TestOrgListCommandCallsBackend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs" {
			t.Fatalf("path = %q, want /orgs", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token-123" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"organization_id":"org_123","organization_name":"Acme","role":"org:admin"}],"total_count":1}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager("token-123", nil)
	defer restore()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "org", "list")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, `"organization_id": "org_123"`) {
		t.Fatalf("output = %q", output)
	}
}

func TestAdminListCommandCallsMemberAndInvitationEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/admin/members":
			_, _ = w.Write([]byte(`{"data":[{"user_id":"user_member","role":"org:member"}],"total_count":1}`))
		case "/admin/invitations":
			_, _ = w.Write([]byte(`{"data":[{"id":"orginv_1","invitation_id":"orginv_1","email_address":"new@example.com","role":"org:member","status":"pending","organization_id":"org_123"}],"total_count":1}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	restore := useFakeTokenManager("token-123", nil)
	defer restore()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "admin", "list")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, `"members"`) || !strings.Contains(output, `"invitations"`) {
		t.Fatalf("output = %q", output)
	}
}

func TestAdminInviteCommandPostsJSON(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin/invite" || r.Method != http.MethodPost {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"orginv_1","invitation_id":"orginv_1","email_address":"new@example.com","role":"org:member","status":"pending","organization_id":"org_123"}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager("token-123", nil)
	defer restore()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "admin", "invite", "--email", "new@example.com")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if requestBody["email_address"] != "new@example.com" {
		t.Fatalf("request body = %#v", requestBody)
	}
	if !strings.Contains(output, `"id": "orginv_1"`) {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, `"invitation_id": "orginv_1"`) {
		t.Fatalf("output = %q", output)
	}
}

func TestAdminRevokeInviteCommandPostsToRevokeEndpoint(t *testing.T) {
	var method string
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"orginv_1","invitation_id":"orginv_1","email_address":"new@example.com","role":"org:member","status":"revoked","organization_id":"org_123"}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager("token-123", nil)
	defer restore()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "admin", "revoke-invite", "orginv_1")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if method != http.MethodPost || path != "/admin/invitations/orginv_1/revoke" {
		t.Fatalf("saw %s %s", method, path)
	}
	if !strings.Contains(output, `"status": "revoked"`) {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, `"invitation_id": "orginv_1"`) {
		t.Fatalf("output = %q", output)
	}
}

func TestAdminRemoveCommandCallsDelete(t *testing.T) {
	var method string
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"user_id":"user_456","role":"org:member"}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager("token-123", nil)
	defer restore()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "admin", "remove", "user_456")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if method != http.MethodDelete || path != "/admin/members/user_456" {
		t.Fatalf("saw %s %s", method, path)
	}
	if !strings.Contains(output, `"user_id": "user_456"`) {
		t.Fatalf("output = %q", output)
	}
}

func TestIndexCommandPostsIndexedChunks(t *testing.T) {
	repoDir := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	sourcePath := filepath.Join(repoDir, "main.go")
	if err := os.WriteFile(sourcePath, []byte("package main\n\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var method string
	var path string
	var sawChunkingMethod bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		switch r.URL.Path {
		case "/index":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			chunks, ok := body["chunks"].([]any)
			if !ok || len(chunks) == 0 {
				t.Fatalf("chunks = %#v", body["chunks"])
			}
			firstChunk, ok := chunks[0].(map[string]any)
			if !ok {
				t.Fatalf("first chunk = %#v", chunks[0])
			}
			metadata, ok := firstChunk["metadata"].(map[string]any)
			if !ok {
				t.Fatalf("metadata = %#v", firstChunk["metadata"])
			}
			sawChunkingMethod = metadata["chunking_method"] == "window"
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","repo_id":"repo-a","indexed_count":1,"accepted":true}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	restore := useFakeTokenManager("token-123", nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, server.URL, "org_123")
	output, err := executeCommand(
		t,
		"--config", configPath,
		"index",
		"--repo-id", "repo-a",
		"--repo-path", repoDir,
		"--window",
	)
	if err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, output)
	}
	if method != http.MethodPost || path != "/index" {
		t.Fatalf("saw %s %s", method, path)
	}
	if !sawChunkingMethod {
		t.Fatalf("expected chunking_method=window in /index payload")
	}
	if !strings.Contains(output, `"indexed_count": 1`) {
		t.Fatalf("output = %q", output)
	}
}

func TestListCommandCallsCollectionsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections" {
			t.Fatalf("path = %q, want /collections", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"repo_id":"repo-a"}],"total_count":1}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager("token-123", nil)
	defer restore()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "list")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, `"repo_id": "repo-a"`) {
		t.Fatalf("output = %q", output)
	}
}

func TestStatsCommandCallsStatsEndpoint(t *testing.T) {
	var requestURI string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURI = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"repo_count":1,"chunk_count":12,"repos":[{"repo_id":"repo-a","chunk_count":12}]}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager("token-123", nil)
	defer restore()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "stats", "--repo-id", "repo-a")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if requestURI != "/stats?repo_id=repo-a" {
		t.Fatalf("requestURI = %q", requestURI)
	}
	if !strings.Contains(output, `"chunk_count": 12`) {
		t.Fatalf("output = %q", output)
	}
}

func TestQueryCommandPostsSearchRequest(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/query" || r.Method != http.MethodPost {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"chunk-1","score":0.9,"repo_id":"repo-a"}],"total_count":1,"page":1,"page_size":10,"has_more":false}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager("token-123", nil)
	defer restore()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "query", "main", "--repo-id", "repo-a", "--lang", "go")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if requestBody["query_text"] != "main" || requestBody["repo_id"] != "repo-a" || requestBody["language"] != "go" {
		t.Fatalf("request body = %#v", requestBody)
	}
	if !strings.Contains(output, `"repo_id": "repo-a"`) {
		t.Fatalf("output = %q", output)
	}
}

func TestTokenizeCommandPostsText(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tokenize" || r.Method != http.MethodPost {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"bert-base-uncased","tokens":["alpha","beta"],"token_count":2}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager("token-123", nil)
	defer restore()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "tokenize", "alpha beta")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if requestBody["text"] != "alpha beta" {
		t.Fatalf("request body = %#v", requestBody)
	}
	if !strings.Contains(output, `"token_count": 2`) {
		t.Fatalf("output = %q", output)
	}
}

func TestLoadAuthenticatedClientRequiresStoredToken(t *testing.T) {
	restore := useFakeTokenManager("", keychain.ErrNotFound)
	defer restore()

	configPath := writeTestConfig(t, "http://127.0.0.1:8080")
	_, err := loadAuthenticatedClient(context.Background(), io.Discard, configPath)
	if err == nil {
		t.Fatal("loadAuthenticatedClient() error = nil, want auth guidance")
	}
	if !strings.Contains(err.Error(), "codefind auth login") {
		t.Fatalf("error = %v", err)
	}
}

func TestAdminInviteCommandRejectsMissingEmail(t *testing.T) {
	restore := useFakeTokenManager("token-123", nil)
	defer restore()

	configPath := writeTestConfig(t, "http://127.0.0.1:8080")
	_, err := executeCommand(t, "--config", configPath, "admin", "invite")
	if err == nil {
		t.Fatal("Execute() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "--email is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestAdminInviteCommandRejectsUnknownRole(t *testing.T) {
	restore := useFakeTokenManager("token-123", nil)
	defer restore()

	configPath := writeTestConfig(t, "http://127.0.0.1:8080")
	_, err := executeCommand(t, "--config", configPath, "admin", "invite", "--email", "new@example.com", "--role", "owner")
	if err == nil {
		t.Fatal("Execute() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "--role must be org:admin or org:member") {
		t.Fatalf("error = %v", err)
	}
}

func TestExecuteCommandUsesCommandContext(t *testing.T) {
	restore := useFakeTokenManager("token-123", nil)
	defer restore()

	configPath := writeTestConfig(t, "http://127.0.0.1:8080")
	cmd := newRootCommand()
	cmd.SetArgs([]string{"--config", configPath, "org", "list"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd.SetContext(ctx)
	err := cmd.Execute()
	if err == nil || (!strings.Contains(err.Error(), "context canceled") && !strings.Contains(err.Error(), "context cancelled")) {
		t.Fatalf("Execute() error = %v, want canceled context", err)
	}
}

func TestFakeKeychainProviderCanReturnWrappedError(t *testing.T) {
	restore := useFakeTokenManager("", errors.New("boom"))
	defer restore()

	configPath := writeTestConfig(t, "http://127.0.0.1:8080")
	_, err := loadAuthenticatedClient(context.Background(), io.Discard, configPath)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %v, want wrapped provider error", err)
	}
}

func TestListCommandRenewsExpiredTokenViaBrowserFlow(t *testing.T) {
	initialToken := makeTestToken(time.Now().UTC().Add(-time.Minute), "org_old")
	provider := &fakeKeychainProvider{
		token: initialToken,
	}
	restoreTokenManager := useMutableTokenManager(provider)
	defer restoreTokenManager()

	restoreLogin := useBrowserLoginRunner(func(_ context.Context, stdout io.Writer, _ string) error {
		if err := provider.Set("", "", makeTestToken(time.Now().UTC().Add(15*time.Minute), "org_123")); err != nil {
			return err
		}
		_, err := io.WriteString(stdout, "authentication stored in keychain\n")
		return err
	})
	defer restoreLogin()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections" {
			t.Fatalf("path = %q, want /collections", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"repo_id":"repo-a"}],"total_count":1}`))
	}))
	defer server.Close()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "list")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, "stored token expired; renewing via browser session...") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, `"repo_id": "repo-a"`) {
		t.Fatalf("output = %q", output)
	}
	token, err := provider.Get("", "")
	if err != nil {
		t.Fatalf("provider.Get() error = %v", err)
	}
	if token == initialToken {
		t.Fatal("stored token was not replaced during renewal")
	}
	claims, err := authflow.DecodeTokenClaims(token)
	if err != nil {
		t.Fatalf("DecodeTokenClaims() error = %v", err)
	}
	if claims.OrgID != "org_123" || claims.OrgRole != "org:admin" {
		t.Fatalf("claims = %+v, want renewed org claims", claims)
	}
}
