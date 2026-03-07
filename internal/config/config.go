package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tk-425/Codefind/internal/pathutil"
)

const (
	configDirName  = ".codefind"
	configFileName = "config.json"
)

type Config struct {
	ServerURL   string `json:"server_url,omitempty"`
	ActiveOrgID string `json:"active_org_id,omitempty"`
	Editor      string `json:"editor,omitempty"`
}

func DefaultPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(homeDir, configDirName, configFileName), nil
}

func Load(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %s: %w", path, err)
	}

	return cfg.Normalize()
}

func LoadOrDefault(path string) (Config, error) {
	cfg, err := Load(path)
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	return Config{}, err
}

func Save(path string, cfg Config) error {
	normalized, err := cfg.Normalize()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir %s: %w", dir, err)
	}

	body, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	body = append(body, '\n')

	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

func (c Config) Normalize() (Config, error) {
	normalized := Config{
		ServerURL:   strings.TrimSpace(c.ServerURL),
		ActiveOrgID: strings.TrimSpace(c.ActiveOrgID),
		Editor:      strings.TrimSpace(c.Editor),
	}

	if normalized.ServerURL != "" {
		serverURL, err := pathutil.NormalizeServerURL(normalized.ServerURL)
		if err != nil {
			return Config{}, err
		}
		normalized.ServerURL = serverURL
	}

	return normalized, nil
}

func (c Config) DisplayMap() map[string]string {
	serverURL := c.ServerURL
	if serverURL == "" {
		serverURL = "<unset>"
	}

	activeOrg := c.ActiveOrgID
	if activeOrg == "" {
		activeOrg = "<unset>"
	}

	editor := c.Editor
	if editor == "" {
		editor = "<unset>"
	}

	return map[string]string{
		"active_org_id": activeOrg,
		"editor":        editor,
		"server_url":    serverURL,
	}
}
