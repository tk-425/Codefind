package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadConfig(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), ".codefind", "config.json")
	input := Config{
		ServerURL:   "http://127.0.0.1:8080",
		WebAppURL:   "http://localhost:5173",
		ActiveOrgID: "org_123",
		Editor:      "nvim",
	}

	if err := Save(configPath, input); err != nil {
		t.Fatalf("save config: %v", err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("config permissions = %o, want 600", got)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if loaded != input {
		t.Fatalf("loaded config = %#v, want %#v", loaded, input)
	}
}

func TestLoadOrDefaultMissingFile(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "missing.json")
	cfg, err := LoadOrDefault(configPath)
	if err != nil {
		t.Fatalf("load or default: %v", err)
	}
	if cfg != (Config{}) {
		t.Fatalf("config = %#v, want zero value", cfg)
	}
}

func TestDisplayMapIncludesWebAppURL(t *testing.T) {
	t.Parallel()

	display := Config{}.DisplayMap()
	if display["web_app_url"] != "<unset>" {
		t.Fatalf("web_app_url display = %q, want <unset>", display["web_app_url"])
	}
}
