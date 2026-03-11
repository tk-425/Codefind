package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var validSHA = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
var repoIDSanitizer = regexp.MustCompile(`[^a-z0-9_-]+`)

func validateRepoPath(repoPath string) error {
	if !filepath.IsAbs(repoPath) {
		return fmt.Errorf("repoPath must be absolute")
	}
	info, err := os.Stat(repoPath)
	if err != nil {
		return fmt.Errorf("repoPath does not exist")
	}
	if !info.IsDir() {
		return fmt.Errorf("repoPath is not a directory")
	}
	return nil
}

func ValidateProjectRoot(repoPath string) error {
	if err := validateRepoPath(repoPath); err != nil {
		return err
	}
	if IsGitRepository(repoPath) {
		return nil
	}

	discovery, err := DiscoverFiles(repoPath)
	if err != nil {
		return err
	}
	if len(discovery.Files) == 0 {
		return fmt.Errorf("repoPath must be a git repo or contain supported source files")
	}
	return nil
}

func DeriveRepoID(repoPath string) (string, error) {
	if err := ValidateProjectRoot(repoPath); err != nil {
		return "", err
	}

	base := strings.ToLower(strings.TrimSpace(filepath.Base(filepath.Clean(repoPath))))
	base = repoIDSanitizer.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-_")
	base = strings.ReplaceAll(base, "--", "-")
	if base == "" {
		return "", fmt.Errorf("could not derive repo_id from path %q", repoPath)
	}
	if err := validateManifestSegment("repo_id", base); err != nil {
		return "", err
	}
	return base, nil
}

func validateCommitHash(hash string) error {
	if !validSHA.MatchString(hash) {
		return fmt.Errorf("invalid commit hash format")
	}
	return nil
}
