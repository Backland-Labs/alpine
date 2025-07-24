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

		// Verify settings.local.json was created
		settingsFile := filepath.Join(claudeDir, "settings.local.json")
		if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
			t.Error("settings.local.json was not created")
		}

		// Verify hook script was created
		hookScriptPath := filepath.Join(claudeDir, "todo-monitor.rs")
		if info, err := os.Stat(hookScriptPath); os.IsNotExist(err) {
			t.Error("Hook script was not created")
		} else if info.Mode()&0111 == 0 {
			t.Error("Hook script is not executable")
		}

		// Verify settings file contains absolute path
		settingsData, err := os.ReadFile(settingsFile)
		if err != nil {
			t.Fatalf("Failed to read settings file: %v", err)
		}

		var settings map[string]interface{}
		if err := json.Unmarshal(settingsData, &settings); err != nil {
			t.Fatalf("Failed to unmarshal settings: %v", err)
		}

		// Navigate through the JSON structure to find the command path
		if hooks, ok := settings["hooks"].(map[string]interface{}); ok {
			if postToolUse, ok := hooks["PostToolUse"].([]interface{}); ok && len(postToolUse) > 0 {
				if matcher, ok := postToolUse[0].(map[string]interface{}); ok {
					if hooksList, ok := matcher["hooks"].([]interface{}); ok && len(hooksList) > 0 {
						if hook, ok := hooksList[0].(map[string]interface{}); ok {
							if command, ok := hook["command"].(string); ok {
								if !filepath.IsAbs(command) {
									t.Errorf("Hook command path is not absolute: %s", command)
								}
							} else {
								t.Error("Command field not found or not a string")
							}
						}
					}
				}
			}
		}

		// Test cleanup
		cleanup()

		// Verify files were removed
		if _, err := os.Stat(todoFile); !os.IsNotExist(err) {
			t.Error("Todo file was not cleaned up")
		}
		if _, err := os.Stat(settingsFile); !os.IsNotExist(err) {
			t.Error("settings.local.json was not cleaned up")
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
		settingsPath := filepath.Join(tmpDir, "settings.local.json")
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

		// Verify SubagentStop hook is present
		subagentStop, ok := hooks["SubagentStop"].([]interface{})
		if !ok {
			t.Fatal("Missing or invalid 'SubagentStop' key")
		}

		if len(subagentStop) != 1 {
			t.Fatalf("Expected 1 SubagentStop entry, got %d", len(subagentStop))
		}

		subagentHook, ok := subagentStop[0].(map[string]interface{})
		if !ok {
			t.Fatal("Invalid SubagentStop hook structure")
		}

		if subagentHook["type"] != "command" {
			t.Errorf("Expected SubagentStop hook type 'command', got '%v'", subagentHook["type"])
		}

		if subagentHook["command"] != hookPath {
			t.Errorf("Expected SubagentStop command '%s', got '%v'", hookPath, subagentHook["command"])
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
		scriptPath := filepath.Join(tmpDir, "todo-monitor.rs")

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

		if !strings.Contains(string(content), "#!/usr/bin/env rust-script") {
			t.Error("Script missing shebang")
		}

		if !strings.Contains(string(content), "TodoWrite") {
			t.Error("Script missing TodoWrite matcher logic")
		}
	})
}
