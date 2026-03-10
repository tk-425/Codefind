package indexer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLanguageFromExtension(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"main.go":   "go",
		"app.py":    "python",
		"app.ts":    "typescript",
		"app.js":    "javascript",
		"Main.java": "java",
		"App.swift": "swift",
		"lib.rs":    "rust",
		"core.ml":   "ocaml",
		"README.md": "unknown",
	}

	for input, want := range tests {
		if got := LanguageFromExtension(input); got != want {
			t.Fatalf("LanguageFromExtension(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDiscoverFilesFindsSupportedCodeFiles(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoPath, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "worker.py"), []byte("print('x')\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# docs\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	discovery, err := DiscoverFiles(repoPath)
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}
	if len(discovery.Files) != 2 {
		t.Fatalf("len(discovery.Files) = %d, want 2", len(discovery.Files))
	}
}

func TestDiscoverFilesSkipsIgnoredDirectories(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	nodeModules := filepath.Join(repoPath, "node_modules")
	if err := os.MkdirAll(nodeModules, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(nodeModules, "bad.js"), []byte("console.log('x')\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "ok.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	discovery, err := DiscoverFiles(repoPath)
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}
	if len(discovery.Files) != 1 || discovery.Files[0].Path != "ok.go" {
		t.Fatalf("discovery.Files = %#v", discovery.Files)
	}
}
