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
	logger.Debug("Starting Claude execution")

	// Merge executor's environment variables into config
	if e.envVars != nil && len(e.envVars) > 0 {
		if config.EnvironmentVariables == nil {
			config.EnvironmentVariables = make(map[string]string)
		}
		for key, value := range e.envVars {
			config.EnvironmentVariables[key] = value
		}
	}

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

	// Check if we should capture stderr for tool logs (without todo monitoring)
	// OR if we have streaming enabled
	if (e.config != nil && e.config.ShowToolUpdates && e.printer != nil) || (e.streamer != nil && e.runID != "") {
		return e.executeClaudeCommand(ctx, config, "")
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
	cmd.Env = append(cmd.Env, fmt.Sprintf("ALPINE_TODO_FILE=%s", todoFile))

	// Check if we should capture stderr for tool logs OR if streaming is enabled
	if (e.config != nil && e.config.ShowToolUpdates && e.printer != nil) || (e.streamer != nil && e.runID != "") {
		return e.executeWithStderrCapture(ctx, cmd)
	}

	// Fallback to combined output
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

		// Create writers based on streaming configuration
		var writer io.Writer = &stdoutBuf

		// If streaming is enabled, use MultiWriter to stream and capture
		if e.streamer != nil && e.runID != "" && messageID != "" {
			streamWriter := NewStreamWriter(e.streamer, e.runID, messageID)
			multiWriter := newMultiWriterWithFlush(&stdoutBuf, streamWriter)
			writer = multiWriter

			// Use TeeReader to read and write simultaneously
			reader := io.TeeReader(stdoutPipe, writer)
			io.Copy(io.Discard, reader)

			// Flush any remaining buffered content
			if err := multiWriter.Flush(); err != nil {
				logger.WithField("error", err).Debug("Error flushing stream writer")
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
			"error":    err,
			"duration": duration,
			"output":   output,
		}).Error("Claude execution failed")
		return "", fmt.Errorf("claude execution failed: %w", err)
	}

	logger.WithField("duration", duration).Debug("Claude execution completed successfully with stderr capture")
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
		logger.WithField("error", err).Info("Failed to get working directory, Claude will use default directory")
	} else {
		cmd.Dir = workDir
		logger.WithField("workDir", workDir).Debug("Set Claude working directory")
	}

	// Set environment variables
	cmd.Env = os.Environ()

	// Add any additional environment variables from config
	if config.EnvironmentVariables != nil {
		for key, value := range config.EnvironmentVariables {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
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
