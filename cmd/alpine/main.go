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
	Short: "Ephemeral dev environments for parallel AI coding agents",
	Long: `Alpine creates fully isolated, containerized development environments
for running parallel AI coding agents. Each environment gets its own
repo clone, branch, services, and Claude Code instance.`,
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
	// - create.go: rootCmd.AddCommand(createCmd)
	// - list.go:   rootCmd.AddCommand(listCmd)
	// - status.go: rootCmd.AddCommand(statusCmd)
}

func execute(ctx context.Context) int {
	rootCmd.SetContext(ctx)
	if err := rootCmd.Execute(); err != nil {
		code := 1
		if ce, ok := err.(*commandError); ok {
			code = ce.exitCode
			outputCommandError(ce)
			return code
		}
		if ee, ok := err.(*exitError); ok {
			code = ee.code
		}
		outputError(err.Error(), code)
		return code
	}
	return 0
}
