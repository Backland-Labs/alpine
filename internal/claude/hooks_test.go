package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxmcd/river/internal/config"
	"github.com/maxmcd/river/internal/output"
)

func TestExecutor_setupTodoHook(t *testing.T) {
	// Test the setupTodoHook functionality
	t.Run("creates hook files and returns cleanup function", func(t *testing.T) {
		executor := &Executor{
			config:  &config.Config{},
			printer: output.NewPrinterWithWriters(os.Stdout, os.Stderr, false),
		}

		todoFile, cleanup, err := executor.setupTodoHook()
		if err != nil {
			t.Fatalf("setupTodoHook failed: %v", err)
		}

		// Verify todo file was created
		if _, err := os.Stat(todoFile); os.IsNotExist(err) {
			t.Error("Todo file was not created")
		}

		// Verify .claude directory was created
		claudeDir := ".claude"
		if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
			t.Error(".claude directory was not created")
		}

		// Verify settings.json was created
		settingsFile := filepath.Join(claudeDir, "settings.json")
		if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
			t.Error("settings.json was not created")
		}

		// Verify hook script was created
		hookScriptPath := filepath.Join(claudeDir, "todo-monitor.sh")
		if info, err := os.Stat(hookScriptPath); os.IsNotExist(err) {
			t.Error("Hook script was not created")
		} else if info.Mode()&0111 == 0 {
			t.Error("Hook script is not executable")
		}

		// Test cleanup
		cleanup()

		// Verify files were removed
		if _, err := os.Stat(todoFile); !os.IsNotExist(err) {
			t.Error("Todo file was not cleaned up")
		}
		if _, err := os.Stat(settingsFile); !os.IsNotExist(err) {
			t.Error("settings.json was not cleaned up")
		}
		if _, err := os.Stat(hookScriptPath); !os.IsNotExist(err) {
			t.Error("Hook script was not cleaned up")
		}
	})

	t.Run("cleanup handles missing files gracefully", func(t *testing.T) {
		executor := &Executor{
			config:  &config.Config{},
			printer: output.NewPrinterWithWriters(os.Stdout, os.Stderr, false),
		}

		todoFile, cleanup, err := executor.setupTodoHook()
		if err != nil {
			t.Fatalf("setupTodoHook failed: %v", err)
		}

		// Remove files manually before cleanup
		os.Remove(todoFile)
		os.RemoveAll(".claude")

		// Cleanup should not panic
		cleanup()
	})
}

func TestExecutor_generateClaudeSettings(t *testing.T) {
	// Test Claude settings generation
	t.Run("generates valid JSON settings", func(t *testing.T) {
		executor := &Executor{
			config:  &config.Config{},
			printer: output.NewPrinterWithWriters(os.Stdout, os.Stderr, false),
		}

		tmpDir := t.TempDir()
		settingsPath := filepath.Join(tmpDir, "settings.json")
		hookPath := "/tmp/test-hook.sh"

		err := executor.generateClaudeSettings(settingsPath, hookPath)
		if err != nil {
			t.Fatalf("generateClaudeSettings failed: %v", err)
		}

		// Read and parse the generated settings
		content, err := os.ReadFile(settingsPath)
		if err != nil {
			t.Fatalf("Failed to read settings file: %v", err)
		}

		// Verify it's valid JSON
		var parsed map[string]interface{}
		err = json.Unmarshal(content, &parsed)
		if err != nil {
			t.Fatalf("Generated settings is not valid JSON: %v", err)
		}

		// Verify structure
		hooks, ok := parsed["hooks"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing or invalid 'hooks' key")
		}

		postToolUse, ok := hooks["PostToolUse"].([]interface{})
		if !ok {
			t.Fatal("Missing or invalid 'PostToolUse' key")
		}

		if len(postToolUse) != 1 {
			t.Fatalf("Expected 1 PostToolUse entry, got %d", len(postToolUse))
		}

		entry, ok := postToolUse[0].(map[string]interface{})
		if !ok {
			t.Fatal("Invalid PostToolUse entry structure")
		}

		if entry["matcher"] != "TodoWrite" {
			t.Errorf("Expected matcher 'TodoWrite', got '%v'", entry["matcher"])
		}

		hooksList, ok := entry["hooks"].([]interface{})
		if !ok || len(hooksList) != 1 {
			t.Fatal("Invalid hooks list in entry")
		}

		hook, ok := hooksList[0].(map[string]interface{})
		if !ok {
			t.Fatal("Invalid hook structure")
		}

		if hook["type"] != "command" {
			t.Errorf("Expected hook type 'command', got '%v'", hook["type"])
		}

		if hook["command"] != hookPath {
			t.Errorf("Expected command '%s', got '%v'", hookPath, hook["command"])
		}
	})
}

func TestExecutor_copyHookScript(t *testing.T) {
	// Test hook script copying
	t.Run("copies script with correct permissions", func(t *testing.T) {
		executor := &Executor{
			config:  &config.Config{},
			printer: output.NewPrinterWithWriters(os.Stdout, os.Stderr, false),
		}

		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "todo-monitor.sh")

		err := executor.copyHookScript(scriptPath)
		if err != nil {
			t.Fatalf("copyHookScript failed: %v", err)
		}

		// Verify file exists
		info, err := os.Stat(scriptPath)
		if err != nil {
			t.Fatalf("Script file not found: %v", err)
		}

		// Verify it's executable
		if info.Mode()&0111 == 0 {
			t.Error("Script is not executable")
		}

		// Verify content
		content, err := os.ReadFile(scriptPath)
		if err != nil {
			t.Fatalf("Failed to read script: %v", err)
		}

		if !strings.Contains(string(content), "#!/bin/bash") {
			t.Error("Script missing shebang")
		}

		if !strings.Contains(string(content), "TodoWrite") {
			t.Error("Script missing TodoWrite matcher logic")
		}
	})
}
