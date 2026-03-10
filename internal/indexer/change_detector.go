package indexer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ChangeDetectionResult struct {
	Added         []string
	Modified      []string
	Deleted       []string
	Renamed       map[string]string
	IsGitRepo     bool
	LastCommit    string
	CurrentCommit string
}

func (c *ChangeDetectionResult) IsFullIndex() bool {
	return c == nil || c.LastCommit == ""
}

func (c *ChangeDetectionResult) HasChanges() bool {
	return len(c.Added) > 0 || len(c.Modified) > 0 || len(c.Deleted) > 0 || len(c.Renamed) > 0
}

func DetectChanges(repoPath string, manifest *Manifest) (*ChangeDetectionResult, error) {
	if manifest == nil {
		manifest = &Manifest{Files: make(map[string]ManifestFile)}
	}
	if IsGitRepository(repoPath) {
		return DetectGitChanges(repoPath, manifest.LastCommit)
	}
	return DetectMtimeChanges(repoPath, manifest)
}

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

	currentCommit, err := getHeadCommit(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	result.CurrentCommit = currentCommit

	if lastCommit == "" || lastCommit == currentCommit {
		return result, nil
	}

	cmd := exec.Command("git", "-C", repoPath, "diff", "--name-status", lastCommit, "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	for rawLine := range strings.SplitSeq(string(output), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		filePath := parts[1]
		switch {
		case status == "A":
			if IsCodeFile(filePath) {
				result.Added = append(result.Added, filePath)
			}
		case status == "M":
			if IsCodeFile(filePath) {
				result.Modified = append(result.Modified, filePath)
			}
		case status == "D":
			if IsCodeFile(filePath) {
				result.Deleted = append(result.Deleted, filePath)
			}
		case strings.HasPrefix(status, "R") && len(parts) >= 3:
			oldPath := parts[1]
			newPath := parts[2]
			if IsCodeFile(newPath) {
				result.Renamed[oldPath] = newPath
			}
		}
	}

	return result, nil
}

func DetectMtimeChanges(repoPath string, manifest *Manifest) (*ChangeDetectionResult, error) {
	if err := validateRepoPath(repoPath); err != nil {
		return nil, fmt.Errorf("invalid repoPath: %w", err)
	}
	if manifest == nil {
		manifest = &Manifest{Files: make(map[string]ManifestFile)}
	}

	result := &ChangeDetectionResult{
		Added:     []string{},
		Modified:  []string{},
		Deleted:   []string{},
		Renamed:   make(map[string]string),
		IsGitRepo: false,
	}

	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	discovery, err := DiscoverFiles(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to discover files: %w", err)
	}

	currentFiles := make(map[string]bool)
	for _, file := range discovery.Files {
		currentFiles[file.Path] = true
		existingInfo, exists := manifest.Files[file.Path]
		if !exists {
			result.Added = append(result.Added, file.Path)
			continue
		}

		fullPath := filepath.Join(absPath, file.Path)
		stat, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		currentMtime := stat.ModTime().UTC().Format(time.RFC3339)
		if currentMtime != existingInfo.LastModTime {
			result.Modified = append(result.Modified, file.Path)
		}
	}

	for filePath := range manifest.Files {
		if !currentFiles[filePath] {
			result.Deleted = append(result.Deleted, filePath)
		}
	}

	return result, nil
}

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
