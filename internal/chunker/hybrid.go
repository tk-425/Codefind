package chunker

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tk-425/Codefind/internal/lsp"
)

// LSP retry configuration
const (
	MaxLSPRetries       = 3  // Maximum retry attempts per file
	MaxConsecutiveFails = 3  // Disable LSP for language after this many consecutive failures
)

// ChunkStats tracks LSP chunking statistics for the session
type ChunkStats struct {
	LSPChunked    int // Files successfully chunked with LSP
	WindowChunked int // Files chunked with window fallback
	LSPTimeouts   int // LSP timeout occurrences
	LSPCrashes    int // LSP crash occurrences
	LSPRetries    int // Total retry attempts
}

// HybridChunker intelligently chooses between symbol and window chunking
type HybridChunker struct {
	config           ChunkConfig
	windowChunker    *WindowChunker
	forceWindow      bool
	rootPath         string
	availableLSPs    map[string]bool // language -> available
	failureCount     map[string]int  // language -> consecutive failures
	disabledLSPs     map[string]bool // language -> disabled after max failures
	Stats            ChunkStats
}

// ChunkResult contains the result of chunking a file
type ChunkResult struct {
	Chunks         []SymbolChunk
	Method         string // "symbol" or "window"
	Language       string
	LSPAvailable   bool
	FallbackReason string // Why fallback occurred: "timeout", "crash", "error", "disabled", ""
	ErrorMessage   string // Detailed error message for logging
}

// NewHybridChunker creates a new hybrid chunker
func NewHybridChunker(config ChunkConfig, rootPath string, forceWindow bool) *HybridChunker {
	return &HybridChunker{
		config:        config,
		windowChunker: NewWindowChunker(config),
		forceWindow:   forceWindow,
		rootPath:      rootPath,
		availableLSPs: make(map[string]bool),
		failureCount:  make(map[string]int),
		disabledLSPs:  make(map[string]bool),
	}
}

// ChunkFile chunks a file using the best available method
func (hc *HybridChunker) ChunkFile(content, filePath string) (*ChunkResult, error) {
	result := &ChunkResult{
		Chunks:   []SymbolChunk{},
		Method:   "window",
		Language: detectLanguage(filePath),
	}

	// If force window mode, skip LSP
	if hc.forceWindow {
		chunks, err := hc.chunkWithWindow(content, filePath)
		if err != nil {
			return nil, err
		}
		result.Chunks = chunks
		hc.Stats.WindowChunked++
		return result, nil
	}

	// Check if LSP is available for this language
	lspLang := getLSPLanguage(filePath)
	if lspLang == "" {
		// No LSP support for this file type
		chunks, err := hc.chunkWithWindow(content, filePath)
		if err != nil {
			return nil, err
		}
		result.Chunks = chunks
		hc.Stats.WindowChunked++
		return result, nil
	}

	// Check if we've already determined LSP availability
	available, cached := hc.availableLSPs[lspLang]
	if !cached {
		// Check if LSP executable exists
		available = hc.checkLSPAvailable(lspLang)
		hc.availableLSPs[lspLang] = available
	}

	result.LSPAvailable = available

	// Check if LSP has been disabled due to repeated failures
	if hc.disabledLSPs[lspLang] {
		chunks, err := hc.chunkWithWindow(content, filePath)
		if err != nil {
			return nil, err
		}
		result.Chunks = chunks
		result.FallbackReason = "disabled"
		result.ErrorMessage = fmt.Sprintf("❌  %s LSP disabled after %d failures, using window chunking", getLSPName(lspLang), MaxConsecutiveFails)
		hc.Stats.WindowChunked++
		return result, nil
	}

	if available {
		// Try symbol chunking with retry logic
		var lastErr error
		for attempt := 1; attempt <= MaxLSPRetries; attempt++ {
			chunks, err := ChunkFileWithLSP(content, filePath, lspLang, hc.rootPath, hc.config)
			if err == nil && len(chunks) > 0 {
				// Success! Reset failure count
				hc.failureCount[lspLang] = 0
				result.Chunks = chunks
				result.Method = "symbol"
				hc.Stats.LSPChunked++
				return result, nil
			}

			lastErr = err

			// Check error type for detailed messaging
			var lspErr *LSPError
			if errors.As(err, &lspErr) {
				switch lspErr.Type {
				case LSPErrorTimeout:
					hc.Stats.LSPTimeouts++
					if attempt < MaxLSPRetries {
						hc.Stats.LSPRetries++
						result.ErrorMessage = fmt.Sprintf("⚠️  LSP timeout for %s (30s), retrying (attempt %d/%d)", filePath, attempt, MaxLSPRetries)
					}
				case LSPErrorCrash:
					hc.Stats.LSPCrashes++
					if attempt < MaxLSPRetries {
						hc.Stats.LSPRetries++
						result.ErrorMessage = fmt.Sprintf("⚠️  %s crashed, restarting (attempt %d/%d)", getLSPName(lspLang), attempt, MaxLSPRetries)
					}
				}
			}

			// If not the last attempt, continue retrying
			if attempt < MaxLSPRetries {
				continue
			}
		}

		// All retries failed - track consecutive failures
		hc.failureCount[lspLang]++

		// Build detailed fallback message
		var lspErr *LSPError
		if errors.As(lastErr, &lspErr) {
			result.FallbackReason = lspErr.Type
			switch lspErr.Type {
			case LSPErrorTimeout:
				result.ErrorMessage = fmt.Sprintf("⚠️  LSP timeout for %s (30s), using window chunking", filePath)
			case LSPErrorCrash:
				result.ErrorMessage = fmt.Sprintf("⚠️  %s crashed, using window chunking", getLSPName(lspLang))
			default:
				result.ErrorMessage = fmt.Sprintf("⚠️  LSP failed for %s: %v, using window chunking", filePath, lastErr)
				result.FallbackReason = "error"
			}
		} else if lastErr != nil {
			result.FallbackReason = "error"
			result.ErrorMessage = fmt.Sprintf("⚠️  LSP failed for %s: %v, using window chunking", filePath, lastErr)
		}

		// Check if we should disable LSP for this language
		if hc.failureCount[lspLang] >= MaxConsecutiveFails {
			hc.disabledLSPs[lspLang] = true
			result.ErrorMessage = fmt.Sprintf("❌  %s failed after %d attempts, falling back to window for %s files", getLSPName(lspLang), MaxLSPRetries, result.Language)
		}
	}

	// Fall back to window chunking
	chunks, err := hc.chunkWithWindow(content, filePath)
	if err != nil {
		return nil, err
	}
	result.Chunks = chunks
	hc.Stats.WindowChunked++
	return result, nil
}

// getLSPName returns the human-readable LSP name for a language
func getLSPName(language string) string {
	info, ok := lsp.KnownLSPs[language]
	if !ok {
		return language + " LSP"
	}
	return info.Name
}

// chunkWithWindow uses window chunker and converts to SymbolChunk format
func (hc *HybridChunker) chunkWithWindow(content, filePath string) ([]SymbolChunk, error) {
	chunks, err := hc.windowChunker.ChunkFile(content, filePath)
	if err != nil {
		return nil, err
	}

	// Convert Chunk to SymbolChunk
	symbolChunks := make([]SymbolChunk, len(chunks))
	for i, c := range chunks {
		symbolChunks[i] = SymbolChunk{
			Chunk:      c,
			SymbolName: "",
			SymbolKind: "window",
			ParentName: "",
		}
	}
	return symbolChunks, nil
}

// checkLSPAvailable checks if the LSP for a language is installed
func (hc *HybridChunker) checkLSPAvailable(language string) bool {
	info, ok := lsp.KnownLSPs[language]
	if !ok {
		return false
	}

	_, err := exec.LookPath(info.Executable)
	return err == nil
}

// getLSPLanguage returns the LSP language identifier for a file
func getLSPLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	langMap := map[string]string{
		".go":    "go",
		".py":    "python",
		".ts":    "typescript/javascript",
		".tsx":   "typescript/javascript",
		".js":    "typescript/javascript",
		".jsx":   "typescript/javascript",
		".rs":    "rust",
		".java":  "java",
		".swift": "swift",
		".ml":    "ocaml",
		".mli":   "ocaml",
	}

	return langMap[ext]
}

// detectLanguage returns the language from file extension
func detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	langMap := map[string]string{
		".go":     "go",
		".py":     "python",
		".ts":     "typescript",
		".tsx":    "typescript",
		".js":     "javascript",
		".jsx":    "javascript",
		".rs":     "rust",
		".java":   "java",
		".swift":  "swift",
		".ml":     "ocaml",
		".mli":    "ocaml",
		".c":      "c",
		".cpp":    "cpp",
		".h":      "c",
		".hpp":    "cpp",
		".rb":     "ruby",
		".php":    "php",
		".cs":     "csharp",
		".md":     "markdown",
		".json":   "json",
		".yaml":   "yaml",
		".yml":    "yaml",
		".toml":   "toml",
		".xml":    "xml",
		".html":   "html",
		".css":    "css",
		".scss":   "scss",
		".sh":     "shell",
		".bash":   "shell",
		".zsh":    "shell",
		".sql":    "sql",
	}

	if lang, ok := langMap[ext]; ok {
		return lang
	}
	return "unknown"
}

// SupportedLSPExtensions returns list of file extensions that have LSP support
func SupportedLSPExtensions() []string {
	return []string{
		".go", ".py", ".ts", ".tsx", ".js", ".jsx",
		".rs", ".java", ".swift", ".ml", ".mli",
	}
}
