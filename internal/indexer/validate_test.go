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
