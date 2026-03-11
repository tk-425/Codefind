package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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

var ErrProjectNotInitialized = errors.New("project is not initialized")

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

func manifestDir(orgID string) (string, error) {
	if err := validateManifestSegment("org_id", orgID); err != nil {
		return "", err
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(homeDir, ".codefind", "manifests", orgID), nil
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
	if manifest.RepoPath != "" {
		manifest.RepoPath = filepath.Clean(manifest.RepoPath)
	}
	return &manifest, nil
}

func LoadInitializedManifest(orgID, repoID, repoPath string) (*Manifest, error) {
	manifest, err := LoadManifest(orgID, repoID)
	if err != nil {
		return nil, err
	}
	if manifest.InitializedAt == "" || manifest.RepoPath == "" {
		return nil, ErrProjectNotInitialized
	}

	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve repo path: %w", err)
	}
	if filepath.Clean(manifest.RepoPath) != filepath.Clean(absRepoPath) {
		return nil, fmt.Errorf("project is initialized for %s; run 'codefind init --repo-path %s' to reinitialize", manifest.RepoPath, absRepoPath)
	}
	return manifest, nil
}

func LoadInitializedManifestForPath(orgID, repoPath string) (*Manifest, error) {
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve repo path: %w", err)
	}

	dir, err := manifestDir(orgID)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrProjectNotInitialized
		}
		return nil, fmt.Errorf("read manifest dir: %w", err)
	}

	var matched *Manifest
	longestPath := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		repoID := strings.TrimSuffix(entry.Name(), ".json")
		manifest, err := LoadManifest(orgID, repoID)
		if err != nil {
			continue
		}
		if manifest.InitializedAt == "" || manifest.RepoPath == "" {
			continue
		}
		cleanManifestPath := filepath.Clean(manifest.RepoPath)
		if absRepoPath != cleanManifestPath && !strings.HasPrefix(absRepoPath, cleanManifestPath+string(os.PathSeparator)) {
			continue
		}
		if len(cleanManifestPath) > longestPath {
			matched = manifest
			longestPath = len(cleanManifestPath)
		}
	}

	if matched == nil {
		return nil, ErrProjectNotInitialized
	}
	return matched, nil
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

// ResetManifest removes the local manifest for the given org/repo pair so the
// project must be re-initialized before future indexing can proceed.
func ResetManifest(orgID, repoID string) error {
	path, err := ManifestPath(orgID, repoID)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove manifest: %w", err)
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

func InitManifest(repoPath, orgID, repoID string, now time.Time) (*Manifest, bool, error) {
	if err := ValidateProjectRoot(repoPath); err != nil {
		return nil, false, err
	}
	if err := validateManifestSegment("org_id", orgID); err != nil {
		return nil, false, err
	}
	if err := validateManifestSegment("repo_id", repoID); err != nil {
		return nil, false, err
	}

	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, false, fmt.Errorf("resolve repo path: %w", err)
	}

	manifest, err := LoadManifest(orgID, repoID)
	if err != nil {
		return nil, false, err
	}
	alreadyInitialized := manifest.InitializedAt != "" && filepath.Clean(manifest.RepoPath) == filepath.Clean(absRepoPath)

	manifest.SchemaVersion = ManifestSchemaVersion
	manifest.RepoID = repoID
	manifest.OrgID = orgID
	manifest.RepoPath = filepath.Clean(absRepoPath)
	if manifest.InitializedAt == "" {
		manifest.InitializedAt = now.UTC().Format(time.RFC3339)
	}
	if manifest.Files == nil {
		manifest.Files = make(map[string]ManifestFile)
	}

	if err := SaveManifest(manifest); err != nil {
		return nil, false, err
	}
	return manifest, alreadyInitialized, nil
}

func validateManifestSegment(label, value string) error {
	if !manifestSegmentPattern.MatchString(value) {
		return fmt.Errorf("%s must match [A-Za-z0-9_-]+", label)
	}
	return nil
}
