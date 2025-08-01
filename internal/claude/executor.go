package claude

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/events"
	"github.com/Backland-Labs/alpine/internal/logger"
	"github.com/Backland-Labs/alpine/internal/output"
)

// ExecuteConfig holds configuration for executing Claude
type ExecuteConfig struct {
	// Prompt is the input prompt to send to Claude
	Prompt string

	// StateFile is the path to the agent_state/agent_state.json file
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

	// EnvironmentVariables are additional environment variables to pass to Claude (optional)
	EnvironmentVariables map[string]string
}

// Executor handles execution of Claude commands
type Executor struct {
	commandRunner CommandRunner
	config        *config.Config
	printer       *output.Printer
	envVars       map[string]string // Additional environment variables to pass to Claude
	streamer      events.Streamer   // Optional streamer for real-time output
	runID         string            // Run ID for stream correlation
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
		envVars:       make(map[string]string),
	}
}

// NewExecutorWithConfig creates a new Claude executor with configuration and printer
func NewExecutorWithConfig(cfg *config.Config, printer *output.Printer) *Executor {
	return &Executor{
		commandRunner: &defaultCommandRunner{},
		config:        cfg,
		printer:       printer,
		envVars:       make(map[string]string),
	}
}

// SetStreamer sets the streamer for real-time output streaming
func (e *Executor) SetStreamer(streamer events.Streamer) {
	e.streamer = streamer
}

// SetRunID sets the run ID for stream correlation
func (e *Executor) SetRunID(runID string) {
	e.runID = runID
}

// Execute runs Claude with the given configuration
func (e *Executor) Execute(ctx context.Context, config ExecuteConfig) (string, error) {
	logger.WithFields(map[string]interface{}{
		"prompt_length": len(config.Prompt),
		"state_file": config.StateFile,
		"mcp_servers": config.MCPServers,
		"allowed_tools": config.AllowedTools,
		"has_system_prompt": config.SystemPrompt != "",
		"timeout": config.Timeout,
		"run_id": e.runID,
	}).Info("Starting Claude execution")

	// Merge executor's environment variables into config
	if e.envVars != nil && len(e.envVars) > 0 {
		logger.WithField("env_var_count", len(e.envVars)).Debug("Merging executor environment variables")
		if config.EnvironmentVariables == nil {
			config.EnvironmentVariables = make(map[string]string)
		}
		for key, value := range e.envVars {
			config.EnvironmentVariables[key] = value
			logger.WithFields(map[string]interface{}{
				"key": key,
				"value_length": len(value),
			}).Debug("Added environment variable")
		}
	}

	// Validate required fields
	if config.Prompt == "" {
		logger.Error("Claude execution validation failed: empty prompt")
		return "", fmt.Errorf("prompt is required")
	}
	if config.StateFile == "" {
		logger.Error("Claude execution validation failed: empty state file")
		return "", fmt.Errorf("state file is required")
	}

	logger.WithFields(map[string]interface{}{
		"state_file":    config.StateFile,
		"mcp_servers":   len(config.MCPServers),
		"allowed_tools": len(config.AllowedTools),
		"prompt_preview": truncateString(config.Prompt, 100),
	}).Info("Claude configuration validated")

	// Check if we should use todo monitoring
	if e.config != nil && e.config.ShowTodoUpdates && e.printer != nil {
		logger.Info("Claude execution will use TODO monitoring")
		return e.executeWithTodoMonitoring(ctx, config)
	}

	// Check if we should capture stderr for tool logs (without todo monitoring)
	// OR if we have streaming enabled
	if (e.config != nil && e.config.ShowToolUpdates && e.printer != nil) || (e.streamer != nil && e.runID != "") {
		logger.WithFields(map[string]interface{}{
			"show_tool_updates": e.config != nil && e.config.ShowToolUpdates,
			"streaming_enabled": e.streamer != nil && e.runID != "",
		}).Info("Claude execution will capture stderr for tool logs or streaming")
		return e.executeClaudeCommand(ctx, config, "")
	}

	// Use command runner (allows for mocking in tests)
	if e.commandRunner != nil {
		logger.Debug("Using injected command runner")
		return e.commandRunner.Run(ctx, config)
	}

	// Default implementation
	logger.Debug("Using default command runner")
	runner := &defaultCommandRunner{}
	return runner.Run(ctx, config)
}

// executeWithTodoMonitoring runs Claude with real-time TODO monitoring
func (e *Executor) executeWithTodoMonitoring(ctx context.Context, config ExecuteConfig) (string, error) {
	logger.WithFields(map[string]interface{}{
		"run_id": e.runID,
		"state_file": config.StateFile,
	}).Debug("Starting Claude execution with TODO monitoring")

	// Setup hook (fallback to normal execution on failure)
	todoFile, cleanup, err := e.setupTodoHook()
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error": err.Error(),
			"run_id": e.runID,
		}).Warn("Failed to setup TODO hook, falling back to normal execution")
		return e.executeWithoutMonitoring(ctx, config)
	}
	logger.WithField("todo_file", todoFile).Debug("TODO hook setup successfully")
	defer func() {
		if cleanup != nil {
			cleanup()
		}
	}()

	// Start monitoring
	monitor := NewTodoMonitor(todoFile)
	monitorCtx, monitorCancel := context.WithCancel(ctx)
	defer monitorCancel()
	logger.WithField("todo_file", todoFile).Debug("Starting TODO monitor")
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
	logger.Debug("Starting printer TODO monitoring")
	e.printer.StartTodoMonitoring()
	defer func() {
		logger.Debug("Stopping printer TODO monitoring")
		e.printer.StopTodoMonitoring()
	}()

	for {
		select {
		case task := <-monitor.Updates():
			logger.WithField("task", task).Debug("Received TODO update")
			e.printer.UpdateCurrentTask(task)
		case result := <-claudeResult:
			logger.WithField("result_length", len(result)).Info("Claude execution completed successfully")
			return result, nil
		case err := <-claudeErr:
			logger.WithField("error", err.Error()).Error("Claude execution failed")
			return "", err
		case <-ctx.Done():
			logger.WithField("error", ctx.Err().Error()).Warn("Claude execution cancelled")
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
		"working_dir": baseCmd.Dir,
		"env_count": len(baseCmd.Env),
	}).Debug("Preparing Claude command with TODO monitoring")

	// Handle timeout
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
		logger.WithFields(map[string]interface{}{
			"timeout": config.Timeout,
			"run_id": e.runID,
		}).Info("Setting Claude execution timeout")
	}

	// Create command with context
	cmd := exec.CommandContext(ctx, baseCmd.Path, baseCmd.Args[1:]...)
	cmd.Env = baseCmd.Env
	cmd.Dir = baseCmd.Dir

	// Add todo file environment variable
	if todoFile != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("ALPINE_TODO_FILE=%s", todoFile))
		logger.WithField("todo_file", todoFile).Debug("Added ALPINE_TODO_FILE environment variable")
	}

	// Check if we should capture stderr for tool logs OR if streaming is enabled
	if (e.config != nil && e.config.ShowToolUpdates && e.printer != nil) || (e.streamer != nil && e.runID != "") {
		return e.executeWithStderrCapture(ctx, cmd)
	}

	// Fallback to combined output
	startTime := time.Now()
	logger.WithFields(map[string]interface{}{
		"command": cmd.Path,
		"args_count": len(cmd.Args),
		"working_dir": cmd.Dir,
	}).Info("Executing Claude command")
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error":    err.Error(),
			"duration": duration.String(),
			"duration_ms": duration.Milliseconds(),
			"output_length": len(output),
			"output_preview": truncateString(string(output), 200),
		}).Error("Claude execution failed")
		return "", fmt.Errorf("claude execution failed: %w", err)
	}

	logger.WithFields(map[string]interface{}{
		"duration": duration.String(),
		"duration_ms": duration.Milliseconds(),
		"output_length": len(output),
	}).Info("Claude execution completed successfully")
	return string(output), nil
}

// executeWithStderrCapture runs a command with separate stdout/stderr handling
// stderr lines are sent to printer.AddToolLog() in real-time
// stdout is streamed in real-time if a streamer is configured
func (e *Executor) executeWithStderrCapture(ctx context.Context, cmd *exec.Cmd) (string, error) {
	startTime := time.Now()

	// Generate message ID for streaming if streamer is available
	var messageID string
	if e.streamer != nil && e.runID != "" {
		messageID = generateMessageID()
		// Start streaming lifecycle
		if err := e.streamer.StreamStart(e.runID, messageID); err != nil {
			logger.WithFields(map[string]interface{}{
				"error":     err,
				"runID":     e.runID,
				"messageID": messageID,
			}).Debug("Failed to start streaming")
		}
	}

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
	logger.WithFields(map[string]interface{}{
		"command": cmd.Path,
		"working_dir": cmd.Dir,
		"streaming": e.streamer != nil,
		"run_id": e.runID,
	}).Info("Starting Claude command with stderr capture")
	if err := cmd.Start(); err != nil {
		logger.WithField("error", err.Error()).Error("Failed to start Claude command")
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	// Capture stdout
	var stdoutBuf bytes.Buffer
	var wg sync.WaitGroup

	// Read stdout in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Create writers based on streaming configuration
		var writer io.Writer = &stdoutBuf

		// If streaming is enabled, use MultiWriter to stream and capture
		if e.streamer != nil && e.runID != "" && messageID != "" {
			streamWriter := NewStreamWriter(e.streamer, e.runID, messageID)
			multiWriter := newMultiWriterWithFlush(&stdoutBuf, streamWriter)
			writer = multiWriter

			// Use TeeReader to read and write simultaneously
			reader := io.TeeReader(stdoutPipe, writer)
			n, err := io.Copy(io.Discard, reader)
			logger.WithFields(map[string]interface{}{
				"bytes_streamed": n,
				"run_id": e.runID,
				"message_id": messageID,
			}).Debug("Finished streaming stdout")
			if err != nil {
				logger.WithField("error", err.Error()).Error("Error during stdout streaming")
			}

			// Flush any remaining buffered content
			if err := multiWriter.Flush(); err != nil {
				logger.WithField("error", err.Error()).Debug("Error flushing stream writer")
			}
		} else {
			// No streaming, just capture to buffer
			if _, err := io.Copy(&stdoutBuf, stdoutPipe); err != nil {
				logger.WithField("error", err).Error("Error reading stdout")
			}
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

	// End streaming lifecycle if enabled
	if e.streamer != nil && e.runID != "" && messageID != "" {
		if err := e.streamer.StreamEnd(e.runID, messageID); err != nil {
			logger.WithFields(map[string]interface{}{
				"error":     err,
				"runID":     e.runID,
				"messageID": messageID,
			}).Debug("Failed to end streaming")
		}
	}

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error":    err.Error(),
			"duration": duration.String(),
			"duration_ms": duration.Milliseconds(),
			"output_length": len(output),
			"output_preview": truncateString(output, 200),
			"run_id": e.runID,
		}).Error("Claude execution failed with stderr capture")
		return "", fmt.Errorf("claude execution failed: %w", err)
	}

	logger.WithFields(map[string]interface{}{
		"duration": duration.String(),
		"duration_ms": duration.Milliseconds(),
		"output_length": len(output),
		"run_id": e.runID,
	}).Info("Claude execution completed successfully with stderr capture")
	return output, nil
}

// generateMessageID generates a unique message ID for streaming correlation
func generateMessageID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("msg-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("msg-%s", hex.EncodeToString(bytes))
}

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
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
		// Claude will use default behavior (inherit from parent process)
		logger.WithField("error", err.Error()).Warn("Failed to get working directory, Claude will use default directory")
	} else {
		cmd.Dir = workDir
		logger.WithFields(map[string]interface{}{
			"workDir": workDir,
			"prompt_preview": truncateString(config.Prompt, 50),
		}).Info("Set Claude working directory")
	}

	// Set environment variables
	cmd.Env = os.Environ()

	// Add any additional environment variables from config
	if config.EnvironmentVariables != nil && len(config.EnvironmentVariables) > 0 {
		logger.WithField("env_var_count", len(config.EnvironmentVariables)).Debug("Adding additional environment variables")
		for key, value := range config.EnvironmentVariables {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
			logger.WithFields(map[string]interface{}{
				"key": key,
				"value_length": len(value),
			}).Debug("Added environment variable to Claude command")
		}
	}

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
		"working_dir": baseCmd.Dir,
		"prompt_preview": truncateString(config.Prompt, 50),
	}).Info("Preparing Claude command")

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
	logger.WithFields(map[string]interface{}{
		"command": cmd.Path,
		"working_dir": cmd.Dir,
	}).Info("Executing Claude command")
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error":    err.Error(),
			"duration": duration.String(),
			"duration_ms": duration.Milliseconds(),
			"output_length": len(output),
			"output_preview": truncateString(string(output), 200),
		}).Error("Claude execution failed")
		return "", fmt.Errorf("claude execution failed: %w", err)
	}

	logger.WithFields(map[string]interface{}{
		"duration": duration.String(),
		"duration_ms": duration.Milliseconds(),
		"output_length": len(output),
		}).Info("Claude execution completed successfully")
	return string(output), nil
}
