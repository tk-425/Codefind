package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/manifoldco/promptui"
	"github.com/tk-425/Codefind/internal/chunker"
	"github.com/tk-425/Codefind/internal/config"
	"github.com/tk-425/Codefind/internal/indexer"
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
	case "help", "-h", "--help":
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
	// Load global config
	cfg, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Run 'codefind init' first")
		os.Exit(1)
	}

	// Use current directory
	repoPath := "."
	if len(os.Args) >= 3 {
		repoPath = os.Args[2]
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

	// Create and run indexer
	indexOpts := indexer.IndexOptions{
		RepoPath:  repoPath,
		ServerURL: cfg.ServerURL,
		AuthKey:   "secret-key-123", // TODO: Load from config in Phase 3A
		Model:     "unclemusclez/jina-embeddings-v2-base-code:latest",
	}

	idx := indexer.NewIndexer(indexOpts)
	if err := idx.Index(); err != nil {
		fmt.Printf("Indexing failed: %v\n", err)
		os.Exit(1)
	}
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

  codefind list-files [repo-path]
    Discover and list all indexable files in a repository
    (defaults to current directory if path not provided)

  codefind chunk-file <file-path>
    Split a file into chunks using window-based strategy
    Shows chunk boundaries, line numbers, and estimated tokens

  codefind index [repo-path]
    Index a repository: discover files, chunk, tokenize, and send to server
    (defaults to current directory if path not provided)
    Requires: 'codefind init' must be run first

  codefind help, -h, --help
    Show this help message

Examples:
  codefind init
  codefind init --server-url=http://localhost:8080
  codefind init --server-url=http://192.168.1.50:8080 --editor=code
  codefind list-files /path/to/repo
  codefind list-files
  codefind chunk-file ./cmd/codefind/main.go
  codefind index
  codefind index /path/to/repo`)
}
