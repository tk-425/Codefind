package indexer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tk-425/Codefind/internal/config"
)

// FileChange represents a changed file with its status
type FileChange struct {
	Path   string
	Status ChangeStatus
}

// ChangeStatus indicates what happened to a file
type ChangeStatus int

const (
	StatusAdded ChangeStatus = iota
	StatusModified
	StatusDeleted
	StatusRenamed
)

// ChangeDetectionResult contains all detected changes
type ChangeDetectionResult struct {
	Added         []string
	Modified      []string
	Deleted       []string
	Renamed       map[string]string // old -> new
	IsGitRepo     bool
	LastCommit    string
	CurrentCommit string
}

// IsFullIndex returns true if no prior commit exists (first-time index)
func (c *ChangeDetectionResult) IsFullIndex() bool {
	return c == nil || c.LastCommit == ""
}

// HasChanges returns true if any files were changed
func (c *ChangeDetectionResult) HasChanges() bool {
	return len(c.Added) > 0 || len(c.Modified) > 0 || len(c.Deleted) > 0 || len(c.Renamed) > 0
}

// DetectGitChanges finds files changed since lastCommit using git diff
func DetectGitChanges(repoPath, lastCommit string) (*ChangeDetectionResult, error) {
	if err := validateRepoPath(repoPath); err != nil {
		return nil, fmt.Errorf("invalid repoPath: %w", err)
	}
	if lastCommit != "" {
		if err := validateCommitHash(lastCommit); err != nil {
			return nil, fmt.Errorf("invalid lastCommit: %w", err)
		}
	}

	result := &ChangeDetectionResult{
		Added:      []string{},
		Modified:   []string{},
		Deleted:    []string{},
		Renamed:    make(map[string]string),
		IsGitRepo:  true,
		LastCommit: lastCommit,
	}

	// Get current HEAD
	currentCommit, err := getHeadCommit(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	result.CurrentCommit = currentCommit

	// If this is first index (no last commit), signal full index needed
	if lastCommit == "" {
		return result, nil
	}

	// If commits are the same, no changes
	if lastCommit == currentCommit {
		return result, nil
	}

	// Run git diff --name-status to get changed files
	cmd := exec.Command("git", "-C", repoPath, "diff", "--name-status", lastCommit, "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	// Parse output format:
	// A   file.go        (added)
	// M   file.go        (modified)
	// D   file.go        (deleted)
	// R100 old.go new.go (renamed with 100% similarity)
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		file := parts[1]

		// Only process code files
		if !IsCodeFile(file) {
			continue
		}

		switch {
		case status == "A":
			result.Added = append(result.Added, file)
		case status == "M":
			result.Modified = append(result.Modified, file)
		case status == "D":
			result.Deleted = append(result.Deleted, file)
		case strings.HasPrefix(status, "R"):
			// Rename format: R100 oldfile newfile
			if len(parts) >= 3 {
				oldFile := parts[1]
				newFile := parts[2]
				if IsCodeFile(newFile) {
					result.Renamed[oldFile] = newFile
				}
			}
		}
	}

	return result, nil
}

// getHeadCommit returns the current HEAD commit hash
func getHeadCommit(repoPath string) (string, error) {
	if err := validateRepoPath(repoPath); err != nil {
		return "", fmt.Errorf("invalid repoPath: %w", err)
	}
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// DetectMtimeChanges finds changed files by comparing mtime
// Used for non-git repositories as a fallback detection method
func DetectMtimeChanges(repoPath string, manifest *config.RepositoryManifest) (*ChangeDetectionResult, error) {
	result := &ChangeDetectionResult{
		Added:     []string{},
		Modified:  []string{},
		Deleted:   []string{},
		Renamed:   make(map[string]string),
		IsGitRepo: false,
	}

	// Get absolute path
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get current files in the repo
	discovery, err := DiscoverFiles(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to discover files: %w", err)
	}

	// Track current files for deletion detection
	currentFiles := make(map[string]bool)

	for _, file := range discovery.Files {
		currentFiles[file.Path] = true

		// Check if file existed in previous index
		existingInfo, exists := manifest.IndexedFiles[file.Path]
		if !exists {
			// New file
			result.Added = append(result.Added, file.Path)
			continue
		}

		// Get current file mtime
		fullPath := filepath.Join(absPath, file.Path)
		stat, err := os.Stat(fullPath)
		if err != nil {
			// File unreadable, skip
			continue
		}
		currentMtime := stat.ModTime().Format(time.RFC3339)

		// Compare mtime with stored value
		if currentMtime != existingInfo.LastModTime {
			result.Modified = append(result.Modified, file.Path)
		}
	}

	// Find deleted files (were in manifest but not in current files)
	for filePath := range manifest.IndexedFiles {
		if !currentFiles[filePath] {
			result.Deleted = append(result.Deleted, filePath)
		}
	}

	return result, nil
}

// DetectChanges returns changes using git (if available) or mtime fallback
// This is the unified entry point for change detection
func DetectChanges(repoPath string, manifest *config.RepositoryManifest) (*ChangeDetectionResult, error) {
	// Use git-based detection if repo is a git repo and has a previous commit
	if IsGitRepository(repoPath) && manifest.LastIndexedCommit != "" {
		return DetectGitChanges(repoPath, manifest.LastIndexedCommit)
	}

	// Fall back to mtime-based detection for non-git repos or first-time indexing
	return DetectMtimeChanges(repoPath, manifest)
}
