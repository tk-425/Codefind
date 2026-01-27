package indexer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tk-425/Codefind/internal/config"
)

func TestGetHeadCommit(t *testing.T) {
	// Get the project root (we're in internal/indexer, so go up 2 levels)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(cwd, "..", "..")

	commit, err := getHeadCommit(projectRoot)
	if err != nil {
		t.Fatalf("getHeadCommit failed: %v", err)
	}

	// Commit hash should be 40 characters (SHA-1)
	if len(commit) != 40 {
		t.Errorf("Expected 40-char commit hash, got %d chars: %s", len(commit), commit)
	}

	t.Logf("Current HEAD: %s", commit)
}

func TestDetectGitChanges_FirstIndex(t *testing.T) {
	// Get the project root
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(cwd, "..", "..")

	// First index (no lastCommit) should return result with IsFullIndex = true
	result, err := DetectGitChanges(projectRoot, "")
	if err != nil {
		t.Fatalf("DetectGitChanges failed: %v", err)
	}

	if !result.IsFullIndex() {
		t.Error("Expected IsFullIndex() to be true for empty lastCommit")
	}

	if result.CurrentCommit == "" {
		t.Error("Expected CurrentCommit to be set")
	}

	t.Logf("Current commit: %s", result.CurrentCommit)
	t.Logf("IsFullIndex: %v", result.IsFullIndex())
}

func TestDetectGitChanges_NoChanges(t *testing.T) {
	// Get the project root
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(cwd, "..", "..")

	// Get current HEAD
	currentCommit, err := getHeadCommit(projectRoot)
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}

	// Detect changes with same commit should show no changes
	result, err := DetectGitChanges(projectRoot, currentCommit)
	if err != nil {
		t.Fatalf("DetectGitChanges failed: %v", err)
	}

	if result.HasChanges() {
		t.Errorf("Expected no changes when comparing same commit, got: +%d ~%d -%d",
			len(result.Added), len(result.Modified), len(result.Deleted))
	}

	t.Logf("Changes: +%d ~%d -%d (expected all zeros)",
		len(result.Added), len(result.Modified), len(result.Deleted))
}

func TestDetectGitChanges_WithHistory(t *testing.T) {
	// Get the project root
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(cwd, "..", "..")

	// Try to detect changes from a few commits back
	// For a more robust test, we'd need to query git log
	// For now, just test that the function handles errors gracefully
	result, err := DetectGitChanges(projectRoot, "HEAD~5")
	if err != nil {
		// This is expected if HEAD~5 doesn't exist or git diff fails
		t.Logf("DetectGitChanges with HEAD~5 returned error (may be expected): %v", err)
		return
	}

	t.Logf("Changes since HEAD~5: +%d added, ~%d modified, -%d deleted",
		len(result.Added), len(result.Modified), len(result.Deleted))

	// Log actual changed files if any
	if len(result.Added) > 0 {
		t.Logf("Added files: %v", result.Added)
	}
	if len(result.Modified) > 0 {
		t.Logf("Modified files: %v", result.Modified)
	}
	if len(result.Deleted) > 0 {
		t.Logf("Deleted files: %v", result.Deleted)
	}
}

func TestIsGitRepository(t *testing.T) {
	// Get the project root
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(cwd, "..", "..")

	if !IsGitRepository(projectRoot) {
		t.Error("Expected project root to be a git repository")
	}

	// /tmp should not be a git repo
	if IsGitRepository("/tmp") {
		t.Error("Expected /tmp to NOT be a git repository")
	}
}

func TestDetectMtimeChanges_EmptyManifest(t *testing.T) {
	// Get the project root
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(cwd, "..", "..")

	// Create an empty manifest (simulating first-time index)
	manifest := &config.RepositoryManifest{
		RepoID:       "test-repo",
		IndexedFiles: make(map[string]config.FileInfo),
	}

	result, err := DetectMtimeChanges(projectRoot, manifest)
	if err != nil {
		t.Fatalf("DetectMtimeChanges failed: %v", err)
	}

	// With empty manifest, all files should be detected as "added"
	if len(result.Added) == 0 {
		t.Error("Expected files to be detected as added with empty manifest")
	}

	t.Logf("Detected %d files as added (first-time index)", len(result.Added))
	t.Logf("IsGitRepo: %v", result.IsGitRepo)
}

func TestDetectMtimeChanges_WithExistingFiles(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "codefind-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test Python file
	testFile := filepath.Join(tempDir, "main.py")
	if err := os.WriteFile(testFile, []byte("def hello(): pass\n"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Get the file's mtime
	stat, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}

	// Create manifest with the file already indexed (same mtime)
	manifest := &config.RepositoryManifest{
		RepoID: "test-repo",
		IndexedFiles: map[string]config.FileInfo{
			"main.py": {
				Language:    "python",
				LastModTime: stat.ModTime().Format("2006-01-02T15:04:05Z07:00"),
			},
		},
	}

	result, err := DetectMtimeChanges(tempDir, manifest)
	if err != nil {
		t.Fatalf("DetectMtimeChanges failed: %v", err)
	}

	// File should NOT be detected as changed (same mtime)
	if len(result.Modified) > 0 {
		t.Errorf("Expected no modified files, got: %v", result.Modified)
	}

	t.Logf("Changes: +%d ~%d -%d (expected all zeros)",
		len(result.Added), len(result.Modified), len(result.Deleted))
}

func TestDetectMtimeChanges_DeletedFile(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "codefind-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create manifest with a file that no longer exists
	manifest := &config.RepositoryManifest{
		RepoID: "test-repo",
		IndexedFiles: map[string]config.FileInfo{
			"deleted.py": {
				Language:    "python",
				LastModTime: "2024-01-01T00:00:00Z",
			},
		},
	}

	result, err := DetectMtimeChanges(tempDir, manifest)
	if err != nil {
		t.Fatalf("DetectMtimeChanges failed: %v", err)
	}

	// File should be detected as deleted
	if len(result.Deleted) != 1 {
		t.Errorf("Expected 1 deleted file, got: %d", len(result.Deleted))
	}

	t.Logf("Deleted files: %v", result.Deleted)
}

func TestDetectChanges_GitRepo(t *testing.T) {
	// Get the project root (which is a git repo)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(cwd, "..", "..")

	// Create manifest with a LastIndexedCommit to trigger git-based detection
	currentCommit, _ := getHeadCommit(projectRoot)
	manifest := &config.RepositoryManifest{
		RepoID:            "test-repo",
		LastIndexedCommit: currentCommit,
		IndexedFiles:      make(map[string]config.FileInfo),
	}

	result, err := DetectChanges(projectRoot, manifest)
	if err != nil {
		t.Fatalf("DetectChanges failed: %v", err)
	}

	// Should use git-based detection
	if !result.IsGitRepo {
		t.Error("Expected git-based detection for git repository with LastIndexedCommit")
	}

	t.Logf("DetectChanges used git-based detection: IsGitRepo=%v", result.IsGitRepo)
	t.Logf("Changes: +%d ~%d -%d", len(result.Added), len(result.Modified), len(result.Deleted))
}

func TestDetectChanges_NonGitRepo(t *testing.T) {
	// Create a temp directory (not a git repo)
	tempDir, err := os.MkdirTemp("", "codefind-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFile := filepath.Join(tempDir, "main.py")
	if err := os.WriteFile(testFile, []byte("def hello(): pass\n"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	manifest := &config.RepositoryManifest{
		RepoID:       "test-repo",
		IndexedFiles: make(map[string]config.FileInfo),
	}

	result, err := DetectChanges(tempDir, manifest)
	if err != nil {
		t.Fatalf("DetectChanges failed: %v", err)
	}

	// Should use mtime-based detection
	if result.IsGitRepo {
		t.Error("Expected mtime-based detection for non-git directory")
	}

	t.Logf("DetectChanges used mtime-based detection: IsGitRepo=%v", result.IsGitRepo)
	t.Logf("Changes: +%d ~%d -%d", len(result.Added), len(result.Modified), len(result.Deleted))
}
