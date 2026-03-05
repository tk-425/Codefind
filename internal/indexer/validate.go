package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// validSHA matches git commit hashes: 7–64 lowercase hex characters.
// Rejects any input containing spaces, semicolons, pipes, or shell metacharacters.
var validSHA = regexp.MustCompile(`^[0-9a-f]{7,64}$`)

// validateRepoPath checks that repoPath is an absolute path to an existing directory.
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

// validateCommitHash checks that hash matches a valid git SHA format.
func validateCommitHash(hash string) error {
	if !validSHA.MatchString(hash) {
		return fmt.Errorf("invalid commit hash format")
	}
	return nil
}
