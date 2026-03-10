package chunker

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tk-425/Codefind/internal/lsp"
)

const MaxLSPRetries = 3

type ChunkStats struct {
	LSPChunked    int
	WindowChunked int
	LSPTimeouts   int
	LSPCrashes    int
	LSPRetries    int
}

type HybridChunker struct {
	config        ChunkConfig
	windowChunker *WindowChunker
	forceWindow   bool
	rootPath      string
	Stats         ChunkStats
}

type ChunkResult struct {
	Chunks         []SymbolChunk
	Method         string
	Language       string
	LSPAvailable   bool
	FallbackReason string
	ErrorMessage   string
}

var chunkFileWithLSP = ChunkFileWithLSP

func NewHybridChunker(config ChunkConfig, rootPath string, forceWindow bool) *HybridChunker {
	return &HybridChunker{
		config:        config,
		windowChunker: NewWindowChunker(config),
		forceWindow:   forceWindow,
		rootPath:      rootPath,
	}
}

func (hc *HybridChunker) ChunkFile(content, filePath string) (*ChunkResult, error) {
	result := &ChunkResult{
		Chunks:   []SymbolChunk{},
		Method:   "window",
		Language: detectLanguage(filePath),
	}

	if hc.forceWindow {
		chunks, err := hc.chunkWithWindow(content, filePath)
		if err != nil {
			return nil, err
		}
		result.Chunks = chunks
		hc.Stats.WindowChunked++
		return result, nil
	}

	lspLang := getLSPLanguage(filePath)
	if lspLang == "" {
		chunks, err := hc.chunkWithWindow(content, filePath)
		if err != nil {
			return nil, err
		}
		result.Chunks = chunks
		result.FallbackReason = "unsupported"
		hc.Stats.WindowChunked++
		return result, nil
	}

	if _, ok := lsp.KnownLSPs[lspLang]; !ok {
		chunks, err := hc.chunkWithWindow(content, filePath)
		if err != nil {
			return nil, err
		}
		result.Chunks = chunks
		result.FallbackReason = "unsupported"
		hc.Stats.WindowChunked++
		return result, nil
	}

	result.LSPAvailable = true
	var lastErr error
	for attempt := 1; attempt <= MaxLSPRetries; attempt++ {
		chunks, err := chunkFileWithLSP(content, filePath, lspLang, hc.rootPath, hc.config)
		if err == nil {
			if len(chunks) == 0 {
				windowChunks, windowErr := hc.chunkWithWindow(content, filePath)
				if windowErr != nil {
					return nil, windowErr
				}
				result.Chunks = windowChunks
				result.FallbackReason = "no_symbols"
				result.ErrorMessage = "LSP returned no useful symbols; using window chunking"
				hc.Stats.WindowChunked++
				return result, nil
			}
			result.Chunks = chunks
			result.Method = "symbol"
			hc.Stats.LSPChunked++
			return result, nil
		}

		lastErr = err
		if attempt < MaxLSPRetries {
			hc.Stats.LSPRetries++
		}
		if lspErr, ok := errors.AsType[*LSPError](err); ok {
			switch lspErr.Type {
			case LSPErrorTimeout:
				hc.Stats.LSPTimeouts++
			case LSPErrorCrash:
				hc.Stats.LSPCrashes++
			}
		}
	}

	windowChunks, err := hc.chunkWithWindow(content, filePath)
	if err != nil {
		return nil, err
	}
	result.Chunks = windowChunks
	hc.Stats.WindowChunked++

	if lspErr, ok := errors.AsType[*LSPError](lastErr); ok {
		result.FallbackReason = lspErr.Type
		result.ErrorMessage = fmt.Sprintf("LSP failed after %d attempts; using window chunking", MaxLSPRetries)
	} else if lastErr != nil {
		result.FallbackReason = "error"
		result.ErrorMessage = fmt.Sprintf("LSP failed: %v; using window chunking", lastErr)
	}

	return result, nil
}

func (hc *HybridChunker) chunkWithWindow(content, filePath string) ([]SymbolChunk, error) {
	chunks, err := hc.windowChunker.ChunkFile(content, filePath)
	if err != nil {
		return nil, err
	}

	symbolChunks := make([]SymbolChunk, len(chunks))
	for i, c := range chunks {
		symbolChunks[i] = SymbolChunk{
			Chunk:      c,
			SymbolKind: "window",
		}
	}
	return symbolChunks, nil
}

func getLSPLanguage(filePath string) string {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".ts", ".tsx", ".js", ".jsx":
		return "typescript/javascript"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".swift":
		return "swift"
	case ".ml", ".mli":
		return "ocaml"
	default:
		return ""
	}
}

func detectLanguage(filePath string) string {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".swift":
		return "swift"
	case ".ml", ".mli":
		return "ocaml"
	default:
		return "unknown"
	}
}
