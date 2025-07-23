// Package config provides configuration management for the River CLI.
// It loads configuration from environment variables with sensible defaults.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Verbosity represents the output verbosity level
type Verbosity string

const (
	// VerbosityNormal shows only essential output
	VerbosityNormal Verbosity = "normal"
	// VerbosityVerbose includes step descriptions and timing
	VerbosityVerbose Verbosity = "verbose"
	// VerbosityDebug provides full debug logging
	VerbosityDebug Verbosity = "debug"
)

// GitConfig holds git-related configuration
type GitConfig struct {
	// WorktreeEnabled controls whether to create git worktrees for tasks
	WorktreeEnabled bool

	// BaseBranch is the branch to base new worktrees on
	BaseBranch string

	// AutoCleanupWT controls whether to clean up worktrees after completion
	AutoCleanupWT bool
}

// Config holds all configuration for the River CLI
type Config struct {
	// WorkDir is the working directory for Claude execution
	WorkDir string

	// Verbosity controls output level
	Verbosity Verbosity

	// ShowOutput controls whether Claude command output is displayed
	ShowOutput bool

	// ShowTodoUpdates controls whether to show real-time TODO progress from Claude
	ShowTodoUpdates bool

	// StateFile is the path to the state file
	StateFile string

	// AutoCleanup controls whether state file is deleted on success
	AutoCleanup bool

	// Git holds git-related configuration
	Git GitConfig
}

// New creates a new Config instance from environment variables
func New() (*Config, error) {
	cfg := &Config{}

	// Load WorkDir - defaults to current directory
	workDir, exists := os.LookupEnv("RIVER_WORKDIR")
	if !exists {
		// Environment variable not set, use current directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		cfg.WorkDir = cwd
	} else {
		// Environment variable exists, validate it
		if workDir == "" {
			return nil, fmt.Errorf("RIVER_WORKDIR cannot be empty")
		}
		// Validate that WorkDir is absolute
		if !filepath.IsAbs(workDir) {
			return nil, fmt.Errorf("RIVER_WORKDIR must be an absolute path, got: %s", workDir)
		}
		cfg.WorkDir = workDir
	}

	// Load Verbosity - defaults to normal
	verbosity := os.Getenv("RIVER_VERBOSITY")
	if verbosity == "" {
		cfg.Verbosity = VerbosityNormal
	} else {
		switch Verbosity(verbosity) {
		case VerbosityNormal, VerbosityVerbose, VerbosityDebug:
			cfg.Verbosity = Verbosity(verbosity)
		default:
			return nil, fmt.Errorf("RIVER_VERBOSITY must be one of: normal, verbose, debug; got: %s", verbosity)
		}
	}

	// Load ShowOutput - defaults to true
	showOutput, err := parseBoolEnv("RIVER_SHOW_OUTPUT", true)
	if err != nil {
		return nil, err
	}
	cfg.ShowOutput = showOutput

	// Load ShowTodoUpdates - defaults to true
	showTodoUpdates, err := parseBoolEnv("RIVER_SHOW_TODO_UPDATES", true)
	if err != nil {
		return nil, err
	}
	cfg.ShowTodoUpdates = showTodoUpdates

	// Load StateFile - defaults to .claude/river/claude_state.json
	stateFile := os.Getenv("RIVER_STATE_FILE")
	if stateFile == "" {
		cfg.StateFile = filepath.Join(".claude", "river", "claude_state.json")
	} else {
		cfg.StateFile = stateFile
	}

	// Load AutoCleanup - defaults to true
	autoCleanup, err := parseBoolEnv("RIVER_AUTO_CLEANUP", true)
	if err != nil {
		return nil, err
	}
	cfg.AutoCleanup = autoCleanup

	// Load Git configuration
	cfg.Git = GitConfig{}

	// Load WorktreeEnabled - defaults to true
	worktreeEnabled, err := parseBoolEnv("RIVER_GIT_ENABLED", true)
	if err != nil {
		return nil, err
	}
	cfg.Git.WorktreeEnabled = worktreeEnabled

	// Load BaseBranch - defaults to "main"
	baseBranch := os.Getenv("RIVER_GIT_BASE_BRANCH")
	if baseBranch == "" {
		cfg.Git.BaseBranch = "main"
	} else {
		cfg.Git.BaseBranch = baseBranch
	}

	// Load AutoCleanupWT - defaults to true
	autoCleanupWT, err := parseBoolEnv("RIVER_GIT_AUTO_CLEANUP", true)
	if err != nil {
		return nil, err
	}
	cfg.Git.AutoCleanupWT = autoCleanupWT

	return cfg, nil
}

// IsVerbose returns true if verbosity is verbose or debug
func (c *Config) IsVerbose() bool {
	return c.Verbosity == VerbosityVerbose || c.Verbosity == VerbosityDebug
}

// IsDebug returns true if verbosity is debug
func (c *Config) IsDebug() bool {
	return c.Verbosity == VerbosityDebug
}

// parseBoolEnv parses a boolean environment variable with a default value
func parseBoolEnv(key string, defaultValue bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}
	switch value {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be true or false, got: %s", key, value)
	}
}
