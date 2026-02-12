package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Set via -ldflags at build time
var version = "dev"

// Global flags
var (
	verbose    bool
	jsonOutput bool
)

// Config represents alpine.yaml configuration
type Config struct {
	Install   string   `yaml:"install"`
	EnvFiles  []string `yaml:"env_files"`
	BaseImage string   `yaml:"base_image"`
	Services  []string `yaml:"services"`
}

// validServices is the set of recognized service names
var validServices = map[string]bool{
	"postgres": true,
	"redis":    true,
}

// loadConfig reads and validates alpine.yaml from the current directory.
// Returns default config with a warning if file is missing.
func loadConfig() (*Config, error) {
	cfg := &Config{
		BaseImage: "ubuntu:24.04",
	}

	data, err := os.ReadFile("alpine.yaml")
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("alpine.yaml not found, using defaults")
			return cfg, nil
		}
		return nil, fmt.Errorf("reading alpine.yaml: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing alpine.yaml: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.BaseImage == "" {
		return fmt.Errorf("base_image cannot be empty")
	}
	for _, svc := range c.Services {
		if !validServices[svc] {
			return fmt.Errorf("unknown service %q (supported: postgres, redis)", svc)
		}
	}
	return nil
}

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

// exitError carries a specific exit code alongside the error message.
// Commands return this to signal user errors (1) vs system errors (2).
type exitError struct {
	msg  string
	code int
}

func (e *exitError) Error() string { return e.msg }

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Handle double Ctrl+C: second signal force-exits
	go func() {
		<-ctx.Done()
		// Context cancelled by first signal. Next signal will force exit.
		ctx2, cancel2 := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel2()
		<-ctx2.Done()
		os.Exit(1)
	}()

	rootCmd.SetContext(ctx)

	if err := rootCmd.Execute(); err != nil {
		code := 1
		if ee, ok := err.(*exitError); ok {
			code = ee.code
		}
		outputError(err.Error(), code)
		os.Exit(code)
	}
}
