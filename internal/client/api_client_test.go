package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListCollectionsSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"collections":["repo-a","repo-b"],"error":""}`))
	}))
	defer ts.Close()

	ac := NewAPIClient(ts.URL)
	resp, err := ac.ListCollections()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(resp.Collections) != 2 || resp.Collections[0] != "repo-a" || resp.Collections[1] != "repo-b" {
		t.Fatalf("unexpected collections: %#v", resp.Collections)
	}
}

func TestListCollectionsServerErrorField(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"collections":[],"error":"chromadb unavailable"}`))
	}))
	defer ts.Close()

	ac := NewAPIClient(ts.URL)
	_, err := ac.ListCollections()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "server error: chromadb unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListCollectionsDecodeError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer ts.Close()

	ac := NewAPIClient(ts.URL)
	_, err := ac.ListCollections()
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode response") {
		t.Fatalf("unexpected error: %v", err)
	}
}
