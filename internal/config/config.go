// Package config provides configuration management for the Alpine CLI.
// It loads configuration from environment variables with sensible defaults.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
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

// GitCloneConfig holds git clone-related configuration for server operations
type GitCloneConfig struct {
	// Enabled controls whether git clone operations are enabled
	Enabled bool

	// AuthToken is the GitHub authentication token for private repositories
	AuthToken string

	// Timeout is the maximum duration to wait for clone operations
	Timeout time.Duration

	// Depth is the depth of the shallow clone (default: 1)
	Depth int
}

// GitConfig holds git-related configuration
type GitConfig struct {
	// WorktreeEnabled controls whether to create git worktrees for tasks
	WorktreeEnabled bool

	// BaseBranch is the branch to base new worktrees on
	BaseBranch string

	// AutoCleanupWT controls whether to clean up worktrees after completion
	AutoCleanupWT bool

	// Clone holds git clone-related configuration
	Clone GitCloneConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	// Enabled controls whether the HTTP server is started
	Enabled bool

	// Port is the HTTP server port
	Port int

	// StreamBufferSize is the size of event buffers for streaming
	StreamBufferSize int

	// MaxClientsPerRun is the maximum number of clients per run
	MaxClientsPerRun int
}

// ToolCallEventsConfig holds tool call event capture configuration
type ToolCallEventsConfig struct {
	// Enabled controls whether tool call events are captured and emitted
	Enabled bool

	// BatchSize is the number of events to batch before sending (default: 10)
	BatchSize int

	// SampleRate is the percentage of events to capture (1-100, default: 100)
	SampleRate int
}

// Config holds all configuration for the Alpine CLI
type Config struct {
	// WorkDir is the working directory for Claude execution
	WorkDir string

	// Verbosity controls output level
	Verbosity Verbosity

	// ShowOutput controls whether Claude command output is displayed
	ShowOutput bool

	// ShowTodoUpdates controls whether to show real-time TODO progress from Claude
	ShowTodoUpdates bool

	// ShowToolUpdates controls whether to show real-time tool usage updates
	ShowToolUpdates bool

	// StateFile is the path to the state file
	StateFile string

	// AutoCleanup controls whether state file is deleted on success
	AutoCleanup bool

	// Git holds git-related configuration
	Git GitConfig

	// Server holds server-related configuration
	Server ServerConfig

	// ToolCallEvents holds tool call event capture configuration
	ToolCallEvents ToolCallEventsConfig
}

// New creates a new Config instance from environment variables
func New() (*Config, error) {
	cfg := &Config{}

	// Load WorkDir - defaults to current directory
	workDir, exists := os.LookupEnv("ALPINE_WORKDIR")
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
			return nil, fmt.Errorf("ALPINE_WORKDIR cannot be empty")
		}
		// Validate that WorkDir is absolute
		if !filepath.IsAbs(workDir) {
			return nil, fmt.Errorf("ALPINE_WORKDIR must be an absolute path, got: %s", workDir)
		}
		cfg.WorkDir = workDir
	}

	// Load Verbosity - defaults to normal
	verbosity := os.Getenv("ALPINE_VERBOSITY")
	if verbosity == "" {
		cfg.Verbosity = VerbosityNormal
	} else {
		switch Verbosity(verbosity) {
		case VerbosityNormal, VerbosityVerbose, VerbosityDebug:
			cfg.Verbosity = Verbosity(verbosity)
		default:
			return nil, fmt.Errorf("ALPINE_VERBOSITY must be one of: normal, verbose, debug; got: %s", verbosity)
		}
	}

	// Load ShowOutput - defaults to true
	showOutput, err := parseBoolEnv("ALPINE_SHOW_OUTPUT", true)
	if err != nil {
		return nil, err
	}
	cfg.ShowOutput = showOutput

	// Load ShowTodoUpdates - defaults to true
	showTodoUpdates, err := parseBoolEnv("ALPINE_SHOW_TODO_UPDATES", true)
	if err != nil {
		return nil, err
	}
	cfg.ShowTodoUpdates = showTodoUpdates

	// Load ShowToolUpdates - defaults to true
	showToolUpdates, err := parseBoolEnv("ALPINE_SHOW_TOOL_UPDATES", true)
	if err != nil {
		return nil, err
	}
	cfg.ShowToolUpdates = showToolUpdates

	// StateFile is always at a fixed location
	cfg.StateFile = filepath.Join("agent_state", "agent_state.json")

	// Load AutoCleanup - defaults to true
	autoCleanup, err := parseBoolEnv("ALPINE_AUTO_CLEANUP", true)
	if err != nil {
		return nil, err
	}
	cfg.AutoCleanup = autoCleanup

	// Load Git configuration
	cfg.Git = GitConfig{}

	// Load WorktreeEnabled - defaults to true
	worktreeEnabled, err := parseBoolEnv("ALPINE_GIT_ENABLED", true)
	if err != nil {
		return nil, err
	}
	cfg.Git.WorktreeEnabled = worktreeEnabled

	// Load BaseBranch - defaults to "main"
	baseBranch := os.Getenv("ALPINE_GIT_BASE_BRANCH")
	if baseBranch == "" {
		cfg.Git.BaseBranch = "main"
	} else {
		cfg.Git.BaseBranch = baseBranch
	}

	// Load AutoCleanupWT - defaults to true
	autoCleanupWT, err := parseBoolEnv("ALPINE_GIT_AUTO_CLEANUP", true)
	if err != nil {
		return nil, err
	}
	cfg.Git.AutoCleanupWT = autoCleanupWT

	// Load Git Clone configuration
	cfg.Git.Clone = GitCloneConfig{}

	// Load Clone.Enabled - defaults to true
	cloneEnabled, err := parseBoolEnv("ALPINE_GIT_CLONE_ENABLED", true)
	if err != nil {
		return nil, err
	}
	cfg.Git.Clone.Enabled = cloneEnabled

	// Load Clone.AuthToken - defaults to empty string
	cfg.Git.Clone.AuthToken = os.Getenv("ALPINE_GIT_CLONE_AUTH_TOKEN")

	// Load Clone.Timeout - defaults to 300 seconds
	cloneTimeoutStr := os.Getenv("ALPINE_GIT_CLONE_TIMEOUT")
	if cloneTimeoutStr == "" {
		cfg.Git.Clone.Timeout = 300 * time.Second
	} else {
		cloneTimeoutSecs, err := strconv.Atoi(cloneTimeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ALPINE_GIT_CLONE_TIMEOUT: %w", err)
		}
		if cloneTimeoutSecs <= 0 {
			return nil, fmt.Errorf("ALPINE_GIT_CLONE_TIMEOUT must be positive, got: %d", cloneTimeoutSecs)
		}
		cfg.Git.Clone.Timeout = time.Duration(cloneTimeoutSecs) * time.Second
	}

	// Load Clone.Depth - defaults to 1
	cloneDepthStr := os.Getenv("ALPINE_GIT_CLONE_DEPTH")
	if cloneDepthStr == "" {
		cfg.Git.Clone.Depth = 1
	} else {
		cloneDepth, err := strconv.Atoi(cloneDepthStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ALPINE_GIT_CLONE_DEPTH: %w", err)
		}
		if cloneDepth <= 0 {
			return nil, fmt.Errorf("ALPINE_GIT_CLONE_DEPTH must be positive, got: %d", cloneDepth)
		}
		cfg.Git.Clone.Depth = cloneDepth
	}

	// Load Server configuration
	cfg.Server = ServerConfig{}

	// Load Server.Enabled - defaults to false
	serverEnabled, err := parseBoolEnv("ALPINE_HTTP_ENABLED", false)
	if err != nil {
		return nil, err
	}
	cfg.Server.Enabled = serverEnabled

	// Load Server.Port - defaults to 3001
	httpPortStr := os.Getenv("ALPINE_HTTP_PORT")
	if httpPortStr == "" {
		cfg.Server.Port = 3001
	} else {
		httpPort, err := parsePort(httpPortStr)
		if err != nil {
			return nil, fmt.Errorf("ALPINE_HTTP_PORT %s", err)
		}
		cfg.Server.Port = httpPort
	}

	// Load Server.StreamBufferSize - defaults to 100
	streamBufferSizeStr := os.Getenv("ALPINE_STREAM_BUFFER_SIZE")
	if streamBufferSizeStr == "" {
		cfg.Server.StreamBufferSize = 100
	} else {
		streamBufferSize, err := strconv.Atoi(streamBufferSizeStr)
		if err != nil || streamBufferSize <= 0 {
			// Invalid value, use default
			cfg.Server.StreamBufferSize = 100
		} else {
			cfg.Server.StreamBufferSize = streamBufferSize
		}
	}

	// Load Server.MaxClientsPerRun - defaults to 100
	maxClientsStr := os.Getenv("ALPINE_MAX_CLIENTS_PER_RUN")
	if maxClientsStr == "" {
		cfg.Server.MaxClientsPerRun = 100
	} else {
		maxClients, err := strconv.Atoi(maxClientsStr)
		if err != nil || maxClients <= 0 {
			// Invalid value, use default
			cfg.Server.MaxClientsPerRun = 100
		} else {
			cfg.Server.MaxClientsPerRun = maxClients
		}
	}

	// Load ToolCallEvents configuration
	cfg.ToolCallEvents = ToolCallEventsConfig{}

	// Load ToolCallEvents.Enabled - defaults to false
	toolCallEventsEnabled, err := parseBoolEnv("ALPINE_TOOL_CALL_EVENTS_ENABLED", false)
	if err != nil {
		return nil, err
	}
	cfg.ToolCallEvents.Enabled = toolCallEventsEnabled

	// Load ToolCallEvents.BatchSize - defaults to 10
	batchSizeStr := os.Getenv("ALPINE_TOOL_CALL_BATCH_SIZE")
	if batchSizeStr == "" {
		cfg.ToolCallEvents.BatchSize = 10
	} else {
		batchSize, err := strconv.Atoi(batchSizeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ALPINE_TOOL_CALL_BATCH_SIZE: %w", err)
		}
		if batchSize <= 0 {
			return nil, fmt.Errorf("ALPINE_TOOL_CALL_BATCH_SIZE must be positive, got: %d", batchSize)
		}
		cfg.ToolCallEvents.BatchSize = batchSize
	}

	// Load ToolCallEvents.SampleRate - defaults to 100
	sampleRateStr := os.Getenv("ALPINE_TOOL_CALL_SAMPLE_RATE")
	if sampleRateStr == "" {
		cfg.ToolCallEvents.SampleRate = 100
	} else {
		sampleRate, err := strconv.Atoi(sampleRateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ALPINE_TOOL_CALL_SAMPLE_RATE: %w", err)
		}
		if sampleRate < 1 || sampleRate > 100 {
			return nil, fmt.Errorf("ALPINE_TOOL_CALL_SAMPLE_RATE must be between 1 and 100, got: %d", sampleRate)
		}
		cfg.ToolCallEvents.SampleRate = sampleRate
	}

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

// parsePort parses and validates a port number string
func parsePort(portStr string) (int, error) {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid port number: %s", portStr)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("must be between 1 and 65535, got: %d", port)
	}
	return port, nil
}
