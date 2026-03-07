package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
