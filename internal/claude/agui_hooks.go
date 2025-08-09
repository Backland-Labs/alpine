package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Backland-Labs/alpine/internal/logger"
)

// SetupAgUIHooks sets up the ag-ui event emitter hooks for HTTP mode
// Returns a cleanup function and error
func (e *Executor) SetupAgUIHooks(eventEndpoint, runID string) (cleanup func(), err error) {
	logger.Debug("Setting up ag-ui event emitter hooks")

	// Create .claude directory in current working directory
	claudeDir := ".claude"
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Get absolute path to ag-ui hook script
	hookScriptPath, err := e.GetAgUIHookScriptPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get ag-ui hook script path: %w", err)
	}

	// Verify hook script exists
	if _, err := os.Stat(hookScriptPath); err != nil {
		return nil, fmt.Errorf("ag-ui hook script not found at %s: %w", hookScriptPath, err)
	}

	// Generate Claude settings
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := e.generateAgUIClaudeSettings(settingsPath, hookScriptPath); err != nil {
		return nil, fmt.Errorf("failed to generate Claude settings: %w", err)
	}

	// Store environment variables needed for hooks in executor
	if e.envVars == nil {
		e.envVars = make(map[string]string)
	}
	e.envVars["ALPINE_EVENTS_ENDPOINT"] = eventEndpoint
	e.envVars["ALPINE_RUN_ID"] = runID

	logger.WithFields(map[string]interface{}{
		"hook_script":    hookScriptPath,
		"settings_file":  settingsPath,
		"event_endpoint": eventEndpoint,
		"run_id":         runID,
	}).Debug("ag-ui hooks setup completed")

	// Return cleanup function
	cleanup = func() {
		logger.Debug("Cleaning up ag-ui hooks")
		_ = os.Remove(settingsPath)
		// Don't remove .claude directory - may contain user's own settings

		// Clear environment variables
		delete(e.envVars, "ALPINE_EVENTS_ENDPOINT")
		delete(e.envVars, "ALPINE_RUN_ID")
	}

	return cleanup, nil
}

// GetAgUIHookScriptPath returns the absolute path to the ag-ui hook script
func (e *Executor) GetAgUIHookScriptPath() (string, error) {
	// Get the current executable path to find the hooks directory
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Get the directory containing the executable
	execDir := filepath.Dir(execPath)

	// Try different paths where the hook script might be located
	possiblePaths := []string{
		// Development environment - hooks at project root
		filepath.Join(execDir, "..", "..", "hooks", "alpine-ag-ui-emitter"),
		// Installed environment - hooks relative to binary
		filepath.Join(execDir, "hooks", "alpine-ag-ui-emitter"),
		// Current working directory (for testing)
		filepath.Join("hooks", "alpine-ag-ui-emitter"),
	}

	for _, path := range possiblePaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			return absPath, nil
		}
	}

	return "", fmt.Errorf("ag-ui hook script not found in any expected location")
}

// SetupToolCallEventHooks sets up hooks for tool call event capture with PreToolUse and PostToolUse
// Returns a cleanup function and error
func (e *Executor) SetupToolCallEventHooks(eventEndpoint, runID string, batchSize, sampleRate int) (cleanup func(), err error) {
	logger.Debug("Setting up tool call event hooks")

	// Create .claude directory in current working directory
	claudeDir := ".claude"
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Get absolute path to ag-ui hook script
	hookScriptPath, err := e.GetAgUIHookScriptPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get ag-ui hook script path: %w", err)
	}

	// Verify hook script exists
	if _, err := os.Stat(hookScriptPath); err != nil {
		return nil, fmt.Errorf("ag-ui hook script not found at %s: %w", hookScriptPath, err)
	}

	// Generate Claude settings with both PreToolUse and PostToolUse hooks
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := e.generateToolCallEventClaudeSettings(settingsPath, hookScriptPath); err != nil {
		return nil, fmt.Errorf("failed to generate Claude settings: %w", err)
	}

	// Store environment variables needed for hooks in executor
	if e.envVars == nil {
		e.envVars = make(map[string]string)
	}
	e.envVars["ALPINE_EVENTS_ENDPOINT"] = eventEndpoint
	e.envVars["ALPINE_RUN_ID"] = runID
	e.envVars["ALPINE_TOOL_CALL_BATCH_SIZE"] = fmt.Sprintf("%d", batchSize)
	e.envVars["ALPINE_TOOL_CALL_SAMPLE_RATE"] = fmt.Sprintf("%d", sampleRate)

	logger.WithFields(map[string]interface{}{
		"hook_script":    hookScriptPath,
		"settings_file":  settingsPath,
		"event_endpoint": eventEndpoint,
		"run_id":         runID,
		"batch_size":     batchSize,
		"sample_rate":    sampleRate,
	}).Debug("tool call event hooks setup completed")

	// Return cleanup function
	cleanup = func() {
		logger.Debug("Cleaning up tool call event hooks")
		_ = os.Remove(settingsPath)
		// Don't remove .claude directory - may contain user's own settings

		// Clear environment variables
		delete(e.envVars, "ALPINE_EVENTS_ENDPOINT")
		delete(e.envVars, "ALPINE_RUN_ID")
		delete(e.envVars, "ALPINE_TOOL_CALL_BATCH_SIZE")
		delete(e.envVars, "ALPINE_TOOL_CALL_SAMPLE_RATE")
	}

	return cleanup, nil
}

// generateAgUIClaudeSettings creates the Claude Code settings.json file with ag-ui hook configuration
func (e *Executor) generateAgUIClaudeSettings(settingsPath, hookScriptPath string) error {
	settings := hookSettings{
		Hooks: map[string]interface{}{
			"PostToolUse": []toolMatcher{
				{
					Matcher: ".*",
					Hooks: []map[string]interface{}{
						{
							"command": hookScriptPath,
							"type":    "command",
						},
					},
				},
			},
		},
	}

	// Marshal to JSON
	settingsJSON, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Write settings file
	if err := os.WriteFile(settingsPath, settingsJSON, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	logger.WithField("settings_path", settingsPath).Debug("Generated ag-ui Claude settings file")
	return nil
}

// generateToolCallEventClaudeSettings creates Claude settings with both PreToolUse and PostToolUse hooks
func (e *Executor) generateToolCallEventClaudeSettings(settingsPath, hookScriptPath string) error {
	settings := hookSettings{
		Hooks: map[string]interface{}{
			"PreToolUse": []toolMatcher{
				{
					Matcher: ".*",
					Hooks: []map[string]interface{}{
						{
							"command": hookScriptPath,
							"type":    "command",
						},
					},
				},
			},
			"PostToolUse": []toolMatcher{
				{
					Matcher: ".*",
					Hooks: []map[string]interface{}{
						{
							"command": hookScriptPath,
							"type":    "command",
						},
					},
				},
			},
		},
	}

	// Marshal to JSON
	settingsJSON, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Write settings file
	if err := os.WriteFile(settingsPath, settingsJSON, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	logger.WithField("settings_path", settingsPath).Debug("Generated tool call event Claude settings file")
	return nil
}

// ExecuteHookWithResilience executes a hook function with circuit breaker protection
// Ensures workflow continues even if hook fails
func (e *Executor) ExecuteHookWithResilience(ctx context.Context, hookName string, hookFunc func() error) error {
	// Initialize circuit breaker if not exists
	if e.hookCircuitBreaker == nil {
		e.hookCircuitBreaker = NewCircuitBreaker(5, 30*time.Second) // 5 failures, 30s recovery
	}

	// Check if circuit breaker allows the call
	if !e.hookCircuitBreaker.CanCall() {
		logger.WithFields(map[string]interface{}{
			"hook_name": hookName,
			"state":     "circuit_open",
		}).Debug("Hook execution blocked by circuit breaker")
		return nil // Don't fail the workflow
	}

	// Execute the hook with timeout
	done := make(chan error, 1)
	go func() {
		done <- hookFunc()
	}()

	select {
	case err := <-done:
		if err != nil {
			e.hookCircuitBreaker.RecordFailure()
			logger.WithFields(map[string]interface{}{
				"hook_name": hookName,
				"error":     err.Error(),
			}).Warn("Hook execution failed, recorded failure")
			return nil // Don't fail the workflow
		}
		e.hookCircuitBreaker.RecordSuccess()
		logger.WithFields(map[string]interface{}{
			"hook_name": hookName,
		}).Debug("Hook executed successfully")
		return nil
	case <-ctx.Done():
		logger.WithFields(map[string]interface{}{
			"hook_name": hookName,
			"error":     "context_cancelled",
		}).Debug("Hook execution cancelled")
		return nil // Don't fail the workflow
	case <-time.After(10 * time.Second): // Hook timeout
		e.hookCircuitBreaker.RecordFailure()
		logger.WithFields(map[string]interface{}{
			"hook_name": hookName,
			"error":     "timeout",
		}).Warn("Hook execution timed out")
		return nil // Don't fail the workflow
	}
}
