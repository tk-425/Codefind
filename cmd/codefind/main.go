package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/tk-425/Codefind/internal/chunker"
	"github.com/tk-425/Codefind/internal/cleanup"
	"github.com/tk-425/Codefind/internal/client"
	"github.com/tk-425/Codefind/internal/config"
	"github.com/tk-425/Codefind/internal/indexer"
	"github.com/tk-425/Codefind/internal/lsp"
	"github.com/tk-425/Codefind/internal/query"
	"github.com/tk-425/Codefind/internal/stats"
	"github.com/tk-425/Codefind/pkg/api"
)

func main() {
	// Define subcommands
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "init":
		handleInit()
	case "list-files":
		handleListFiles()
	case "chunk-file":
		handleChunkFile()
	case "index":
		handleIndex()
	case "query":
		handleQuery(os.Args[2:])
	case "list":
		handleList()
	case "open":
		if len(os.Args) < 3 {
			fmt.Println("Error: result ID required")
			fmt.Println("Usage: codefind open <id>")
			os.Exit(1)
		}
		handleOpen(os.Args[2])
	case "lsp":
		handleLSP(os.Args[2:])
	case "stats":
		handleStats(os.Args[2:])
	case "cleanup":
		handleCleanup(os.Args[2:])
	case "clear":
		repoPath := "."
		if len(os.Args) >= 3 {
			repoPath = os.Args[2]
		}
		handleClear(repoPath)
	case "help", "-h", "--help", "":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleInit() {
	// Parse flags
	serverURL := ""
	editor := ""

	// Create a flag set for the init command
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	initCmd.StringVar(&serverURL, "server-url", "", "Server URL (e.g., http://localhost:8080)")
	initCmd.StringVar(&editor, "editor", "", "Editor to use (vim, code, nano, etc.)")

	// Parse arguments after "init"
	initCmd.Parse(os.Args[2:])

	// Prompt for server URL if not provided
	if serverURL == "" {
		serverURL = promptFor("Server URL", "")
	}
	if serverURL == "" {
		fmt.Println("Error: server_url is required")
		os.Exit(1)
	}

	// Prompt for editor if not provided
	if editor == "" {
		// Try to get default editor from $EDITOR environment variable
		defaultEditor := os.Getenv("EDITOR")
		if defaultEditor == "" {
			defaultEditor = "nvim"
		}
		editor = promptFor("Editor", defaultEditor)
	}
	if editor == "" {
		fmt.Println("Error: editor is required")
		os.Exit(1)
	}

	// Create config object
	cfg := &config.GlobalConfig{
		ServerURL: serverURL,
		Editor:    editor,
	}

	// Validate
	if err := config.ValidateGlobalConfig(cfg); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Save config
	if err := config.SaveGlobalConfig(cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	// Get config path for display
	configPath, _ := config.ConfigPath()
	fmt.Printf("✓ Config saved to %s\n", configPath)
	fmt.Printf("  server_url: %s\n", cfg.ServerURL)
	fmt.Printf("  editor: %s\n", cfg.Editor)
}

func handleListFiles() {
	// Use current directory if path not provided
	repoPath := "."
	if len(os.Args) >= 3 {
		repoPath = os.Args[2]
	}

	// Check if path exists
	info, err := os.Stat(repoPath)
	if err != nil {
		fmt.Printf("Error: cannot access repository path: %v\n", err)
		os.Exit(1)
	}

	if !info.IsDir() {
		fmt.Printf("Error: path is not a directory: %s\n", repoPath)
		os.Exit(1)
	}

	// Discover files
	result, err := indexer.DiscoverFiles(repoPath)
	if err != nil {
		fmt.Printf("Error discovering files: %v\n", err)
		os.Exit(1)
	}

	// Group files by language
	filesByLanguage := make(map[string][]indexer.DiscoveredFile)
	for _, file := range result.Files {
		filesByLanguage[file.Language] = append(filesByLanguage[file.Language], file)
	}

	// Get sorted language keys
	languages := make([]string, 0, len(filesByLanguage))
	for lang := range filesByLanguage {
		languages = append(languages, lang)
	}
	sort.Strings(languages)

	// Display results
	fmt.Printf("\n📂 File Discovery Results\n")
	fmt.Printf("Repository: %s\n\n", repoPath)

	totalFiles := len(result.Files)
	fmt.Printf("Total files: %d\n", totalFiles)
	fmt.Printf("Total size: %.2f MB\n\n", float64(result.TotalSize)/(1024*1024))

	// Show files grouped by language
	for _, lang := range languages {
		files := filesByLanguage[lang]
		fmt.Printf("📝 %s (%d files)\n", lang, len(files))

		// Sort files by path
		sort.Slice(files, func(i, j int) bool {
			return files[i].Path < files[j].Path
		})

		for _, file := range files {
			fmt.Printf("  %s (%d lines, %.2f KB)\n",
				file.Path,
				file.Lines,
				float64(file.Size)/1024)
		}
		fmt.Println()
	}
}

func handleChunkFile() {
	if len(os.Args) < 3 {
		fmt.Println("Error: file path required")
		fmt.Println("Usage: codefind chunk-file <file-path>")
		os.Exit(1)
	}

	filePath := os.Args[2]

	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("Error: cannot access file: %v\n", err)
		os.Exit(1)
	}

	if info.IsDir() {
		fmt.Printf("Error: path is a directory, not a file: %s\n", filePath)
		os.Exit(1)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Create chunker with default config
	wc := chunker.NewWindowChunker(chunker.DefaultConfig())

	// Chunk the file
	chunks, err := wc.ChunkFile(string(content), filePath)
	if err != nil {
		fmt.Printf("Error chunking file: %v\n", err)
		os.Exit(1)
	}

	// Display results
	fmt.Printf("\n📦 File Chunking Results\n")
	fmt.Printf("File: %s\n", filePath)
	fmt.Printf("File size: %d bytes\n\n", len(content))

	fmt.Printf("Chunks: %d\n", len(chunks))
	fmt.Printf("Target chunk size: %d characters (~%.0f tokens)\n",
		int(float32(chunker.DefaultConfig().TargetTokens)*chunker.DefaultConfig().CharsPerToken),
		float32(chunker.DefaultConfig().TargetTokens))
	fmt.Printf("Overlap: %d characters (~%d tokens)\n\n",
		int(float32(chunker.DefaultConfig().OverlapTokens)*chunker.DefaultConfig().CharsPerToken),
		chunker.DefaultConfig().OverlapTokens)

	for i, chunk := range chunks {
		estimatedTokens := int(float32(len(chunk.Content)) / chunker.DefaultConfig().CharsPerToken)
		fmt.Printf("Chunk %d: Lines %d-%d (%d chars, ~%d tokens)\n",
			i+1,
			chunk.StartLine,
			chunk.EndLine,
			len(chunk.Content),
			estimatedTokens)
		fmt.Printf("  Hash: %s\n", chunk.Hash[:8]+"...")
	}
}

func handleIndex() {
	// Parse flags
	indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
	windowOnly := indexCmd.Bool("window-only", false, "Force window-based chunking (skip LSP)")
	
	// Find where flags start (after "index" and optional path)
	args := os.Args[2:] // Skip "codefind" and "index"
	
	// Separate path from flags
	repoPath := "."
	flagStart := 0
	for i, arg := range args {
		if strings.HasPrefix(arg, "-") {
			flagStart = i
			break
		}
		if i == 0 && !strings.HasPrefix(arg, "-") {
			repoPath = arg
			flagStart = 1
		}
	}
	
	// Parse remaining flags
	if flagStart < len(args) {
		indexCmd.Parse(args[flagStart:])
	}

	// Load global config
	cfg, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Run 'codefind init' first")
		os.Exit(1)
	}

	// Validate repo path
	info, err := os.Stat(repoPath)
	if err != nil {
		fmt.Printf("Error: cannot access repository: %v\n", err)
		os.Exit(1)
	}

	if !info.IsDir() {
		fmt.Printf("Error: path is not a directory: %s\n", repoPath)
		os.Exit(1)
	}

	// Show chunking mode
	if *windowOnly {
		fmt.Println("📦 Chunking mode: Window-only (LSP disabled)")
	} else {
		fmt.Println("📦 Chunking mode: Hybrid (LSP when available)")
	}

	// Create and run indexer
	indexOpts := indexer.IndexOptions{
		RepoPath:   repoPath,
		ServerURL:  cfg.ServerURL,
		AuthKey:    "secret-key-123", // TODO: Load from config in Phase 3A
		Model:      "unclemusclez/jina-embeddings-v2-base-code:latest",
		WindowOnly: *windowOnly,
	}

	idx := indexer.NewIndexer(indexOpts)
	if err := idx.Index(); err != nil {
		fmt.Printf("Indexing failed: %v\n", err)
		os.Exit(1)
	}
}

// handleCleanup handles the cleanup command for purging old deleted chunks
func handleCleanup(args []string) {
	// Load global config
	cfg, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Printf("Error: Not initialized. Run 'codefind init' first.\n")
		os.Exit(1)
	}

	// Get repo ID for current directory
	absPath, err := filepath.Abs(".")
	if err != nil {
		fmt.Printf("Error: cannot determine current directory: %v\n", err)
		os.Exit(1)
	}
	repoID := indexer.GenerateRepoID(absPath)

	// Parse cleanup options
	opts := cleanup.CleanupOptions{
		Project: repoID,
	}

	for _, arg := range args {
		if arg == "--dry-run" {
			opts.DryRun = true
		} else if arg == "--list" {
			opts.ListOnly = true
		} else if strings.HasPrefix(arg, "--older-than=") {
			fmt.Sscanf(arg, "--older-than=%d", &opts.OlderThan)
		}
	}

	// Create API client and cleanup client
	apiClient := client.NewAPIClient(cfg.ServerURL)
	apiClient.SetAuthKey("secret-key-123") // TODO: Load from config in Phase 3A
	cc := cleanup.NewCleanupClient(apiClient)

	// Run cleanup
	result, err := cc.Cleanup(opts)
	if err != nil {
		fmt.Printf("Cleanup failed: %v\n", err)
		os.Exit(1)
	}

	// Display result
	fmt.Println(cleanup.FormatResult(result, opts.ListOnly))
}

// handleStats handles the stats command
func handleStats(args []string) {
	// Load global config
	cfg, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Printf("Error: Not initialized. Run 'codefind init' first.\n")
		os.Exit(1)
	}

	// Get repo ID for current directory
	absPath, err := filepath.Abs(".")
	if err != nil {
		fmt.Printf("Error: cannot determine current directory: %v\n", err)
		os.Exit(1)
	}
	repoID := indexer.GenerateRepoID(absPath)

	// Try to get project name from manifest
	projectName := "Unknown"
	if manifest, err := config.LoadManifest(repoID); err == nil {
		projectName = manifest.ProjectName
	}

	// Create API client and stats client
	apiClient := client.NewAPIClient(cfg.ServerURL)
	sc := stats.NewStatsClient(apiClient)

	// Get stats
	statsResp, err := sc.GetStats(repoID)
	if err != nil {
		fmt.Printf("Failed to get stats: %v\n", err)
		os.Exit(1)
	}

	// Display formatted stats
	fmt.Println(stats.FormatStats(projectName, statsResp))
}

// handleLSP handles the lsp command
func handleLSP(args []string) {
	// Parse subcommand
	subcommand := "check"
	if len(args) > 0 {
		subcommand = args[0]
	}

	switch subcommand {
	case "check":
		// Force refresh if --refresh flag
		forceRefresh := false
		for _, arg := range args {
			if arg == "--refresh" || arg == "-r" {
				forceRefresh = true
				break
			}
		}

		lsps, err := lsp.GetOrDiscoverLSPs(forceRefresh)
		if err != nil {
			fmt.Printf("Error discovering LSPs: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(lsp.FormatLSPStatus(lsps))

	case "symbols":
		// Extract symbols from a file
		if len(args) < 3 {
			fmt.Println("Usage: codefind lsp symbols <language> <file>")
			fmt.Println("Example: codefind lsp symbols go ./cmd/codefind/main.go")
			os.Exit(1)
		}
		language := args[1]
		filePath := args[2]

		// Get absolute path
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			fmt.Printf("Error resolving path: %v\n", err)
			os.Exit(1)
		}

		// Get workspace root (current directory)
		workspaceRoot, _ := os.Getwd()

		fmt.Printf("Starting %s LSP for %s...\n", language, absPath)

		// Create LSP client
		client, err := lsp.NewLSPClient(language, workspaceRoot)
		if err != nil {
			fmt.Printf("Error creating LSP client: %v\n", err)
			os.Exit(1)
		}

		// Initialize with timeout (30s for slower LSPs like pyright)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		fmt.Println("Initializing LSP...")
		if err := client.Initialize(ctx); err != nil {
			fmt.Printf("Error initializing LSP: %v\n", err)
			client.Shutdown(ctx)
			os.Exit(1)
		}

		fmt.Println("Requesting document symbols...")
		symbols, err := client.DocumentSymbols(ctx, absPath)
		if err != nil {
			fmt.Printf("Error getting symbols: %v\n", err)
			client.Shutdown(ctx)
			os.Exit(1)
		}

		// Shutdown
		client.Shutdown(ctx)

		// Display symbols
		fmt.Printf("\nFound %d symbols:\n", len(symbols))
		fmt.Println("──────────────────────────────────────────")
		printSymbols(symbols, 0)

	case "chunks":
		// Chunk a file using LSP symbols
		if len(args) < 3 {
			fmt.Println("Usage: codefind lsp chunks <language> <file>")
			fmt.Println("Example: codefind lsp chunks go ./cmd/codefind/main.go")
			os.Exit(1)
		}
		language := args[1]
		filePath := args[2]

		// Get absolute path
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			fmt.Printf("Error resolving path: %v\n", err)
			os.Exit(1)
		}

		// Read file content
		content, err := os.ReadFile(absPath)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			os.Exit(1)
		}

		// Get workspace root
		workspaceRoot, _ := os.Getwd()

		fmt.Printf("Chunking %s with %s LSP...\n", absPath, language)

		// Create symbol chunker
		chunks, err := chunker.ChunkFileWithLSP(string(content), absPath, language, workspaceRoot, chunker.DefaultConfig())
		if err != nil {
			fmt.Printf("Error chunking file: %v\n", err)
			os.Exit(1)
		}

		// Display chunks
		fmt.Printf("\nGenerated %d chunks:\n", len(chunks))
		fmt.Println("──────────────────────────────────────────")
		for i, c := range chunks {
			parent := ""
			if c.ParentName != "" {
				parent = fmt.Sprintf(" (in %s)", c.ParentName)
			}
			fmt.Printf("\n[%d] %s (%s)%s\n", i+1, c.SymbolName, c.SymbolKind, parent)
			fmt.Printf("    Lines: %d-%d | Tokens: ~%d\n", c.StartLine, c.EndLine, c.TokenCount)
			fmt.Printf("    Hash: %s\n", c.Hash[:12])

			// Show preview (first 100 chars)
			preview := c.Content
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			preview = strings.ReplaceAll(preview, "\n", "↵")
			fmt.Printf("    Preview: %s\n", preview)
		}

	case "hybrid":
		// Test hybrid chunking on a file
		if len(args) < 2 {
			fmt.Println("Usage: codefind lsp hybrid <file> [--window-only]")
			fmt.Println("Example: codefind lsp hybrid ./cmd/codefind/main.go")
			os.Exit(1)
		}
		filePath := args[1]

		// Check for --window-only flag
		windowOnly := false
		for _, arg := range args {
			if arg == "--window-only" {
				windowOnly = true
				break
			}
		}

		// Get absolute path
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			fmt.Printf("Error resolving path: %v\n", err)
			os.Exit(1)
		}

		// Read file content
		content, err := os.ReadFile(absPath)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			os.Exit(1)
		}

		// Get workspace root
		workspaceRoot, _ := os.Getwd()

		fmt.Printf("Hybrid chunking: %s\n", absPath)
		if windowOnly {
			fmt.Println("Mode: Window-only (forced)")
		} else {
			fmt.Println("Mode: Auto-detect (LSP if available)")
		}

		// Create hybrid chunker
		hybridChunker := chunker.NewHybridChunker(chunker.DefaultConfig(), workspaceRoot, windowOnly)
		result, err := hybridChunker.ChunkFile(string(content), absPath)
		if err != nil {
			fmt.Printf("Error chunking file: %v\n", err)
			os.Exit(1)
		}

		// Display result
		fmt.Printf("\nResult:\n")
		fmt.Println("──────────────────────────────────────────")
		fmt.Printf("Language: %s\n", result.Language)
		fmt.Printf("Method: %s\n", result.Method)
		fmt.Printf("LSP Available: %v\n", result.LSPAvailable)
		fmt.Printf("Chunks: %d\n", len(result.Chunks))

		// Show first few chunks
		fmt.Printf("\nFirst 5 chunks:\n")
		for i, c := range result.Chunks {
			if i >= 5 {
				fmt.Printf("\n... and %d more chunks\n", len(result.Chunks)-5)
				break
			}
			name := c.SymbolName
			if name == "" {
				name = fmt.Sprintf("window-%d", i+1)
			}
			fmt.Printf("[%d] %s (%s) - Lines %d-%d, ~%d tokens\n",
				i+1, name, c.SymbolKind, c.StartLine, c.EndLine, c.TokenCount)
		}

	default:
		fmt.Printf("Unknown lsp subcommand: %s\n", subcommand)
		fmt.Println("Usage:")
		fmt.Println("  codefind lsp check [--refresh]       - Check available LSP servers")
		fmt.Println("  codefind lsp symbols <lang> <file>   - Extract symbols from file")
		fmt.Println("  codefind lsp chunks <lang> <file>    - Chunk file using symbols")
		fmt.Println("  codefind lsp hybrid <file> [--window-only] - Test hybrid chunking")
		os.Exit(1)
	}
}

// printSymbols recursively prints document symbols with indentation
func printSymbols(symbols []lsp.DocumentSymbol, indent int) {
	for _, sym := range symbols {
		prefix := strings.Repeat("  ", indent)
		fmt.Printf("%s• %s (%s) [L%d-%d]\n", prefix, sym.Name, sym.Kind.String(), sym.Range.Start.Line+1, sym.Range.End.Line+1)
		if len(sym.Children) > 0 {
			printSymbols(sym.Children, indent+1)
		}
	}
}

// savedResult stores query results for 'codefind open' command
type savedResult struct {
	ID        string `json:"id"`
	RepoID    string `json:"repo_id"`
	FilePath  string `json:"file_path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Content   string `json:"content"`
	Language  string `json:"language"`
}

// queryArgs holds parsed query arguments
type queryArgs struct {
	projectID      string
	lang           string
	pathPrefix     string
	excludePath    string
	topK           int
	page           int
	pageSize       int
	includeDeleted bool
	deletedOnly    bool
}

func handleQuery(args []string) {
	// Separate flags from positional arguments
	// This allows: codefind query "search text" --lang=python
	// OR: codefind query --lang=python "search text"
	var queryText string
	var flagArgs []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			flagArgs = append(flagArgs, arg)
		} else if queryText == "" {
			queryText = arg
		}
	}

	if queryText == "" {
		fmt.Println("Error: query text required")
		fmt.Println("\nUsage: codefind query <text> [options]")
		fmt.Println("\nOptions:")
		fmt.Println("  --project=<id>    Limit to specific project")
		fmt.Println("  --lang=<lang>     Filter by language (python, go, typescript)")
		fmt.Println("  --path=<prefix>   Filter by file path prefix")
		fmt.Println("  --exclude=<pat>   Exclude paths matching regex pattern")
		fmt.Println("  --top-k=<n>       Number of results (default 10)")
		fmt.Println("  --limit=<n>       Alias for --top-k")
		fmt.Println("  --page=<n>        Page number for pagination (default 1)")
		fmt.Println("  --page-size=<n>   Results per page (default 20)")
		os.Exit(1)
	}

	// Load global config
	cfg, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Run 'codefind init' first")
		os.Exit(1)
	}

	// Parse flag arguments
	qa := parseQueryArgs(flagArgs)

	// Create API client and query client
	apiClient := client.NewAPIClient(cfg.ServerURL)
	qc := query.NewQueryClient(apiClient)

	// Parse languages from comma-separated string
	var languages []string
	if qa.lang != "" {
		languages = strings.Split(qa.lang, ",")
	}

	// Execute search with new filter fields
	resp, err := qc.Search(queryText, qa.topK, languages, qa.pathPrefix, qa.excludePath, qa.includeDeleted, qa.deletedOnly)
	if err != nil {
		fmt.Printf("Query failed: %v\n", err)
		os.Exit(1)
	}

	if resp.Error != "" {
		fmt.Printf("Server error: %s\n", resp.Error)
		os.Exit(1)
	}

	// Display results using package-level function
	fmt.Println(query.FormatResults(resp))

	// Save results for 'codefind open' command
	if err := saveLastResults(resp); err != nil {
		fmt.Printf("Warning: could not save results: %v\n", err)
	}
}

// parseQueryArgs parses query command arguments
func parseQueryArgs(args []string) queryArgs {
	qa := queryArgs{
		topK:     10,
		page:     1,
		pageSize: 20,
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, "--project=") {
			qa.projectID = strings.TrimPrefix(arg, "--project=")
		} else if strings.HasPrefix(arg, "--lang=") {
			qa.lang = strings.TrimPrefix(arg, "--lang=")
		} else if strings.HasPrefix(arg, "--path=") {
			qa.pathPrefix = strings.TrimPrefix(arg, "--path=")
		} else if strings.HasPrefix(arg, "--top-k=") {
			fmt.Sscanf(arg, "--top-k=%d", &qa.topK)
		} else if strings.HasPrefix(arg, "--limit=") {
			fmt.Sscanf(arg, "--limit=%d", &qa.topK)
		} else if strings.HasPrefix(arg, "--page=") {
			fmt.Sscanf(arg, "--page=%d", &qa.page)
		} else if strings.HasPrefix(arg, "--page-size=") {
			fmt.Sscanf(arg, "--page-size=%d", &qa.pageSize)
		} else if strings.HasPrefix(arg, "--exclude=") {
			qa.excludePath = strings.TrimPrefix(arg, "--exclude=")
		} else if arg == "--include-deleted" {
			qa.includeDeleted = true
		} else if arg == "--deleted-only" {
			qa.deletedOnly = true
		}
	}

	return qa
}


// saveLastResults saves results for 'codefind open' command
func saveLastResults(resp *api.QueryResponse) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get home directory: %w", err)
	}

	resultsDir := filepath.Join(home, ".codefind")
	resultsFile := filepath.Join(resultsDir, "last-results.json")

	// Ensure directory exists
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return fmt.Errorf("could not create directory: %w", err)
	}

	results := make([]savedResult, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = savedResult{
			ID:        r.ChunkID,
			RepoID:    r.Metadata.RepoID,
			FilePath:  r.Metadata.FilePath,
			StartLine: r.Metadata.StartLine,
			EndLine:   r.Metadata.EndLine,
			Content:   r.Content,
			Language:  r.Metadata.Language,
		}
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(resultsFile, data, 0644)
}

func handleOpen(resultID string) {
	// Load last results
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error: could not get home directory: %v\n", err)
		os.Exit(1)
	}

	resultsFile := filepath.Join(home, ".codefind", "last-results.json")
	data, err := os.ReadFile(resultsFile)
	if err != nil {
		fmt.Println("Error: no recent query results")
		fmt.Println("Run 'codefind query' first")
		os.Exit(1)
	}

	var results []savedResult
	if err := json.Unmarshal(data, &results); err != nil {
		fmt.Println("Error: could not parse results")
		os.Exit(1)
	}

	// Find result by index or ID
	var result *savedResult

	// Try parsing as index (1-based)
	if idx, err := strconv.Atoi(resultID); err == nil {
		if idx > 0 && idx <= len(results) {
			result = &results[idx-1]
		}
	}

	// Try matching by ID if index didn't work
	if result == nil {
		for i := range results {
			if strings.HasPrefix(results[i].ID, resultID) {
				result = &results[i]
				break
			}
		}
	}

	if result == nil {
		fmt.Printf("Error: result not found\n")
		os.Exit(1)
	}

	// Load config to get editor
	cfg, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Println("Error: run 'codefind init' first")
		os.Exit(1)
	}

	// Load manifest to get repo path
	manifest, err := config.LoadManifest(result.RepoID)
	if err != nil {
		fmt.Printf("Error: could not load manifest for %s\n", result.RepoID)
		os.Exit(1)
	}

	// Resolve full file path
	fullPath := filepath.Join(manifest.RepoPath, result.FilePath)

	// If repo path is relative, make it absolute
	if !filepath.IsAbs(fullPath) {
		absPath, err := filepath.Abs(fullPath)
		if err == nil {
			fullPath = absPath
		}
	}

	// Verify file exists
	if _, err := os.Stat(fullPath); err != nil {
		fmt.Printf("Error: file not found: %s\n", fullPath)
		os.Exit(1)
	}

	// Open in editor with line number
	editor := cfg.Editor
	if editor == "" {
		editor = os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}
	}

	// Build editor-specific arguments
	var cmdArgs []string
	editorBase := filepath.Base(editor)

	switch {
	case strings.Contains(editorBase, "code"):
		// VS Code: code --goto file:line
		cmdArgs = []string{"--goto", fmt.Sprintf("%s:%d", fullPath, result.StartLine)}
	case strings.Contains(editorBase, "subl"):
		// Sublime Text: subl file:line
		cmdArgs = []string{fmt.Sprintf("%s:%d", fullPath, result.StartLine)}
	case strings.Contains(editorBase, "idea") || strings.Contains(editorBase, "goland"):
		// JetBrains IDEs: idea --line N file
		cmdArgs = []string{"--line", strconv.Itoa(result.StartLine), fullPath}
	default:
		// vim, nvim, nano, emacs, etc.: editor +line file
		cmdArgs = []string{fmt.Sprintf("+%d", result.StartLine), fullPath}
	}

	cmd := exec.Command(editor, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error: could not open editor: %v\n", err)
		os.Exit(1)
	}
}

func handleList() {
	// Read all manifests from ~/.codefind/manifests/
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error: could not get home directory: %v\n", err)
		os.Exit(1)
	}

	manifestDir := filepath.Join(home, ".codefind", "manifests")

	entries, err := os.ReadDir(manifestDir)
	if err != nil {
		fmt.Println("No projects indexed yet")
		return
	}

	fmt.Println("Indexed Projects:")
	fmt.Printf("%-40s %-12s %-8s %s\n", "PROJECT", "REPO ID", "CHUNKS", "INDEXED AT")
	fmt.Println(strings.Repeat("-", 80))

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		repoID := strings.TrimSuffix(entry.Name(), ".json")
		manifest, err := config.LoadManifest(repoID)
		if err != nil {
			continue
		}

		// Parse timestamp
		indexedAt := "unknown"
		if manifest.IndexedAt != "" {
			t, err := time.Parse(time.RFC3339, manifest.IndexedAt)
			if err == nil {
				indexedAt = t.Format("2006-01-02")
			}
		}

		// Safely truncate repo ID
		displayRepoID := repoID
		if len(displayRepoID) > 12 {
			displayRepoID = displayRepoID[:12]
		}

		fmt.Printf("%-40s %-12s %-8d %s\n",
			manifest.ProjectName,
			displayRepoID,
			manifest.ActiveChunkCount,
			indexedAt)
	}
}

// handleClear clears all chunks from a repository's collection
func handleClear(repoPath string) {
	// Get absolute path
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		fmt.Printf("Error: failed to resolve path: %v\n", err)
		os.Exit(1)
	}

	// Load global config for server URL and auth
	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Printf("Error: codefind not initialized. Run 'codefind init' first.\n")
		os.Exit(1)
	}

	// Load manifest to get repo_id
	repoID := indexer.GenerateRepoID(absPath)
	manifest, err := config.LoadManifest(repoID)
	if err != nil {
		fmt.Printf("Error: no manifest found for %s. Has this repo been indexed?\n", absPath)
		os.Exit(1)
	}

	// Confirm with user
	fmt.Printf("⚠️  This will delete ALL indexed chunks for project '%s'\n", manifest.ProjectName)
	fmt.Printf("   Repo ID: %s\n", manifest.RepoID)
	fmt.Print("   Continue? [y/N]: ")

	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Cancelled.")
		return
	}

	// Call server to delete collection
	apiClient := client.NewAPIClient(globalCfg.ServerURL)
	apiClient.SetAuthKey("secret-key-123") // TODO: Load from config in Phase 3A

	err = apiClient.ClearCollection(manifest.RepoID)
	if err != nil {
		fmt.Printf("Error: failed to clear collection: %v\n", err)
		os.Exit(1)
	}

	// Delete local manifest
	manifestPath, _ := config.ManifestPath(repoID)
	if err := os.Remove(manifestPath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: failed to delete manifest: %v\n", err)
	}

	fmt.Printf("✅ Cleared all chunks for '%s'\n", manifest.ProjectName)
	fmt.Println("   Run 'codefind index' to re-index from scratch.")
}

// promptFor displays a prompt and reads user input
// Returns the input or defaultValue if input is empty
func promptFor(label string, defaultValue string) string {
	prompt := promptui.Prompt{
		Label:   label,
		Default: defaultValue,
	}

	result, err := prompt.Run()
	if err != nil {
		return defaultValue
	}
	return result
}

func printUsage() {
	fmt.Println(`Codefind - Code semantic search tool

Usage:
  codefind init [--server-url=<url>] [--editor=<editor>]
    Initialize configuration (sets up global config)

  codefind index [repo-path]
    Index a repository: discover files, chunk, tokenize, and send to server
    (defaults to current directory if path not provided)

  codefind query <text> [--options]
    Search indexed code with semantic query
    Options:
      --project=<id>    Limit to specific project
      --lang=<lang>     Filter by language
      --path=<prefix>   Filter by file path
      --top-k=<n>       Number of results (default 10)
      --page=<n>        Page number (default 1)
      --page-size=<n>   Results per page (default 20)

  codefind list
    Show all indexed projects

  codefind open <id>
    Open query result in editor at the correct line
    id: Result number (1, 2, 3...) or UUID prefix

  codefind list-files [repo-path]
    Discover and list all indexable files in a repository

  codefind chunk-file <file-path>
    Split a file into chunks using window-based strategy

  codefind clear [repo-path]
    Delete all indexed chunks for a repository
    (defaults to current directory if path not provided)

  codefind help, -h, --help
    Show this help message

Examples:
  codefind init --server-url=http://localhost:8080
  codefind index
  codefind query "authentication logic"
  codefind query "error handling" --lang=python
  codefind query "API" --project=my-api --top-k=20`)
}
