package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	var payload api.HealthResponse
	if err := c.doJSON(ctx, http.MethodGet, "/health", nil, http.StatusOK, &payload); err != nil {
		return api.HealthResponse{}, err
	}
	return payload, nil
}

func (c *Client) GetCollections(ctx context.Context) (api.CollectionListResponse, error) {
	var payload api.CollectionListResponse
	if err := c.doJSON(ctx, http.MethodGet, "/collections", nil, http.StatusOK, &payload); err != nil {
		return api.CollectionListResponse{}, err
	}
	return payload, nil
}

func (c *Client) GetStats(ctx context.Context, repoID string) (api.StatsResponse, error) {
	var payload api.StatsResponse
	requestPath := "/stats"
	if strings.TrimSpace(repoID) != "" {
		requestPath += "?repo_id=" + url.QueryEscape(repoID)
	}
	if err := c.doJSON(ctx, http.MethodGet, requestPath, nil, http.StatusOK, &payload); err != nil {
		return api.StatsResponse{}, err
	}
	return payload, nil
}

func (c *Client) Query(ctx context.Context, request api.QueryRequest) (api.QueryResponse, error) {
	var payload api.QueryResponse
	if err := c.doJSON(ctx, http.MethodPost, "/query", request, http.StatusOK, &payload); err != nil {
		return api.QueryResponse{}, err
	}
	return payload, nil
}

func (c *Client) Tokenize(ctx context.Context, request api.TokenizeRequest) (api.TokenizeResponse, error) {
	var payload api.TokenizeResponse
	if err := c.doJSON(ctx, http.MethodPost, "/tokenize", request, http.StatusOK, &payload); err != nil {
		return api.TokenizeResponse{}, err
	}
	return payload, nil
}

func (c *Client) Index(ctx context.Context, request api.IndexRequest) (api.IndexResponse, error) {
	var payload api.IndexResponse
	if err := c.doJSON(ctx, http.MethodPost, "/index", request, http.StatusAccepted, &payload); err != nil {
		return api.IndexResponse{}, err
	}
	return payload, nil
}

func (c *Client) UpdateChunkStatus(
	ctx context.Context,
	request api.ChunkStatusUpdateRequest,
) (api.ChunkStatusUpdateResponse, error) {
	var payload api.ChunkStatusUpdateResponse
	if err := c.doJSON(ctx, http.MethodPatch, "/chunks/status", request, http.StatusOK, &payload); err != nil {
		return api.ChunkStatusUpdateResponse{}, err
	}
	return payload, nil
}

func (c *Client) PurgeChunks(ctx context.Context, request api.ChunkPurgeRequest) (api.ChunkPurgeResponse, error) {
	var payload api.ChunkPurgeResponse
	if err := c.doJSON(ctx, http.MethodDelete, "/chunks/purge", request, http.StatusOK, &payload); err != nil {
		return api.ChunkPurgeResponse{}, err
	}
	return payload, nil
}

func (c *Client) GetOrganizations(ctx context.Context) (api.OrgListResponse, error) {
	var payload api.OrgListResponse
	if err := c.doJSON(ctx, http.MethodGet, "/orgs", nil, http.StatusOK, &payload); err != nil {
		return api.OrgListResponse{}, err
	}
	return payload, nil
}

func (c *Client) GetAdminMembers(ctx context.Context) (api.OrganizationMemberListResponse, error) {
	var payload api.OrganizationMemberListResponse
	if err := c.doJSON(ctx, http.MethodGet, "/admin/members", nil, http.StatusOK, &payload); err != nil {
		return api.OrganizationMemberListResponse{}, err
	}
	return payload, nil
}

func (c *Client) GetAdminInvitations(ctx context.Context) (api.OrganizationInvitationListResponse, error) {
	var payload api.OrganizationInvitationListResponse
	if err := c.doJSON(ctx, http.MethodGet, "/admin/invitations", nil, http.StatusOK, &payload); err != nil {
		return api.OrganizationInvitationListResponse{}, err
	}
	return payload, nil
}

func (c *Client) CreateAdminInvitation(
	ctx context.Context,
	request api.CreateOrganizationInvitationRequest,
) (api.OrganizationInvitation, error) {
	var payload api.OrganizationInvitation
	if err := c.doJSON(ctx, http.MethodPost, "/admin/invite", request, http.StatusCreated, &payload); err != nil {
		return api.OrganizationInvitation{}, err
	}
	return payload, nil
}

func (c *Client) RevokeAdminInvitation(ctx context.Context, invitationID string) (api.OrganizationInvitation, error) {
	var payload api.OrganizationInvitation
	if err := c.doJSON(ctx, http.MethodPost, "/admin/invitations/"+invitationID+"/revoke", nil, http.StatusOK, &payload); err != nil {
		return api.OrganizationInvitation{}, err
	}
	return payload, nil
}

func (c *Client) RemoveAdminMember(ctx context.Context, userID string) (api.OrganizationMember, error) {
	var payload api.OrganizationMember
	if err := c.doJSON(ctx, http.MethodDelete, "/admin/members/"+userID, nil, http.StatusOK, &payload); err != nil {
		return api.OrganizationMember{}, err
	}
	return payload, nil
}

func (c *Client) newRequest(ctx context.Context, method, requestPath string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+requestPath, body)
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

func (c *Client) doJSON(
	ctx context.Context,
	method string,
	requestPath string,
	requestBody any,
	expectedStatus int,
	out any,
) error {
	var body io.Reader
	if requestBody != nil {
		encoded, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	req, err := c.newRequest(ctx, method, requestPath, body)
	if err != nil {
		return err
	}
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s: %w", method, requestPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("%s %s failed: %s", method, requestPath, resp.Status)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s response: %w", requestPath, err)
	}
	return nil
}
