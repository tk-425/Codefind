package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- validateBaseURL tests ---

func TestValidateBaseURL_Localhost(t *testing.T) {
	if err := validateBaseURL("http://localhost:8080"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateBaseURL_127(t *testing.T) {
	if err := validateBaseURL("http://127.0.0.1:8080"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateBaseURL_RFC1918(t *testing.T) {
	if err := validateBaseURL("http://192.168.1.10:8080"); err != nil {
		t.Fatalf("expected nil for RFC 1918 address, got %v", err)
	}
}

func TestValidateBaseURL_TailscaleCGNAT(t *testing.T) {
	if err := validateBaseURL("http://100.108.160.103:8080"); err != nil {
		t.Fatalf("expected nil for Tailscale CGNAT address, got %v", err)
	}
}

func TestValidateBaseURL_ExternalHostRejected(t *testing.T) {
	if err := validateBaseURL("http://evil.com/path"); err == nil {
		t.Fatal("expected error for external host, got nil")
	}
}

func TestValidateBaseURL_PublicIPRejected(t *testing.T) {
	if err := validateBaseURL("http://8.8.8.8:8080"); err == nil {
		t.Fatal("expected error for public IP, got nil")
	}
}

func TestValidateBaseURL_EmptyRejected(t *testing.T) {
	if err := validateBaseURL(""); err == nil {
		t.Fatal("expected error for empty URL, got nil")
	}
}

// --- buildURL tests ---

func TestBuildURL_ValidEndpoint(t *testing.T) {
	got, err := buildURL("http://localhost:8080", "/collections")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != "http://localhost:8080/collections" {
		t.Fatalf("unexpected URL: %s", got)
	}
}

func TestBuildURL_EndpointSchemeOverrideRejected(t *testing.T) {
	_, err := buildURL("http://localhost:8080", "http://evil.com/steal")
	if err == nil {
		t.Fatal("expected error for scheme-override endpoint, got nil")
	}
}

func TestBuildURL_UserInfoHostInjectionRejected(t *testing.T) {
	// "@evil.com/path" causes url.Parse to treat "evil.com" as the host
	// via user-info injection: http://localhost:8080@evil.com/path
	_, err := buildURL("http://localhost:8080", "@evil.com/path")
	if err == nil {
		t.Fatal("expected error for user-info host injection, got nil")
	}
}

func TestListCollectionsSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"collections":["repo-a","repo-b"],"error":""}`))
	}))
	defer ts.Close()

	ac, err := NewAPIClient(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}
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

	ac, err := NewAPIClient(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}
	_, err = ac.ListCollections()
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

	ac, err := NewAPIClient(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}
	_, err = ac.ListCollections()
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode response") {
		t.Fatalf("unexpected error: %v", err)
	}
}
