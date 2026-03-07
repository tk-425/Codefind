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

	"github.com/spf13/cobra"
	"github.com/tk-425/Codefind/internal/client"
	"github.com/tk-425/Codefind/internal/config"
	"github.com/tk-425/Codefind/internal/keychain"
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

Use 'codefind config' to set or show local CLI config, and 'codefind health'
to check the configured server. Build with 'go build -o ./bin/codefind ./cmd/codefind'
and install globally with the documented /usr/local/bin flow.`),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		if configPath != "" {
			return nil
		}
		defaultPath, err := config.DefaultPath()
		if err != nil {
			return err
		}
		configPath = defaultPath
		return nil
	}
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "override config file path")

	rootCmd.AddCommand(newConfigCommand(&configPath))
	rootCmd.AddCommand(newHealthCommand(&configPath))

	return rootCmd
}

func newConfigCommand(configPath *string) *cobra.Command {
	var (
		show        bool
		serverURL   string
		activeOrgID string
		editor      string
	)

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Show or update CLI configuration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			hasUpdates := strings.TrimSpace(serverURL) != "" || strings.TrimSpace(activeOrgID) != "" || strings.TrimSpace(editor) != ""
			if show {
				if hasUpdates {
					return errors.New("--show cannot be combined with update flags")
				}
				return runConfigShow(cmd.OutOrStdout(), *configPath)
			}
			if !hasUpdates {
				return cmd.Help()
			}
			return runConfigUpdate(cmd.OutOrStdout(), *configPath, serverURL, activeOrgID, editor)
		},
	}

	configCmd.Flags().BoolVar(&show, "show", false, "show current config")
	configCmd.Flags().StringVar(&serverURL, "server-url", "", "set the server URL")
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

			apiClient, err := client.New(cfg.ServerURL, keychain.DefaultManager())
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

func runConfigShow(stdout io.Writer, path string) error {
	cfg, err := config.LoadOrDefault(path)
	if err != nil {
		return err
	}
	return writeJSON(stdout, cfg.DisplayMap())
}

func runConfigUpdate(stdout io.Writer, path, serverURL, activeOrgID, editor string) error {
	cfg, err := config.LoadOrDefault(path)
	if err != nil {
		return err
	}

	if strings.TrimSpace(serverURL) != "" {
		cfg.ServerURL = serverURL
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

func writeJSON(stdout io.Writer, value any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
