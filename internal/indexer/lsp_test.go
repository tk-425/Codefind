package indexer

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/tk-425/Codefind/internal/lsp"
)

func TestIndexerWarmLSPsUsesDiscoveredLanguages(t *testing.T) {
	repoPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoPath, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "app.ts"), []byte("export const x = 1;\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	originalWarmLanguages := warmLanguages
	warmLanguages = func(rootPath string, languages []string) (*lsp.WarmState, error) {
		return &lsp.WarmState{
			RequestedLanguages: lsp.UniqueLSPKeys(languages),
			Languages:          map[string]lsp.LanguageWarmState{},
			Clients:            map[string]*lsp.Client{},
		}, nil
	}
	defer func() { warmLanguages = originalWarmLanguages }()

	indexer, err := New(repoPath, &Manifest{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	state, err := indexer.WarmLSPs()
	if err != nil {
		t.Fatalf("WarmLSPs() error = %v", err)
	}

	if state == nil {
		t.Fatalf("WarmLSPs() returned nil state")
	}
	if indexer.LSPState() == nil {
		t.Fatalf("LSPState() should be set after WarmLSPs()")
	}
	if got, want := len(state.RequestedLanguages), 2; got != want {
		t.Fatalf("len(RequestedLanguages) = %d, want %d", got, want)
	}
	if !slices.Contains(state.RequestedLanguages, "go") {
		t.Fatalf("expected go in RequestedLanguages, got %v", state.RequestedLanguages)
	}
	if !slices.Contains(state.RequestedLanguages, "typescript/javascript") {
		t.Fatalf("expected typescript/javascript in RequestedLanguages, got %v", state.RequestedLanguages)
	}
}
