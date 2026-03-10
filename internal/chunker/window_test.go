package chunker

import (
	"strings"
	"testing"
)

func TestGenerateChunkIDIsStable(t *testing.T) {
	t.Parallel()

	id1 := GenerateChunkID("repo1", "main.go", 1, 10, "package main")
	id2 := GenerateChunkID("repo1", "main.go", 1, 10, "package main")
	if id1 != id2 {
		t.Fatalf("GenerateChunkID() should be stable: %s != %s", id1, id2)
	}
}

func TestWindowChunkerSplitsContent(t *testing.T) {
	t.Parallel()

	chunker := NewWindowChunker(ChunkConfig{TargetTokens: 4, OverlapTokens: 1, CharsPerToken: 1})
	content := strings.Repeat("a", 12)

	chunks, err := chunker.ChunkFile(content, "main.go")
	if err != nil {
		t.Fatalf("ChunkFile() error = %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
}
