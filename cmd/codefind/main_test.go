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
	"github.com/tk-425/Codefind/internal/indexer"
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
	return makeTestTokenWithRole(expiry, orgID, "org:admin")
}

func makeTestTokenWithRole(expiry time.Time, orgID, orgRole string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(
		`{"org_id":"` + orgID + `","org_role":"` + orgRole + `","exp":` + fmt.Sprint(expiry.Unix()) + `}`,
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

func initTestProject(t *testing.T, configPath, repoDir string, extraArgs ...string) {
	t.Helper()
	t.Chdir(repoDir)
	args := []string{"--config", configPath, "init"}
	args = append(args, extraArgs...)
	if _, err := executeCommand(t, args...); err != nil {
		t.Fatalf("init Execute() error = %v", err)
	}
}

func TestOrgListCommandCallsBackend(t *testing.T) {
	token := makeTestToken(time.Now().UTC().Add(time.Hour), "org_123")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs" {
			t.Fatalf("path = %q, want /orgs", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"organization_id":"org_123","organization_name":"Acme","role":"org:admin"}],"total_count":1}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager(token, nil)
	defer restore()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "org", "list")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, "Organizations: 1") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Acme") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "org_123") {
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

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "admin", "list")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, "Members: 1") || !strings.Contains(output, "Invitations: 1") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "user_member") || !strings.Contains(output, "new@example.com") {
		t.Fatalf("output = %q", output)
	}
}

func TestOrgListCommandJSONFlagPrintsJSONOnly(t *testing.T) {
	token := makeTestToken(time.Now().UTC().Add(time.Hour), "org_123")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"organization_id":"org_123","organization_name":"Acme","role":"org:admin"}],"total_count":1}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager(token, nil)
	defer restore()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "org", "list", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, `"organization_id": "org_123"`) || strings.Contains(output, "Organizations:") {
		t.Fatalf("output = %q", output)
	}
}

func TestAdminListCommandJSONFlagPrintsJSONOnly(t *testing.T) {
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

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "admin", "list", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, `"members"`) || strings.Contains(output, "Members:") {
		t.Fatalf("output = %q", output)
	}
}

func TestAuthStatusCommandPrintsHumanReadableOutput(t *testing.T) {
	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, "http://127.0.0.1:8080", "org_123")
	output, err := executeCommand(t, "--config", configPath, "auth", "status")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, "Authenticated: YES") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Active Org: org_123") || !strings.Contains(output, "Token Org: org_123") {
		t.Fatalf("output = %q", output)
	}
}

func TestAuthStatusCommandJSONFlagPrintsJSONOnly(t *testing.T) {
	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, "http://127.0.0.1:8080", "org_123")
	output, err := executeCommand(t, "--config", configPath, "auth", "status", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, `"authenticated": true`) || strings.Contains(output, "Authenticated:") {
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

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
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

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
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

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
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

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, server.URL, "org_123")
	initTestProject(t, configPath, repoDir, "--repo-id", "repo-a")
	output, err := executeCommand(t, "--config", configPath, "index", "run", "--window")
	if err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, output)
	}
	if method != http.MethodPost || path != "/index" {
		t.Fatalf("saw %s %s", method, path)
	}
	if !sawChunkingMethod {
		t.Fatalf("expected chunking_method=window in /index payload")
	}
	if !strings.Contains(output, "┏━╸┏━┓╺┳┓┏━╸") ||
		!strings.Contains(output, "Chunking Mode: window-only") ||
		!strings.Contains(output, "• Discovering files...") ||
		!strings.Contains(output, "• Building chunks for 1 files...") ||
		!strings.Contains(output, "• [WINDOW] [1/1] main.go: 1 chunks") ||
		!strings.Contains(output, "• [SEND] 1/1 send: 1 chunks") ||
		!strings.Contains(output, "• Index complete.") ||
		!strings.Contains(output, "Total Time: ") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, `"indexed_count": 1`) {
		t.Fatalf("output = %q", output)
	}
}

func TestIndexRunRequiresInit(t *testing.T) {
	repoDir := t.TempDir()
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, "http://127.0.0.1:8080", "org_123")
	_, err := executeCommand(t, "--config", configPath, "index", "run", "--window")
	if err == nil {
		t.Fatal("Execute() error = nil, want init guidance")
	}
	if !strings.Contains(err.Error(), "project is not initialized") || !strings.Contains(err.Error(), "codefind init") {
		t.Fatalf("error = %v", err)
	}
}

func TestIndexRunRequiresAdminRole(t *testing.T) {
	repoDir := t.TempDir()
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	restore := useFakeTokenManager(makeTestTokenWithRole(time.Now().UTC().Add(time.Hour), "org_123", "org:member"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, "http://127.0.0.1:8080", "org_123")
	_, err := executeCommand(t, "--config", configPath, "index", "run", "--window")
	if err == nil {
		t.Fatal("Execute() error = nil, want admin-role guidance")
	}
	if !strings.Contains(err.Error(), "org:admin role required") {
		t.Fatalf("error = %v", err)
	}
}

func TestCleanupListCommandCallsTombstonedEndpoint(t *testing.T) {
	repoDir := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var method string
	var path string
	var rawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","repo_id":"repo-a","found_count":1,"files":[{"path":"main.go","chunk_count":1,"tombstoned_at":"2026-03-09T00:00:00Z"}]}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, server.URL, "org_123")
	initTestProject(t, configPath, repoDir, "--repo-id", "repo-a")
	output, err := executeCommand(t, "--config", configPath, "cleanup", "--list")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if method != http.MethodGet || path != "/chunks/tombstoned" || rawQuery != "repo_id=repo-a" {
		t.Fatalf("saw %s %s?%s", method, path, rawQuery)
	}
	if !strings.Contains(output, `"found_count": 1`) {
		t.Fatalf("output = %q", output)
	}
}

func TestCleanupPurgeCommandCallsDeleteEndpoint(t *testing.T) {
	repoDir := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var method string
	var path string
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","repo_id":"repo-a","found_count":2,"purged_count":2,"files":[{"path":"main.go","chunk_count":2,"tombstoned_at":"2026-03-01T00:00:00Z"}]}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, server.URL, "org_123")
	initTestProject(t, configPath, repoDir, "--repo-id", "repo-a")
	output, err := executeCommand(t, "--config", configPath, "cleanup", "--older-than", "30")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if method != http.MethodDelete || path != "/chunks/purge" {
		t.Fatalf("saw %s %s", method, path)
	}
	if requestBody["older_than_days"] != float64(30) || requestBody["repo_id"] != "repo-a" {
		t.Fatalf("request body = %#v", requestBody)
	}
	if !strings.Contains(output, `"purged_count": 2`) {
		t.Fatalf("output = %q", output)
	}
}

func TestCleanupCommandRequiresAdminRole(t *testing.T) {
	restore := useFakeTokenManager(makeTestTokenWithRole(time.Now().UTC().Add(time.Hour), "org_123", "org:member"), nil)
	defer restore()

	configPath := writeTestConfig(t, "http://127.0.0.1:8080")
	_, err := executeCommand(t, "--config", configPath, "cleanup", "--repo-id", "repo-a", "--list")
	if err == nil {
		t.Fatal("Execute() error = nil, want admin-role guidance")
	}
	if !strings.Contains(err.Error(), "org:admin role required") {
		t.Fatalf("error = %v", err)
	}
}

func TestCleanupCommandRequiresInitializedProject(t *testing.T) {
	repoDir := t.TempDir()
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, "http://127.0.0.1:8080", "org_123")
	_, err := executeCommand(t, "--config", configPath, "cleanup", "--list")
	if err == nil {
		t.Fatal("Execute() error = nil, want init guidance")
	}
	if !strings.Contains(err.Error(), "project is not initialized") {
		t.Fatalf("error = %v", err)
	}
}

func TestLSPStatusCommandPrintsSupportedLanguages(t *testing.T) {
	output, err := executeCommand(t, "lsp", "status")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, "LSP availability:") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "go: available") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "typescript/javascript: available") {
		t.Fatalf("output = %q", output)
	}
}

func TestLSPStatusCommandJSONFlagPrintsJSONOnly(t *testing.T) {
	output, err := executeCommand(t, "lsp", "status", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, `"supported_count": 7`) {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, `"language": "go"`) {
		t.Fatalf("output = %q", output)
	}
	if strings.Contains(output, "LSP availability:") {
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

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "list")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, "Indexed repos: 1") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "repo-a") {
		t.Fatalf("output = %q", output)
	}
}

func TestHealthCommandPrintsHumanReadableOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("path = %q, want /health", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","ollama_status":"ok","qdrant_status":"ok","timestamp":"2026-03-11T14:35:00Z"}`))
	}))
	defer server.Close()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "health")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, "Server: OK") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Ollama: OK") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Qdrant: OK") {
		t.Fatalf("output = %q", output)
	}
}

func TestStatsCommandCallsStatsEndpoint(t *testing.T) {
	repoDir := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var requestURI string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURI = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"repo_id":"repo-a","repo_count":1,"chunk_count":12,"active_chunks":12,"deleted_chunks":3,"total_chunks":15,"overhead_percent":25.0,"repos":[{"repo_id":"repo-a","chunk_count":12}]}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, server.URL, "org_123")
	initTestProject(t, configPath, repoDir, "--repo-id", "repo-a")
	output, err := executeCommand(t, "--config", configPath, "stats")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if requestURI != "/stats?repo_id=repo-a" {
		t.Fatalf("requestURI = %q", requestURI)
	}
	if !strings.Contains(output, "Stats for repo repo-a") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Active Chunks: 12") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Deleted Chunks: 3") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Total Chunks: 15") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Storage Overhead: 25.0%") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Consider running: codefind cleanup --older-than=30") {
		t.Fatalf("output = %q", output)
	}
	if strings.Contains(output, "Repos: 1") {
		t.Fatalf("output = %q", output)
	}
}

func TestStatsCommandRequiresInitializedProject(t *testing.T) {
	repoDir := t.TempDir()
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, "http://127.0.0.1:8080", "org_123")
	_, err := executeCommand(t, "--config", configPath, "stats")
	if err == nil {
		t.Fatal("Execute() error = nil, want init guidance")
	}
	if !strings.Contains(err.Error(), "project is not initialized") {
		t.Fatalf("error = %v", err)
	}
}

func TestQueryCommandPostsSearchRequest(t *testing.T) {
	repoDir := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/query" || r.Method != http.MethodPost {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"chunk-1","score":0.9,"repo_id":"repo-a","language":"go","path":"main.go","start_line":4,"end_line":7,"snippet":"func main() { println(\"hi\") }"}],"total_count":1,"page":1,"page_size":10,"has_more":false}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, server.URL, "org_123")
	initTestProject(t, configPath, repoDir, "--repo-id", "repo-a")
	output, err := executeCommand(t, "--config", configPath, "query", "main", "--lang", "go")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if requestBody["query_text"] != "main" || requestBody["repo_id"] != "repo-a" || requestBody["language"] != "go" {
		t.Fatalf("request body = %#v", requestBody)
	}
	if !strings.Contains(output, "Results: 1 (page 1, page size 10)") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "1. main.go:4-7") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Score: 0.900") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "repo repo-a | lang go") {
		t.Fatalf("output = %q", output)
	}
}

func TestQueryCommandRequiresInitializedProject(t *testing.T) {
	repoDir := t.TempDir()
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, "http://127.0.0.1:8080", "org_123")
	_, err := executeCommand(t, "--config", configPath, "query", "main")
	if err == nil {
		t.Fatal("Execute() error = nil, want init guidance")
	}
	if !strings.Contains(err.Error(), "project is not initialized") {
		t.Fatalf("error = %v", err)
	}
}

func TestListCommandJSONFlagPrintsJSONOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"repo_id":"repo-a"}],"total_count":1}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "list", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, `"repo_id": "repo-a"`) {
		t.Fatalf("output = %q", output)
	}
	if strings.Contains(output, "Indexed repos:") {
		t.Fatalf("output = %q", output)
	}
}

func TestHealthCommandJSONFlagPrintsJSONOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","ollama_status":"ok","qdrant_status":"ok"}`))
	}))
	defer server.Close()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "health", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, `"status": "ok"`) {
		t.Fatalf("output = %q", output)
	}
	if strings.Contains(output, "Server:") {
		t.Fatalf("output = %q", output)
	}
}

func TestStatsCommandJSONFlagPrintsJSONOnly(t *testing.T) {
	repoDir := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"repo_id":"repo-a","repo_count":1,"chunk_count":12,"active_chunks":12,"deleted_chunks":3,"total_chunks":15,"overhead_percent":25.0,"repos":[{"repo_id":"repo-a","chunk_count":12}]}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, server.URL, "org_123")
	initTestProject(t, configPath, repoDir, "--repo-id", "repo-a")
	output, err := executeCommand(t, "--config", configPath, "stats", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, `"chunk_count": 12`) {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, `"deleted_chunks": 3`) {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, `"overhead_percent": 25`) {
		t.Fatalf("output = %q", output)
	}
	if strings.Contains(output, "Stats for") {
		t.Fatalf("output = %q", output)
	}
}

func TestQueryCommandJSONFlagPrintsJSONOnly(t *testing.T) {
	repoDir := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"chunk-1","score":0.9,"repo_id":"repo-a","path":"main.go","snippet":"func main() {}"}],"total_count":1,"page":1,"page_size":10,"has_more":false}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, server.URL, "org_123")
	initTestProject(t, configPath, repoDir, "--repo-id", "repo-a")
	output, err := executeCommand(t, "--config", configPath, "query", "main", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output, `"repo_id": "repo-a"`) {
		t.Fatalf("output = %q", output)
	}
	if strings.Contains(output, "Results:") {
		t.Fatalf("output = %q", output)
	}
}

func TestTokenizeCommandPostsText(t *testing.T) {
	repoDir := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

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

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, server.URL, "org_123")
	initTestProject(t, configPath, repoDir)
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

func TestTokenizeCommandRequiresInitializedProject(t *testing.T) {
	repoDir := t.TempDir()
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, "http://127.0.0.1:8080", "org_123")
	_, err := executeCommand(t, "--config", configPath, "tokenize", "alpha beta")
	if err == nil {
		t.Fatal("Execute() error = nil, want init guidance")
	}
	if !strings.Contains(err.Error(), "project is not initialized") {
		t.Fatalf("error = %v", err)
	}
}

func TestTokenizeCommandRequiresAdminRole(t *testing.T) {
	restore := useFakeTokenManager(makeTestTokenWithRole(time.Now().UTC().Add(time.Hour), "org_123", "org:member"), nil)
	defer restore()

	configPath := writeTestConfig(t, "http://127.0.0.1:8080")
	_, err := executeCommand(t, "--config", configPath, "tokenize", "alpha beta")
	if err == nil {
		t.Fatal("Execute() error = nil, want admin-role guidance")
	}
	if !strings.Contains(err.Error(), "org:admin role required") {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadAuthenticatedClientRequiresStoredToken(t *testing.T) {
	restore := useFakeTokenManager("", keychain.ErrNotFound)
	defer restore()

	configPath := writeTestConfig(t, "http://127.0.0.1:8080")
	_, err := loadAuthenticatedClient(context.Background(), io.Discard, configPath, false)
	if err == nil {
		t.Fatal("loadAuthenticatedClient() error = nil, want auth guidance")
	}
	if !strings.Contains(err.Error(), "codefind auth login") {
		t.Fatalf("error = %v", err)
	}
}

func TestAdminInviteCommandRejectsMissingEmail(t *testing.T) {
	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
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
	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
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
	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
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
	_, err := loadAuthenticatedClient(context.Background(), io.Discard, configPath, false)
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
	if !strings.Contains(output, "Indexed repos: 1") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "repo-a") {
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

func TestListCommandJSONFlagSuppressesRenewalBanner(t *testing.T) {
	initialToken := makeTestToken(time.Now().UTC().Add(-time.Minute), "org_old")
	provider := &fakeKeychainProvider{
		token: initialToken,
	}
	restoreTokenManager := useMutableTokenManager(provider)
	defer restoreTokenManager()

	restoreLogin := useBrowserLoginRunner(func(_ context.Context, _ io.Writer, _ string) error {
		return provider.Set("", "", makeTestToken(time.Now().UTC().Add(15*time.Minute), "org_123"))
	})
	defer restoreLogin()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"repo_id":"repo-a"}],"total_count":1}`))
	}))
	defer server.Close()

	configPath := writeTestConfig(t, server.URL)
	output, err := executeCommand(t, "--config", configPath, "list", "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if strings.Contains(output, "stored token expired; renewing via browser session...") {
		t.Fatalf("output = %q", output)
	}
	if strings.Contains(output, "authentication stored in keychain") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, `"repo_id": "repo-a"`) {
		t.Fatalf("output = %q", output)
	}
}

func TestReadCommandsAdvertiseJSONFlagInHelp(t *testing.T) {
	tests := [][]string{
		{"health", "--help"},
		{"list", "--help"},
		{"stats", "--help"},
		{"query", "--help"},
		{"init", "--help"},
		{"lsp", "status", "--help"},
		{"org", "list", "--help"},
		{"admin", "list", "--help"},
		{"auth", "status", "--help"},
	}

	for _, args := range tests {
		output, err := executeCommand(t, args...)
		if err != nil {
			t.Fatalf("Execute(%v) error = %v", args, err)
		}
		if !strings.Contains(output, "--json") {
			t.Fatalf("help output for %v = %q", args, output)
		}
	}
}

func TestInitCommandCreatesLocalBootstrapState(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("init should not contact backend, saw %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, server.URL, "org_123")
	output, err := executeCommand(t, "--config", configPath, "init", "--repo-path", repoDir)
	if err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, output)
	}

	repoID, err := indexer.DeriveRepoID(repoDir)
	if err != nil {
		t.Fatalf("DeriveRepoID() error = %v", err)
	}
	manifest, err := indexer.LoadManifest("org_123", repoID)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}

	if requests != 0 {
		t.Fatalf("backend requests = %d, want 0", requests)
	}
	if manifest.RepoPath != repoDir {
		t.Fatalf("manifest.RepoPath = %q, want %q", manifest.RepoPath, repoDir)
	}
	if manifest.InitializedAt == "" {
		t.Fatal("manifest.InitializedAt = empty, want timestamp")
	}
	if !strings.Contains(output, "Status: project initialized") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, repoID) {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, repoDir) {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Next Step: codefind index run") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "┏━╸┏━┓╺┳┓┏━╸") {
		t.Fatalf("output = %q", output)
	}
	if strings.Contains(output, "Manifest Path:") || strings.Contains(output, "\"manifest\"") {
		t.Fatalf("output should be compact, got %q", output)
	}
}

func TestInitCommandVerboseIncludesManifestDetails(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, "http://127.0.0.1:8080", "org_123")
	output, err := executeCommand(t, "--config", configPath, "init", "--repo-path", repoDir, "--verbose")
	if err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, output)
	}

	repoID, err := indexer.DeriveRepoID(repoDir)
	if err != nil {
		t.Fatalf("DeriveRepoID() error = %v", err)
	}
	manifestPath, err := indexer.ManifestPath("org_123", repoID)
	if err != nil {
		t.Fatalf("ManifestPath() error = %v", err)
	}

	if !strings.Contains(output, "Manifest Path: "+manifestPath) {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "Next Steps") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "codefind index run") || !strings.Contains(output, "codefind query <text>") || !strings.Contains(output, "codefind stats") {
		t.Fatalf("output = %q", output)
	}
	if strings.Contains(output, "--repo-id") || strings.Contains(output, "--repo-path") {
		t.Fatalf("verbose next steps should not require flags, got %q", output)
	}
}

func TestInitCommandRejectsInvalidProjectRoot(t *testing.T) {
	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, "http://127.0.0.1:8080", "org_123")
	emptyDir := t.TempDir()
	_, err := executeCommand(t, "--config", configPath, "init", "--repo-path", emptyDir)
	if err == nil {
		t.Fatal("Execute() error = nil, want invalid project root error")
	}
	if !strings.Contains(err.Error(), "invalid project root") {
		t.Fatalf("error = %v", err)
	}
}

func TestInitCommandRequiresAdminRole(t *testing.T) {
	restore := useFakeTokenManager(makeTestTokenWithRole(time.Now().UTC().Add(time.Hour), "org_123", "org:member"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, "http://127.0.0.1:8080", "org_123")
	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := executeCommand(t, "--config", configPath, "init", "--repo-path", repoDir)
	if err == nil {
		t.Fatal("Execute() error = nil, want admin-role guidance")
	}
	if !strings.Contains(err.Error(), "org:admin role required") {
		t.Fatalf("error = %v", err)
	}
}

func TestInitCommandRequiresConfig(t *testing.T) {
	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	missingConfigPath := filepath.Join(t.TempDir(), "missing-config.json")
	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := executeCommand(t, "--config", missingConfigPath, "init", "--repo-path", repoDir)
	if err == nil {
		t.Fatal("Execute() error = nil, want config guidance")
	}
	if !strings.Contains(err.Error(), "codefind config --server-url") {
		t.Fatalf("error = %v", err)
	}
}

func TestInitCommandRequiresAuthentication(t *testing.T) {
	restore := useFakeTokenManager("", keychain.ErrNotFound)
	defer restore()

	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	configPath := writeTestConfigWithOrg(t, "http://127.0.0.1:8080", "org_123")
	_, err := executeCommand(t, "--config", configPath, "init", "--repo-path", repoDir)
	if err == nil {
		t.Fatal("Execute() error = nil, want auth guidance")
	}
	if !strings.Contains(err.Error(), "codefind auth login") {
		t.Fatalf("error = %v", err)
	}
}

func TestInitCommandIsIdempotent(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, "http://127.0.0.1:8080", "org_123")
	if _, err := executeCommand(t, "--config", configPath, "init", "--repo-path", repoDir); err != nil {
		t.Fatalf("initial Execute() error = %v", err)
	}

	repoID, err := indexer.DeriveRepoID(repoDir)
	if err != nil {
		t.Fatalf("DeriveRepoID() error = %v", err)
	}
	manifest, err := indexer.LoadManifest("org_123", repoID)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	manifest.LastCommit = "baseline"
	manifest.Files["main.go"] = indexer.ManifestFile{Path: "main.go", ContentHash: "abc123"}
	initializedAt := manifest.InitializedAt
	if err := indexer.SaveManifest(manifest); err != nil {
		t.Fatalf("SaveManifest() error = %v", err)
	}

	secondOutput, err := executeCommand(t, "--config", configPath, "init", "--repo-path", repoDir)
	if err != nil {
		t.Fatalf("second Execute() error = %v", err)
	}

	reloaded, err := indexer.LoadManifest("org_123", repoID)
	if err != nil {
		t.Fatalf("LoadManifest() after rerun error = %v", err)
	}
	if reloaded.LastCommit != "baseline" {
		t.Fatalf("reloaded.LastCommit = %q, want baseline", reloaded.LastCommit)
	}
	if _, ok := reloaded.Files["main.go"]; !ok {
		t.Fatalf("reloaded.Files = %#v, want existing file metadata preserved", reloaded.Files)
	}
	if reloaded.InitializedAt != initializedAt {
		t.Fatalf("reloaded.InitializedAt = %q, want %q", reloaded.InitializedAt, initializedAt)
	}
	if !strings.Contains(secondOutput, "Status: project already initialized") {
		t.Fatalf("output = %q", secondOutput)
	}
	if !strings.Contains(secondOutput, "Next Step: codefind index run") {
		t.Fatalf("output = %q", secondOutput)
	}
}

func TestInitCommandJSONFlagPrintsJSONOnly(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, "http://127.0.0.1:8080", "org_123")
	output, err := executeCommand(t, "--config", configPath, "init", "--repo-path", repoDir, "--json")
	if err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, output)
	}
	if !strings.Contains(output, `"message": "project initialized"`) {
		t.Fatalf("output = %q", output)
	}
	if strings.Contains(output, "Status:") || strings.Contains(output, "┏━╸┏━┓╺┳┓┏━╸") {
		t.Fatalf("output = %q", output)
	}
}

func TestIndexRemoveCommandCallsClearRepoAndRemovesManifest(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	repoDir := t.TempDir()
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var sawMethod, sawPath string
	var sawBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawMethod = r.Method
		sawPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&sawBody); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","repo_id":"repo-a","cleared":true}`))
	}))
	defer server.Close()

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, server.URL, "org_123")
	initTestProject(t, configPath, repoDir, "--repo-id", "repo-a")
	manifestPath, err := indexer.ManifestPath("org_123", "repo-a")
	if err != nil {
		t.Fatalf("ManifestPath() error = %v", err)
	}
	output, err := executeCommand(t, "--config", configPath, "index", "remove")
	if err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, output)
	}
	if sawMethod != http.MethodDelete || sawPath != "/index/remove" {
		t.Fatalf("saw %s %s, want DELETE /index/remove", sawMethod, sawPath)
	}
	if sawBody["repo_id"] != "repo-a" {
		t.Fatalf("body repo_id = %v", sawBody["repo_id"])
	}
	if !strings.Contains(output, `"cleared": true`) {
		t.Fatalf("output = %q", output)
	}
	if _, statErr := os.Stat(manifestPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("manifest still exists after remove: %v", statErr)
	}
}

func TestIndexRemoveCommandDoesNotResetManifestOnBackendFailure(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	repoDir := t.TempDir()
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer server.Close()

	restore := useFakeTokenManager(makeTestToken(time.Now().UTC().Add(time.Hour), "org_123"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, server.URL, "org_123")
	initTestProject(t, configPath, repoDir, "--repo-id", "repo-a")
	manifestPath, err := indexer.ManifestPath("org_123", "repo-a")
	if err != nil {
		t.Fatalf("ManifestPath() error = %v", err)
	}
	seededContent := []byte(`{"schema_version":1,"repo_id":"repo-a","org_id":"org_123","repo_path":"` + repoDir + `","initialized_at":"2026-03-11T00:00:00Z","files":{"main.go":{}}}`)
	if err := os.WriteFile(manifestPath, seededContent, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err = executeCommand(t, "--config", configPath, "index", "remove")
	if err == nil {
		t.Fatal("expected error from backend failure, got nil")
	}

	// Manifest must remain unchanged — still has the original file entry.
	content, readErr := os.ReadFile(manifestPath)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if !strings.Contains(string(content), `"main.go"`) {
		t.Fatalf("manifest was reset despite backend failure: %s", content)
	}
}

func TestIndexRemoveCommandRequiresAdminRole(t *testing.T) {
	repoDir := t.TempDir()
	t.Chdir(repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	restore := useFakeTokenManager(makeTestTokenWithRole(time.Now().UTC().Add(time.Hour), "org_123", "org:member"), nil)
	defer restore()

	configPath := writeTestConfigWithOrg(t, "http://127.0.0.1:8080", "org_123")
	_, err := executeCommand(t, "--config", configPath, "index", "remove")
	if err == nil {
		t.Fatal("Execute() error = nil, want admin-role guidance")
	}
	if !strings.Contains(err.Error(), "org:admin role required") {
		t.Fatalf("error = %v", err)
	}
}
