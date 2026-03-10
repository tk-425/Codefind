package lsp

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestSupportedLSPKeysAreStableAndSorted(t *testing.T) {
	t.Parallel()

	keys := SupportedLSPKeys()
	if len(keys) != len(KnownLSPs) {
		t.Fatalf("len(keys) = %d, want %d", len(keys), len(KnownLSPs))
	}
	if !slices.IsSorted(keys) {
		t.Fatalf("keys should be sorted: %#v", keys)
	}
	if !slices.Contains(keys, "go") || !slices.Contains(keys, "typescript/javascript") {
		t.Fatalf("expected known keys in %#v", keys)
	}
}

func TestWarmLanguagesOnlyUsesUniqueLSPKeys(t *testing.T) {
	originalInitializeClient := initializeClient
	initializeClient = func(ctx context.Context, client *Client) error { return nil }
	defer func() { initializeClient = originalInitializeClient }()

	attempts := map[string]int{}
	factory := func(language, rootPath string) (*Client, error) {
		attempts[language]++
		return &Client{}, nil
	}

	state, err := WarmLanguagesWithFactory("/tmp/repo", []string{"typescript", "javascript", "go"}, factory)
	if err != nil {
		t.Fatalf("WarmLanguagesWithFactory() error = %v", err)
	}

	if got, want := len(state.RequestedLanguages), 2; got != want {
		t.Fatalf("len(RequestedLanguages) = %d, want %d", got, want)
	}
	if attempts["typescript/javascript"] != 1 {
		t.Fatalf("typescript/javascript attempts = %d, want 1", attempts["typescript/javascript"])
	}
	if attempts["go"] != 1 {
		t.Fatalf("go attempts = %d, want 1", attempts["go"])
	}
}

func TestWarmLanguagesRetriesFailures(t *testing.T) {
	attempts := 0
	factory := func(language, rootPath string) (*Client, error) {
		attempts++
		return nil, errors.New("boom")
	}

	state, err := WarmLanguagesWithFactory("/tmp/repo", []string{"go"}, factory)
	if err != nil {
		t.Fatalf("WarmLanguagesWithFactory() error = %v", err)
	}

	if attempts != MaxWarmupRetries {
		t.Fatalf("attempts = %d, want %d", attempts, MaxWarmupRetries)
	}
	if state.Languages["go"].Ready {
		t.Fatalf("go should not be ready")
	}
	if state.Languages["go"].RetryCount != MaxWarmupRetries {
		t.Fatalf("RetryCount = %d, want %d", state.Languages["go"].RetryCount, MaxWarmupRetries)
	}
}

func TestWarmStateShutdownClearsClients(t *testing.T) {
	t.Parallel()

	state := &WarmState{
		Clients: map[string]*Client{
			"go": {},
		},
	}

	if err := state.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if len(state.Clients) != 0 {
		t.Fatalf("Clients should be cleared after Shutdown")
	}
}
