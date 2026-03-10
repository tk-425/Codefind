package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/tk-425/Codefind/internal/client"
	"github.com/tk-425/Codefind/pkg/api"
)

type ChunkStore interface {
	Index(context.Context, api.IndexRequest) (api.IndexResponse, error)
	UpdateChunkStatus(context.Context, api.ChunkStatusUpdateRequest) (api.ChunkStatusUpdateResponse, error)
}

type ClientStore struct {
	client *client.Client
}

func NewClientStore(client *client.Client) *ClientStore {
	return &ClientStore{client: client}
}

func (s *ClientStore) Index(ctx context.Context, request api.IndexRequest) (api.IndexResponse, error) {
	return s.client.Index(ctx, request)
}

func (s *ClientStore) UpdateChunkStatus(ctx context.Context, request api.ChunkStatusUpdateRequest) (api.ChunkStatusUpdateResponse, error) {
	return s.client.UpdateChunkStatus(ctx, request)
}

var manifestSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func ManifestPath(orgID, repoID string) (string, error) {
	if err := validateManifestSegment("org_id", orgID); err != nil {
		return "", err
	}
	if err := validateManifestSegment("repo_id", repoID); err != nil {
		return "", err
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(homeDir, ".codefind", "manifests", orgID, repoID+".json"), nil
}

func LoadManifest(orgID, repoID string) (*Manifest, error) {
	path, err := ManifestPath(orgID, repoID)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultManifest(orgID, repoID), nil
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}

	if manifest.SchemaVersion != ManifestSchemaVersion {
		return defaultManifest(orgID, repoID), nil
	}
	if manifest.Files == nil {
		manifest.Files = make(map[string]ManifestFile)
	}
	if manifest.RepoID == "" {
		manifest.RepoID = repoID
	}
	if manifest.OrgID == "" {
		manifest.OrgID = orgID
	}
	return &manifest, nil
}

func SaveManifest(manifest *Manifest) error {
	if manifest == nil {
		return fmt.Errorf("manifest is required")
	}
	path, err := ManifestPath(manifest.OrgID, manifest.RepoID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create manifest dir: %w", err)
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	body = append(body, '\n')
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

func defaultManifest(orgID, repoID string) *Manifest {
	return &Manifest{
		SchemaVersion: ManifestSchemaVersion,
		RepoID:        repoID,
		OrgID:         orgID,
		Files:         make(map[string]ManifestFile),
	}
}

func validateManifestSegment(label, value string) error {
	if !manifestSegmentPattern.MatchString(value) {
		return fmt.Errorf("%s must match [A-Za-z0-9_-]+", label)
	}
	return nil
}
