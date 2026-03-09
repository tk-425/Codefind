package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tk-425/Codefind/pkg/api"
)

type fakeTokenLoader struct {
	token string
	err   error
}

func (f fakeTokenLoader) LoadToken() (string, error) {
	return f.token, f.err
}

func TestClientInjectsAuthorizationHeaderFromTokenLoader(t *testing.T) {
	t.Parallel()

	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","ollama_status":"ok","qdrant_status":"ok"}`))
	}))
	defer server.Close()

	client, err := New(server.URL, fakeTokenLoader{token: "token-123"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if _, err := client.Health(context.Background()); err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	if authHeader != "Bearer token-123" {
		t.Fatalf("Authorization header = %q, want %q", authHeader, "Bearer token-123")
	}
}

func TestClientHealthDecodesResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","ollama_status":"ok","qdrant_status":"ok","timestamp":"2026-03-07T00:00:00Z"}`))
	}))
	defer server.Close()

	client, err := New(server.URL, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	response, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	if response.Status != "ok" || response.Timestamp == "" {
		t.Fatalf("Health() = %#v, want decoded status and timestamp", response)
	}
}

func TestClientGetOrganizationsDecodesResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs" {
			t.Fatalf("path = %q, want /orgs", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"organization_id":"org_123","organization_name":"Acme","role":"org:admin"}],"total_count":1}`))
	}))
	defer server.Close()

	client, err := New(server.URL, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	response, err := client.GetOrganizations(context.Background())
	if err != nil {
		t.Fatalf("GetOrganizations() error = %v", err)
	}

	if response.TotalCount != 1 || response.Data[0].OrganizationID != "org_123" {
		t.Fatalf("GetOrganizations() = %#v, want decoded org list", response)
	}
}

func TestClientCreateAdminInvitationSendsJSONBody(t *testing.T) {
	t.Parallel()

	var contentType string
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		decoded, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		body = string(decoded)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"orginv_1","email_address":"new@example.com","role":"org:member","status":"pending","organization_id":"org_123"}`))
	}))
	defer server.Close()

	client, err := New(server.URL, fakeTokenLoader{token: "token-123"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	response, err := client.CreateAdminInvitation(context.Background(), api.CreateOrganizationInvitationRequest{
		EmailAddress: "new@example.com",
		Role:         "org:member",
	})
	if err != nil {
		t.Fatalf("CreateAdminInvitation() error = %v", err)
	}

	if contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}
	if body != `{"email_address":"new@example.com","role":"org:member"}` {
		t.Fatalf("body = %q", body)
	}
	if response.ID != "orginv_1" {
		t.Fatalf("response = %#v, want decoded invitation", response)
	}
}

func TestClientRevokeAdminInvitationPostsToRevokeEndpoint(t *testing.T) {
	t.Parallel()

	var method string
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"orginv_1","email_address":"new@example.com","role":"org:member","status":"revoked","organization_id":"org_123"}`))
	}))
	defer server.Close()

	client, err := New(server.URL, fakeTokenLoader{token: "token-123"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	response, err := client.RevokeAdminInvitation(context.Background(), "orginv_1")
	if err != nil {
		t.Fatalf("RevokeAdminInvitation() error = %v", err)
	}

	if method != http.MethodPost || path != "/admin/invitations/orginv_1/revoke" {
		t.Fatalf("saw %s %s", method, path)
	}
	if response.Status != "revoked" {
		t.Fatalf("response = %#v, want revoked invitation", response)
	}
}

func TestClientGetCollectionsDecodesResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections" {
			t.Fatalf("path = %q, want /collections", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"repo_id":"repo-a"}],"total_count":1}`))
	}))
	defer server.Close()

	client, err := New(server.URL, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	response, err := client.GetCollections(context.Background())
	if err != nil {
		t.Fatalf("GetCollections() error = %v", err)
	}

	if response.TotalCount != 1 || response.Data[0].RepoID != "repo-a" {
		t.Fatalf("GetCollections() = %#v", response)
	}
}

func TestClientQueryPostsJSONBody(t *testing.T) {
	t.Parallel()

	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoded, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		body = string(decoded)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"chunk-1","score":0.9,"repo_id":"repo-a"}],"total_count":1,"page":1,"page_size":10,"has_more":false}`))
	}))
	defer server.Close()

	client, err := New(server.URL, fakeTokenLoader{token: "token-123"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	response, err := client.Query(context.Background(), api.QueryRequest{
		QueryText: "main",
		RepoID:    "repo-a",
		Page:      1,
		PageSize:  10,
		TopK:      10,
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if body != `{"query_text":"main","repo_id":"repo-a","page":1,"page_size":10,"top_k":10}` {
		t.Fatalf("body = %q", body)
	}
	if response.TotalCount != 1 || response.Data[0].RepoID != "repo-a" {
		t.Fatalf("Query() = %#v", response)
	}
}
