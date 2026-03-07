package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tk-425/Codefind/internal/keychain"
	"github.com/tk-425/Codefind/internal/pathutil"
	"github.com/tk-425/Codefind/pkg/api"
)

const defaultTimeout = 5 * time.Second

type TokenLoader interface {
	LoadToken() (string, error)
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	tokenStore TokenLoader
}

func New(baseURL string, tokenStore TokenLoader) (*Client, error) {
	return NewWithHTTPClient(baseURL, tokenStore, &http.Client{Timeout: defaultTimeout})
}

func NewWithHTTPClient(baseURL string, tokenStore TokenLoader, httpClient *http.Client) (*Client, error) {
	normalizedURL, err := pathutil.NormalizeServerURL(baseURL)
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	return &Client{
		baseURL:    normalizedURL,
		httpClient: httpClient,
		tokenStore: tokenStore,
	}, nil
}

func (c *Client) Health(ctx context.Context) (api.HealthResponse, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/health")
	if err != nil {
		return api.HealthResponse{}, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return api.HealthResponse{}, fmt.Errorf("request /health: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return api.HealthResponse{}, fmt.Errorf("health request failed: %s", resp.Status)
	}

	var payload api.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return api.HealthResponse{}, fmt.Errorf("decode health response: %w", err)
	}

	return payload, nil
}

func (c *Client) newRequest(ctx context.Context, method, requestPath string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+requestPath, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	if c.tokenStore == nil {
		return req, nil
	}

	token, err := c.tokenStore.LoadToken()
	if err != nil {
		if err == keychain.ErrNotFound {
			return req, nil
		}
		return nil, err
	}
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return req, nil
}
