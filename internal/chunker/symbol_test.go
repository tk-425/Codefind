package chunker

import (
	"testing"

	"github.com/tk-425/Codefind/internal/lsp"
)

func TestSymbolsToChunksPreservesMetadata(t *testing.T) {
	t.Parallel()

	chunker := &SymbolChunker{
		config:    DefaultConfig(),
		fileLines: []string{"class A {", "func child() {}", "}"},
	}

	symbols := []lsp.DocumentSymbol{
		{
			Name: "A",
			Kind: lsp.SymbolKindClass,
			Range: lsp.Range{
				Start: lsp.Position{Line: 0},
				End:   lsp.Position{Line: 2},
			},
			Children: []lsp.DocumentSymbol{
				{
					Name: "child",
					Kind: lsp.SymbolKindMethod,
					Range: lsp.Range{
						Start: lsp.Position{Line: 1},
						End:   lsp.Position{Line: 1},
					},
				},
			},
		},
	}

	chunks := chunker.symbolsToChunks(symbols, "")
	if len(chunks) < 2 {
		t.Fatalf("expected parent and child chunks, got %d", len(chunks))
	}
	if chunks[0].SymbolName != "A" || chunks[0].SymbolKind != "class" {
		t.Fatalf("unexpected parent chunk metadata: %+v", chunks[0])
	}
	if chunks[1].ParentName != "A" || chunks[1].SymbolName != "child" {
		t.Fatalf("unexpected child chunk metadata: %+v", chunks[1])
	}
}

func TestSymbolsToChunksPreservesSignificantChildrenWhenParentIsSplit(t *testing.T) {
	t.Parallel()

	contentLines := []string{
		"class IndexingService:",
		"    def __init__(self):",
		"        self.value = 1",
		"",
		"    async def purge_tombstoned_chunks(self):",
		"        return self.value",
	}

	chunker := &SymbolChunker{
		config: ChunkConfig{
			TargetTokens:  1,
			OverlapTokens: 0,
			CharsPerToken: 4.0,
		},
		fileLines: contentLines,
	}

	symbols := []lsp.DocumentSymbol{
		{
			Name: "IndexingService",
			Kind: lsp.SymbolKindClass,
			Range: lsp.Range{
				Start: lsp.Position{Line: 0},
				End:   lsp.Position{Line: 5},
			},
			Children: []lsp.DocumentSymbol{
				{
					Name: "purge_tombstoned_chunks",
					Kind: lsp.SymbolKindMethod,
					Range: lsp.Range{
						Start: lsp.Position{Line: 4},
						End:   lsp.Position{Line: 5},
					},
				},
			},
		},
	}

	chunks := chunker.symbolsToChunks(symbols, "")

	foundParentPart := false
	foundChildMethod := false
	for _, chunk := range chunks {
		if chunk.SymbolName == "IndexingService (part 1)" && chunk.SymbolKind == "class" {
			foundParentPart = true
		}
		if chunk.SymbolName == "purge_tombstoned_chunks (part 1)" && chunk.SymbolKind == "method" && chunk.ParentName == "IndexingService" {
			foundChildMethod = true
		}
	}

	if !foundParentPart {
		t.Fatalf("expected split parent chunk, got %+v", chunks)
	}
	if !foundChildMethod {
		t.Fatalf("expected child method chunk to be preserved, got %+v", chunks)
	}
}
