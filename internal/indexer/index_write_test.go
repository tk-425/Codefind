package indexer

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tk-425/Codefind/pkg/api"
)

type fakeChunkStore struct {
	indexRequests        []api.IndexRequest
	updateStatusRequests []api.ChunkStatusUpdateRequest
}

func (f *fakeChunkStore) Index(_ context.Context, request api.IndexRequest) (api.IndexResponse, error) {
	f.indexRequests = append(f.indexRequests, request)
	return api.IndexResponse{
		Status:       "ok",
		RepoID:       request.RepoID,
		IndexedCount: len(request.Chunks),
		Accepted:     true,
	}, nil
}

func (f *fakeChunkStore) UpdateChunkStatus(_ context.Context, request api.ChunkStatusUpdateRequest) (api.ChunkStatusUpdateResponse, error) {
	f.updateStatusRequests = append(f.updateStatusRequests, request)
	return api.ChunkStatusUpdateResponse{
		Status:       "ok",
		RepoID:       request.RepoID,
		UpdatedCount: len(request.ChunkIDs),
	}, nil
}

func TestIndexerIndexIndexesChangedFilesAndPersistsManifest(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	repoDir := t.TempDir()
	sourcePath := filepath.Join(repoDir, "main.go")
	if err := os.WriteFile(sourcePath, []byte("package main\n\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manifest := &Manifest{
		SchemaVersion: ManifestSchemaVersion,
		RepoID:        "repo-a",
		OrgID:         "org_123",
		LastCommit:    "baseline",
		Files: map[string]ManifestFile{
			"main.go": {
				Path:               "main.go",
				ContentHash:        "old-hash",
				LastModTime:        "2000-01-01T00:00:00Z",
				LastChunkingMethod: "window",
				ChunkIDs:           []string{"chunk-old-1", "chunk-old-2"},
			},
		},
	}

	indexer, err := New(repoDir, manifest)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	store := &fakeChunkStore{}
	response, err := indexer.Index(context.Background(), RunOptions{
		RepoID:      "repo-a",
		OrgID:       "org_123",
		Window:      true,
		Concurrency: 1,
	}, store)
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	if response.IndexedCount == 0 {
		t.Fatalf("Index() = %#v, want indexed chunks", response)
	}
	if len(store.updateStatusRequests) != 1 {
		t.Fatalf("UpdateChunkStatus calls = %d, want 1", len(store.updateStatusRequests))
	}
	if got := store.updateStatusRequests[0].ChunkIDs; len(got) != 2 {
		t.Fatalf("tombstoned chunk ids = %#v, want prior chunk ids", got)
	}
	if len(store.indexRequests) != 1 || len(store.indexRequests[0].Chunks) == 0 {
		t.Fatalf("Index requests = %#v, want chunk payloads", store.indexRequests)
	}

	manifestPath, err := ManifestPath("org_123", "repo-a")
	if err != nil {
		t.Fatalf("ManifestPath() error = %v", err)
	}
	persisted, err := LoadManifest("org_123", "repo-a")
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if persisted.Files["main.go"].ContentHash == "old-hash" {
		t.Fatalf("persisted manifest file was not updated: %#v", persisted.Files["main.go"])
	}
	if persisted.Files["main.go"].LastIndexMode != IndexModeForceWindow {
		t.Fatalf("persisted LastIndexMode = %q, want %q", persisted.Files["main.go"].LastIndexMode, IndexModeForceWindow)
	}
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest file missing: %v", err)
	}
}

func TestIndexerIndexRetryLSPReindexesUnchangedDegradedFiles(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	repoDir := t.TempDir()
	sourcePath := filepath.Join(repoDir, "main.go")
	content := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(sourcePath, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	manifest := &Manifest{
		SchemaVersion: ManifestSchemaVersion,
		RepoID:        "repo-a",
		OrgID:         "org_123",
		LastCommit:    "baseline",
		Files: map[string]ManifestFile{
			"main.go": {
				Path:               "main.go",
				ContentHash:        "same-hash",
				LastModTime:        info.ModTime().UTC().Format(time.RFC3339),
				LastIndexMode:      IndexModeHybrid,
				LastChunkingMethod: "window",
				FallbackReason:     "timeout",
				ChunkIDs:           []string{"chunk-old-1"},
			},
		},
	}

	indexer, err := New(repoDir, manifest)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	store := &fakeChunkStore{}
	response, err := indexer.Index(context.Background(), RunOptions{
		RepoID:      "repo-a",
		OrgID:       "org_123",
		RetryLSP:    true,
		Concurrency: 1,
	}, store)
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	if response.IndexedCount == 0 {
		t.Fatalf("Index() = %#v, want retried chunks", response)
	}
	if len(store.updateStatusRequests) != 1 {
		t.Fatalf("UpdateChunkStatus calls = %d, want 1", len(store.updateStatusRequests))
	}
	if got := store.updateStatusRequests[0].ChunkIDs; len(got) != 1 || got[0] != "chunk-old-1" {
		t.Fatalf("tombstoned chunk ids = %#v, want prior degraded chunk id", got)
	}
	if len(store.indexRequests) != 1 || len(store.indexRequests[0].Chunks) == 0 {
		t.Fatalf("Index requests = %#v, want retried chunk payloads", store.indexRequests)
	}
}

func TestIndexerIndexRetryLSPSkipsBenignWindowFallbacks(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	repoDir := t.TempDir()
	sourcePath := filepath.Join(repoDir, "tiny.go")
	content := []byte("package main\n")
	if err := os.WriteFile(sourcePath, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	manifest := &Manifest{
		SchemaVersion: ManifestSchemaVersion,
		RepoID:        "repo-a",
		OrgID:         "org_123",
		LastCommit:    "baseline",
		Files: map[string]ManifestFile{
			"tiny.go": {
				Path:               "tiny.go",
				ContentHash:        "same-hash",
				LastModTime:        info.ModTime().UTC().Format(time.RFC3339),
				LastIndexMode:      IndexModeHybrid,
				LastChunkingMethod: "window",
				FallbackReason:     "no_symbols",
				ChunkIDs:           []string{"chunk-old-1"},
			},
		},
	}

	indexer, err := New(repoDir, manifest)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	store := &fakeChunkStore{}
	response, err := indexer.Index(context.Background(), RunOptions{
		RepoID:      "repo-a",
		OrgID:       "org_123",
		RetryLSP:    true,
		Concurrency: 1,
	}, store)
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	if response.IndexedCount != 0 || response.Detail != "no changed files to index" {
		t.Fatalf("Index() = %#v, want no retry work for benign fallback", response)
	}
	if len(store.updateStatusRequests) != 0 {
		t.Fatalf("UpdateChunkStatus calls = %#v, want none", store.updateStatusRequests)
	}
	if len(store.indexRequests) != 0 {
		t.Fatalf("Index requests = %#v, want none", store.indexRequests)
	}
}

func TestIndexerIndexRetryLSPSkipsExplicitForceWindowRuns(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	repoDir := t.TempDir()
	sourcePath := filepath.Join(repoDir, "main.go")
	content := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(sourcePath, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	manifest := &Manifest{
		SchemaVersion: ManifestSchemaVersion,
		RepoID:        "repo-a",
		OrgID:         "org_123",
		LastCommit:    "baseline",
		Files: map[string]ManifestFile{
			"main.go": {
				Path:               "main.go",
				ContentHash:        "same-hash",
				LastModTime:        info.ModTime().UTC().Format(time.RFC3339),
				LastIndexMode:      IndexModeForceWindow,
				LastChunkingMethod: "window",
				ChunkIDs:           []string{"chunk-old-1"},
			},
		},
	}

	indexer, err := New(repoDir, manifest)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	store := &fakeChunkStore{}
	response, err := indexer.Index(context.Background(), RunOptions{
		RepoID:      "repo-a",
		OrgID:       "org_123",
		RetryLSP:    true,
		Concurrency: 1,
	}, store)
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	if response.IndexedCount != 0 || response.Detail != "no changed files to index" {
		t.Fatalf("Index() = %#v, want no retry work for prior force-window indexing", response)
	}
	if len(store.updateStatusRequests) != 0 {
		t.Fatalf("UpdateChunkStatus calls = %#v, want none", store.updateStatusRequests)
	}
	if len(store.indexRequests) != 0 {
		t.Fatalf("Index requests = %#v, want none", store.indexRequests)
	}
}

func TestIndexerIndexForceRebuildTombstonesPreviouslyIndexedFiles(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	repoDir := t.TempDir()
	sourcePath := filepath.Join(repoDir, "main.go")
	if err := os.WriteFile(sourcePath, []byte("package main\n\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manifest := &Manifest{
		SchemaVersion: ManifestSchemaVersion,
		RepoID:        "repo-a",
		OrgID:         "org_123",
		LastCommit:    "baseline",
		Files: map[string]ManifestFile{
			"main.go": {
				Path:               "main.go",
				ContentHash:        "old-hash",
				LastIndexMode:      IndexModeForceWindow,
				LastChunkingMethod: "window",
				ChunkIDs:           []string{"chunk-old-1", "chunk-old-2"},
			},
		},
	}

	indexer, err := New(repoDir, manifest)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	store := &fakeChunkStore{}
	response, err := indexer.Index(context.Background(), RunOptions{
		RepoID:      "repo-a",
		OrgID:       "org_123",
		Force:       true,
		Concurrency: 1,
	}, store)
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	if response.IndexedCount == 0 {
		t.Fatalf("Index() = %#v, want indexed chunks", response)
	}
	if len(store.updateStatusRequests) != 1 {
		t.Fatalf("UpdateChunkStatus calls = %d, want 1", len(store.updateStatusRequests))
	}
	got := append([]string(nil), store.updateStatusRequests[0].ChunkIDs...)
	sort.Strings(got)
	if strings.Join(got, ",") != "chunk-old-1,chunk-old-2" {
		t.Fatalf("tombstoned chunk ids = %#v, want prior chunk ids", got)
	}
	if len(store.indexRequests) != 1 || len(store.indexRequests[0].Chunks) == 0 {
		t.Fatalf("Index requests = %#v, want chunk payloads", store.indexRequests)
	}

	persisted, err := LoadManifest("org_123", "repo-a")
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if persisted.Files["main.go"].LastIndexMode != IndexModeHybrid {
		t.Fatalf("persisted LastIndexMode = %q, want %q", persisted.Files["main.go"].LastIndexMode, IndexModeHybrid)
	}
}

func TestIndexerIndexSupportsDeterministicParallelChunkBuilds(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "a.go"), []byte("package main\n\nfunc a() {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(a.go) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "b.go"), []byte("package main\n\nfunc b() {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(b.go) error = %v", err)
	}

	indexer, err := New(repoDir, &Manifest{
		SchemaVersion: ManifestSchemaVersion,
		RepoID:        "repo-a",
		OrgID:         "org_123",
		Files:         map[string]ManifestFile{},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	store := &fakeChunkStore{}
	response, err := indexer.Index(context.Background(), RunOptions{
		RepoID:      "repo-a",
		OrgID:       "org_123",
		Force:       true,
		Window:      true,
		Concurrency: 2,
	}, store)
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}
	if response.IndexedCount == 0 {
		t.Fatalf("Index() = %#v, want indexed chunks", response)
	}
	if len(store.indexRequests) != 1 {
		t.Fatalf("Index requests = %d, want 1", len(store.indexRequests))
	}

	paths := make([]string, 0, len(store.indexRequests[0].Chunks))
	for _, chunk := range store.indexRequests[0].Chunks {
		paths = append(paths, chunk.Metadata.Path)
	}
	if !sort.StringsAreSorted(paths) {
		t.Fatalf("chunk paths should be deterministic and sorted, got %#v", paths)
	}
}

func TestIndexerIndexSplitsLargeChunkPayloadsIntoMultipleSendBatches(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv(indexSendBatchSizeEnvVar, "8")

	repoDir := t.TempDir()
	var content strings.Builder
	content.WriteString("package main\n\n")
	for i := range 5000 {
		content.WriteString("func f")
		content.WriteString(strconv.Itoa(i))
		content.WriteString("() {}\n\n")
	}
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte(content.String()), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	indexer, err := New(repoDir, &Manifest{
		SchemaVersion: ManifestSchemaVersion,
		RepoID:        "repo-a",
		OrgID:         "org_123",
		Files:         map[string]ManifestFile{},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	store := &fakeChunkStore{}
	response, err := indexer.Index(context.Background(), RunOptions{
		RepoID:      "repo-a",
		OrgID:       "org_123",
		Force:       true,
		Window:      false,
		Concurrency: 1,
	}, store)
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}
	if response.IndexedCount == 0 {
		t.Fatalf("Index() = %#v, want indexed chunks", response)
	}
	if len(store.indexRequests) < 2 {
		t.Fatalf("Index requests = %d, want multiple batches", len(store.indexRequests))
	}
	batchSize, err := indexSendBatchSizeFromEnv()
	if err != nil {
		t.Fatalf("indexSendBatchSizeFromEnv() error = %v", err)
	}
	totalChunks := 0
	for _, request := range store.indexRequests {
		if len(request.Chunks) > batchSize {
			t.Fatalf("batch size = %d, want <= %d", len(request.Chunks), batchSize)
		}
		totalChunks += len(request.Chunks)
	}
	if totalChunks != response.IndexedCount {
		t.Fatalf("total sent chunks = %d, response indexed_count = %d", totalChunks, response.IndexedCount)
	}
}

func TestIndexSendBatchSizeFromEnvRejectsInvalidValue(t *testing.T) {
	t.Setenv(indexSendBatchSizeEnvVar, "invalid")

	_, err := indexSendBatchSizeFromEnv()
	if err == nil {
		t.Fatal("indexSendBatchSizeFromEnv() error = nil, want validation failure")
	}
	if !strings.Contains(err.Error(), indexSendBatchSizeEnvVar) {
		t.Fatalf("error = %v", err)
	}
}
