package indexer

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tk-425/Codefind/internal/chunker"
	"github.com/tk-425/Codefind/internal/lsp"
	"github.com/tk-425/Codefind/internal/pathutil"
	"github.com/tk-425/Codefind/pkg/api"
)

type Indexer struct {
	repoPath string
	manifest *Manifest
	lspState *lsp.WarmState
}

type RunOptions struct {
	RepoID      string
	OrgID       string
	Force       bool
	Window      bool
	RetryLSP    bool
	Concurrency int
}

var warmLanguages = lsp.WarmLanguages

func New(repoPath string, manifest *Manifest) (*Indexer, error) {
	if err := validateRepoPath(repoPath); err != nil {
		return nil, fmt.Errorf("invalid repo path: %w", err)
	}
	if manifest == nil {
		manifest = &Manifest{
			SchemaVersion: ManifestSchemaVersion,
			Files:         make(map[string]ManifestFile),
		}
	}
	if manifest.SchemaVersion == 0 {
		manifest.SchemaVersion = ManifestSchemaVersion
	}
	if manifest.Files == nil {
		manifest.Files = make(map[string]ManifestFile)
	}
	return &Indexer{
		repoPath: repoPath,
		manifest: manifest,
	}, nil
}

func (i *Indexer) Discover() (*DiscoveryResult, error) {
	return DiscoverFiles(i.repoPath)
}

func (i *Indexer) DetectChanges() (*ChangeDetectionResult, error) {
	return DetectChanges(i.repoPath, i.manifest)
}

func (i *Indexer) Manifest() *Manifest {
	return i.manifest
}

func (i *Indexer) WarmLSPs() (*lsp.WarmState, error) {
	discovery, err := i.Discover()
	if err != nil {
		return nil, err
	}

	languages := make([]string, 0, len(discovery.Files))
	for _, file := range discovery.Files {
		if file.Language == "" || slices.Contains(languages, file.Language) {
			continue
		}
		languages = append(languages, file.Language)
	}

	state, err := warmLanguages(i.repoPath, languages)
	if err != nil {
		return nil, err
	}
	i.lspState = state
	return state, nil
}

func (i *Indexer) LSPState() *lsp.WarmState {
	return i.lspState
}

func (i *Indexer) Index(ctx context.Context, options RunOptions, store ChunkStore) (api.IndexResponse, error) {
	if store == nil {
		return api.IndexResponse{}, fmt.Errorf("chunk store is required")
	}
	i.manifest.RepoID = options.RepoID
	i.manifest.OrgID = options.OrgID

	currentCommit := ""
	if IsGitRepository(i.repoPath) {
		currentCommit, _ = getHeadCommit(i.repoPath)
	}

	var added []string
	var modified []string
	var deleted []string
	var retryCandidates []string

	if options.Force || i.manifest.LastCommit == "" {
		discovery, err := i.Discover()
		if err != nil {
			return api.IndexResponse{}, err
		}
		for _, file := range discovery.Files {
			added = append(added, file.Path)
		}
	} else {
		changes, err := i.DetectChanges()
		if err != nil {
			return api.IndexResponse{}, err
		}
		added = append(added, changes.Added...)
		modified = append(modified, changes.Modified...)
		deleted = append(deleted, changes.Deleted...)
		currentCommit = changes.CurrentCommit
		if options.RetryLSP {
			retryCandidates, err = i.retryLSPCandidates(append(append([]string{}, added...), modified...), deleted)
			if err != nil {
				return api.IndexResponse{}, err
			}
		}
	}

	if !options.Window && len(added)+len(modified)+len(retryCandidates) > 0 {
		_, _ = i.WarmLSPs()
	}

	tombstonedIDs := i.collectChunkIDs(append(append(append([]string{}, modified...), deleted...), retryCandidates...))
	if len(tombstonedIDs) > 0 {
		if _, err := store.UpdateChunkStatus(ctx, api.ChunkStatusUpdateRequest{
			RepoID:   options.RepoID,
			ChunkIDs: tombstonedIDs,
			Status:   "tombstoned",
		}); err != nil {
			return api.IndexResponse{}, err
		}
	}

	filesToIndex := append(append(append([]string{}, added...), modified...), retryCandidates...)
	filesToIndex = uniquePaths(filesToIndex)
	indexChunks, manifestFiles, err := i.buildChunks(filesToIndex, options, currentCommit)
	if err != nil {
		return api.IndexResponse{}, err
	}

	var response api.IndexResponse
	if len(indexChunks) > 0 {
		response, err = store.Index(ctx, api.IndexRequest{
			RepoID: options.RepoID,
			Chunks: indexChunks,
		})
		if err != nil {
			return api.IndexResponse{}, err
		}
	} else {
		response = api.IndexResponse{
			Status:       "ok",
			RepoID:       options.RepoID,
			IndexedCount: 0,
			Accepted:     true,
			Detail:       "no changed files to index",
		}
	}

	for _, path := range deleted {
		delete(i.manifest.Files, path)
	}
	maps.Copy(i.manifest.Files, manifestFiles)
	i.manifest.LastCommit = currentCommit
	if err := SaveManifest(i.manifest); err != nil {
		return api.IndexResponse{}, err
	}

	return response, nil
}

func (i *Indexer) retryLSPCandidates(changedPaths, deletedPaths []string) ([]string, error) {
	discovery, err := i.Discover()
	if err != nil {
		return nil, err
	}

	currentFiles := make(map[string]DiscoveredFile, len(discovery.Files))
	for _, file := range discovery.Files {
		currentFiles[file.Path] = file
	}

	changed := make(map[string]struct{}, len(changedPaths))
	for _, path := range changedPaths {
		changed[path] = struct{}{}
	}
	deleted := make(map[string]struct{}, len(deletedPaths))
	for _, path := range deletedPaths {
		deleted[path] = struct{}{}
	}

	candidates := make([]string, 0)
	for path, file := range i.manifest.Files {
		if _, ok := currentFiles[path]; !ok {
			continue
		}
		if _, ok := changed[path]; ok {
			continue
		}
		if _, ok := deleted[path]; ok {
			continue
		}
		if !shouldRetryLSP(file) {
			continue
		}
		candidates = append(candidates, path)
	}

	slices.Sort(candidates)
	return candidates, nil
}

func shouldRetryLSP(file ManifestFile) bool {
	if file.LastChunkingMethod != "window" {
		return false
	}
	switch file.FallbackReason {
	case "", "no_symbols", "unsupported":
		return false
	default:
		return true
	}
}

func (i *Indexer) buildChunks(files []string, options RunOptions, currentCommit string) ([]api.IndexChunk, map[string]ManifestFile, error) {
	if options.Concurrency <= 1 {
		return i.buildChunksSerial(files, options, currentCommit)
	}

	type fileBuildResult struct {
		path         string
		chunks       []api.IndexChunk
		manifestFile ManifestFile
	}

	results := make([]fileBuildResult, 0, len(files))
	resultsCh := make(chan fileBuildResult, len(files))
	errCh := make(chan error, len(files))
	semaphore := make(chan struct{}, options.Concurrency)

	var wg sync.WaitGroup
	for _, relPath := range files {
		relPath := relPath
		wg.Go(func() {
			select {
			case semaphore <- struct{}{}:
			case <-context.Background().Done():
				return
			}
			defer func() { <-semaphore }()

			fileChunks, manifestFile, err := i.buildChunkFile(relPath, options, currentCommit)
			if err != nil {
				errCh <- err
				return
			}
			resultsCh <- fileBuildResult{
				path:         relPath,
				chunks:       fileChunks,
				manifestFile: manifestFile,
			}
		})
	}

	wg.Wait()
	close(resultsCh)
	close(errCh)

	for err := range errCh {
		if err != nil {
			return nil, nil, err
		}
	}
	for result := range resultsCh {
		results = append(results, result)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].path < results[j].path
	})

	chunks := make([]api.IndexChunk, 0)
	manifestFiles := make(map[string]ManifestFile, len(results))
	for _, result := range results {
		chunks = append(chunks, result.chunks...)
		manifestFiles[result.path] = result.manifestFile
	}

	return chunks, manifestFiles, nil
}

func (i *Indexer) buildChunksSerial(files []string, options RunOptions, currentCommit string) ([]api.IndexChunk, map[string]ManifestFile, error) {
	chunks := make([]api.IndexChunk, 0)
	manifestFiles := make(map[string]ManifestFile, len(files))

	for _, relPath := range files {
		fileChunks, manifestFile, err := i.buildChunkFile(relPath, options, currentCommit)
		if err != nil {
			return nil, nil, err
		}
		chunks = append(chunks, fileChunks...)
		manifestFiles[relPath] = manifestFile
	}

	return chunks, manifestFiles, nil
}

func (i *Indexer) buildChunkFile(relPath string, options RunOptions, currentCommit string) ([]api.IndexChunk, ManifestFile, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	hybrid := chunker.NewHybridChunker(chunker.DefaultConfig(), i.repoPath, options.Window)
	fullPath := filepath.Join(i.repoPath, relPath)
	if !validatePathWithinRepo(i.repoPath, fullPath) {
		return nil, ManifestFile{}, fmt.Errorf("file path outside repo root: %s", relPath)
	}

	contentBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, ManifestFile{}, fmt.Errorf("read %s: %w", relPath, err)
	}
	content := string(contentBytes)
	result, err := hybrid.ChunkFile(content, relPath)
	if err != nil {
		return nil, ManifestFile{}, fmt.Errorf("chunk %s: %w", relPath, err)
	}

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return nil, ManifestFile{}, fmt.Errorf("stat %s: %w", relPath, err)
	}

	entry := ManifestFile{
		Path:               relPath,
		ContentHash:        chunker.GenerateContentHash(content),
		Language:           result.Language,
		SizeBytes:          fileInfo.Size(),
		LineCount:          countLines(content),
		LastIndexedCommit:  currentCommit,
		LastModTime:        fileInfo.ModTime().UTC().Format(time.RFC3339),
		LastChunkingMethod: result.Method,
		FallbackReason:     result.FallbackReason,
		ChunkingVersion:    "1",
		LastIndexedAt:      now,
		ChunkIDs:           make([]string, 0, len(result.Chunks)),
	}

	chunks := make([]api.IndexChunk, 0, len(result.Chunks))
	for _, symbolChunk := range result.Chunks {
		chunkID := chunker.GenerateChunkID(options.RepoID, relPath, symbolChunk.StartLine, symbolChunk.EndLine, symbolChunk.Content)
		entry.ChunkIDs = append(entry.ChunkIDs, chunkID)
		chunks = append(chunks, api.IndexChunk{
			ID:      chunkID,
			Content: symbolChunk.Content,
			Metadata: api.ChunkMetadata{
				RepoID:         options.RepoID,
				Path:           relPath,
				Language:       result.Language,
				StartLine:      symbolChunk.StartLine,
				EndLine:        symbolChunk.EndLine,
				ContentHash:    chunker.GenerateContentHash(symbolChunk.Content),
				Status:         "active",
				SymbolName:     symbolChunk.SymbolName,
				SymbolKind:     symbolChunk.SymbolKind,
				ParentName:     symbolChunk.ParentName,
				IndexedAt:      now,
				ChunkingMethod: result.Method,
				FallbackReason: result.FallbackReason,
			},
		})
	}

	return chunks, entry, nil
}

func (i *Indexer) collectChunkIDs(paths []string) []string {
	ids := make([]string, 0)
	for _, path := range paths {
		file, ok := i.manifest.Files[path]
		if !ok {
			continue
		}
		for _, id := range file.ChunkIDs {
			if !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func validatePathWithinRepo(repoPath, target string) bool {
	return pathutil.IsWithinDir(repoPath, target)
}

func countLines(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}

func uniquePaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(paths))
	unique := make([]string, 0, len(paths))
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		unique = append(unique, path)
	}
	return unique
}
