package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tk-425/Codefind/internal/indexer"
	"github.com/tk-425/Codefind/internal/lsp"
	"github.com/tk-425/Codefind/pkg/api"
)

func newInitCommand(configPath *string) *cobra.Command {
	var (
		repoID     string
		repoPath   string
		verbose    bool
		jsonOutput bool
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

			response := initCommandResponse{
				Status:             "ok",
				Message:            message,
				Initialized:        true,
				AlreadyInitialized: alreadyInitialized,
				RepoID:             resolvedRepoID,
				RepoPath:           resolvedRepoPath,
				NextStep:           "codefind index run",
			}
			if verbose {
				manifestPath, err := indexer.ManifestPath(cfg.ActiveOrgID, resolvedRepoID)
				if err != nil {
					return err
				}
				response.ManifestPath = manifestPath
				response.Manifest = manifest
				response.NextSteps = []string{
					"codefind index run",
					"codefind query <text>",
					"codefind stats",
				}
			}

			return writeCommandOutput(cmd.OutOrStdout(), jsonOutput, response, func(stdout io.Writer) error {
				return writeInitOutput(stdout, response, verbose)
			})
		},
	}

	command.Flags().StringVar(&repoID, "repo-id", "", "override the derived repo identifier")
	command.Flags().StringVar(&repoPath, "repo-path", "", "project path to initialize (defaults to current directory)")
	command.Flags().BoolVar(&verbose, "verbose", false, "include manifest details in output")
	addJSONFlag(command, &jsonOutput)
	return command
}

type initCommandResponse struct {
	Status             string            `json:"status"`
	Message            string            `json:"message"`
	Initialized        bool              `json:"initialized"`
	AlreadyInitialized bool              `json:"already_initialized"`
	RepoID             string            `json:"repo_id"`
	RepoPath           string            `json:"repo_path"`
	NextStep           string            `json:"next_step"`
	ManifestPath       string            `json:"manifest_path,omitempty"`
	Manifest           *indexer.Manifest `json:"manifest,omitempty"`
	NextSteps          []string          `json:"next_steps,omitempty"`
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
	command.Flags().BoolVar(&retryLSP, "retry-lsp", false, "retry degraded hybrid LSP fallbacks for unchanged files")
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
