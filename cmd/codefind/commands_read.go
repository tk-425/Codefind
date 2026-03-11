package main

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tk-425/Codefind/internal/query"
	"github.com/tk-425/Codefind/internal/stats"
	"github.com/tk-425/Codefind/pkg/api"
)

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
