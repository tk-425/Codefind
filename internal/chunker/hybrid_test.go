package chunker

import (
	"errors"
	"testing"
)

func TestHybridChunkerFallsBackForNoSymbols(t *testing.T) {
	original := chunkFileWithLSP
	chunkFileWithLSP = func(content, filePath, language, rootPath string, config ChunkConfig) ([]SymbolChunk, error) {
		return nil, nil
	}
	defer func() { chunkFileWithLSP = original }()

	chunker := NewHybridChunker(DefaultConfig(), t.TempDir(), false)
	result, err := chunker.ChunkFile("package main\nfunc main() {}\n", "main.go")
	if err != nil {
		t.Fatalf("ChunkFile() error = %v", err)
	}
	if result.Method != "window" || result.FallbackReason != "no_symbols" {
		t.Fatalf("unexpected fallback result: %+v", result)
	}
}

func TestHybridChunkerRetriesLSPFailures(t *testing.T) {
	attempts := 0
	original := chunkFileWithLSP
	chunkFileWithLSP = func(content, filePath, language, rootPath string, config ChunkConfig) ([]SymbolChunk, error) {
		attempts++
		return nil, NewLSPError(LSPErrorTimeout, "timeout", errors.New("boom"))
	}
	defer func() { chunkFileWithLSP = original }()

	chunker := NewHybridChunker(DefaultConfig(), t.TempDir(), false)
	result, err := chunker.ChunkFile("package main\nfunc main() {}\n", "main.go")
	if err != nil {
		t.Fatalf("ChunkFile() error = %v", err)
	}
	if attempts != MaxLSPRetries {
		t.Fatalf("attempts = %d, want %d", attempts, MaxLSPRetries)
	}
	if result.FallbackReason != LSPErrorTimeout {
		t.Fatalf("unexpected fallback reason: %s", result.FallbackReason)
	}
	if chunker.Stats.WindowChunked != 1 || chunker.Stats.LSPRetries != MaxLSPRetries-1 {
		t.Fatalf("unexpected stats: %+v", chunker.Stats)
	}
}

func TestHybridChunkerUsesWindowOverride(t *testing.T) {
	t.Parallel()

	chunker := NewHybridChunker(DefaultConfig(), t.TempDir(), true)
	result, err := chunker.ChunkFile("package main\nfunc main() {}\n", "main.go")
	if err != nil {
		t.Fatalf("ChunkFile() error = %v", err)
	}
	if result.Method != "window" {
		t.Fatalf("expected window method, got %s", result.Method)
	}
}
