package chunker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tk-425/Codefind/internal/lsp"
)

const (
	LSPErrorTimeout      = "timeout"
	LSPErrorCrash        = "crash"
	LSPErrorInitFailed   = "init_failed"
	LSPErrorSymbolFailed = "symbol_failed"
	LSPRequestTimeout    = 30 * time.Second
)

type LSPError struct {
	Type    string
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

func NewLSPError(errType, message string, err error) *LSPError {
	return &LSPError{Type: errType, Message: message, Err: err}
}

type SymbolChunker struct {
	config      ChunkConfig
	lspClient   *lsp.Client
	fileLines   []string
	fileContent string
}

type SymbolChunk struct {
	Chunk
	SymbolName string
	SymbolKind string
	ParentName string
}

var newLSPClient = lsp.NewClient

func NewSymbolChunker(config ChunkConfig, language, rootPath string) (*SymbolChunker, error) {
	client, err := newLSPClient(language, rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create LSP client: %w", err)
	}

	return &SymbolChunker{
		config:    config,
		lspClient: client,
	}, nil
}

func (sc *SymbolChunker) ChunkFileWithSymbols(content, filePath string) ([]SymbolChunk, error) {
	sc.fileContent = content
	sc.fileLines = strings.Split(content, "\n")

	ctx, cancel := context.WithTimeout(context.Background(), LSPRequestTimeout)
	defer cancel()

	if err := sc.lspClient.Initialize(ctx); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, NewLSPError(LSPErrorTimeout, "LSP initialization timed out", err)
		}
		if !sc.lspClient.IsAlive() {
			return nil, NewLSPError(LSPErrorCrash, "LSP process crashed during initialization", err)
		}
		return nil, NewLSPError(LSPErrorInitFailed, "LSP initialize failed", err)
	}

	symbols, err := sc.lspClient.DocumentSymbols(ctx, filePath)
	if err != nil {
		_ = sc.lspClient.Shutdown(ctx)
		if ctx.Err() == context.DeadlineExceeded {
			return nil, NewLSPError(LSPErrorTimeout, "document symbols request timed out", err)
		}
		if !sc.lspClient.IsAlive() {
			return nil, NewLSPError(LSPErrorCrash, "LSP process crashed during symbol extraction", err)
		}
		return nil, NewLSPError(LSPErrorSymbolFailed, "document symbols failed", err)
	}

	_ = sc.lspClient.Shutdown(ctx)
	return sc.symbolsToChunks(symbols, ""), nil
}

func ChunkFileWithLSP(content, filePath, language, rootPath string, config ChunkConfig) ([]SymbolChunk, error) {
	chunker, err := NewSymbolChunker(config, language, rootPath)
	if err != nil {
		return nil, err
	}
	defer chunker.Close()
	return chunker.ChunkFileWithSymbols(content, filePath)
}

func (sc *SymbolChunker) Close() {
	if sc.lspClient == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = sc.lspClient.Shutdown(ctx)
}

func (sc *SymbolChunker) symbolsToChunks(symbols []lsp.DocumentSymbol, parentName string) []SymbolChunk {
	var chunks []SymbolChunk

	for _, sym := range symbols {
		startLine := sym.Range.Start.Line + 1
		endLine := sym.Range.End.Line + 1
		content := sc.extractLines(startLine, endLine)
		estimatedTokens := int(float32(len(content)) / sc.config.CharsPerToken)

		if estimatedTokens > sc.config.TargetTokens*2 {
			chunks = append(chunks, sc.splitLargeSymbol(sym, parentName)...)
		} else {
			chunks = append(chunks, SymbolChunk{
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
			})
		}

		if len(sym.Children) > 0 {
			chunks = append(chunks, sc.significantChildChunks(sym.Children, sym.Name)...)
		}
	}

	return chunks
}

func (sc *SymbolChunker) significantChildChunks(symbols []lsp.DocumentSymbol, parentName string) []SymbolChunk {
	childChunks := sc.symbolsToChunks(symbols, parentName)
	filtered := make([]SymbolChunk, 0, len(childChunks))
	for _, child := range childChunks {
		if isSignificantSymbol(child.SymbolKind) {
			filtered = append(filtered, child)
		}
	}
	return filtered
}

func (sc *SymbolChunker) splitLargeSymbol(sym lsp.DocumentSymbol, parentName string) []SymbolChunk {
	var chunks []SymbolChunk

	startLine := sym.Range.Start.Line + 1
	endLine := sym.Range.End.Line + 1
	targetChars := int(float32(sc.config.TargetTokens) * sc.config.CharsPerToken)
	overlapLines := 3
	currentLine := startLine
	partNum := 1

	for currentLine <= endLine {
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

		if chunkEndLine == currentLine {
			chunkEndLine = currentLine + 1
			chunkContent = sc.getLine(currentLine) + "\n"
		}

		chunks = append(chunks, SymbolChunk{
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
		})

		currentLine = chunkEndLine - overlapLines
		if currentLine <= chunks[len(chunks)-1].StartLine {
			currentLine = chunkEndLine
		}
		partNum++
	}

	return chunks
}

func (sc *SymbolChunker) extractLines(startLine, endLine int) string {
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(sc.fileLines) {
		endLine = len(sc.fileLines)
	}
	return strings.Join(sc.fileLines[startLine-1:endLine], "\n")
}

func (sc *SymbolChunker) getLine(lineNum int) string {
	if lineNum < 1 || lineNum > len(sc.fileLines) {
		return ""
	}
	return sc.fileLines[lineNum-1]
}

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
