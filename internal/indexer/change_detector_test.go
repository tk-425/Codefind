package indexer

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDetectMtimeChangesWithEmptyManifestTreatsFilesAsAdded(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoPath, "main.py"), []byte("print('x')\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := DetectMtimeChanges(repoPath, &Manifest{Files: map[string]ManifestFile{}})
	if err != nil {
		t.Fatalf("DetectMtimeChanges() error = %v", err)
	}
	if len(result.Added) != 1 {
		t.Fatalf("len(result.Added) = %d, want 1", len(result.Added))
	}
}

func TestDetectMtimeChangesDetectsDeletedFiles(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	result, err := DetectMtimeChanges(repoPath, &Manifest{
		Files: map[string]ManifestFile{
			"deleted.py": {Path: "deleted.py", LastModTime: "2024-01-01T00:00:00Z"},
		},
	})
	if err != nil {
		t.Fatalf("DetectMtimeChanges() error = %v", err)
	}
	if len(result.Deleted) != 1 || result.Deleted[0] != "deleted.py" {
		t.Fatalf("result.Deleted = %#v", result.Deleted)
	}
}

func TestDetectGitChangesFirstIndex(t *testing.T) {
	t.Parallel()

	repoPath := createGitRepoForTest(t)
	result, err := DetectGitChanges(repoPath, "")
	if err != nil {
		t.Fatalf("DetectGitChanges() error = %v", err)
	}
	if !result.IsFullIndex() {
		t.Fatalf("IsFullIndex() = false, want true")
	}
	if result.CurrentCommit == "" {
		t.Fatalf("CurrentCommit is empty")
	}
}

func TestDetectGitChangesNoChangesOnSameCommit(t *testing.T) {
	t.Parallel()

	repoPath := createGitRepoForTest(t)
	head, err := getHeadCommit(repoPath)
	if err != nil {
		t.Fatalf("getHeadCommit() error = %v", err)
	}

	result, err := DetectGitChanges(repoPath, head)
	if err != nil {
		t.Fatalf("DetectGitChanges() error = %v", err)
	}
	if result.HasChanges() {
		t.Fatalf("HasChanges() = true, want false")
	}
}

func createGitRepoForTest(t *testing.T) string {
	t.Helper()

	repoPath := t.TempDir()
	runGit(t, repoPath, "init")
	runGit(t, repoPath, "config", "user.email", "test@example.com")
	runGit(t, repoPath, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repoPath, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	runGit(t, repoPath, "add", ".")
	runGit(t, repoPath, "commit", "-m", "initial")
	return repoPath
}

func runGit(t *testing.T, repoPath string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}
