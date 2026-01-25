package indexer

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tk-425/Codefind/internal/chunker"
	"github.com/tk-425/Codefind/internal/client"
	"github.com/tk-425/Codefind/internal/config"
	"github.com/tk-425/Codefind/pkg/api"
)

// IndexOptions contains options for indexing
type IndexOptions struct {
	RepoPath  string // Repository path
	ServerURL string // Server URL for API calls
	AuthKey   string // Authentication key
	Model     string // Embedding model name
}

// Indexer orchestrates the indexing pipeline
type Indexer struct {
	options   IndexOptions
	client    *client.APIClient
	tokenizer *client.Tokenizer
	manifest  *config.RepositoryManifest
}

// NewIndexer creates a new indexer
func NewIndexer(opts IndexOptions) *Indexer {
	apiClient := client.NewAPIClient(opts.ServerURL)
	apiClient.SetAuthKey(opts.AuthKey)
	tokenizer := client.NewTokenizer(apiClient, opts.Model)

	return &Indexer{
		options:   opts,
		client:    apiClient,
		tokenizer: tokenizer,
	}
}

// Index performs a full index of the repository
func (idx *Indexer) Index() error {
	// Validate server connectivity
	fmt.Println("🔍 Validating server connectivity...")
	health, err := idx.client.Health()
	if err != nil {
		return fmt.Errorf("server unreachable: %w", err)
	}

	if health.Status != "ok" {
		return fmt.Errorf("server unhealthy: ollama=%s, chromadb=%s",
			health.OllamaStatus, health.ChromaDBStatus)
	}

	fmt.Println("✓ Server is healthy")

	// Create or load manifest
	fmt.Println("📋 Loading/creating repository manifest...")

	repoID := generateRepoID(idx.options.RepoPath)
	manifest, err := idx.loadOrCreateManifest(repoID)
	if err != nil {
		return fmt.Errorf("failed to load/create manifest: %w", err)
	}
	idx.manifest = manifest
	fmt.Printf("✓ Repository: %s\n", manifest.ProjectName)

	// Discover files
	fmt.Println("📂 Discovering files...")
	result, err := DiscoverFiles(idx.options.RepoPath)
	if err != nil {
		return fmt.Errorf("file discovery failed: %w", err)
	}
	fmt.Printf("✓ Found %d files (%.2f MB)\n", len(result.Files), float64(result.TotalSize)/(1024*1024))

	// Process files and collect chunks
	fmt.Println("⚙️ Chunking and tokenizing files...")
	allChunks := []api.Chunk{}

	for i, file := range result.Files {
		// Read file
		content, err := os.ReadFile(filepath.Join(idx.options.RepoPath, file.Path))
		if err != nil {
			fmt.Printf("⚠️ Skipped %s: %v\n", file.Path, err)
			continue
		}

		// Chunk the file
		wc := chunker.NewWindowChunker(chunker.DefaultConfig())
		chunks, err := wc.ChunkFile(string(content), file.Path)
		if err != nil {
			fmt.Printf("⚠️ Failed to chunk %s: %v\n", file.Path, err)
			continue
		}

		// Tokenize chunks (300 to stay well under BERT's 512 max)
		verifiedChunks, err := idx.tokenizer.TokenizeChunks(chunks, 300)
		if err != nil {
			fmt.Printf("⚠️ Failed to tokenize %s: %v\n", file.Path, err)
			continue
		}

		// Convert to API chunks with metadata
		for _, chunk := range verifiedChunks {
			apiChunk := api.Chunk{
				Content: chunk.Content,
				Metadata: api.ChunkMetadata{
					RepoID:      manifest.RepoID,
					ProjectName: manifest.ProjectName,
					FilePath:    file.Path,
					Language:    file.Language,
					StartLine:   chunk.StartLine,
					EndLine:     chunk.EndLine,
					ContentHash: chunk.Hash,
					ChunkTokens: chunk.TokenCount,
					ModelID:     idx.options.Model,
					IndexedAt:   time.Now(),
					Status:      "active",
					IsSplit:     false,
				},
			}
			allChunks = append(allChunks, apiChunk)
		}

		fmt.Printf("  [%d/%d] %s: %d chunks\n", i+1, len(result.Files),
			file.Path, len(verifiedChunks))
	}

	fmt.Printf("✓ Total chunks to index: %d\n", len(allChunks))

	// Send chunks to server in batches with retry logic
	fmt.Println("📤 Sending chunks to server...")
	const batchSize = 8  // Reduced from 16 for faster embedding on CPU
	const maxRetries = 3
	totalInserted := 0
	totalBatches := (len(allChunks) + batchSize - 1) / batchSize

	for i := 0; i < len(allChunks); i += batchSize {
		end := min(i + batchSize, len(allChunks))
		batch := allChunks[i:end]
		batchNum := (i / batchSize) + 1

		// Progress indicator
		fmt.Printf("  Batch %d/%d: %d chunks", batchNum, totalBatches, len(batch))

		// Send with retry logic
		err := idx.sendBatchWithRetry(batch, maxRetries)
		if err != nil {
			fmt.Printf(" ❌\n")

			// Check if we should save partial progress
			if isNetworkError(err) {
				fmt.Printf("\n⚠️  Network error detected. Saving partial progress...\n")
				idx.manifest.ActiveChunkCount = totalInserted
				if saveErr := config.SaveManifest(idx.manifest); saveErr != nil {
					return fmt.Errorf("failed to save partial progress: %w", saveErr)
				}
				fmt.Printf("✓ Saved %d/%d chunks to manifest\n\n", totalInserted, len(allChunks))
				return fmt.Errorf("indexing paused due to network error: %w", err)
			}

			return fmt.Errorf("batch %d failed: %w", batchNum, err)
		}

		totalInserted += len(batch)

		// Progress percentage
		progress := float64(totalInserted) / float64(len(allChunks)) * 100
		fmt.Printf(" ✓ (%.0f%%)\n", progress)
	}

	fmt.Printf("\n✅ All chunks sent successfully!\n")

	// Update manifest
	fmt.Println("💾 Updating manifest...")
	now := time.Now().Format(time.RFC3339)
	manifest.IndexedAt = now
	manifest.ActiveChunkCount = len(allChunks)
	manifest.DeletedChunkCount = 0

	// For git repos, store commit hash
	if IsGitRepository(idx.options.RepoPath) {
		// TODO: Implement in Phase 2B: Get current HEAD commit hash
		// manifest.LastIndexedCommit = getCommitHash(idx.options.RepoPath)
	}

	// Store indexed files
	manifest.IndexedFiles = make(map[string]config.FileInfo)
	for _, file := range result.Files {
		manifest.IndexedFiles[file.Path] = config.FileInfo{
			Language:    file.Language,
			LineCount:   file.Lines,
			ChunkCount:  0, // TODO: Calculate actual chunk count per file
			LastModTime: now,
			ContentHash: "", // TODO: Compute file content hash in Phase 2B
		}
	}

	if err := config.SaveManifest(manifest); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}
	fmt.Println("✓ Manifest saved")

	// Success
	fmt.Printf("\n✅ Indexing complete!\n")
	fmt.Printf("   Total chunks: %d\n", totalInserted)
	fmt.Printf("   Indexed at: %s\n", manifest.IndexedAt)

	return nil
}

// loadOrCreateManifest loads existing manifest or creates new one
func (idx *Indexer) loadOrCreateManifest(repoID string) (*config.RepositoryManifest, error) {
	manifest, err := config.LoadManifest(repoID)
	if err == nil {
		return manifest, nil
	}

	// Create new manifest
	projectName := filepath.Base(idx.options.RepoPath)
	manifest = &config.RepositoryManifest{
		RepoID:            repoID,
		ProjectName:       projectName,
		RepoPath:          idx.options.RepoPath,
		IndexedFiles:      make(map[string]config.FileInfo),
		ActiveChunkCount:  0,
		DeletedChunkCount: 0,
	}

	return manifest, nil
}

// sendBatch sends a single batch of chunks to the server
func (idx *Indexer) sendBatch(batch []api.Chunk) error {
	indexReq := api.IndexRequest{
		Chunks:     batch,
		Collection: idx.manifest.RepoID,
	}

	resp, err := idx.client.Index(indexReq)
	if err != nil {
		return err
	}

	if resp.InsertedCount != len(batch) {
		return fmt.Errorf("expected %d chunks inserted, got %d", len(batch), resp.InsertedCount)
	}

	return nil
}

// sendBatchWithRetry sends a batch with exponential backoff retry logic
func (idx *Indexer) sendBatchWithRetry(batch []api.Chunk, maxRetries int) error {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := idx.sendBatch(batch)
		if err == nil {
			return nil
		}

		// Check if error is retryable (network or server errors)
		if !isRetryableError(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		if attempt == maxRetries {
			return fmt.Errorf("failed after %d attempts: %w", maxRetries, err)
		}

		// Exponential backoff: 1s, 2s, 4s...
		backoff := time.Duration(1<<uint(attempt-1)) * time.Second
		fmt.Printf("  ⚠️  Attempt %d/%d failed: %v\n", attempt, maxRetries, err)
		fmt.Printf("  ⏳ Retrying in %v...\n", backoff)
		time.Sleep(backoff)
	}
	return nil
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network-related errors
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network unreachable") ||
		strings.Contains(errStr, "dial tcp") {
		return true
	}

	// Server errors (5xx) are retryable, client errors (4xx) are not
	if strings.Contains(errStr, "status code: 5") {
		return true
	}

	return false
}

// isNetworkError checks if error is a network connectivity issue
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network unreachable") ||
		strings.Contains(errStr, "dial tcp")
}

// generateRepoID creates a unique ID for a repository
func generateRepoID(repoPath string) string {
	// Hash the absolute path to guarantee uniqueness
	// This ensures repos with same directory name in different locations get different IDs
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		absPath = repoPath
	}

	// Create SHA256 hash of the absolute path
	hash := sha256.Sum256([]byte(absPath))
	hashStr := fmt.Sprintf("%x", hash)

	// Use first 12 characters of the hash as repo ID
	// 12 chars = 48 bits, sufficient for collision-free IDs in practice
	return hashStr[:12]
}
