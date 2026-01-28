package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GlobalConfig represents the global configuration
type GlobalConfig struct {
	ServerURL string `json:"server_url"`
	Editor    string `json:"editor"`
	AuthKey   string `json:"auth_key,omitempty"`
}

// RepositoryManifest represents per-repository metadata
type RepositoryManifest struct {
	RepoID            string              `json:"repo_id"`
	ProjectName       string              `json:"project_name"`
	RepoPath          string              `json:"repo_path"`
	LastIndexedCommit string              `json:"last_indexed_commit,omitempty"`
	IndexedAt         string              `json:"indexed_at,omitempty"`
	IndexedFiles      map[string]FileInfo `json:"indexed_files"`
	ActiveChunkCount  int                 `json:"active_chunk_count"`
	DeletedChunkCount int                 `json:"deleted_chunk_count"`
}

// FileInfo tracks metadata for indexed files
type FileInfo struct {
	Language    string `json:"language"`
	LineCount   int    `json:"line_count"`
	ChunkCount  int    `json:"chunk_count"`
	LastModTime string `json:"last_mod_time"`
	ContentHash string `json:"content_hash"`
}

// ConfigPath returns the path to global config file
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codefind", "config.json"), nil
}

// ManifestPath returns the path to a repository manifest
func ManifestPath(repoID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	manifestDir := filepath.Join(home, ".codefind", "manifests")
	return filepath.Join(manifestDir, repoID+".json"), nil
}

// LoadGlobalConfig reads the global configuration
func LoadGlobalConfig() (*GlobalConfig, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at %s", configPath)
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg GlobalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// SaveGlobalConfig writes the global configuration
func SaveGlobalConfig(cfg *GlobalConfig) error {
	configPath, err := ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// LoadManifest reads a repository manifest
func LoadManifest(repoID string) (*RepositoryManifest, error) {
	manifestPath, err := ManifestPath(repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest path: %w", err)
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("manifest not found for repo %s", repoID)
		}
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest RepositoryManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// SaveManifest writes a repository manifest
func SaveManifest(manifest *RepositoryManifest) error {
	manifestPath, err := ManifestPath(manifest.RepoID)
	if err != nil {
		return fmt.Errorf("failed to get manifest path: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// ValidateGlobalConfig checks if configuration is valid
func ValidateGlobalConfig(cfg *GlobalConfig) error {
	if cfg.ServerURL == "" {
		return fmt.Errorf("server_url is required")
	}
	if cfg.Editor == "" {
		return fmt.Errorf("editor is required")
	}
	return nil
}
// test incremental indexing
// comment
