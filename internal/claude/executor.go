package claude

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/maxmcd/river/internal/logger"
)

// ExecuteConfig holds configuration for executing Claude
type ExecuteConfig struct {
	// Prompt is the input prompt to send to Claude
	Prompt string

	// StateFile is the path to the claude_state.json file
	StateFile string


	// MCPServers is a list of MCP servers to enable (optional)
	MCPServers []string

	// AllowedTools restricts which tools Claude can use (optional)
	AllowedTools []string

	// SystemPrompt overrides the default system prompt (optional)
	SystemPrompt string

	// Timeout for the Claude execution (optional, defaults to no timeout)
	Timeout time.Duration
}

// Executor handles execution of Claude commands
type Executor struct {
	commandRunner CommandRunner
}

// CommandRunner interface for testing
type CommandRunner interface {
	Run(ctx context.Context, config ExecuteConfig) (string, error)
}

// NewExecutor creates a new Claude executor
func NewExecutor() *Executor {
	return &Executor{
		commandRunner: &defaultCommandRunner{},
	}
}

// Execute runs Claude with the given configuration
func (e *Executor) Execute(ctx context.Context, config ExecuteConfig) (string, error) {
	logger.Debug("Starting Claude execution")
	
	// Validate required fields
	if config.Prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}
	if config.StateFile == "" {
		return "", fmt.Errorf("state file is required")
	}

	logger.WithFields(map[string]interface{}{
		"state_file": config.StateFile,
		"mcp_servers": len(config.MCPServers),
		"allowed_tools": len(config.AllowedTools),
	}).Debug("Claude configuration validated")

	// Use command runner (allows for mocking in tests)
	if e.commandRunner != nil {
		return e.commandRunner.Run(ctx, config)
	}

	// Default implementation
	runner := &defaultCommandRunner{}
	return runner.Run(ctx, config)
}

// DefaultSystemPrompt is the default system prompt used when none is provided
const DefaultSystemPrompt = "You are an expert software engineer with deep knowledge of TDD, Python, Typescript. Execute the following tasks with surgical precision while taking care not to overengineer solutions."

// DefaultAllowedTools are the default tools allowed when none are specified
var DefaultAllowedTools = []string{
	"mcp__context7__*",
	"Bash",
	"Read",
	"Write",
	"Edit",
	"Remove",
}

// buildCommand constructs the exec.Cmd for Claude
func (e *Executor) buildCommand(config ExecuteConfig) *exec.Cmd {
	args := []string{}

	// Add output format
	args = append(args, "--output-format", "text")

	// Add MCP servers
	if len(config.MCPServers) > 0 {
		for _, server := range config.MCPServers {
			args = append(args, "--mcp-server", server)
		}
	}

	// Add allowed tools restriction
	allowedTools := config.AllowedTools
	if len(allowedTools) == 0 {
		allowedTools = DefaultAllowedTools
	}
	if len(allowedTools) > 0 {
		args = append(args, "--allowedTools")
		args = append(args, allowedTools...)
	}

	// Add system prompt
	systemPrompt := config.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = DefaultSystemPrompt
	}
	args = append(args, "--append-system-prompt", systemPrompt)

	// Note: Claude CLI doesn't have a --project flag
	// It uses the current working directory by default
	// TODO: Consider using --add-dir flag or changing working directory

	// Add the prompt with -p flag
	args = append(args, "-p", config.Prompt)

	// Create command
	cmd := exec.Command("claude", args...)

	// Set working directory to current directory (enables worktree isolation)
	workDir, err := os.Getwd()
	if err != nil {
		// Log warning but continue without setting Dir
		// Claude will use default behavior (inherit from parent process)
		logger.WithField("error", err).Info("Failed to get working directory, Claude will use default directory")
	} else {
		cmd.Dir = workDir
		logger.WithField("workDir", workDir).Debug("Set Claude working directory")
	}

	// Set environment variables
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("RIVER_STATE_FILE=%s", config.StateFile))

	return cmd
}

// defaultCommandRunner is the actual implementation
type defaultCommandRunner struct{}

func (r *defaultCommandRunner) Run(ctx context.Context, config ExecuteConfig) (string, error) {
	// Create executor to access buildCommand
	executor := &Executor{}
	baseCmd := executor.buildCommand(config)

	logger.WithFields(map[string]interface{}{
		"command": baseCmd.Path,
		"args": baseCmd.Args,
	}).Debug("Preparing Claude command")

	// Handle timeout
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
		logger.WithField("timeout", config.Timeout).Debug("Setting execution timeout")
	}

	// Create command with context
	cmd := exec.CommandContext(ctx, baseCmd.Path, baseCmd.Args[1:]...)
	cmd.Env = baseCmd.Env
	cmd.Dir = baseCmd.Dir  // Preserve working directory from buildCommand

	// Run the command
	startTime := time.Now()
	logger.Debug("Executing Claude command")
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)
	
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error": err,
			"duration": duration,
			"output": string(output),
		}).Error("Claude execution failed")
		return "", fmt.Errorf("claude execution failed: %w", err)
	}

	logger.WithField("duration", duration).Debug("Claude execution completed successfully")
	return string(output), nil
}
