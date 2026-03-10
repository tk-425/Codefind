package indexer

import (
	"os"
	"testing"
)

func TestResetManifestWritesCleanDefaultManifest(t *testing.T) {
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

	got, err := LoadManifest(orgID, repoID)
	if err != nil {
		t.Fatalf("LoadManifest() after reset error = %v", err)
	}
	if len(got.Files) != 0 {
		t.Errorf("expected empty Files after reset, got %d entries", len(got.Files))
	}
	if got.RepoID != repoID {
		t.Errorf("expected RepoID %q, got %q", repoID, got.RepoID)
	}
	if got.OrgID != orgID {
		t.Errorf("expected OrgID %q, got %q", orgID, got.OrgID)
	}
}

func TestResetManifestCreatesManifestWhenNoneExists(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	orgID := "org_test"
	repoID := "repo-new"

	if err := ResetManifest(orgID, repoID); err != nil {
		t.Fatalf("ResetManifest() error = %v", err)
	}

	manifestPath, err := ManifestPath(orgID, repoID)
	if err != nil {
		t.Fatalf("ManifestPath() error = %v", err)
	}
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Errorf("expected manifest file to exist at %s after reset", manifestPath)
	}
}
