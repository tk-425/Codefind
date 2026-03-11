package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tk-425/Codefind/internal/authflow"
	"github.com/tk-425/Codefind/internal/client"
	"github.com/tk-425/Codefind/internal/config"
	"github.com/tk-425/Codefind/internal/indexer"
	"github.com/tk-425/Codefind/internal/keychain"
)

var (
	newAPIClient        = client.New
	defaultTokenManager = keychain.DefaultManager
	defaultPathResolver = config.DefaultPath
	browserLoginRunner  = runBrowserLogin
	newCallbackListener = authflow.NewLocalCallbackListener
	startCallbackServer = authflow.StartCallbackServer
	buildSignInURL      = authflow.BuildSignInURL
	openBrowser         = authflow.DefaultBrowserOpener
)

func runConfigShow(stdout io.Writer, path string) error {
	cfg, err := config.LoadOrDefault(path)
	if err != nil {
		return err
	}
	return writeJSON(stdout, cfg.DisplayMap())
}

func runConfigUpdate(stdout io.Writer, path, serverURL, webAppURL, activeOrgID, editor string) error {
	cfg, err := config.LoadOrDefault(path)
	if err != nil {
		return err
	}

	if strings.TrimSpace(serverURL) != "" {
		cfg.ServerURL = serverURL
	}
	if strings.TrimSpace(webAppURL) != "" {
		cfg.WebAppURL = webAppURL
	}
	if strings.TrimSpace(activeOrgID) != "" {
		cfg.ActiveOrgID = activeOrgID
	}
	if strings.TrimSpace(editor) != "" {
		cfg.Editor = editor
	}

	if err := config.Save(path, cfg); err != nil {
		return err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	_, err = fmt.Fprintf(stdout, "config saved to %s\n", absPath)
	return err
}

func loadRequiredConfig(path string) (config.Config, error) {
	cfg, err := config.Load(path)
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return config.Config{}, fmt.Errorf("config file not found at %s; run 'codefind config --server-url <url>'", path)
	}
	return config.Config{}, err
}

func resolveInitializedProject(cfg config.Config, repoPath string) (*indexer.Manifest, string, error) {
	if strings.TrimSpace(cfg.ActiveOrgID) == "" {
		return nil, "", errors.New("active_org_id is not configured; run 'codefind auth login'")
	}

	resolvedRepoPath := strings.TrimSpace(repoPath)
	if resolvedRepoPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, "", fmt.Errorf("resolve current directory: %w", err)
		}
		resolvedRepoPath = cwd
	}

	absRepoPath, err := filepath.Abs(resolvedRepoPath)
	if err != nil {
		return nil, "", fmt.Errorf("resolve repo path: %w", err)
	}

	manifest, err := indexer.LoadInitializedManifestForPath(cfg.ActiveOrgID, absRepoPath)
	if err != nil {
		if errors.Is(err, indexer.ErrProjectNotInitialized) {
			return nil, absRepoPath, fmt.Errorf("project is not initialized; run 'codefind init --repo-path %s' first", absRepoPath)
		}
		return nil, absRepoPath, err
	}

	return manifest, manifest.RepoPath, nil
}

func resolveScopedRepoID(manifest *indexer.Manifest, repoID string) (string, error) {
	resolvedRepoID := strings.TrimSpace(repoID)
	if resolvedRepoID == "" {
		return manifest.RepoID, nil
	}
	if resolvedRepoID != manifest.RepoID {
		return "", fmt.Errorf("current directory is initialized as repo %s; omit --repo-id or use --repo-id %s", manifest.RepoID, manifest.RepoID)
	}
	return resolvedRepoID, nil
}

func loadAuthenticatedClient(ctx context.Context, stdout io.Writer, path string, quiet bool) (*client.Client, error) {
	cfg, err := loadRequiredConfig(path)
	if err != nil {
		return nil, err
	}
	if cfg.ServerURL == "" {
		return nil, errors.New("server_url is not configured; run 'codefind config --server-url <url>'")
	}

	token, err := defaultTokenManager().LoadToken()
	if err != nil {
		if errors.Is(err, keychain.ErrNotFound) {
			return nil, errors.New("not authenticated; run 'codefind auth login'")
		}
		return nil, err
	}
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("stored token is empty; run 'codefind auth login'")
	}
	if expiry, err := authflow.TokenExpiryTime(token); err == nil && time.Now().UTC().After(expiry) {
		if !quiet {
			if _, writeErr := fmt.Fprintln(stdout, "stored token expired; renewing via browser session..."); writeErr != nil {
				return nil, writeErr
			}
		}
		if err := browserLoginRunner(ctx, stdout, path); err != nil {
			return nil, err
		}
	}

	return newAPIClient(cfg.ServerURL, defaultTokenManager())
}

func requireAdminClient(ctx context.Context, stdout io.Writer, path string) (*client.Client, error) {
	apiClient, err := loadAuthenticatedClient(ctx, stdout, path, false)
	if err != nil {
		return nil, err
	}

	token, err := defaultTokenManager().LoadToken()
	if err != nil {
		return nil, err
	}
	claims, err := authflow.DecodeTokenClaims(token)
	if err != nil {
		return nil, fmt.Errorf("decode stored token claims: %w", err)
	}
	if claims.OrgRole != "org:admin" {
		return nil, errors.New("org:admin role required for this command")
	}
	return apiClient, nil
}

func writeJSON(stdout io.Writer, value any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
