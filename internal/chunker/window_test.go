package chunker

import (
	"regexp"
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

	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidPattern.MatchString(id1) {
		t.Fatalf("GenerateChunkID() = %q, want deterministic UUID", id1)
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
