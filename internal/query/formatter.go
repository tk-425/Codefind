package query

import (
	"fmt"
	"strings"
	"time"

	"github.com/tk-425/Codefind/pkg/api"
)

// FormatResults formats query results for display
func FormatResults(resp *api.QueryResponse) string {
	if len(resp.Results) == 0 {
		return "No results found"
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d results (page %d/%d, %d per page)\n\n", resp.TotalCount, resp.Page, (resp.TotalCount+resp.PageSize-1)/resp.PageSize, resp.PageSize))

	for i, result := range resp.Results {
		idx := (resp.Page-1)*resp.PageSize + i + 1
		output.WriteString(formatResult(idx, result))
		output.WriteString("\n")
	}

	return output.String()
}

// formatResult formats a single result
func formatResult(index int, result api.QueryResult) string {
	var sb strings.Builder

	// Header with index, file, and similarity
	similarity := int(result.Distance * 100)
	sb.WriteString(fmt.Sprintf("[%d] %s:%d-%d (similarity: %d%%)\n", index,
		result.Metadata.FilePath,
		result.Metadata.StartLine,
		result.Metadata.EndLine,
		similarity))

	// Metadata line (includes project info for multi-project support)
	meta := result.Metadata
	sb.WriteString(fmt.Sprintf("    [%s] %s | %s\n",
		meta.RepoID[:8],
		meta.ProjectName,
		meta.Language))

	// Content preview (truncated)
	contentPreview := strings.TrimSpace(result.Content)
	if len(contentPreview) > 200 {
		contentPreview = contentPreview[:200] + "..."
	}
	contentPreview = strings.ReplaceAll(contentPreview, "\n", "\n    ")
	sb.WriteString(fmt.Sprintf("    %s\n", contentPreview))

	// Metadata timestamp
	sb.WriteString(fmt.Sprintf("    indexed: %s\n",
		formatTime(meta.IndexedAt)))

	return sb.String()
}

// formatTime formats a timestamp for display
func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		return fmt.Sprintf("%d min ago", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(diff.Hours()))
	}
	if diff < 30*24*time.Hour {
		return fmt.Sprintf("%d days ago", int(diff.Hours()/24))
	}

	return t.Format("2006-01-02")
}

// FormatResultsJSON formats as JSON (for scripting)
func FormatResultsJSON(resp *api.QueryResponse) string {
	// User can pipe output to jq if they want structured format
	// For now, keep human-readable default
	return FormatResults(resp)
}
