package indexer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRepoPath(t *testing.T) {
	// Create a real temp dir for valid path tests
	validDir, err := os.MkdirTemp("", "validate-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(validDir)

	// Create a real file (not a directory)
	validFile := filepath.Join(validDir, "file.txt")
	if err := os.WriteFile(validFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid absolute directory",
			path:    validDir,
			wantErr: false,
		},
		{
			name:    "relative path",
			path:    "relative/path",
			wantErr: true,
		},
		{
			name:    "non-existent path",
			path:    "/tmp/does-not-exist-xyz-123",
			wantErr: true,
		},
		{
			name:    "path is a file not a directory",
			path:    validFile,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRepoPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRepoPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCommitHash(t *testing.T) {
	tests := []struct {
		name    string
		hash    string
		wantErr bool
	}{
		{
			name:    "full 40-char SHA",
			hash:    "a3f1b2c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0",
			wantErr: false,
		},
		{
			name:    "short 7-char SHA",
			hash:    "abc1234",
			wantErr: false,
		},
		{
			name:    "empty string",
			hash:    "",
			wantErr: true,
		},
		{
			name:    "uppercase letters",
			hash:    "A3F1B2C4D5E6F7A8",
			wantErr: true,
		},
		{
			name:    "input with semicolon",
			hash:    "abc1234; rm -rf /",
			wantErr: true,
		},
		{
			name:    "input with space",
			hash:    "abc1234 HEAD",
			wantErr: true,
		},
		{
			name:    "input with pipe",
			hash:    "abc1234|cat /etc/passwd",
			wantErr: true,
		},
		{
			name:    "too short (6 chars)",
			hash:    "abc123",
			wantErr: true,
		},
		{
			name:    "too long (65 chars)",
			hash:    "a3f1b2c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommitHash(tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCommitHash(%q) error = %v, wantErr %v", tt.hash, err, tt.wantErr)
			}
		})
	}
}
