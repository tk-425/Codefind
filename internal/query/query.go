package query

import (
	"github.com/tk-425/Codefind/internal/client"
	"github.com/tk-425/Codefind/pkg/api"
)

// QueryClient wraps query operations
type QueryClient struct {
	apiClient *client.APIClient
}

// NewQueryClient creates a new query client
func NewQueryClient(apiClient *client.APIClient) *QueryClient {
	return &QueryClient{
		apiClient: apiClient,
	}
}

// Search performs a semantic search query with optional filters
func (qc *QueryClient) Search(query string, topK int, languages []string, pathPrefix string, excludePath string) (*api.QueryResponse, error) {
	req := api.QueryRequest{
		Query:       query,
		TopK:        topK,
		Languages:   languages,
		PathPrefix:  pathPrefix,
		ExcludePath: excludePath,
	}
	return qc.apiClient.Query(req)
}

// SearchProject performs a query limited to a specific project
func (qc *QueryClient) SearchProject(query string, projectID string, topK int) (*api.QueryResponse, error) {
	req := api.QueryRequest{
		Query:      query,
		TopK:       topK,
		Collection: projectID,
	}

	return qc.apiClient.Query(req)
}
