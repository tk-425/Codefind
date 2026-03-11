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
	"github.com/tk-425/Codefind/internal/indexer"
	"github.com/tk-425/Codefind/internal/keychain"
	"github.com/tk-425/Codefind/internal/lsp"
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
	rootCmd.AddCommand(newInitCommand(&configPath))
	rootCmd.AddCommand(newIndexCommand(&configPath))
	rootCmd.AddCommand(newCleanupCommand(&configPath))
	rootCmd.AddCommand(newLSPCommand(&configPath))

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
	var jsonOutput bool

	command := &cobra.Command{
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

			return writeCommandOutput(cmd.OutOrStdout(), jsonOutput, response, func(stdout io.Writer) error {
				return writeHealthOutput(stdout, response)
			})
		},
	}

	addJSONFlag(command, &jsonOutput)
	return command
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
	var jsonOutput bool

	command := &cobra.Command{
		Use:   "list",
		Short: "List organizations available to the authenticated user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath, jsonOutput)
			if err != nil {
				return err
			}

			response, err := apiClient.GetOrganizations(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd.OutOrStdout(), jsonOutput, response, func(stdout io.Writer) error {
				return writeOrgListOutput(stdout, response)
			})
		},
	}

	addJSONFlag(command, &jsonOutput)
	return command
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
	var jsonOutput bool

	command := &cobra.Command{
		Use:   "list",
		Short: "List members and invitations for the active token organization",
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath, jsonOutput)
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
			payload := map[string]any{
				"members":     members,
				"invitations": invitations,
			}
			return writeCommandOutput(cmd.OutOrStdout(), jsonOutput, payload, func(stdout io.Writer) error {
				return writeAdminListOutput(stdout, members, invitations)
			})
		},
	}

	addJSONFlag(command, &jsonOutput)
	return command
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

			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath, false)
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
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath, false)
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
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath, false)
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
	var jsonOutput bool

	command := &cobra.Command{
		Use:   "list",
		Short: "List indexed repos available to the current organization",
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath, jsonOutput)
			if err != nil {
				return err
			}
			response, err := apiClient.GetCollections(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd.OutOrStdout(), jsonOutput, response, func(stdout io.Writer) error {
				return writeCollectionListOutput(stdout, response)
			})
		},
	}

	addJSONFlag(command, &jsonOutput)
	return command
}

func newStatsCommand(configPath *string) *cobra.Command {
	var options stats.Options
	var jsonOutput bool

	command := &cobra.Command{
		Use:   "stats",
		Short: "Show chunk stats for the current initialized repo",
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath, jsonOutput)
			if err != nil {
				return err
			}
			cfg, err := loadRequiredConfig(*configPath)
			if err != nil {
				return err
			}
			manifest, _, err := resolveInitializedProject(cfg, "")
			if err != nil {
				return err
			}

			repoID, err := resolveScopedRepoID(manifest, options.RepoID)
			if err != nil {
				return err
			}
			response, err := apiClient.GetStats(cmd.Context(), repoID)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd.OutOrStdout(), jsonOutput, response, func(stdout io.Writer) error {
				return writeStatsOutput(stdout, response)
			})
		},
	}

	command.Flags().StringVar(&options.RepoID, "repo-id", "", "repo identifier to inspect")
	addJSONFlag(command, &jsonOutput)
	return command
}

func newQueryCommand(configPath *string) *cobra.Command {
	var options query.Options
	var jsonOutput bool

	command := &cobra.Command{
		Use:   "query <text>",
		Short: "Search the current organization or a specific repo",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if options.All && strings.TrimSpace(options.RepoID) != "" {
				return errors.New("--all cannot be combined with --repo-id")
			}

			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath, jsonOutput)
			if err != nil {
				return err
			}
			cfg, err := loadRequiredConfig(*configPath)
			if err != nil {
				return err
			}
			manifest, _, err := resolveInitializedProject(cfg, "")
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
				payload.RepoID, err = resolveScopedRepoID(manifest, options.RepoID)
				if err != nil {
					return err
				}
			}

			response, err := apiClient.Query(cmd.Context(), payload)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd.OutOrStdout(), jsonOutput, response, func(stdout io.Writer) error {
				return writeQueryOutput(stdout, response)
			})
		},
	}

	command.Flags().StringVar(&options.RepoID, "repo-id", "", "repo identifier to search")
	command.Flags().StringVar(&options.Project, "project", "", "exact project filter")
	command.Flags().StringVar(&options.Language, "lang", "", "exact language filter")
	command.Flags().BoolVar(&options.All, "all", false, "search all repos in the current org")
	command.Flags().IntVar(&options.Page, "page", 1, "1-based result page")
	command.Flags().IntVar(&options.PageSize, "page-size", 10, "results per page")
	command.Flags().IntVar(&options.TopK, "top-k", 10, "per-collection match limit before pagination")
	addJSONFlag(command, &jsonOutput)
	return command
}

func newTokenizeCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "tokenize <text>",
		Short: "Tokenize text using the server tokenizer",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := requireAdminClient(cmd.Context(), cmd.OutOrStdout(), *configPath)
			if err != nil {
				return err
			}
			cfg, err := loadRequiredConfig(*configPath)
			if err != nil {
				return err
			}
			if _, _, err := resolveInitializedProject(cfg, ""); err != nil {
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

func newInitCommand(configPath *string) *cobra.Command {
	var (
		repoID   string
		repoPath string
		verbose  bool
	)

	command := &cobra.Command{
		Use:   "init",
		Short: "Initialize the current project for future indexing",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := requireAdminClient(cmd.Context(), cmd.OutOrStdout(), *configPath); err != nil {
				return err
			}

			cfg, err := loadRequiredConfig(*configPath)
			if err != nil {
				return err
			}
			if strings.TrimSpace(cfg.ActiveOrgID) == "" {
				return errors.New("active_org_id is not configured; run 'codefind auth login'")
			}

			resolvedRepoPath := strings.TrimSpace(repoPath)
			if resolvedRepoPath == "" {
				resolvedRepoPath, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("resolve current directory: %w", err)
				}
			}
			resolvedRepoPath, err = filepath.Abs(resolvedRepoPath)
			if err != nil {
				return fmt.Errorf("resolve repo path: %w", err)
			}
			if err := indexer.ValidateProjectRoot(resolvedRepoPath); err != nil {
				return fmt.Errorf("invalid project root: %w", err)
			}

			resolvedRepoID := strings.TrimSpace(repoID)
			if resolvedRepoID == "" {
				resolvedRepoID, err = indexer.DeriveRepoID(resolvedRepoPath)
				if err != nil {
					return fmt.Errorf("derive repo_id: %w", err)
				}
			} else if _, err := indexer.ManifestPath(cfg.ActiveOrgID, resolvedRepoID); err != nil {
				return err
			}

			manifest, alreadyInitialized, err := indexer.InitManifest(resolvedRepoPath, cfg.ActiveOrgID, resolvedRepoID, time.Now().UTC())
			if err != nil {
				return err
			}
			message := "project initialized"
			if alreadyInitialized {
				message = "project already initialized"
			}

			response := map[string]any{
				"status":              "ok",
				"message":             message,
				"initialized":         true,
				"already_initialized": alreadyInitialized,
				"repo_id":             resolvedRepoID,
				"repo_path":           resolvedRepoPath,
				"next_step":           "codefind index run",
			}
			if verbose {
				manifestPath, err := indexer.ManifestPath(cfg.ActiveOrgID, resolvedRepoID)
				if err != nil {
					return err
				}
				response["manifest_path"] = manifestPath
				response["manifest"] = manifest
				response["next_steps"] = []string{
					"codefind index run",
					"codefind query <text>",
					"codefind stats",
				}
			}

			return writeJSON(cmd.OutOrStdout(), response)
		},
	}

	command.Flags().StringVar(&repoID, "repo-id", "", "override the derived repo identifier")
	command.Flags().StringVar(&repoPath, "repo-path", "", "project path to initialize (defaults to current directory)")
	command.Flags().BoolVar(&verbose, "verbose", false, "include manifest details in the response")
	return command
}

func newIndexCommand(configPath *string) *cobra.Command {
	indexCmd := &cobra.Command{
		Use:   "index",
		Short: "Manage repo indexing for the current organization",
	}
	indexCmd.AddCommand(newIndexRunCommand(configPath))
	indexCmd.AddCommand(newIndexRemoveCommand(configPath))
	return indexCmd
}

func newIndexRunCommand(configPath *string) *cobra.Command {
	var (
		repoID      string
		repoPath    string
		force       bool
		window      bool
		retryLSP    bool
		concurrency int
	)

	command := &cobra.Command{
		Use:   "run",
		Short: "Index a repo for the current organization",
		RunE: func(cmd *cobra.Command, _ []string) error {
			startedAt := time.Now()
			if strings.TrimSpace(repoPath) == "" {
				repoPath = "."
			}
			if concurrency < 1 {
				return errors.New("--concurrency must be >= 1")
			}
			if window && retryLSP {
				return errors.New("--window cannot be combined with --retry-lsp")
			}

			apiClient, err := requireAdminClient(cmd.Context(), cmd.OutOrStdout(), *configPath)
			if err != nil {
				return err
			}
			cfg, err := loadRequiredConfig(*configPath)
			if err != nil {
				return err
			}
			if strings.TrimSpace(cfg.ActiveOrgID) == "" {
				return errors.New("active_org_id is not configured; run 'codefind auth login'")
			}

			manifest, resolvedRepoPath, err := resolveInitializedProject(cfg, repoPath)
			if err != nil {
				return err
			}
			resolvedRepoID, err := resolveScopedRepoID(manifest, repoID)
			if err != nil {
				return err
			}
			modeLabel := "hybrid (LSP when available)"
			if window {
				modeLabel = "window-only"
			}
			reporter := newIndexRunReporter(cmd.OutOrStdout())
			reporter.Start(resolvedRepoID, resolvedRepoPath, modeLabel, concurrency)
			idx, err := indexer.New(resolvedRepoPath, manifest)
			if err != nil {
				return err
			}

			response, err := idx.Index(cmd.Context(), indexer.RunOptions{
				RepoID:      resolvedRepoID,
				OrgID:       cfg.ActiveOrgID,
				Force:       force,
				Window:      window,
				RetryLSP:    retryLSP,
				Concurrency: concurrency,
				Progress:    reporter.Progress,
			}, indexer.NewClientStore(apiClient))
			if err != nil {
				return err
			}
			reporter.Complete(time.Since(startedAt))
			return writeJSON(cmd.OutOrStdout(), response)
		},
	}

	command.Flags().StringVar(&repoID, "repo-id", "", "repo identifier to index (defaults to the current initialized project)")
	command.Flags().StringVar(&repoPath, "repo-path", "", "project path to index (defaults to current directory)")
	command.Flags().BoolVar(&force, "force", false, "reindex the full repo")
	command.Flags().BoolVar(&window, "window", false, "use window chunking only")
	command.Flags().BoolVar(&retryLSP, "retry-lsp", false, "retry LSP chunking for previously degraded unchanged files")
	command.Flags().IntVar(&concurrency, "concurrency", 1, "parallel file processing inside one indexing job")
	return command
}

func newIndexRemoveCommand(configPath *string) *cobra.Command {
	var (
		repoID   string
		repoPath string
	)

	command := &cobra.Command{
		Use:   "remove",
		Short: "Remove all indexed backend data for a repo and reset its local manifest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiClient, err := requireAdminClient(cmd.Context(), cmd.OutOrStdout(), *configPath)
			if err != nil {
				return err
			}
			cfg, err := loadRequiredConfig(*configPath)
			if err != nil {
				return err
			}
			if strings.TrimSpace(cfg.ActiveOrgID) == "" {
				return errors.New("active_org_id is not configured; run 'codefind auth login'")
			}

			manifest, _, err := resolveInitializedProject(cfg, repoPath)
			if err != nil {
				return err
			}
			resolvedRepoID, err := resolveScopedRepoID(manifest, repoID)
			if err != nil {
				return err
			}

			response, err := apiClient.ClearRepo(cmd.Context(), api.RepoClearRequest{RepoID: resolvedRepoID})
			if err != nil {
				return err
			}

			if resetErr := indexer.ResetManifest(cfg.ActiveOrgID, resolvedRepoID); resetErr != nil {
				if writeErr := writeJSON(cmd.OutOrStdout(), response); writeErr != nil {
					return writeErr
				}
				return fmt.Errorf("backend cleared but manifest reset failed: %w", resetErr)
			}

			return writeJSON(cmd.OutOrStdout(), response)
		},
	}

	command.Flags().StringVar(&repoID, "repo-id", "", "repo identifier to remove from the index (defaults to the current initialized project)")
	command.Flags().StringVar(&repoPath, "repo-path", "", "project path to remove from the index (defaults to current directory)")
	return command
}

func newCleanupCommand(configPath *string) *cobra.Command {
	var (
		repoID        string
		listMode      bool
		olderThanDays int
	)

	command := &cobra.Command{
		Use:   "cleanup",
		Short: "Inspect or purge tombstoned chunks for a repo",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if listMode && olderThanDays > 0 {
				return errors.New("--list cannot be combined with --older-than")
			}
			if !listMode && olderThanDays == 0 {
				return errors.New("either --list or --older-than must be provided")
			}

			apiClient, err := requireAdminClient(cmd.Context(), cmd.OutOrStdout(), *configPath)
			if err != nil {
				return err
			}
			cfg, err := loadRequiredConfig(*configPath)
			if err != nil {
				return err
			}
			manifest, _, err := resolveInitializedProject(cfg, "")
			if err != nil {
				return err
			}
			resolvedRepoID, err := resolveScopedRepoID(manifest, repoID)
			if err != nil {
				return err
			}
			if listMode {
				response, err := apiClient.ListTombstonedChunks(cmd.Context(), resolvedRepoID)
				if err != nil {
					return err
				}
				return writeJSON(cmd.OutOrStdout(), response)
			}
			response, err := apiClient.PurgeChunks(cmd.Context(), api.ChunkPurgeRequest{
				RepoID:        resolvedRepoID,
				OlderThanDays: olderThanDays,
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), response)
		},
	}

	command.Flags().StringVar(&repoID, "repo-id", "", "repo identifier to inspect or purge (defaults to the current initialized project)")
	command.Flags().BoolVar(&listMode, "list", false, "list tombstoned chunks")
	command.Flags().IntVar(&olderThanDays, "older-than", 0, "purge tombstoned chunks older than the given number of days")
	return command
}

func newLSPCommand(_ *string) *cobra.Command {
	lspCmd := &cobra.Command{
		Use:   "lsp",
		Short: "Inspect LSP availability for indexing",
	}

	lspCmd.AddCommand(newLSPStatusCommand())

	return lspCmd
}

type lspLanguageStatus struct {
	Language   string `json:"language"`
	Name       string `json:"name"`
	Executable string `json:"executable"`
	Path       string `json:"path,omitempty"`
	Available  bool   `json:"available"`
}

type lspStatusResponse struct {
	SupportedCount int                 `json:"supported_count"`
	AvailableCount int                 `json:"available_count"`
	Languages      []lspLanguageStatus `json:"languages"`
}

func newLSPStatusCommand() *cobra.Command {
	var jsonOutput bool

	command := &cobra.Command{
		Use:   "status",
		Short: "Show the current LSP status for supported languages",
		RunE: func(cmd *cobra.Command, _ []string) error {
			discovered := lsp.DiscoverAvailability()
			response := lspStatusResponse{
				SupportedCount: len(discovered),
				Languages:      make([]lspLanguageStatus, 0, len(discovered)),
			}
			for _, entry := range discovered {
				if entry.Available {
					response.AvailableCount++
				}
				response.Languages = append(response.Languages, lspLanguageStatus{
					Language:   entry.Language,
					Name:       entry.Name,
					Executable: entry.Executable,
					Path:       entry.Path,
					Available:  entry.Available,
				})
			}

			return writeCommandOutput(cmd.OutOrStdout(), jsonOutput, response, func(stdout io.Writer) error {
				return writeLSPStatusOutput(stdout, response)
			})
		},
	}

	addJSONFlag(command, &jsonOutput)
	return command
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
	var jsonOutput bool

	command := &cobra.Command{
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
					return writeCommandOutput(cmd.OutOrStdout(), jsonOutput, status, func(stdout io.Writer) error {
						return writeAuthStatusOutput(stdout, status)
					})
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

			return writeCommandOutput(cmd.OutOrStdout(), jsonOutput, status, func(stdout io.Writer) error {
				return writeAuthStatusOutput(stdout, status)
			})
		},
	}

	addJSONFlag(command, &jsonOutput)
	return command
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

type indexRunReporter struct {
	stdout io.Writer
}

const indexRunBanner = "┏━╸┏━┓╺┳┓┏━╸   ┏━╸╻┏┓╻╺┳┓\n┃  ┃ ┃ ┃┃┣╸ ╺━╸┣╸ ┃┃┗┫ ┃┃\n┗━╸┗━┛╺┻┛┗━╸   ╹  ╹╹ ╹╺┻┛"

func newIndexRunReporter(stdout io.Writer) *indexRunReporter {
	return &indexRunReporter{stdout: stdout}
}

func (r *indexRunReporter) Start(repoID, repoPath, mode string, concurrency int) {
	fmt.Fprintf(r.stdout, "%s\n\n", indexRunBanner)
	fmt.Fprintf(r.stdout, "Repo ID: %s\n", repoID)
	fmt.Fprintf(r.stdout, "Repo Path: %s\n", repoPath)
	fmt.Fprintf(r.stdout, "Chunking Mode: %s\n", mode)
	fmt.Fprintf(r.stdout, "Concurrency: %d\n", concurrency)
}

func (r *indexRunReporter) Progress(message string) {
	fmt.Fprintf(r.stdout, "• %s\n", message)
}

func (r *indexRunReporter) Complete(duration time.Duration) {
	fmt.Fprintf(r.stdout, "Total Time: %.1fs\n", duration.Seconds())
}
