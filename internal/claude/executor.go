package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	// defaultTimeout is the default command timeout in seconds if not specified
	defaultTimeout = 120
)

// Execute runs a Claude command with the given options and returns the response
func (c *commandBuilder) Execute(ctx context.Context, cmd Command, opts CommandOptions) (*Response, error) {
	// Validate working directory if specified
	if opts.WorkingDir != "" {
		if _, err := os.Stat(opts.WorkingDir); err != nil {
			return nil, fmt.Errorf("invalid working directory: %w", err)
		}
	}

	// Build the command arguments
	args, err := c.BuildCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	// Remove the binary name from args since exec.Command expects it separately
	if len(args) > 0 && args[0] == "claude" {
		args = args[1:]
	}

	// Create the command
	execCmd := exec.CommandContext(ctx, "claude", args...)

	// Set working directory if specified
	if opts.WorkingDir != "" {
		execCmd.Dir = opts.WorkingDir
	}

	// Set up output buffers
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	// Create a channel to signal completion
	done := make(chan error, 1)

	// Start the command
	err = execCmd.Start()
	if err != nil {
		// Check if the command was not found
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("claude command not found: %w", err)
		}
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Wait for command completion in a goroutine
	go func() {
		done <- execCmd.Wait()
	}()

	// Create a timer for timeout
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	timer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timer.Stop()

	// Wait for completion, timeout, or context cancellation
	select {
	case <-ctx.Done():
		// Context was cancelled
		if killErr := execCmd.Process.Kill(); killErr != nil {
			// Log kill error but return context error
			fmt.Fprintf(os.Stderr, "failed to kill process: %v\n", killErr)
		}
		return nil, ctx.Err()

	case <-timer.C:
		// Command timed out
		if killErr := execCmd.Process.Kill(); killErr != nil {
			fmt.Fprintf(os.Stderr, "failed to kill process: %v\n", killErr)
		}
		return nil, fmt.Errorf("command timed out after %d seconds", timeout)

	case err := <-done:
		// Command completed
		if err != nil {
			// Check if it's an exit error
			if exitErr, ok := err.(*exec.ExitError); ok {
				return nil, fmt.Errorf("command failed with exit code %d: %s", exitErr.ExitCode(), stderr.String())
			}
			return nil, fmt.Errorf("command failed: %w", err)
		}
	}

	// Parse the response
	output := stdout.String()
	resp, err := c.ParseResponse(ctx, output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// If there was stderr output, add it to the response error
	if stderrStr := stderr.String(); stderrStr != "" {
		resp.Error = strings.TrimSpace(stderrStr)
	}

	return resp, nil
}

// ParseResponse parses the Claude command output into a Response struct
func (c *commandBuilder) ParseResponse(ctx context.Context, output string) (*Response, error) {
	// Handle empty output
	if output == "" {
		return &Response{
			Content:      "",
			ContinueFlag: false,
		}, nil
	}

	// Try to parse as JSON first
	var jsonResp struct {
		Content  string `json:"content"`
		Continue bool   `json:"continue"`
	}

	if err := json.Unmarshal([]byte(output), &jsonResp); err == nil {
		// Successfully parsed as JSON
		return &Response{
			Content:      jsonResp.Content,
			ContinueFlag: jsonResp.Continue,
		}, nil
	}

	// If JSON parsing failed, treat as text and look for continue signals
	continueFlag := parseTextContinueFlag(output)
	
	return &Response{
		Content:      output,
		ContinueFlag: continueFlag,
	}, nil
}

// parseTextContinueFlag extracts continue flag from text output
func parseTextContinueFlag(text string) bool {
	lowerText := strings.ToLower(text)
	
	// Look for explicit continue signals
	if strings.Contains(lowerText, "continue=true") {
		return true
	}
	if strings.Contains(lowerText, "continue=false") {
		return false
	}
	
	// Default to false if no explicit signal found
	return false
}

// CheckStatusFile reads claude_status.json to determine continuation status
func CheckStatusFile(workingDir string) bool {
	statusPath := filepath.Join(workingDir, "claude_status.json")
	
	data, err := os.ReadFile(statusPath)
	if err != nil {
		// Continue if no status file (matches Python behavior)
		return true
	}
	
	var status StatusFile
	if err := json.Unmarshal(data, &status); err != nil {
		// Continue if invalid JSON (matches Python behavior)
		return true
	}
	
	return strings.ToLower(status.Continue) == "yes"
}

// CleanupStatusFile removes claude_status.json from the working directory
func CleanupStatusFile(workingDir string) {
	statusPath := filepath.Join(workingDir, "claude_status.json")
	os.Remove(statusPath) // Ignore errors like Python version
}
