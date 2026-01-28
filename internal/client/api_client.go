package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tk-425/Codefind/pkg/api"
)

// APIClient communicates with the Codefind server
type APIClient struct {
	baseURL string
	client  *http.Client
	authKey string
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 5 * time.Minute, // 5 minutes for batch embedding and storage
		},
	}
}

// SetAuthKey sets the authentication key for protected endpoints
func (ac *APIClient) SetAuthKey(authKey string) {
	ac.authKey = authKey
}

// Tokenize sends texts to server for tokenization
func (ac *APIClient) Tokenize(model string, texts []string) ([]int, error) {
	req := api.TokenizeRequest{
		Model: model,
		Input: texts,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := ac.post("/tokenize", data, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp api.TokenizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("server error: %s", tokenResp.Error)
	}

	return tokenResp.Tokens, nil
}

// Health checks server health
func (ac *APIClient) Health() (*api.HealthResponse, error) {
	resp, err := ac.get("/health")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var healthResp api.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &healthResp, nil
}

// Index sends chunks to the server for embedding and storage
func (ac *APIClient) Index(req api.IndexRequest) (*api.IndexResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := ac.post("/index", data, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var indexResp api.IndexResponse
	if err := json.NewDecoder(resp.Body).Decode(&indexResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if indexResp.Error != "" {
		return &indexResp, fmt.Errorf("server error: %s", indexResp.Error)
	}

	return &indexResp, nil
}

// Query sends a search query to the server
func (ac *APIClient) Query(req api.QueryRequest) (*api.QueryResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := ac.post("/query", data, false) // false = no auth required
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var queryResp api.QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if queryResp.Error != "" {
		return &queryResp, fmt.Errorf("server error: %s", queryResp.Error)
	}

	return &queryResp, nil
}

// UpdateChunkStatus marks chunks as deleted or active (tombstone mode)
func (ac *APIClient) UpdateChunkStatus(req api.ChunkStatusRequest) (*api.ChunkStatusResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := ac.patch("/chunks/status", data, true) // true = requires auth
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var statusResp api.ChunkStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if statusResp.Error != "" {
		return &statusResp, fmt.Errorf("server error: %s", statusResp.Error)
	}

	return &statusResp, nil
}

// PurgeChunks permanently removes deleted chunks older than cutoff date
func (ac *APIClient) PurgeChunks(req api.PurgeRequest) (*api.PurgeResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := ac.deleteWithBody("/chunks/purge", data, true) // true = requires auth
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var purgeResp api.PurgeResponse
	if err := json.NewDecoder(resp.Body).Decode(&purgeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if purgeResp.Error != "" {
		return &purgeResp, fmt.Errorf("server error: %s", purgeResp.Error)
	}

	return &purgeResp, nil
}

// GetStats retrieves statistics for a collection
func (ac *APIClient) GetStats(collection string) (*api.StatsResponse, error) {
	resp, err := ac.get("/stats/" + collection)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var statsResp api.StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&statsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if statsResp.Error != "" {
		return &statsResp, fmt.Errorf("server error: %s", statsResp.Error)
	}

	return &statsResp, nil
}

// post sends a POST request
func (ac *APIClient) post(endpoint string, data []byte, requireAuth bool) (*http.Response, error) {
	url := ac.baseURL + endpoint

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if requireAuth && ac.authKey != "" {
		req.Header.Set("X-Auth-Key", ac.authKey)
	}

	resp, err := ac.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// patch sends a PATCH request
func (ac *APIClient) patch(endpoint string, data []byte, requireAuth bool) (*http.Response, error) {
	url := ac.baseURL + endpoint

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if requireAuth && ac.authKey != "" {
		req.Header.Set("X-Auth-Key", ac.authKey)
	}

	resp, err := ac.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// get sends a GET request
func (ac *APIClient) get(endpoint string) (*http.Response, error) {
	url := ac.baseURL + endpoint

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ac.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// delete sends a DELETE request
func (ac *APIClient) delete(endpoint string) (*http.Response, error) {
	url := ac.baseURL + endpoint

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if ac.authKey != "" {
		req.Header.Set("X-Auth-Key", ac.authKey)
	}

	resp, err := ac.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// deleteWithBody sends a DELETE request with JSON body
func (ac *APIClient) deleteWithBody(endpoint string, data []byte, requireAuth bool) (*http.Response, error) {
	url := ac.baseURL + endpoint

	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if requireAuth && ac.authKey != "" {
		req.Header.Set("X-Auth-Key", ac.authKey)
	}

	resp, err := ac.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// ClearCollection deletes all chunks in a collection
func (ac *APIClient) ClearCollection(repoID string) error {
	resp, err := ac.delete("/clear/" + repoID)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
