package authflow

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBuildSignInURL(t *testing.T) {
	t.Parallel()

	got, err := BuildSignInURL("http://100.64.0.5:8080", "http://127.0.0.1:49152/callback")
	if err != nil {
		t.Fatalf("BuildSignInURL() error = %v", err)
	}

	want := "http://100.64.0.5:8080/auth/signin?redirect_uri=http%3A%2F%2F127.0.0.1%3A49152%2Fcallback"
	if got != want {
		t.Fatalf("BuildSignInURL() = %q, want %q", got, want)
	}
}

func TestCallbackHandlerAcceptsTokenPost(t *testing.T) {
	t.Parallel()

	tokenCh := make(chan string, 1)
	handler := callbackHandler(tokenCh, "http://localhost:5173")
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, callbackPath, strings.NewReader(`{"token":"jwt-123"}`))
	request.Header.Set("Origin", "http://localhost:5173")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := <-tokenCh; got != "jwt-123" {
		t.Fatalf("token = %q, want %q", got, "jwt-123")
	}
}

func TestCallbackHandlerRejectsGet(t *testing.T) {
	t.Parallel()

	tokenCh := make(chan string, 1)
	handler := callbackHandler(tokenCh, "http://localhost:5173")
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, callbackPath, nil)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}

func TestCallbackHandlerSupportsCorsPreflight(t *testing.T) {
	t.Parallel()

	tokenCh := make(chan string, 1)
	handler := callbackHandler(tokenCh, "http://localhost:5173")
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, callbackPath, nil)
	request.Header.Set("Origin", "http://localhost:5173")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("allow origin = %q, want %q", got, "http://localhost:5173")
	}
}

func TestCallbackHandlerRejectsUnexpectedOrigin(t *testing.T) {
	t.Parallel()

	tokenCh := make(chan string, 1)
	handler := callbackHandler(tokenCh, "http://localhost:5173")
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, callbackPath, strings.NewReader(`{"token":"jwt-123"}`))
	request.Header.Set("Origin", "http://malicious.example")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestDecodeTokenClaimsExtractsOrgID(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(TokenClaims{
		OrgID: "org_123",
		Exp:   time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	token := "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"

	claims, err := DecodeTokenClaims(token)
	if err != nil {
		t.Fatalf("DecodeTokenClaims() error = %v", err)
	}
	if claims.OrgID != "org_123" {
		t.Fatalf("OrgID = %q, want %q", claims.OrgID, "org_123")
	}
}

func TestDecodeTokenClaimsExtractsNestedOrgClaims(t *testing.T) {
	t.Parallel()

	payload := map[string]any{
		"exp": time.Now().Add(time.Hour).Unix(),
		"o": map[string]any{
			"id":  "org_456",
			"rol": "admin",
		},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	token := "header." + base64.RawURLEncoding.EncodeToString(encoded) + ".signature"

	claims, err := DecodeTokenClaims(token)
	if err != nil {
		t.Fatalf("DecodeTokenClaims() error = %v", err)
	}
	if claims.OrgID != "org_456" {
		t.Fatalf("OrgID = %q, want %q", claims.OrgID, "org_456")
	}
	if claims.OrgRole != "org:admin" {
		t.Fatalf("OrgRole = %q, want %q", claims.OrgRole, "org:admin")
	}
}

func TestStartCallbackServerTimesOutWithContext(t *testing.T) {
	t.Parallel()

	listener, err := NewLocalCallbackListener()
	if err != nil {
		t.Fatalf("NewLocalCallbackListener() error = %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, waitForToken, err := StartCallbackServer(ctx, listener, "http://localhost:5173")
	if err != nil {
		t.Fatalf("StartCallbackServer() error = %v", err)
	}

	if _, err := waitForToken(); err == nil {
		t.Fatal("waitForToken() error = nil, want context error")
	}
}
