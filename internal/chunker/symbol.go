package chunker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tk-425/Codefind/internal/lsp"
)

// LSP Error types for classifying failures
const (
	LSPErrorTimeout      = "timeout"
	LSPErrorCrash        = "crash"
	LSPErrorInitFailed   = "init_failed"
	LSPErrorSymbolFailed = "symbol_failed"
)

// LSPError wraps LSP-related errors with type classification
type LSPError struct {
	Type    string // timeout, crash, init_failed, symbol_failed
	Message string
	Err     error
}

func (e *LSPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *LSPError) Unwrap() error {
	return e.Err
}

// NewLSPError creates a new LSP error with the given type and message
func NewLSPError(errType, message string, err error) *LSPError {
	return &LSPError{
		Type:    errType,
		Message: message,
		Err:     err,
	}
}

// SymbolChunker chunks code based on LSP document symbols
type SymbolChunker struct {
	config      ChunkConfig
	lspClient   *lsp.LSPClient
	language    string
	fileContent string
	fileLines   []string
}

// SymbolChunk represents a chunk derived from a code symbol
type SymbolChunk struct {
	Chunk
	SymbolName string // Name of the symbol (e.g., "handleQuery")
	SymbolKind string // Kind of symbol (e.g., "function", "class")
	ParentName string // Parent symbol name (e.g., class name for methods)
}

// NewSymbolChunker creates a new symbol-based chunker
func NewSymbolChunker(config ChunkConfig, language, rootPath string) (*SymbolChunker, error) {
	client, err := lsp.NewLSPClient(language, rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create LSP client: %w", err)
	}

	return &SymbolChunker{
		config:    config,
		lspClient: client,
		language:  language,
	}, nil
}

// ChunkFileWithSymbols extracts symbols and creates chunks from them
func (sc *SymbolChunker) ChunkFileWithSymbols(content, filePath string) ([]SymbolChunk, error) {
	sc.fileContent = content
	sc.fileLines = strings.Split(content, "\n")

	// Initialize LSP with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := sc.lspClient.Initialize(ctx); err != nil {
		// Check if context was cancelled (timeout)
		if ctx.Err() == context.DeadlineExceeded {
			return nil, NewLSPError(LSPErrorTimeout, "LSP initialization timed out (30s)", err)
		}
		// Check if process crashed
		if !sc.lspClient.IsAlive() {
			return nil, NewLSPError(LSPErrorCrash, "LSP process crashed during initialization", err)
		}
		return nil, NewLSPError(LSPErrorInitFailed, "LSP initialize failed", err)
	}

	// Get symbols
	symbols, err := sc.lspClient.DocumentSymbols(ctx, filePath)
	if err != nil {
		sc.lspClient.Shutdown(ctx)
		// Check if context was cancelled (timeout)
		if ctx.Err() == context.DeadlineExceeded {
			return nil, NewLSPError(LSPErrorTimeout, "document symbols request timed out (30s)", err)
		}
		// Check if process crashed
		if !sc.lspClient.IsAlive() {
			return nil, NewLSPError(LSPErrorCrash, "LSP process crashed during symbol extraction", err)
		}
		return nil, NewLSPError(LSPErrorSymbolFailed, "document symbols failed", err)
	}

	// Shutdown LSP
	sc.lspClient.Shutdown(ctx)

	// Convert symbols to chunks
	chunks := sc.symbolsToChunks(symbols, "")

	return chunks, nil
}

// symbolsToChunks converts LSP symbols to chunks recursively
func (sc *SymbolChunker) symbolsToChunks(symbols []lsp.DocumentSymbol, parentName string) []SymbolChunk {
	var chunks []SymbolChunk

	for _, sym := range symbols {
		// Get the line range (LSP uses 0-indexed, we use 1-indexed)
		startLine := sym.Range.Start.Line + 1
		endLine := sym.Range.End.Line + 1

		// Extract content for this symbol
		content := sc.extractLines(startLine, endLine)

		// Estimate token count
		estimatedTokens := int(float32(len(content)) / sc.config.CharsPerToken)

		// If symbol is too large, split it
		if estimatedTokens > sc.config.TargetTokens*2 {
			// Split large symbol into smaller chunks
			subChunks := sc.splitLargeSymbol(sym, parentName)
			chunks = append(chunks, subChunks...)
		} else {
			// Create single chunk for this symbol
			chunk := SymbolChunk{
				Chunk: Chunk{
					Content:    content,
					StartLine:  startLine,
					EndLine:    endLine,
					Hash:       hashContent(content),
					TokenCount: estimatedTokens,
				},
				SymbolName: sym.Name,
				SymbolKind: sym.Kind.String(),
				ParentName: parentName,
			}
			chunks = append(chunks, chunk)
		}

		// Process children (e.g., methods in a class)
		// But only if the parent wasn't already split
		if len(sym.Children) > 0 && estimatedTokens <= sc.config.TargetTokens*2 {
			childChunks := sc.symbolsToChunks(sym.Children, sym.Name)
			// Don't add child chunks separately if they're already part of parent
			// Only add if children are significant (like methods)
			for _, child := range childChunks {
				if isSignificantSymbol(child.SymbolKind) {
					chunks = append(chunks, child)
				}
			}
		}
	}

	return chunks
}

// splitLargeSymbol splits a large symbol into multiple chunks
func (sc *SymbolChunker) splitLargeSymbol(sym lsp.DocumentSymbol, parentName string) []SymbolChunk {
	var chunks []SymbolChunk

	startLine := sym.Range.Start.Line + 1
	endLine := sym.Range.End.Line + 1

	// Calculate lines per chunk
	targetChars := int(float32(sc.config.TargetTokens) * sc.config.CharsPerToken)
	overlapLines := 3 // Overlap a few lines for context

	currentLine := startLine
	partNum := 1

	for currentLine <= endLine {
		// Find end of this chunk
		chunkContent := ""
		chunkEndLine := currentLine

		for chunkEndLine <= endLine {
			lineContent := sc.getLine(chunkEndLine)
			if len(chunkContent)+len(lineContent) > targetChars {
				break
			}
			chunkContent += lineContent + "\n"
			chunkEndLine++
		}

		// Ensure we made progress
		if chunkEndLine == currentLine {
			chunkEndLine = currentLine + 1
			chunkContent = sc.getLine(currentLine) + "\n"
		}

		// Create chunk
		chunk := SymbolChunk{
			Chunk: Chunk{
				Content:    strings.TrimSuffix(chunkContent, "\n"),
				StartLine:  currentLine,
				EndLine:    chunkEndLine - 1,
				Hash:       hashContent(chunkContent),
				TokenCount: int(float32(len(chunkContent)) / sc.config.CharsPerToken),
			},
			SymbolName: fmt.Sprintf("%s (part %d)", sym.Name, partNum),
			SymbolKind: sym.Kind.String(),
			ParentName: parentName,
		}
		chunks = append(chunks, chunk)

		// Move to next chunk with overlap
		currentLine = chunkEndLine - overlapLines
		if currentLine <= chunks[len(chunks)-1].StartLine {
			currentLine = chunkEndLine
		}
		partNum++
	}

	return chunks
}

// extractLines extracts lines from fileLines (1-indexed)
func (sc *SymbolChunker) extractLines(startLine, endLine int) string {
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(sc.fileLines) {
		endLine = len(sc.fileLines)
	}

	lines := sc.fileLines[startLine-1 : endLine]
	return strings.Join(lines, "\n")
}

// getLine returns a single line (1-indexed)
func (sc *SymbolChunker) getLine(lineNum int) string {
	if lineNum < 1 || lineNum > len(sc.fileLines) {
		return ""
	}
	return sc.fileLines[lineNum-1]
}

// isSignificantSymbol returns true if the symbol kind is worth chunking separately
func isSignificantSymbol(kind string) bool {
	significant := map[string]bool{
		"function":    true,
		"method":      true,
		"class":       true,
		"struct":      true,
		"interface":   true,
		"enum":        true,
		"constructor": true,
		"module":      true,
		"namespace":   true,
	}
	return significant[kind]
}

// Close shuts down the LSP client
func (sc *SymbolChunker) Close() {
	if sc.lspClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		sc.lspClient.Shutdown(ctx)
	}
}

// ChunkFileWithLSP is a convenience function that creates a chunker, chunks a file, and cleans up
func ChunkFileWithLSP(content, filePath, language, rootPath string, config ChunkConfig) ([]SymbolChunk, error) {
	chunker, err := NewSymbolChunker(config, language, rootPath)
	if err != nil {
		return nil, err
	}
	defer chunker.Close()

	return chunker.ChunkFileWithSymbols(content, filePath)
}
