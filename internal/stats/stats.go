package stats

import (
	"fmt"
	"strings"
	"time"

	"github.com/tk-425/Codefind/internal/client"
	"github.com/tk-425/Codefind/pkg/api"
)

// StatsClient handles stats operations
type StatsClient struct {
	apiClient *client.APIClient
}

// NewStatsClient creates a new stats client
func NewStatsClient(apiClient *client.APIClient) *StatsClient {
	return &StatsClient{
		apiClient: apiClient,
	}
}

// GetStats retrieves statistics for a project
func (s *StatsClient) GetStats(projectID string) (*api.StatsResponse, error) {
	resp, err := s.apiClient.GetStats(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	return resp, nil
}

// FormatStats formats stats for display
func FormatStats(projectName string, stats *api.StatsResponse) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Project: %s\n", projectName))
	sb.WriteString("┌─────────────────┬────────┐\n")
	sb.WriteString(fmt.Sprintf("│ Active Chunks   │ %6d │\n", stats.ActiveChunks))
	sb.WriteString(fmt.Sprintf("│ Deleted Chunks  │ %6d │\n", stats.DeletedChunks))
	sb.WriteString(fmt.Sprintf("│ Total Chunks    │ %6d │\n", stats.TotalChunks))
	sb.WriteString(fmt.Sprintf("│ Storage Overhead│ %5.1f%% │\n", stats.OverheadPercent))
	sb.WriteString("└─────────────────┴────────┘")

	// Add cleanup suggestion if overhead > 15%
	if stats.OverheadPercent > 15 {
		sb.WriteString("\n\n⚠️  Consider running: codefind cleanup --older-than=30")
	}

	return sb.String()
}

// FormatOldestDeleted formats the oldest deleted date
func formatOldestDeleted(t time.Time) string {
	if t.IsZero() {
		return "none"
	}
	return t.Format("2006-01-02")
}
