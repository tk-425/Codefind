package cleanup

import (
	"fmt"
	"strings"
	"time"

	"github.com/tk-425/Codefind/internal/client"
	"github.com/tk-425/Codefind/pkg/api"
)

// CleanupOptions contains cleanup command options
type CleanupOptions struct {
	DryRun    bool   // Preview without removing
	OlderThan int    // Days since deletion (0 = all deleted)
	Project   string // Specific project or empty for current
	ListOnly  bool   // Just list deleted chunks
}

// CleanupResult contains cleanup operation results
type CleanupResult struct {
	ChunksFound   int                  `json:"chunks_found"`
	ChunksRemoved int                  `json:"chunks_removed"`
	BytesFreed    int64                `json:"bytes_freed"`
	Files         []api.DeletedFileInfo `json:"files,omitempty"`
	DryRun        bool                 `json:"dry_run"`
}

// CleanupClient handles cleanup operations
type CleanupClient struct {
	apiClient *client.APIClient
}

// NewCleanupClient creates a new cleanup client
func NewCleanupClient(apiClient *client.APIClient) *CleanupClient {
	return &CleanupClient{
		apiClient: apiClient,
	}
}

// Cleanup performs cleanup of deleted chunks
func (c *CleanupClient) Cleanup(opts CleanupOptions) (*CleanupResult, error) {
	// Calculate cutoff date
	cutoffDate := time.Now().AddDate(0, 0, -opts.OlderThan)

	// Call purge endpoint
	req := api.PurgeRequest{
		Collection:  opts.Project,
		OlderThan:   opts.OlderThan,
		CutoffDate:  cutoffDate.Format(time.RFC3339),
		DryRun:      opts.DryRun || opts.ListOnly,
	}

	resp, err := c.apiClient.PurgeChunks(req)
	if err != nil {
		return nil, fmt.Errorf("cleanup failed: %w", err)
	}

	return &CleanupResult{
		ChunksFound:   resp.ChunksFound,
		ChunksRemoved: resp.ChunksRemoved,
		BytesFreed:    resp.BytesFreed,
		Files:         resp.Files,
		DryRun:        opts.DryRun || opts.ListOnly,
	}, nil
}

// FormatResult formats cleanup result for display
func FormatResult(result *CleanupResult, listOnly bool) string {
	var sb strings.Builder

	if listOnly {
		sb.WriteString(fmt.Sprintf("Found %d deleted chunks\n", result.ChunksFound))
		// Show per-file details
		for _, f := range result.Files {
			deletedDate := formatDeletedDate(f.DeletedAt)
			sb.WriteString(fmt.Sprintf("  - %s (%d chunks, deleted %s)\n", f.FilePath, f.ChunkCount, deletedDate))
		}
		return strings.TrimSpace(sb.String())
	}

	if result.DryRun {
		sb.WriteString(fmt.Sprintf("Would remove %d chunks (dry-run)\n", result.ChunksFound))
		// Show per-file details
		for _, f := range result.Files {
			deletedDate := formatDeletedDate(f.DeletedAt)
			sb.WriteString(fmt.Sprintf("  - %s (%d chunks, deleted %s)\n", f.FilePath, f.ChunkCount, deletedDate))
		}
		return strings.TrimSpace(sb.String())
	}

	return fmt.Sprintf("✅ Removed %d chunks", result.ChunksRemoved)
}

// formatDeletedDate extracts just the date from an RFC3339 timestamp
func formatDeletedDate(deletedAt string) string {
	t, err := time.Parse(time.RFC3339, deletedAt)
	if err != nil {
		return "unknown"
	}
	return t.Format("2006-01-02")
}
