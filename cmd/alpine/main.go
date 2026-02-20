package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

// Set via -ldflags at build time
var version = "dev"

// Global flags
var (
	verbose    bool
	jsonOutput bool
)

var rootCmd = &cobra.Command{
	Use:   "alpine",
	Short: "Cloudflare-native orchestrator for OpenCode sandboxes",
	Long: `Alpine orchestrates OpenCode sandbox sessions on Cloudflare.
Launch sandboxes, resume durable checkpoints, export branches,
and tear down safely from a single CLI.`,
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Configure slog based on verbose flag
		level := slog.LevelWarn
		if verbose {
			level = slog.LevelDebug
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})))
	},
	SilenceUsage:      true,
	SilenceErrors:     true,
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "machine-readable JSON output")

	// Subcommands are registered via init() in their respective files:
	// - launch.go:   rootCmd.AddCommand(launchCmd)
	// - list.go:     rootCmd.AddCommand(listCmd)
	// - status.go:   rootCmd.AddCommand(statusCmd)
	// - open.go:     rootCmd.AddCommand(openCmd)
	// - export.go:   rootCmd.AddCommand(exportCmd)
	// - teardown.go: rootCmd.AddCommand(teardownCmd)
	// - create.go: legacy hidden command
}

func execute(ctx context.Context) int {
	rootCmd.SetContext(ctx)
	if err := rootCmd.Execute(); err != nil {
		code := 1
		reasonCode := ""
		retryable := false
		if ee, ok := err.(*exitError); ok {
			code = ee.code
			reasonCode = ee.reasonCode
			retryable = ee.retryable
		}
		outputErrorWithReason(err.Error(), code, reasonCode, retryable)
		return code
	}
	return 0
}
