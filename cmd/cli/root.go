package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/znth-cx/zentag/internal/config"
	"github.com/znth-cx/zentag/internal/logging"
	"github.com/znth-cx/zentag/internal/version"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	cfg     *config.Config
	logger  *slog.Logger
)

var rootCmd = &cobra.Command{
	Use:   "zentag",
	Short: "zentag renames, retags, and lints audiobook metadata",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		logger = logging.New(verbose)
		loaded, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		cfg = loaded
		return nil
	},
}

func init() {
	rootCmd.Version = version.Version
	rootCmd.SilenceErrors = true
	// Suppresses usage block on check violations, also on genuine arg/flag
	// errors for any subcommand: acceptable since --help still works.
	rootCmd.SilenceUsage = true
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "path to config file (default: ./zentag.yaml or <user config dir>/zentag/zentag.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")
}

// Run executes the root command against ctx; cancels in-flight
// ffmpeg/mediainfo subprocess calls on SIGINT/SIGTERM.
func Run(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}
