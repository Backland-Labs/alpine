package claude

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ExecuteConfig holds configuration for executing Claude
type ExecuteConfig struct {
	// Prompt is the input prompt to send to Claude
	Prompt string

	// StateFile is the path to the claude_state.json file
	StateFile string

	// LinearIssue is the Linear issue ID (optional)
	LinearIssue string

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
	// Validate required fields
	if config.Prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}
	if config.StateFile == "" {
		return "", fmt.Errorf("state file is required")
	}

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
	"mcp__linear-server__*",
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
	} else if config.LinearIssue != "" {
		// Default to linear-server if Linear issue is provided
		args = append(args, "--mcp-server", "linear-server")
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

	// Add project directory (directory of state file)
	projectDir := filepath.Dir(config.StateFile)
	args = append(args, "--project", projectDir)

	// Add the prompt with -p flag
	args = append(args, "-p", config.Prompt)

	// Create command
	cmd := exec.Command("claude", args...)

	// Set environment variables
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("RIVER_STATE_FILE=%s", config.StateFile))
	if config.LinearIssue != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RIVER_LINEAR_ISSUE=%s", config.LinearIssue))
	}

	return cmd
}

// defaultCommandRunner is the actual implementation
type defaultCommandRunner struct{}

func (r *defaultCommandRunner) Run(ctx context.Context, config ExecuteConfig) (string, error) {
	// Create executor to access buildCommand
	executor := &Executor{}
	baseCmd := executor.buildCommand(config)

	// Handle timeout
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	// Create command with context
	cmd := exec.CommandContext(ctx, baseCmd.Path, baseCmd.Args[1:]...)
	cmd.Env = baseCmd.Env

	// Run the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("claude execution failed: %w", err)
	}

	return string(output), nil
}
