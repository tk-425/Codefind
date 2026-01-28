package query

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/tk-425/Codefind/pkg/api"
)

// Color definitions for consistent styling
var (
	indexColor    = color.New(color.FgCyan, color.Bold)
	fileColor     = color.New(color.FgWhite)
	similarityColor = color.New(color.FgYellow)
	repoColor     = color.New(color.FgHiBlack)
	langColor     = color.New(color.FgMagenta)
	contentColor  = color.New(color.FgHiBlack)
	timestampColor = color.New(color.FgHiBlack, color.Italic)
	headerColor   = color.New(color.FgGreen, color.Bold)
	errorColor    = color.New(color.FgRed)
)

// FormatResults formats query results for display with colors
func FormatResults(resp *api.QueryResponse) string {
	if len(resp.Results) == 0 {
		return "No results found"
	}

	var output strings.Builder
	
	// Header with result count and pagination
	header := fmt.Sprintf("Found %d results (page %d/%d, %d per page)\n\n",
		resp.TotalCount,
		resp.Page,
		(resp.TotalCount+resp.PageSize-1)/resp.PageSize,
		resp.PageSize)
	output.WriteString(headerColor.Sprint(header))

	for i, result := range resp.Results {
		idx := (resp.Page-1)*resp.PageSize + i + 1
		output.WriteString(formatResult(idx, result))
		output.WriteString("\n")
	}

	return output.String()
}

// formatResult formats a single result with colors
func formatResult(index int, result api.QueryResult) string {
	var sb strings.Builder

	// Line 1: [index] file:lines (similarity: X%)
	similarity := int(result.Distance * 100)
	sb.WriteString(indexColor.Sprintf("[%d] ", index))
	sb.WriteString(fileColor.Sprintf("%s:%d-%d ",
		result.Metadata.FilePath,
		result.Metadata.StartLine,
		result.Metadata.EndLine))
	sb.WriteString(similarityColor.Sprintf("(similarity: %d%%)\n", similarity))

	// Line 2: [repo_id] project_name | language
	meta := result.Metadata
	repoIDShort := meta.RepoID
	if len(repoIDShort) > 8 {
		repoIDShort = repoIDShort[:8]
	}
	sb.WriteString("    ")
	sb.WriteString(repoColor.Sprintf("[%s] ", repoIDShort))
	sb.WriteString(fmt.Sprintf("%s | ", meta.ProjectName))
	sb.WriteString(langColor.Sprintf("%s", meta.Language))
	
	// Show [DELETED] prefix for deleted chunks (tombstone mode)
	if meta.Status == "deleted" {
		deletedColor := color.New(color.FgRed, color.Bold)
		deletedAt := ""
		if meta.DeletedAt != nil {
			deletedAt = meta.DeletedAt.Format("2006-01-02")
		}
		sb.WriteString(deletedColor.Sprintf(" [DELETED %s]", deletedAt))
	}
	sb.WriteString("\n")

	// Lines 3+: Content preview (first 4 lines)
	preview := getContentPreview(result.Content, 4)
	for _, line := range preview {
		sb.WriteString("    ")
		sb.WriteString(contentColor.Sprint(line))
		sb.WriteString("\n")
	}

	// Last line: indexed timestamp
	sb.WriteString("    ")
	sb.WriteString(timestampColor.Sprintf("indexed: %s\n", formatTime(meta.IndexedAt)))

	return sb.String()
}

// getContentPreview returns the first N lines of content
func getContentPreview(content string, maxLines int) []string {
	lines := strings.Split(content, "\n")
	
	// Trim empty lines from start
	startIdx := 0
	for startIdx < len(lines) && strings.TrimSpace(lines[startIdx]) == "" {
		startIdx++
	}
	
	if startIdx >= len(lines) {
		return []string{"(empty)"}
	}
	
	lines = lines[startIdx:]
	
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, "...")
	}
	
	// Truncate long lines
	for i, line := range lines {
		if len(line) > 100 {
			lines[i] = line[:100] + "..."
		}
	}
	
	return lines
}

// formatTime formats a timestamp as relative time
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

// FormatResultsPlain formats results without colors (for piping/scripting)
func FormatResultsPlain(resp *api.QueryResponse) string {
	// Disable colors temporarily
	color.NoColor = true
	result := FormatResults(resp)
	color.NoColor = false
	return result
}

// FormatError formats an error message with color
func FormatError(msg string) string {
	return errorColor.Sprint(msg)
}
