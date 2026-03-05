package pathutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

// IsWithinDir reports whether path is contained within baseDir.
// Both paths are resolved to absolute form before comparison.
// A trailing separator is appended to baseDir to prevent prefix
// collision attacks (e.g. /foo/bar matching /foo/barbaz).
func IsWithinDir(path, baseDir string) (bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("pathutil: cannot resolve path %q: %w", path, err)
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return false, fmt.Errorf("pathutil: cannot resolve baseDir %q: %w", baseDir, err)
	}

	return strings.HasPrefix(absPath, absBase+string(filepath.Separator)), nil
}
