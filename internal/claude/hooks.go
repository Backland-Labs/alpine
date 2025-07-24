package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/maxmcd/river/internal/hooks"
	"github.com/maxmcd/river/internal/logger"
)

// hookSettings represents Claude Code settings for hooks
type hookSettings struct {
	Hooks map[string]interface{} `json:"hooks"`
}

// toolMatcher represents a hook configuration for a specific tool
type toolMatcher struct {
	Matcher string                   `json:"matcher"`
	Hooks   []map[string]interface{} `json:"hooks"`
}

// setupTodoHook sets up the TodoWrite PostToolUse hook for Claude Code
// Returns the todo file path and cleanup function
func (e *Executor) setupTodoHook() (todoFilePath string, cleanup func(), err error) {
	logger.Debug("Setting up TodoWrite hook")

	// Create temporary file for todo updates
	todoFile, err := os.CreateTemp("", "river-todo-*.txt")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create todo file: %w", err)
	}
	todoFilePath = todoFile.Name()
	if err := todoFile.Close(); err != nil {
		_ = os.Remove(todoFilePath)
		return "", nil, fmt.Errorf("failed to close todo file: %w", err)
	}

	// Create .claude directory in current working directory
	claudeDir := ".claude"
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		_ = os.Remove(todoFilePath)
		return "", nil, fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Copy hook script to .claude directory
	hookScriptPath := filepath.Join(claudeDir, "todo-monitor.rs")
	if err := e.copyHookScript(hookScriptPath); err != nil {
		_ = os.Remove(todoFilePath)
		return "", nil, fmt.Errorf("failed to copy hook script: %w", err)
	}

	// Make hook script executable
	if err := os.Chmod(hookScriptPath, 0755); err != nil {
		_ = os.Remove(todoFilePath)
		_ = os.Remove(hookScriptPath)
		return "", nil, fmt.Errorf("failed to make hook script executable: %w", err)
	}

	// Convert hook script path to absolute path
	absHookScriptPath, err := filepath.Abs(hookScriptPath)
	if err != nil {
		_ = os.Remove(todoFilePath)
		_ = os.Remove(hookScriptPath)
		return "", nil, fmt.Errorf("failed to get absolute path of hook script: %w", err)
	}

	// Generate Claude settings
	settingsPath := filepath.Join(claudeDir, "settings.local.json")
	if err := e.generateClaudeSettings(settingsPath, absHookScriptPath); err != nil {
		_ = os.Remove(todoFilePath)
		_ = os.Remove(hookScriptPath)
		return "", nil, fmt.Errorf("failed to generate Claude settings: %w", err)
	}

	logger.WithFields(map[string]interface{}{
		"todo_file":     todoFilePath,
		"hook_script":   absHookScriptPath,
		"settings_file": settingsPath,
	}).Debug("TodoWrite hook setup completed")

	// Return cleanup function
	cleanup = func() {
		logger.Debug("Cleaning up TodoWrite hook")
		_ = os.Remove(todoFilePath)
		_ = os.Remove(hookScriptPath)
		_ = os.Remove(settingsPath)
		// Don't remove .claude directory - may contain user's own settings
	}

	return todoFilePath, cleanup, nil
}

// copyHookScript copies the embedded hook script to the specified path
func (e *Executor) copyHookScript(destPath string) error {
	// Get the hook script content
	scriptContent, err := hooks.GetTodoMonitorScript()
	if err != nil {
		return fmt.Errorf("failed to get hook script: %w", err)
	}

	// Write script to destination
	if err := os.WriteFile(destPath, []byte(scriptContent), 0644); err != nil {
		return fmt.Errorf("failed to write hook script: %w", err)
	}

	// Explicitly set executable permissions
	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	return nil
}

// generateClaudeSettings creates the Claude Code settings.local.json file with hook configuration
func (e *Executor) generateClaudeSettings(settingsPath, hookScriptPath string) error {
	settings := hookSettings{
		Hooks: map[string]interface{}{
			"PostToolUse": []toolMatcher{
				{
					Matcher: "",
					Hooks: []map[string]interface{}{
						{
							"command": hookScriptPath,
							"type":    "command",
						},
					},
				},
			},
			"SubagentStop": []toolMatcher{
				{
					Matcher: "",
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

	logger.WithField("settings_path", settingsPath).Debug("Generated Claude settings file")
	return nil
}
