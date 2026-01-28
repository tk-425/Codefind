package chunker

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tk-425/Codefind/internal/lsp"
)

// HybridChunker intelligently chooses between symbol and window chunking
type HybridChunker struct {
	config         ChunkConfig
	windowChunker  *WindowChunker
	forceWindow    bool
	rootPath       string
	availableLSPs  map[string]bool // language -> available
}

// ChunkResult contains the result of chunking a file
type ChunkResult struct {
	Chunks       []SymbolChunk
	Method       string // "symbol" or "window"
	Language     string
	LSPAvailable bool
}

// NewHybridChunker creates a new hybrid chunker
func NewHybridChunker(config ChunkConfig, rootPath string, forceWindow bool) *HybridChunker {
	return &HybridChunker{
		config:        config,
		windowChunker: NewWindowChunker(config),
		forceWindow:   forceWindow,
		rootPath:      rootPath,
		availableLSPs: make(map[string]bool),
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

	if available {
		// Try symbol chunking
		chunks, err := ChunkFileWithLSP(content, filePath, lspLang, hc.rootPath, hc.config)
		if err == nil && len(chunks) > 0 {
			result.Chunks = chunks
			result.Method = "symbol"
			return result, nil
		}
		// If LSP failed, fall back to window
	}

	// Fall back to window chunking
	chunks, err := hc.chunkWithWindow(content, filePath)
	if err != nil {
		return nil, err
	}
	result.Chunks = chunks
	return result, nil
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
