package indexer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRepoPath(t *testing.T) {
	t.Parallel()

	validDir := t.TempDir()
	validFile := filepath.Join(validDir, "file.txt")
	if err := os.WriteFile(validFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "valid absolute directory", path: validDir},
		{name: "relative path", path: "relative/path", wantErr: true},
		{name: "non-existent path", path: filepath.Join(validDir, "missing"), wantErr: true},
		{name: "path is file", path: validFile, wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateRepoPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateRepoPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateProjectRoot(t *testing.T) {
	t.Parallel()

	gitRepo := t.TempDir()
	if err := os.Mkdir(filepath.Join(gitRepo, ".git"), 0o755); err != nil {
		t.Fatalf("Mkdir(.git) error = %v", err)
	}

	codeRepo := t.TempDir()
	if err := os.WriteFile(filepath.Join(codeRepo, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}

	emptyDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "git repo", path: gitRepo},
		{name: "supported source files", path: codeRepo},
		{name: "empty directory", path: emptyDir, wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateProjectRoot(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateProjectRoot(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestDeriveRepoID(t *testing.T) {
	t.Parallel()

	repoPath := filepath.Join(t.TempDir(), "Code Find.v2")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}

	repoID, err := DeriveRepoID(repoPath)
	if err != nil {
		t.Fatalf("DeriveRepoID() error = %v", err)
	}
	if repoID != "code-find-v2" {
		t.Fatalf("DeriveRepoID() = %q, want code-find-v2", repoID)
	}
}

func TestValidateCommitHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		hash    string
		wantErr bool
	}{
		{name: "full sha", hash: "a3f1b2c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0"},
		{name: "short sha", hash: "abc1234"},
		{name: "empty", hash: "", wantErr: true},
		{name: "uppercase", hash: "A3F1B2C4", wantErr: true},
		{name: "semicolon", hash: "abc1234; rm -rf /", wantErr: true},
		{name: "space", hash: "abc1234 HEAD", wantErr: true},
		{name: "pipe", hash: "abc1234|cat /etc/passwd", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateCommitHash(tt.hash)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateCommitHash(%q) error = %v, wantErr %v", tt.hash, err, tt.wantErr)
			}
		})
	}
}
