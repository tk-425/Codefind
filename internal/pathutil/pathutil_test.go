package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsWithinDir(t *testing.T) {
	// Use a real temp dir so filepath.Abs resolves correctly
	base, err := os.MkdirTemp("", "pathutil-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(base)

	sub := filepath.Join(base, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		baseDir string
		want    bool
		wantErr bool
	}{
		{
			name:    "valid path inside base",
			path:    filepath.Join(base, "file.go"),
			baseDir: base,
			want:    true,
		},
		{
			name:    "valid nested path inside base",
			path:    filepath.Join(base, "sub", "file.go"),
			baseDir: base,
			want:    true,
		},
		{
			name:    "path is exactly base dir (not within)",
			path:    base,
			baseDir: base,
			want:    false,
		},
		{
			name:    "path traversal escape via ..",
			path:    filepath.Join(base, "..", "escape.go"),
			baseDir: base,
			want:    false,
		},
		{
			name:    "prefix collision — base/foo vs base/foobar",
			path:    base + "bar" + string(filepath.Separator) + "file.go",
			baseDir: base,
			want:    false,
		},
		{
			name:    "absolute path outside base",
			path:    "/etc/passwd",
			baseDir: base,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsWithinDir(tt.path, tt.baseDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsWithinDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsWithinDir(%q, %q) = %v, want %v", tt.path, tt.baseDir, got, tt.want)
			}
		})
	}
}
