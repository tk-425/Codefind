package lsp

import "slices"

type ServerInfo struct {
	Name       string
	Executable string
	Args       []string
}

var KnownLSPs = map[string]ServerInfo{
	"typescript/javascript": {Name: "TypeScript Language Server", Executable: "typescript-language-server", Args: []string{"--stdio"}},
	"python":                {Name: "Pyright", Executable: "pyright-langserver", Args: []string{"--stdio"}},
	"go":                    {Name: "gopls", Executable: "gopls"},
	"java":                  {Name: "Eclipse JDT LS", Executable: "jdtls"},
	"swift":                 {Name: "SourceKit-LSP", Executable: "sourcekit-lsp"},
	"rust":                  {Name: "rust-analyzer", Executable: "rust-analyzer"},
	"ocaml":                 {Name: "OCaml LSP", Executable: "ocamllsp"},
}

func LSPKeyForLanguage(language string) string {
	switch language {
	case "typescript", "javascript":
		return "typescript/javascript"
	default:
		return language
	}
}

func UniqueLSPKeys(languages []string) []string {
	keys := make([]string, 0, len(languages))
	for _, language := range languages {
		key := LSPKeyForLanguage(language)
		if key == "" || slices.Contains(keys, key) {
			continue
		}
		keys = append(keys, key)
	}
	return keys
}
