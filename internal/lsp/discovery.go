package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// LSPInfo contains information about an LSP server
type LSPInfo struct {
	Language   string `json:"language"`
	Name       string `json:"name"`
	Executable string `json:"executable"`
	Path       string `json:"path"`       // Full path to executable
	Version    string `json:"version"`
	Available  bool   `json:"available"`
}

// LSPCache stores discovered LSP servers with TTL
type LSPCache struct {
	Servers  map[string]LSPInfo `json:"servers"`
	CachedAt time.Time          `json:"cached_at"`
}

// CacheTTL is the time-to-live for the LSP cache (7 days)
const CacheTTL = 7 * 24 * time.Hour

// VersionTimeout is the timeout for getting LSP version
const VersionTimeout = 3 * time.Second

// KnownLSPs maps language to LSP executable names and version flags
// Empty VersionFlag means skip version check
// Args are additional arguments needed to start the LSP in stdio mode
var KnownLSPs = map[string]struct {
	Name        string
	Executable  string
	VersionFlag string
	Args        []string
}{
	"typescript/javascript": {"TypeScript Language Server", "typescript-language-server", "--version", []string{"--stdio"}},
	"python":                {"Pyright", "pyright-langserver", "", []string{"--stdio"}},  // pyright-langserver needs --stdio
	"go":                    {"gopls", "gopls", "version", nil},
	"java":                  {"Eclipse JDT LS", "jdtls", "", nil},           // jdtls doesn't have clean version flag
	"swift":                 {"SourceKit-LSP", "sourcekit-lsp", "", nil},    // sourcekit-lsp doesn't support --version
	"rust":                  {"rust-analyzer", "rust-analyzer", "--version", nil},
	"ocaml":                 {"OCaml LSP", "ocamllsp", "--version", nil},
}

// DiscoverLSPs searches PATH for known LSP executables
func DiscoverLSPs() []LSPInfo {
	var results []LSPInfo

	for lang, info := range KnownLSPs {
		lspInfo := LSPInfo{
			Language:   lang,
			Name:       info.Name,
			Executable: info.Executable,
			Available:  false,
		}

		// Search for executable in PATH
		path, err := exec.LookPath(info.Executable)
		if err != nil {
			results = append(results, lspInfo)
			continue
		}

		lspInfo.Path = path
		lspInfo.Available = true

		// Get version with timeout (skip if no version flag)
		if info.VersionFlag != "" {
			version, err := getLSPVersion(path, info.VersionFlag)
			if err == nil {
				lspInfo.Version = version
			} else {
				lspInfo.Version = "unknown"
			}
		} else {
			lspInfo.Version = "installed" // No version check available
		}

		results = append(results, lspInfo)
	}

	return results
}

// getLSPVersion runs the LSP with version flag and extracts version string
func getLSPVersion(execPath, versionFlag string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), VersionTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, execPath, versionFlag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("timeout")
		}
		return "", err
	}

	// Extract version from output (various formats)
	version := extractVersion(string(output))
	return version, nil
}

// extractVersion tries to extract a version number from command output
func extractVersion(output string) string {
	// Common patterns: v1.2.3, 1.2.3, version 1.2.3
	patterns := []string{
		`v?(\d+\.\d+\.\d+)`,       // v1.2.3 or 1.2.3
		`version\s+(\d+\.\d+\.\d+)`, // version 1.2.3
		`(\d+\.\d+\.\d+[-\w]*)`,   // 1.2.3-beta
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(output)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	// Return first line if no version pattern found
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return "unknown"
}

// GetCachePath returns the path to the LSP cache file
func GetCachePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".codefind", "lsp-cache.json"), nil
}

// LoadCache loads the LSP cache from disk
func LoadCache() (*LSPCache, error) {
	cachePath, err := GetCachePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cache LSPCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

// SaveCache saves the LSP cache to disk
func SaveCache(cache *LSPCache) error {
	cachePath, err := GetCachePath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(cachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

// IsCacheValid checks if the cache is still within TTL
func IsCacheValid(cache *LSPCache) bool {
	return time.Since(cache.CachedAt) < CacheTTL
}

// GetOrDiscoverLSPs returns cached LSPs if valid, otherwise discovers and caches
func GetOrDiscoverLSPs(forceRefresh bool) ([]LSPInfo, error) {
	// Try to load from cache
	if !forceRefresh {
		cache, err := LoadCache()
		if err == nil && IsCacheValid(cache) {
			// Convert map to slice
			var results []LSPInfo
			for _, info := range cache.Servers {
				results = append(results, info)
			}
			return results, nil
		}
	}

	// Discover LSPs
	lsps := DiscoverLSPs()

	// Build cache
	cache := &LSPCache{
		Servers:  make(map[string]LSPInfo),
		CachedAt: time.Now(),
	}
	for _, lsp := range lsps {
		cache.Servers[lsp.Language] = lsp
	}

	// Save cache
	if err := SaveCache(cache); err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to save LSP cache: %v\n", err)
	}

	return lsps, nil
}

// FormatLSPStatus returns a formatted string showing LSP availability
func FormatLSPStatus(lsps []LSPInfo) string {
	var sb strings.Builder

	sb.WriteString("LSP Server Status:\n")
	sb.WriteString("──────────────────────────────────────────\n")

	for _, lsp := range lsps {
		if lsp.Available {
			sb.WriteString(fmt.Sprintf("✅ %s (%s)\n", lsp.Name, lsp.Language))
			sb.WriteString(fmt.Sprintf("   Version: %s\n", lsp.Version))
			sb.WriteString(fmt.Sprintf("   Path: %s\n", lsp.Path))
		} else {
			sb.WriteString(fmt.Sprintf("❌ %s (%s) - not found\n", lsp.Name, lsp.Language))
		}
		sb.WriteString("\n") // Blank line between entries
	}

	// Summary
	available := 0
	for _, lsp := range lsps {
		if lsp.Available {
			available++
		}
	}
	sb.WriteString("──────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("Found: %d/%d LSP servers\n", available, len(lsps)))

	return sb.String()
}
