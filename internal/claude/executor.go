package claude

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
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

	// AdditionalArgs allows passing custom CLI arguments to Claude (optional)
	AdditionalArgs []string
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

// executionOptions represents unified options for command execution
type executionOptions struct {
	enableTodoMonitoring bool
	enableStderrCapture  bool
	todoFile            string
	timeout             time.Duration
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

	// Validate configuration
	if err := e.validateConfig(config); err != nil {
		return "", err
	}

	logger.WithFields(map[string]interface{}{
		"state_file":    config.StateFile,
		"mcp_servers":   len(config.MCPServers),
		"allowed_tools": len(config.AllowedTools),
	}).Debug("Claude configuration validated")

	// Determine execution options based on configuration
	opts := e.determineExecutionOptions(config)

	// Setup TODO monitoring if enabled
	if opts.enableTodoMonitoring {
		todoFile, cleanup, err := e.setupTodoHook()
		if err != nil {
			logger.WithField("error", err).Info("Failed to setup TODO hook, disabling monitoring")
			opts.enableTodoMonitoring = false
		} else {
			opts.todoFile = todoFile
			defer func() {
				if cleanup != nil {
					cleanup()
				}
			}()
		}
	}

	// Execute command with unified pipeline
	return e.executeCommand(ctx, config, opts)
}

// validateConfig validates the required fields in ExecuteConfig
func (e *Executor) validateConfig(config ExecuteConfig) error {
	if config.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	if config.StateFile == "" {
		return fmt.Errorf("state file is required")
	}
	return nil
}

// determineExecutionOptions determines the execution options based on configuration
func (e *Executor) determineExecutionOptions(config ExecuteConfig) executionOptions {
	opts := executionOptions{
		timeout: config.Timeout,
	}

	// Only enable features if we have both config and printer
	if e.config != nil && e.printer != nil {
		opts.enableTodoMonitoring = e.config.ShowTodoUpdates
		opts.enableStderrCapture = e.config.ShowToolUpdates
	}

	return opts
}

// executeCommand executes Claude with the given options using a unified pipeline
func (e *Executor) executeCommand(ctx context.Context, config ExecuteConfig, opts executionOptions) (string, error) {
	// For mocking in tests
	if e.commandRunner != nil && !opts.enableTodoMonitoring {
		return e.commandRunner.Run(ctx, config)
	}

	// Build command once
	cmd := e.buildCommandWithValidation(config)

	// Apply timeout if specified
	if opts.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.timeout)
		defer cancel()
		logger.WithField("timeout", opts.timeout).Debug("Setting execution timeout")
	}

	// Create exec command with context
	execCmd := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	execCmd.Env = cmd.Env
	execCmd.Dir = cmd.Dir

	// Add TODO file environment variable if monitoring is enabled
	if opts.enableTodoMonitoring && opts.todoFile != "" {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("RIVER_TODO_FILE=%s", opts.todoFile))
	}

	// Handle TODO monitoring if enabled
	if opts.enableTodoMonitoring {
		return e.executeWithMonitoring(ctx, execCmd, config, opts)
	}

	// Handle stderr capture if enabled
	if opts.enableStderrCapture {
		return e.executeWithStderrCapture(ctx, execCmd)
	}

	// Default execution with combined output
	return e.runCommand(ctx, execCmd)
}

// executeWithMonitoring handles execution with TODO monitoring
func (e *Executor) executeWithMonitoring(ctx context.Context, cmd *exec.Cmd, config ExecuteConfig, opts executionOptions) (string, error) {
	// Start monitoring
	monitor := NewTodoMonitor(opts.todoFile)
	monitorCtx, monitorCancel := context.WithCancel(ctx)
	defer monitorCancel()
	go monitor.Start(monitorCtx)

	// Start Claude execution
	claudeResult := make(chan string, 1)
	claudeErr := make(chan error, 1)
	go func() {
		var result string
		var err error
		if opts.enableStderrCapture {
			result, err = e.executeWithStderrCapture(ctx, cmd)
		} else {
			result, err = e.runCommand(ctx, cmd)
		}
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

// runCommand executes a command and returns combined output
func (e *Executor) runCommand(ctx context.Context, cmd *exec.Cmd) (string, error) {
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


// executeWithStderrCapture runs a command with separate stdout/stderr handling
// stderr lines are sent to printer.AddToolLog() in real-time
func (e *Executor) executeWithStderrCapture(ctx context.Context, cmd *exec.Cmd) (string, error) {
	startTime := time.Now()

	// Get stdout pipe
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Get stderr pipe
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	logger.Debug("Starting Claude command with stderr capture")
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	// Capture stdout
	var stdoutBuf bytes.Buffer
	var wg sync.WaitGroup

	// Read stdout in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := io.Copy(&stdoutBuf, stdoutPipe); err != nil {
			logger.WithField("error", err).Error("Error reading stdout")
		}
	}()

	// Read stderr line-by-line and send to AddToolLog
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if e.printer != nil {
				e.printer.AddToolLog(line)
				e.printer.RenderToolLogs()
			}
			logger.WithField("stderr", line).Debug("Captured stderr line")
		}
		if err := scanner.Err(); err != nil {
			logger.WithField("error", err).Error("Error reading stderr")
		}
	}()

	// Wait for command to complete
	err = cmd.Wait()
	duration := time.Since(startTime)

	// Wait for all goroutines to finish reading
	wg.Wait()

	output := stdoutBuf.String()

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error":    err,
			"duration": duration,
			"output":   output,
		}).Error("Claude execution failed")
		return "", fmt.Errorf("claude execution failed: %w", err)
	}

	logger.WithField("duration", duration).Debug("Claude execution completed successfully with stderr capture")
	return output, nil
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

// buildCommand constructs the exec.Cmd for Claude with validation
func (e *Executor) buildCommand(config ExecuteConfig) (*exec.Cmd, error) {
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

	// Add any additional arguments
	if len(config.AdditionalArgs) > 0 {
		args = append(args, config.AdditionalArgs...)
	}

	// Add the prompt with -p flag
	args = append(args, "-p", config.Prompt)

	// Create command
	cmd := exec.Command("claude", args...)

	// Set working directory to current directory (enables worktree isolation)
	workDir, err := os.Getwd()
	if err != nil {
		// Log warning but continue without setting Dir
		logger.WithField("error", err).Info("Failed to get working directory, Claude will use default directory")
	} else {
		// Validate the directory exists and is accessible
		if _, err := os.Stat(workDir); err != nil {
			if os.IsNotExist(err) {
				logger.WithField("workDir", workDir).Info("Working directory does not exist, using default directory")
			} else if os.IsPermission(err) {
				logger.WithField("workDir", workDir).Info("Working directory permission denied, using default directory")
			} else {
				// For other errors, set the directory anyway
				cmd.Dir = workDir
				logger.WithField("workDir", workDir).Debug("Set Claude working directory")
			}
		} else {
			cmd.Dir = workDir
			logger.WithField("workDir", workDir).Debug("Set Claude working directory")
		}
	}

	// Set environment variables
	cmd.Env = os.Environ()

	return cmd, nil
}

// buildCommandWithValidation is a compatibility wrapper that returns *exec.Cmd
func (e *Executor) buildCommandWithValidation(config ExecuteConfig) *exec.Cmd {
	cmd, _ := e.buildCommand(config)
	return cmd
}

// defaultCommandRunner is the actual implementation
type defaultCommandRunner struct{}

func (r *defaultCommandRunner) Run(ctx context.Context, config ExecuteConfig) (string, error) {
	// Create a temporary executor to leverage the unified execution pipeline
	executor := &Executor{commandRunner: nil}
	
	// Determine options (without special features since we don't have config/printer)
	opts := executionOptions{
		timeout: config.Timeout,
	}
	
	// Use the unified execution command
	return executor.executeCommand(ctx, config, opts)
}
