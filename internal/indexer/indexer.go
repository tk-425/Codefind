package indexer

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tk-425/Codefind/internal/chunker"
	"github.com/tk-425/Codefind/internal/client"
	"github.com/tk-425/Codefind/internal/config"
	"github.com/tk-425/Codefind/pkg/api"
)

// IndexOptions contains options for indexing
type IndexOptions struct {
	RepoPath    string // Repository path
	ServerURL   string // Server URL for API calls
	AuthKey     string // Authentication key
	Model       string // Embedding model name
	WindowOnly  bool   // Force window-based chunking (skip LSP)
	Concurrency int    // Number of concurrent batch requests (default: 2)
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

// Index performs a full or incremental index of the repository
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

	repoID := GenerateRepoID(idx.options.RepoPath)
	manifest, err := idx.loadOrCreateManifest(repoID)
	if err != nil {
		return fmt.Errorf("failed to load/create manifest: %w", err)
	}
	idx.manifest = manifest
	fmt.Printf("✓ Repository: %s\n", manifest.ProjectName)

	// Detect changes (NEW for Phase 2A)
	fmt.Println("🔄 Detecting changes...")
	changes, err := DetectChanges(idx.options.RepoPath, manifest)
	if err != nil {
		fmt.Printf("⚠️  Change detection failed, doing full index: %v\n", err)
		return idx.fullIndex()
	}

	// If first-time index or no LastIndexedCommit, do full index
	if changes.IsFullIndex() {
		fmt.Println("📊 First-time index detected, performing full index...")
		return idx.fullIndex()
	}

	// If no changes, skip indexing
	if !changes.HasChanges() {
		fmt.Println("✓ No changes detected, repository is up to date!")
		return nil
	}

	// Perform incremental index
	return idx.incrementalIndex(changes)
}

// incrementalIndex processes only changed files
func (idx *Indexer) incrementalIndex(changes *ChangeDetectionResult) error {
	fmt.Printf("📊 Incremental index: +%d added, ~%d modified, -%d deleted\n",
		len(changes.Added), len(changes.Modified), len(changes.Deleted))

	// Step 1: Handle deleted files
	if len(changes.Deleted) > 0 {
		fmt.Printf("🗑️  Marking %d deleted file(s)...\n", len(changes.Deleted))
		if err := idx.markChunksDeleted(changes.Deleted); err != nil {
			return fmt.Errorf("failed to mark deleted chunks: %w", err)
		}
	}

	// Step 2: Handle modified files (mark old chunks deleted first)
	if len(changes.Modified) > 0 {
		fmt.Printf("♻️  Re-indexing %d modified file(s)...\n", len(changes.Modified))
		if err := idx.markChunksDeleted(changes.Modified); err != nil {
			return fmt.Errorf("failed to mark old chunks deleted: %w", err)
		}
	}

	// Step 3: Index added + modified files
	filesToIndex := append(changes.Added, changes.Modified...)
	if len(filesToIndex) > 0 {
		fmt.Printf("📤 Indexing %d file(s)...\n", len(filesToIndex))
		if err := idx.indexFiles(filesToIndex); err != nil {
			return err
		}
	}

	// Step 4: Update manifest
	fmt.Println("💾 Updating manifest...")
	idx.manifest.LastIndexedCommit = changes.CurrentCommit
	idx.manifest.IndexedAt = time.Now().Format(time.RFC3339)
	if err := config.SaveManifest(idx.manifest); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	fmt.Println("✅ Incremental index complete!")
	return nil
}

// markChunksDeleted marks chunks as deleted without removing from ChromaDB
// This preserves history and allows recovery (tombstone mode)
func (idx *Indexer) markChunksDeleted(filePaths []string) error {
	if len(filePaths) == 0 {
		return nil
	}

	// Get current commit for tracking deletion context
	commit := ""
	if IsGitRepository(idx.options.RepoPath) {
		commit, _ = getHeadCommit(idx.options.RepoPath)
	}

	// Call server to mark chunks as deleted
	req := api.ChunkStatusRequest{
		AuthKey:         idx.options.AuthKey,
		Collection:      idx.manifest.RepoID,
		FilePaths:       filePaths,
		Status:          "deleted",
		DeletedInCommit: commit,
	}

	resp, err := idx.client.UpdateChunkStatus(req)
	if err != nil {
		return fmt.Errorf("failed to mark chunks as deleted: %w", err)
	}

	// Update local manifest
	for _, path := range filePaths {
		delete(idx.manifest.IndexedFiles, path)
	}
	idx.manifest.DeletedChunkCount += resp.UpdatedCount

	if resp.UpdatedCount > 0 {
		fmt.Printf("🗑️  Marked %d chunks as deleted from %d files\n", resp.UpdatedCount, len(filePaths))
	}

	return nil
}

// indexFiles processes specific files and sends chunks to server
func (idx *Indexer) indexFiles(filePaths []string) error {
	allChunks := []api.Chunk{}

	for i, filePath := range filePaths {
		// Read file
		fullPath := filepath.Join(idx.options.RepoPath, filePath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			fmt.Printf("⚠️ Skipped %s: %v\n", filePath, err)
			continue
		}

		// Get file info for language detection
		language := LanguageFromExtension(filePath)

		// Chunk the file using hybrid chunker (LSP if available, else window)
		hc := chunker.NewHybridChunker(chunker.DefaultConfig(), idx.options.RepoPath, idx.options.WindowOnly)
		result, err := hc.ChunkFile(string(content), filePath)
		if err != nil {
			fmt.Printf("⚠️ Failed to chunk %s: %v\n", filePath, err)
			continue
		}

		// Convert SymbolChunks to Chunks for tokenization
		chunks := make([]chunker.Chunk, len(result.Chunks))
		for j, sc := range result.Chunks {
			chunks[j] = sc.Chunk
		}

		// Tokenize chunks (300 to stay well under BERT's 512 max)
		verifiedChunks, err := idx.tokenizer.TokenizeChunks(chunks, 300)
		if err != nil {
			fmt.Printf("⚠️ Failed to tokenize %s: %v\n", filePath, err)
			continue
		}

		// Convert to API chunks with metadata
		for _, chunk := range verifiedChunks {
			// Generate stable chunk ID (instead of UUID)
			chunkID := chunker.GenerateChunkID(
				idx.manifest.RepoID,
				filePath,
				chunk.StartLine,
				chunk.EndLine,
				chunk.Content,
			)

			apiChunk := api.Chunk{
				ID:      chunkID,
				Content: chunk.Content,
				Metadata: api.ChunkMetadata{
					RepoID:      idx.manifest.RepoID,
					ProjectName: idx.manifest.ProjectName,
					FilePath:    filePath,
					Language:    language,
					StartLine:   chunk.StartLine,
					EndLine:     chunk.EndLine,
					ContentHash: chunker.GenerateContentHash(chunk.Content),
					ChunkTokens: chunk.TokenCount,
					ModelID:     idx.options.Model,
					IndexedAt:   time.Now(),
					Status:      "active",
					IsSplit:     false,
				},
			}
			allChunks = append(allChunks, apiChunk)
		}

		// Update manifest for this file
		stat, _ := os.Stat(fullPath)
		idx.manifest.IndexedFiles[filePath] = config.FileInfo{
			Language:    language,
			LineCount:   strings.Count(string(content), "\n") + 1,
			ChunkCount:  len(verifiedChunks),
			LastModTime: stat.ModTime().Format(time.RFC3339),
			ContentHash: chunker.GenerateContentHash(string(content)), // File content hash
		}

		// Log with chunking method
		methodTag := "[WINDOW]"
		if result.Method == "symbol" {
			methodTag = "[LSP]"
		}
		fmt.Printf("  %s [%d/%d] %s: %d chunks\n", methodTag, i+1, len(filePaths),
			filePath, len(verifiedChunks))
	}

	if len(allChunks) == 0 {
		fmt.Println("  No chunks to index")
		return nil
	}

	// Deduplicate chunks before sending (prevents duplicate ID errors)
	allChunks = deduplicateChunks(allChunks)

	// Send chunks to server in batches
	return idx.sendChunksInBatches(allChunks)
}

// sendChunksInBatches sends chunks to server using parallel batching
func (idx *Indexer) sendChunksInBatches(allChunks []api.Chunk) error {
	fmt.Println("📤 Sending chunks (parallel)...")

	if err := idx.sendChunksParallel(allChunks); err != nil {
		return err
	}

	idx.manifest.ActiveChunkCount += len(allChunks)
	fmt.Printf("✓ Indexed %d chunks\n", len(allChunks))
	return nil
}

// fullIndex performs a complete index of all files
func (idx *Indexer) fullIndex() error {
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
	chunkCounts := make(map[string]int) // Track chunk count per file path

	for i, file := range result.Files {
		// Read file
		content, err := os.ReadFile(filepath.Join(idx.options.RepoPath, file.Path))
		if err != nil {
			fmt.Printf("⚠️ Skipped %s: %v\n", file.Path, err)
			continue
		}

		// Chunk the file using hybrid chunker (LSP if available, else window)
		hc := chunker.NewHybridChunker(chunker.DefaultConfig(), idx.options.RepoPath, idx.options.WindowOnly)
		chunkResult, err := hc.ChunkFile(string(content), file.Path)
		if err != nil {
			fmt.Printf("⚠️ Failed to chunk %s: %v\n", file.Path, err)
			continue
		}

		// Convert SymbolChunks to Chunks for tokenization
		chunks := make([]chunker.Chunk, len(chunkResult.Chunks))
		for j, sc := range chunkResult.Chunks {
			chunks[j] = sc.Chunk
		}

		// Tokenize chunks (300 to stay well under BERT's 512 max)
		verifiedChunks, err := idx.tokenizer.TokenizeChunks(chunks, 300)
		if err != nil {
			fmt.Printf("⚠️ Failed to tokenize %s: %v\n", file.Path, err)
			continue
		}

		// Convert to API chunks with metadata
		for _, chunk := range verifiedChunks {
			// Generate stable chunk ID (instead of UUID)
			chunkID := chunker.GenerateChunkID(
				idx.manifest.RepoID,
				file.Path,
				chunk.StartLine,
				chunk.EndLine,
				chunk.Content,
			)

			apiChunk := api.Chunk{
				ID:      chunkID,
				Content: chunk.Content,
				Metadata: api.ChunkMetadata{
					RepoID:      idx.manifest.RepoID,
					ProjectName: idx.manifest.ProjectName,
					FilePath:    file.Path,
					Language:    file.Language,
					StartLine:   chunk.StartLine,
					EndLine:     chunk.EndLine,
					ContentHash: chunker.GenerateContentHash(chunk.Content),
					ChunkTokens: chunk.TokenCount,
					ModelID:     idx.options.Model,
					IndexedAt:   time.Now(),
					Status:      "active",
					IsSplit:     false,
				},
			}
			allChunks = append(allChunks, apiChunk)
		}

		// Log with chunking method
		methodTag := "[WINDOW]"
		if chunkResult.Method == "symbol" {
			methodTag = "[LSP]"
		}
		fmt.Printf("  %s [%d/%d] %s: %d chunks\n", methodTag, i+1, len(result.Files),
			file.Path, len(verifiedChunks))
		chunkCounts[file.Path] = len(verifiedChunks) // Store chunk count for this file
	}

	// Deduplicate chunks before sending (prevents duplicate ID errors)
	allChunks = deduplicateChunks(allChunks)
	fmt.Printf("✓ Total chunks to index: %d\n", len(allChunks))

	// Send chunks to server in parallel batches
	fmt.Println("📤 Sending chunks to server (parallel)...")
	if err := idx.sendChunksParallel(allChunks); err != nil {
		// On error, try to save partial progress
		if isNetworkError(err) {
			fmt.Printf("\n⚠️  Network error detected. Saving partial progress...\n")
			if saveErr := config.SaveManifest(idx.manifest); saveErr != nil {
				return fmt.Errorf("failed to save partial progress: %w", saveErr)
			}
			return fmt.Errorf("indexing paused due to network error: %w", err)
		}
		return err
	}

	fmt.Printf("✅ All chunks sent successfully!\n")

	// Update manifest
	fmt.Println("💾 Updating manifest...")
	now := time.Now().Format(time.RFC3339)
	idx.manifest.IndexedAt = now
	idx.manifest.ActiveChunkCount = len(allChunks)
	idx.manifest.DeletedChunkCount = 0

	// For git repos, store commit hash
	if IsGitRepository(idx.options.RepoPath) {
		commit, err := getHeadCommit(idx.options.RepoPath)
		if err == nil {
			idx.manifest.LastIndexedCommit = commit
		}
	}

	// Store indexed files with mtime and content hash
	idx.manifest.IndexedFiles = make(map[string]config.FileInfo)
	for _, file := range result.Files {
		fullPath := filepath.Join(idx.options.RepoPath, file.Path)
		content, _ := os.ReadFile(fullPath)
		stat, _ := os.Stat(fullPath)
		mtime := ""
		if stat != nil {
			mtime = stat.ModTime().Format(time.RFC3339)
		}

		idx.manifest.IndexedFiles[file.Path] = config.FileInfo{
			Language:    file.Language,
			LineCount:   file.Lines,
			ChunkCount:  chunkCounts[file.Path], // Actual chunk count per file
			LastModTime: mtime,
			ContentHash: chunker.GenerateContentHash(string(content)),
		}
	}

	if err := config.SaveManifest(idx.manifest); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}
	fmt.Println("✓ Manifest saved")

	// Success
	fmt.Printf("\n✅ Indexing complete!\n")
	fmt.Printf("   Total chunks: %d\n", len(allChunks))
	fmt.Printf("   Indexed at: %s\n", idx.manifest.IndexedAt)

	return nil
}

// loadOrCreateManifest loads existing manifest or creates new one
func (idx *Indexer) loadOrCreateManifest(repoID string) (*config.RepositoryManifest, error) {
	manifest, err := config.LoadManifest(repoID)
	if err == nil {
		return manifest, nil
	}

	// Create new manifest
	// Use absolute path to get proper directory name (handles "." case)
	absPath, _ := filepath.Abs(idx.options.RepoPath)
	projectName := filepath.Base(absPath)
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

// splitIntoBatches divides chunks into batches of specified size
func splitIntoBatches(chunks []api.Chunk, batchSize int) [][]api.Chunk {
	var batches [][]api.Chunk
	for i := 0; i < len(chunks); i += batchSize {
		end := min(i+batchSize, len(chunks))
		batches = append(batches, chunks[i:end])
	}
	return batches
}

// sendChunksParallel sends chunks to server with concurrent goroutines
func (idx *Indexer) sendChunksParallel(allChunks []api.Chunk) error {
	const batchSize = 8 // Smaller batches for stability
	const maxRetries = 3

	// Use configured concurrency with bounds (default: 2, max: 8)
	maxConcurrent := idx.options.Concurrency
	if maxConcurrent <= 0 {
		maxConcurrent = 2
	}
	if maxConcurrent > 8 {
		fmt.Printf("⚠️  Concurrency capped to 8 (was %d)\n", maxConcurrent)
		maxConcurrent = 8
	}

	batches := splitIntoBatches(allChunks, batchSize)
	totalBatches := len(batches)

	if totalBatches == 0 {
		return nil
	}

	// Track errors from goroutines
	errChan := make(chan error, totalBatches)
	sem := make(chan struct{}, maxConcurrent) // Semaphore for concurrency limit

	var wg sync.WaitGroup
	var mu sync.Mutex
	completed := 0
	var firstErr error

	for i, batch := range batches {
		wg.Add(1)
		go func(batchNum int, b []api.Chunk) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			err := idx.sendBatchWithRetry(b, maxRetries)

			mu.Lock()
			completed++
			progress := float64(completed) / float64(totalBatches) * 100
			if err != nil {
				fmt.Printf("\r  Batch %d/%d ❌ Error: %v\n", batchNum, totalBatches, err)
			} else {
				fmt.Printf("\r  Progress: %d/%d batches ✓ (%.0f%%)          ", completed, totalBatches, progress)
			}
			mu.Unlock()

			errChan <- err
		}(i+1, batch)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Print newline after progress
	fmt.Println()

	if firstErr != nil {
		return fmt.Errorf("batch failed: %w", firstErr)
	}

	return nil
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Network-related errors (case-insensitive)
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
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

// deduplicateChunks removes duplicate chunks by ID, keeping the first occurrence
// This prevents "duplicate ID in upsert" errors from ChromaDB when LSP chunking
// produces overlapping symbols or tokenization creates identical chunks
func deduplicateChunks(chunks []api.Chunk) []api.Chunk {
	seen := make(map[string]bool)
	result := []api.Chunk{}

	for _, chunk := range chunks {
		if !seen[chunk.ID] {
			seen[chunk.ID] = true
			result = append(result, chunk)
		}
	}
	return result
}

// GenerateRepoID creates a unique ID for a repository
func GenerateRepoID(repoPath string) string {
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
