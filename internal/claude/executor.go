package claude

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/maxmcd/river/internal/config"
	"github.com/maxmcd/river/internal/logger"
	"github.com/maxmcd/river/internal/output"
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
	config        *config.Config
	printer       *output.Printer
}

// CommandRunner interface for testing
type CommandRunner interface {
	Run(ctx context.Context, config ExecuteConfig) (string, error)
}

// NewExecutor creates a new Claude executor
func NewExecutor() *Executor {
	return &Executor{
		commandRunner: &defaultCommandRunner{},
		config:        nil,
		printer:       nil,
	}
}

// NewExecutorWithConfig creates a new Claude executor with configuration and printer
func NewExecutorWithConfig(cfg *config.Config, printer *output.Printer) *Executor {
	return &Executor{
		commandRunner: &defaultCommandRunner{},
		config:        cfg,
		printer:       printer,
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
		"state_file":    config.StateFile,
		"mcp_servers":   len(config.MCPServers),
		"allowed_tools": len(config.AllowedTools),
	}).Debug("Claude configuration validated")

	// Check if we should use todo monitoring
	if e.config != nil && e.config.ShowTodoUpdates && e.printer != nil {
		return e.executeWithTodoMonitoring(ctx, config)
	}

	// Use command runner (allows for mocking in tests)
	if e.commandRunner != nil {
		return e.commandRunner.Run(ctx, config)
	}

	// Default implementation
	runner := &defaultCommandRunner{}
	return runner.Run(ctx, config)
}

// executeWithTodoMonitoring runs Claude with real-time TODO monitoring
func (e *Executor) executeWithTodoMonitoring(ctx context.Context, config ExecuteConfig) (string, error) {
	logger.Debug("Starting Claude execution with TODO monitoring")

	// Setup hook (fallback to normal execution on failure)
	todoFile, cleanup, err := e.setupTodoHook()
	if err != nil {
		logger.WithField("error", err).Info("Failed to setup TODO hook, falling back to normal execution")
		return e.executeWithoutMonitoring(ctx, config)
	}
	defer func() {
		if cleanup != nil {
			cleanup()
		}
	}()

	// Start monitoring
	monitor := NewTodoMonitor(todoFile)
	monitorCtx, monitorCancel := context.WithCancel(ctx)
	defer monitorCancel()
	go monitor.Start(monitorCtx)

	// Start Claude execution
	claudeResult := make(chan string, 1)
	claudeErr := make(chan error, 1)
	go func() {
		result, err := e.executeClaudeCommand(ctx, config, todoFile)
		if err != nil {
			claudeErr <- err
		} else {
			claudeResult <- result
		}
	}()

	// Show updates
	e.printer.StartTodoMonitoring()
	defer e.printer.StopTodoMonitoring()

	for {
		select {
		case task := <-monitor.Updates():
			e.printer.UpdateCurrentTask(task)
		case result := <-claudeResult:
			return result, nil
		case err := <-claudeErr:
			return "", err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

// executeWithoutMonitoring runs Claude using the standard command runner
func (e *Executor) executeWithoutMonitoring(ctx context.Context, config ExecuteConfig) (string, error) {
	if e.commandRunner != nil {
		return e.commandRunner.Run(ctx, config)
	}
	runner := &defaultCommandRunner{}
	return runner.Run(ctx, config)
}

// executeClaudeCommand runs Claude with environment variables for hook integration
func (e *Executor) executeClaudeCommand(ctx context.Context, config ExecuteConfig, todoFile string) (string, error) {
	// Create executor to access buildCommand
	baseCmd := e.buildCommandWithValidation(config)

	logger.WithFields(map[string]interface{}{
		"command": baseCmd.Path,
		"args":    baseCmd.Args,
	}).Debug("Preparing Claude command with TODO monitoring")

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
	cmd.Dir = baseCmd.Dir

	// Add todo file environment variable
	cmd.Env = append(cmd.Env, fmt.Sprintf("RIVER_TODO_FILE=%s", todoFile))

	// Run the command
	startTime := time.Now()
	logger.Debug("Executing Claude command with TODO monitoring")
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error":    err,
			"duration": duration,
			"output":   string(output),
		}).Error("Claude execution failed")
		return "", fmt.Errorf("claude execution failed: %w", err)
	}

	logger.WithField("duration", duration).Debug("Claude execution completed successfully")
	return string(output), nil
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
	"TodoWrite",
	"Task",
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

// buildCommandWithValidation builds a command and validates the working directory
// before setting it on the command. This ensures Claude is not executed in a
// non-existent or inaccessible directory.
func (e *Executor) buildCommandWithValidation(config ExecuteConfig) *exec.Cmd {
	// Build the command using the existing method
	cmd := e.buildCommand(config)

	// Only validate and modify Dir if it was set by buildCommand
	if cmd.Dir != "" {
		// Validate the directory exists and is accessible
		if _, err := os.Stat(cmd.Dir); err != nil {
			if os.IsNotExist(err) {
				logger.WithField("workDir", cmd.Dir).
					WithField("error", err.Error()).
					Info("Working directory does not exist, using default directory")
				cmd.Dir = "" // Clear invalid directory
			} else if os.IsPermission(err) {
				logger.WithField("workDir", cmd.Dir).
					WithField("error", err.Error()).
					Info("Working directory permission denied, using default directory")
				cmd.Dir = "" // Clear inaccessible directory
			} else {
				// For other errors, log but keep the directory set
				logger.WithField("workDir", cmd.Dir).
					WithField("error", err.Error()).
					Debug("Working directory validation encountered error")
			}
		}
	}

	return cmd
}

// defaultCommandRunner is the actual implementation
type defaultCommandRunner struct{}

func (r *defaultCommandRunner) Run(ctx context.Context, config ExecuteConfig) (string, error) {
	// Create executor to access buildCommand
	executor := &Executor{}
	baseCmd := executor.buildCommandWithValidation(config)

	logger.WithFields(map[string]interface{}{
		"command": baseCmd.Path,
		"args":    baseCmd.Args,
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
	cmd.Dir = baseCmd.Dir // Preserve working directory from buildCommand

	// Run the command
	startTime := time.Now()
	logger.Debug("Executing Claude command")
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error":    err,
			"duration": duration,
			"output":   string(output),
		}).Error("Claude execution failed")
		return "", fmt.Errorf("claude execution failed: %w", err)
	}

	logger.WithField("duration", duration).Debug("Claude execution completed successfully")
	return string(output), nil
}
