package indexer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResetManifestRemovesManifestFile(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	orgID := "org_test"
	repoID := "repo-a"

	// Seed a manifest with existing file state.
	initial := &Manifest{
		SchemaVersion: ManifestSchemaVersion,
		RepoID:        repoID,
		OrgID:         orgID,
		Files: map[string]ManifestFile{
			"main.go": {ContentHash: "abc123"},
		},
	}
	if err := SaveManifest(initial); err != nil {
		t.Fatalf("SaveManifest() error = %v", err)
	}

	if err := ResetManifest(orgID, repoID); err != nil {
		t.Fatalf("ResetManifest() error = %v", err)
	}

	manifestPath, err := ManifestPath(orgID, repoID)
	if err != nil {
		t.Fatalf("ManifestPath() error = %v", err)
	}
	if _, err := os.Stat(manifestPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected manifest file to be removed, stat error = %v", err)
	}
}

func TestResetManifestIgnoresMissingManifest(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	orgID := "org_test"
	repoID := "repo-new"

	if err := ResetManifest(orgID, repoID); err != nil {
		t.Fatalf("ResetManifest() error = %v", err)
	}
}

func TestLoadInitializedManifestRequiresInitMetadata(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	manifest := &Manifest{
		SchemaVersion: ManifestSchemaVersion,
		RepoID:        "repo-a",
		OrgID:         "org_test",
		Files:         map[string]ManifestFile{},
	}
	if err := SaveManifest(manifest); err != nil {
		t.Fatalf("SaveManifest() error = %v", err)
	}

	_, err := LoadInitializedManifest("org_test", "repo-a", t.TempDir())
	if !errors.Is(err, ErrProjectNotInitialized) {
		t.Fatalf("LoadInitializedManifest() error = %v, want ErrProjectNotInitialized", err)
	}
}

func TestLoadInitializedManifestRejectsPathMismatch(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	repoDir := t.TempDir()
	otherRepoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(repoDir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(otherRepoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(otherRepoDir) error = %v", err)
	}
	manifest, _, err := InitManifest(repoDir, "org_test", "repo-a", testNow())
	if err != nil {
		t.Fatalf("InitManifest() error = %v", err)
	}
	if manifest.RepoPath == "" {
		t.Fatal("manifest.RepoPath = empty, want initialized path")
	}

	_, err = LoadInitializedManifest("org_test", "repo-a", otherRepoDir)
	if err == nil {
		t.Fatal("LoadInitializedManifest() error = nil, want path mismatch")
	}
}

func TestLoadInitializedManifestForPathFindsCustomRepoID(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	subdir := filepath.Join(repoDir, "nested", "pkg")
	if err := os.MkdirAll(subdir, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	manifest, _, err := InitManifest(repoDir, "org_test", "custom-repo-id", testNow())
	if err != nil {
		t.Fatalf("InitManifest() error = %v", err)
	}

	resolved, err := LoadInitializedManifestForPath("org_test", subdir)
	if err != nil {
		t.Fatalf("LoadInitializedManifestForPath() error = %v", err)
	}
	if resolved.RepoID != manifest.RepoID {
		t.Fatalf("resolved.RepoID = %q, want %q", resolved.RepoID, manifest.RepoID)
	}
	if resolved.RepoPath != manifest.RepoPath {
		t.Fatalf("resolved.RepoPath = %q, want %q", resolved.RepoPath, manifest.RepoPath)
	}
}

func TestLoadInitializedManifestForPathRequiresInit(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := LoadInitializedManifestForPath("org_test", repoDir)
	if !errors.Is(err, ErrProjectNotInitialized) {
		t.Fatalf("LoadInitializedManifestForPath() error = %v, want ErrProjectNotInitialized", err)
	}
}

func TestLoadManifestDefaultsLastIndexModeForOlderEntries(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	manifestPath, err := ManifestPath("org_test", "repo-a")
	if err != nil {
		t.Fatalf("ManifestPath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	body := `{
  "schema_version": 1,
  "repo_id": "repo-a",
  "org_id": "org_test",
  "files": {
    "main.go": {
      "path": "main.go",
      "last_chunking_method": "window",
      "fallback_reason": ""
    },
    "retry.go": {
      "path": "retry.go",
      "last_chunking_method": "window",
      "fallback_reason": "timeout"
    }
  }
}
`
	if err := os.WriteFile(manifestPath, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manifest, err := LoadManifest("org_test", "repo-a")
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if manifest.Files["main.go"].LastIndexMode != IndexModeForceWindow {
		t.Fatalf("main.go LastIndexMode = %q, want %q", manifest.Files["main.go"].LastIndexMode, IndexModeForceWindow)
	}
	if manifest.Files["retry.go"].LastIndexMode != IndexModeHybrid {
		t.Fatalf("retry.go LastIndexMode = %q, want %q", manifest.Files["retry.go"].LastIndexMode, IndexModeHybrid)
	}
}

func testNow() time.Time {
	return time.Date(2026, time.March, 11, 12, 0, 0, 0, time.UTC)
}
