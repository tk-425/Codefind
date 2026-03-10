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
