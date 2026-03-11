package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
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
