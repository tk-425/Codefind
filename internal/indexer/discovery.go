package indexer

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tk-425/Codefind/internal/pathutil"
)

type DiscoveryResult struct {
	Files     []DiscoveredFile
	TotalSize int64
}

type DiscoveredFile struct {
	Path     string
	Language string
	Size     int64
	Lines    int
}

var languagePatterns = map[string][]string{
	"go": {
		"bin/", "vendor/", "*.o", "*.a", "*.so", "*.exe", "*.dll", "*.dylib",
		"*.test", "dist/", "build/", "coverage/", "*.out", ".env", ".env.local",
	},
	"python": {
		"__pycache__/", "*.pyc", "*.pyo", ".pyc", ".pyo",
		"venv/", ".venv/", "env/", ".env/", "ENV/",
		"*.egg-info/", "dist/", "build/", ".pytest_cache/", ".tox/",
		"htmlcov/", ".coverage", "*.egg",
	},
	"javascript": {
		"node_modules/", "dist/", "build/", ".next/", ".nuxt/",
		"coverage/", "npm-debug.log", "yarn-error.log", ".env.local",
		"*.min.js", "*.min.css", ".parcel-cache/", ".cache/",
	},
	"typescript": {
		"node_modules/", "dist/", "build/", ".next/", ".nuxt/",
		"coverage/", "npm-debug.log", "yarn-error.log", ".env.local",
		"*.min.js", "*.min.css", ".parcel-cache/", ".cache/",
	},
	"java": {
		"target/", ".gradle/", ".settings/", "bin/", "out/",
		"*.class", "*.jar", ".classpath", ".project", ".vscode/",
		"*.swp", "build/", ".idea/",
	},
	"swift": {
		".build/", "Pods/", ".xcodeproj/", ".xcworkspace/",
		"Carthage/", "DerivedData/", "*.xcarchive", ".swiftpm/",
		"build/", "dist/",
	},
	"rust": {
		"target/", "Cargo.lock", "dist/", "build/",
	},
	"ocaml": {
		"_build/", "_opam/", "*.cmi", "*.cmo", "*.cmx", "*.cmxa", "*.a",
		"dist/", "build/",
	},
}

var defaultSkipDirs = map[string]bool{
	".":             true,
	"..":            true,
	".git":          true,
	".github":       true,
	".idea":         true,
	".vscode":       true,
	".venv":         true,
	".pytest_cache": true,
	".tox":          true,
	".next":         true,
	".nuxt":         true,
	"vendor":        true,
	"node_modules":  true,
	"dist":          true,
	"build":         true,
	"target":        true,
	"bin":           true,
	"pkg":           true,
	"Pods":          true,
	".build":        true,
	"__pycache__":   true,
}

func LanguageFromExtension(filePath string) string {
	switch filepath.Ext(filePath) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".java":
		return "java"
	case ".swift":
		return "swift"
	case ".rs":
		return "rust"
	case ".ml", ".mli":
		return "ocaml"
	default:
		return "unknown"
	}
}

func IsCodeFile(filePath string) bool {
	return LanguageFromExtension(filePath) != "unknown"
}

func DiscoverFiles(repoPath string) (*DiscoveryResult, error) {
	if err := validateRepoPath(repoPath); err != nil {
		return nil, fmt.Errorf("invalid repoPath: %w", err)
	}

	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	isGitRepo := IsGitRepository(absPath)
	supportedLanguages := []string{"go", "python", "typescript", "javascript", "java", "swift", "rust", "ocaml"}
	result := &DiscoveryResult{Files: []DiscoveredFile{}}

	err = filepath.WalkDir(absPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !pathutil.IsWithinDir(absPath, path) {
			return filepath.SkipDir
		}

		if entry.IsDir() {
			dirName := entry.Name()
			if isGitRepo && isIgnoredByGit(absPath, path) {
				return filepath.SkipDir
			}
			if defaultSkipDirs[dirName] || (strings.HasPrefix(dirName, ".") && dirName != ".") {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(absPath, path)
		if err != nil {
			return nil
		}
		if !IsCodeFile(relPath) {
			return nil
		}
		if isGitRepo && isIgnoredByGit(absPath, path) {
			return nil
		}
		if !isGitRepo && isIgnoredByLanguagePatterns(relPath, supportedLanguages) {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		result.Files = append(result.Files, DiscoveredFile{
			Path:     relPath,
			Language: LanguageFromExtension(relPath),
			Size:     info.Size(),
			Lines:    strings.Count(string(content), "\n") + 1,
		})
		result.TotalSize += info.Size()
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to discover files: %w", err)
	}

	return result, nil
}

func IsGitRepository(repoPath string) bool {
	gitDir := filepath.Join(repoPath, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}

func isIgnoredByGit(repoPath, filePath string) bool {
	if err := validateRepoPath(repoPath); err != nil {
		return false
	}

	relPath, err := filepath.Rel(repoPath, filePath)
	if err != nil {
		return false
	}

	cmd := exec.Command("git", "-C", repoPath, "check-ignore", relPath)
	return cmd.Run() == nil
}

func isIgnoredByLanguagePatterns(filePath string, foundLanguages []string) bool {
	var allPatterns []string
	for _, lang := range foundLanguages {
		if patterns, ok := languagePatterns[lang]; ok {
			allPatterns = append(allPatterns, patterns...)
		}
	}
	return matchesPattern(filePath, allPatterns)
}

func matchesPattern(filePath string, patterns []string) bool {
	for _, pattern := range patterns {
		if dirPattern, ok := strings.CutSuffix(pattern, "/"); ok {
			if strings.Contains(filePath, "/"+dirPattern+"/") || strings.HasPrefix(filePath, dirPattern+"/") {
				return true
			}
			continue
		}
		if strings.HasPrefix(pattern, "*.") {
			if strings.HasSuffix(filePath, pattern[1:]) {
				return true
			}
			continue
		}
		if strings.HasSuffix(filePath, pattern) {
			return true
		}
	}
	return false
}
