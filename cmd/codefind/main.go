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

	"github.com/spf13/cobra"
	"github.com/tk-425/Codefind/internal/authflow"
	"github.com/tk-425/Codefind/internal/client"
	"github.com/tk-425/Codefind/internal/config"
	"github.com/tk-425/Codefind/internal/keychain"
	"github.com/tk-425/Codefind/internal/query"
	"github.com/tk-425/Codefind/internal/stats"
	"github.com/tk-425/Codefind/pkg/api"
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

func main() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	var configPath string

	rootCmd := &cobra.Command{
		Use:   "codefind",
		Short: "Code-Find v2 CLI",
		Long: strings.TrimSpace(`Code-Find v2 CLI foundation.

Use 'codefind auth', 'codefind org', 'codefind admin', 'codefind list',
'codefind stats', and 'codefind query' against the configured server.
Build with 'go build -o ./bin/codefind ./cmd/codefind' and install
globally with the documented /usr/local/bin flow.`),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		if configPath != "" {
			return nil
		}
		defaultPath, err := defaultPathResolver()
		if err != nil {
			return err
		}
		configPath = defaultPath
		return nil
	}
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "override config file path")

	rootCmd.AddCommand(newConfigCommand(&configPath))
	rootCmd.AddCommand(newHealthCommand(&configPath))
	rootCmd.AddCommand(newAuthCommand(&configPath))
	rootCmd.AddCommand(newOrgCommand(&configPath))
	rootCmd.AddCommand(newAdminCommand(&configPath))
	rootCmd.AddCommand(newListCommand(&configPath))
	rootCmd.AddCommand(newStatsCommand(&configPath))
	rootCmd.AddCommand(newQueryCommand(&configPath))
	rootCmd.AddCommand(newTokenizeCommand(&configPath))

	return rootCmd
}

func newConfigCommand(configPath *string) *cobra.Command {
	var (
		show        bool
		serverURL   string
		webAppURL   string
		activeOrgID string
		editor      string
	)

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Show or update CLI configuration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			hasUpdates := strings.TrimSpace(serverURL) != "" || strings.TrimSpace(webAppURL) != "" || strings.TrimSpace(activeOrgID) != "" || strings.TrimSpace(editor) != ""
			if show {
				if hasUpdates {
					return errors.New("--show cannot be combined with update flags")
				}
				return runConfigShow(cmd.OutOrStdout(), *configPath)
			}
			if !hasUpdates {
				return cmd.Help()
			}
			return runConfigUpdate(cmd.OutOrStdout(), *configPath, serverURL, webAppURL, activeOrgID, editor)
		},
	}

	configCmd.Flags().BoolVar(&show, "show", false, "show current config")
	configCmd.Flags().StringVar(&serverURL, "server-url", "", "set the server URL")
	configCmd.Flags().StringVar(&webAppURL, "web-app-url", "", "set the local web app URL")
	configCmd.Flags().StringVar(&activeOrgID, "active-org-id", "", "set the active org ID")
	configCmd.Flags().StringVar(&editor, "editor", "", "set the preferred editor")

	return configCmd
}

func newHealthCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check the configured server health endpoint",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadRequiredConfig(*configPath)
			if err != nil {
				return err
			}
			if cfg.ServerURL == "" {
				return errors.New("server_url is not configured; run 'codefind config --server-url <url>'")
			}

			apiClient, err := newAPIClient(cfg.ServerURL, defaultTokenManager())
			if err != nil {
				return err
			}

			response, err := apiClient.Health(context.Background())
			if err != nil {
				return err
			}

			return writeJSON(cmd.OutOrStdout(), response)
		},
	}
}

func newAuthCommand(configPath *string) *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage CLI authentication",
	}

	authCmd.AddCommand(newAuthLoginCommand(configPath))
	authCmd.AddCommand(newAuthLogoutCommand(configPath))
	authCmd.AddCommand(newAuthStatusCommand(configPath))

	return authCmd
}

func newOrgCommand(configPath *string) *cobra.Command {
	orgCmd := &cobra.Command{
		Use:   "org",
		Short: "Inspect organization access for the current token",
	}
	orgCmd.AddCommand(newOrgListCommand(configPath))
	return orgCmd
}

func newOrgListCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List organizations available to the authenticated user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath)
			if err != nil {
				return err
			}

			response, err := apiClient.GetOrganizations(cmd.Context())
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), response)
		},
	}
}

func newAdminCommand(configPath *string) *cobra.Command {
	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "Manage organization members and invitations for the current token org",
	}
	adminCmd.AddCommand(newAdminListCommand(configPath))
	adminCmd.AddCommand(newAdminInviteCommand(configPath))
	adminCmd.AddCommand(newAdminRevokeInviteCommand(configPath))
	adminCmd.AddCommand(newAdminRemoveCommand(configPath))
	return adminCmd
}

func newAdminListCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List members and invitations for the active token organization",
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath)
			if err != nil {
				return err
			}

			members, err := apiClient.GetAdminMembers(cmd.Context())
			if err != nil {
				return err
			}
			invitations, err := apiClient.GetAdminInvitations(cmd.Context())
			if err != nil {
				return err
			}

			return writeJSON(cmd.OutOrStdout(), map[string]any{
				"members":     members,
				"invitations": invitations,
			})
		},
	}
}

func newAdminInviteCommand(configPath *string) *cobra.Command {
	var (
		email string
		role  string
	)

	command := &cobra.Command{
		Use:   "invite",
		Short: "Invite a user into the current token organization",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(email) == "" {
				return errors.New("--email is required")
			}
			if role != "org:admin" && role != "org:member" {
				return errors.New("--role must be org:admin or org:member")
			}

			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath)
			if err != nil {
				return err
			}

			response, err := apiClient.CreateAdminInvitation(cmd.Context(), api.CreateOrganizationInvitationRequest{
				EmailAddress: email,
				Role:         role,
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), response)
		},
	}
	command.Flags().StringVar(&email, "email", "", "email address to invite")
	command.Flags().StringVar(&role, "role", "org:member", "organization role for the invitee")
	return command
}

func newAdminRevokeInviteCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "revoke-invite <invitation-id>",
		Short: "Revoke a pending invitation in the current token organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath)
			if err != nil {
				return err
			}

			response, err := apiClient.RevokeAdminInvitation(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), response)
		},
	}
}

func newAdminRemoveCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <user-id>",
		Short: "Remove a member from the current token organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath)
			if err != nil {
				return err
			}

			response, err := apiClient.RemoveAdminMember(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), response)
		},
	}
}

func newListCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List indexed repos available to the current organization",
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath)
			if err != nil {
				return err
			}
			response, err := apiClient.GetCollections(cmd.Context())
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), response)
		},
	}
}

func newStatsCommand(configPath *string) *cobra.Command {
	var options stats.Options

	command := &cobra.Command{
		Use:   "stats",
		Short: "Show chunk stats for a repo or the whole active organization",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if options.All && strings.TrimSpace(options.RepoID) != "" {
				return errors.New("--all cannot be combined with --repo-id")
			}

			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath)
			if err != nil {
				return err
			}

			repoID := ""
			if !options.All {
				repoID = strings.TrimSpace(options.RepoID)
			}
			response, err := apiClient.GetStats(cmd.Context(), repoID)
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), response)
		},
	}

	command.Flags().StringVar(&options.RepoID, "repo-id", "", "repo identifier to inspect")
	command.Flags().BoolVar(&options.All, "all", false, "show org-wide stats")
	return command
}

func newQueryCommand(configPath *string) *cobra.Command {
	var options query.Options

	command := &cobra.Command{
		Use:   "query <text>",
		Short: "Search the current organization or a specific repo",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if options.All && strings.TrimSpace(options.RepoID) != "" {
				return errors.New("--all cannot be combined with --repo-id")
			}

			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath)
			if err != nil {
				return err
			}

			payload := api.QueryRequest{
				QueryText: args[0],
				Project:   strings.TrimSpace(options.Project),
				Language:  strings.TrimSpace(options.Language),
				Page:      options.Page,
				PageSize:  options.PageSize,
				TopK:      options.TopK,
			}
			if !options.All {
				payload.RepoID = strings.TrimSpace(options.RepoID)
			}

			response, err := apiClient.Query(cmd.Context(), payload)
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), response)
		},
	}

	command.Flags().StringVar(&options.RepoID, "repo-id", "", "repo identifier to search")
	command.Flags().StringVar(&options.Project, "project", "", "exact project filter")
	command.Flags().StringVar(&options.Language, "lang", "", "exact language filter")
	command.Flags().BoolVar(&options.All, "all", false, "search all repos in the current org")
	command.Flags().IntVar(&options.Page, "page", 1, "1-based result page")
	command.Flags().IntVar(&options.PageSize, "page-size", 10, "results per page")
	command.Flags().IntVar(&options.TopK, "top-k", 10, "per-collection match limit before pagination")
	return command
}

func newTokenizeCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "tokenize <text>",
		Short: "Tokenize text using the server tokenizer",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath)
			if err != nil {
				return err
			}
			response, err := apiClient.Tokenize(cmd.Context(), api.TokenizeRequest{Text: args[0]})
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), response)
		},
	}
}

func newAuthLoginCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Open the browser and authenticate with Clerk",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return browserLoginRunner(cmd.Context(), cmd.OutOrStdout(), *configPath)
		},
	}
}

func newAuthLogoutCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Delete the stored CLI token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager := defaultTokenManager()
			err := manager.DeleteToken()
			if err != nil && !errors.Is(err, keychain.ErrNotFound) {
				return err
			}

			cfg, loadErr := config.LoadOrDefault(*configPath)
			if loadErr != nil {
				return loadErr
			}
			cfg.ActiveOrgID = ""
			if saveErr := config.Save(*configPath, cfg); saveErr != nil {
				return saveErr
			}

			if errors.Is(err, keychain.ErrNotFound) {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "no stored token was present")
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "stored token deleted")
			return err
		},
	}
}

func newAuthStatusCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show CLI authentication status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.LoadOrDefault(*configPath)
			if err != nil {
				return err
			}

			status := map[string]any{
				"authenticated": false,
				"active_org_id": cfg.DisplayMap()["active_org_id"],
			}

			token, err := defaultTokenManager().LoadToken()
			if err != nil {
				if errors.Is(err, keychain.ErrNotFound) {
					return writeJSON(cmd.OutOrStdout(), status)
				}
				return err
			}

			status["authenticated"] = true
			if claims, err := authflow.DecodeTokenClaims(token); err == nil {
				if claims.OrgID != "" {
					status["token_org_id"] = claims.OrgID
				}
			}
			if expiry, err := authflow.TokenExpiryTime(token); err == nil {
				status["expires_at"] = expiry.Format(time.RFC3339)
				status["expired"] = time.Now().UTC().After(expiry)
			}

			return writeJSON(cmd.OutOrStdout(), status)
		},
	}
}

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

func loadAuthenticatedClient(ctx context.Context, stdout io.Writer, path string) (*client.Client, error) {
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
		if _, writeErr := fmt.Fprintln(stdout, "stored token expired; renewing via browser session..."); writeErr != nil {
			return nil, writeErr
		}
		if err := browserLoginRunner(ctx, stdout, path); err != nil {
			return nil, err
		}
	}

	return newAPIClient(cfg.ServerURL, defaultTokenManager())
}

func runBrowserLogin(ctx context.Context, stdout io.Writer, configPath string) error {
	cfg, err := loadRequiredConfig(configPath)
	if err != nil {
		return err
	}
	if cfg.ServerURL == "" {
		return errors.New("server_url is not configured; run 'codefind config --server-url <url>'")
	}
	webAppURL := cfg.WebAppURL
	if webAppURL == "" {
		webAppURL = "http://localhost:5173"
	}

	listener, err := newCallbackListener()
	if err != nil {
		return err
	}
	defer listener.Close()

	timeoutCtx, cancel := context.WithTimeout(ctx, authflow.LoginTimeout())
	defer cancel()

	redirectURI, waitForToken, err := startCallbackServer(timeoutCtx, listener, webAppURL)
	if err != nil {
		return err
	}

	signInURL, err := buildSignInURL(cfg.ServerURL, redirectURI)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(stdout, "opening browser: %s\n", signInURL); err != nil {
		return err
	}
	if err := openBrowser(signInURL); err != nil {
		return fmt.Errorf("open browser manually with %s: %w", signInURL, err)
	}

	token, err := waitForToken()
	if err != nil {
		return err
	}

	manager := defaultTokenManager()
	if err := manager.SaveToken(token); err != nil {
		return err
	}

	if claims, err := authflow.DecodeTokenClaims(token); err == nil {
		cfg.ActiveOrgID = claims.OrgID
	} else {
		cfg.ActiveOrgID = ""
	}
	if err := config.Save(configPath, cfg); err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout, "authentication stored in keychain")
	return err
}

func writeJSON(stdout io.Writer, value any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
