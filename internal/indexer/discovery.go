package indexer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DiscoveryResult contains discovered files
type DiscoveryResult struct {
	Files     []DiscoveredFile
	TotalSize int64
}

// DiscoveredFile represents a file to be indexed
type DiscoveredFile struct {
	Path     string // Relative path from repo root
	Language string
	Size     int64
	Lines    int
}

// LanguageFromExtension detects programming language
// Currently supports: Go, Python, TypeScript/JavaScript, Java, Swift
// More languages will be added gradually in future phases
func LanguageFromExtension(filePath string) string {
	ext := filepath.Ext(filePath)

	switch ext {
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
	default:
		return "unknown"
	}
}

// IsCodeFile checks if a file should be indexed
func IsCodeFile(filePath string) bool {
	lang := LanguageFromExtension(filePath)
	return lang != "unknown"
}

// isIgnoredByGit checks if a file is ignored by .gitignore
// Only works if the path is within a git repository
func isIgnoredByGit(repoPath, filePath string) bool {
	// Get relative path from repo root for git check-ignore
	relPath, err := filepath.Rel(repoPath, filePath)
	if err != nil {
		return false
	}

	// Run git check-ignore command
	// Returns exit code 0 if ignored, non-zero if not ignored
	cmd := exec.Command("git", "-C", repoPath, "check-ignore", relPath)
	err = cmd.Run()

	// Exit code 0 means the file is ignored
	return err == nil
}

// languagePatterns contains ignore patterns for each supported language
// Used for non-git repositories to filter language-specific build artifacts
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
}

// matchesPattern checks if a file path matches any of the given patterns
// Supports simple patterns: directory names, file extensions, and wildcards
func matchesPattern(filePath string, patterns []string) bool {
	for _, pattern := range patterns {
		// Check exact directory name match
		if pattern[len(pattern)-1:] == "/" {
			dirPattern := pattern[:len(pattern)-1]
			// Check if directory appears in path
			if strings.Contains(filePath, "/"+dirPattern+"/") || strings.HasPrefix(filePath, dirPattern+"/") {
				return true
			}
		} else if strings.HasPrefix(pattern, "*.") {
			// Check file extension match
			ext := pattern[1:] // Remove the *
			if strings.HasSuffix(filePath, ext) {
				return true
			}
		} else {
			// Exact filename match
			if strings.HasSuffix(filePath, pattern) {
				return true
			}
		}
	}
	return false
}

// isIgnoredByLanguagePatterns checks if a file should be ignored based on
// language-specific patterns (for non-git repositories)
func isIgnoredByLanguagePatterns(filePath string, foundLanguages []string) bool {
	// Collect all patterns for found languages
	var allPatterns []string
	for _, lang := range foundLanguages {
		if patterns, ok := languagePatterns[lang]; ok {
			allPatterns = append(allPatterns, patterns...)
		}
	}

	return matchesPattern(filePath, allPatterns)
}

// DiscoverFiles recursively finds all indexable files in a repository
func DiscoverFiles(repoPath string) (*DiscoveryResult, error) {
	// Convert relative path to absolute path for consistent handling
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if this is a git repository
	isGitRepo := IsGitRepository(absPath)

	result := &DiscoveryResult{
		Files: []DiscoveredFile{},
	}

	// Skip directories map (fallback for non-git repos)
	skipDirs := map[string]bool{
		// Common to all
		".":                   true,
		"..":                  true,
		".git":                true,
		".github":             true,
		"vendor":              true,
		// Python
		"__pycache__":         true,
		".pytest_cache":       true,
		"venv":                true,
		"env":                 true,
		".venv":               true,
		// JavaScript/TypeScript
		"node_modules":        true,
		".next":               true,
		"dist":                true,
		"build":               true,
		".vscode":             true,
		// Java
		"target":              true,
		".gradle":             true,
		// Go
		"bin":                 true,
		"pkg":                 true,
		// Swift
		".build":              true,
		"Pods":                true,
		".xcodeproj":          true,
		".xcworkspace":        true,
	}

	// Collect all language patterns for non-git repos
	// Use patterns for all supported languages since we don't know what's in the repo yet
	var supportedLanguages = []string{"go", "python", "typescript", "javascript", "java", "swift"}

	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-code files
		if info.IsDir() {
			dirName := info.Name()

			// Priority 1: Check gitignore first (if in git repo)
			if isGitRepo && isIgnoredByGit(absPath, path) {
				return filepath.SkipDir
			}

			// Priority 2: Fallback to hardcoded skip list (non-git repos or safety net)
			if skipDirs[dirName] || strings.HasPrefix(dirName, ".") {
				return filepath.SkipDir
			}

			return nil
		}

		// Check if file is code
		if !IsCodeFile(path) {
			return nil
		}

		// Get relative path from absolute base (needed for pattern matching)
		relPath, err := filepath.Rel(absPath, path)
		if err != nil {
			return nil
		}

		// Priority 1: Check gitignore first (if in git repo)
		if isGitRepo && isIgnoredByGit(absPath, path) {
			return nil
		}

		// Priority 2: Check language patterns (for non-git repos)
		if !isGitRepo && isIgnoredByLanguagePatterns(relPath, supportedLanguages) {
			return nil
		}

		// Read file to count lines
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip unreadable files
		}

		lineCount := strings.Count(string(content), "\n") + 1

		discovered := DiscoveredFile{
			Path:     relPath,
			Language: LanguageFromExtension(path),
			Size:     info.Size(),
			Lines:    lineCount,
		}

		result.Files = append(result.Files, discovered)
		result.TotalSize += info.Size()

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to discover files: %w", err)
	}

	return result, nil
}

// IsGitRepository checks if directory is a git repository
func IsGitRepository(repoPath string) bool {
	gitDir := filepath.Join(repoPath, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}
