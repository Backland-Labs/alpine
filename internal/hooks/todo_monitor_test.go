package hooks_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxmcd/alpine/internal/hooks"
)

// TestTodoMonitorConsoleOutput tests that the todo-monitor hook outputs to stderr
func TestTodoMonitorConsoleOutput(t *testing.T) {
	// Get the hook script
	script, err := hooks.GetTodoMonitorScript()
	if err != nil {
		t.Fatalf("Failed to get todo monitor script: %v", err)
	}

	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "todo-monitor.rs")

	// Write the script to a file
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	// Test cases for different tool types
	tests := []struct {
		name           string
		input          map[string]interface{}
		expectedOutput []string
	}{
		{
			name: "TodoWrite with task counts",
			input: map[string]interface{}{
				"tool_name": "TodoWrite",
				"tool_input": map[string]interface{}{
					"todos": []map[string]interface{}{
						{"content": "Task 1", "status": "completed"},
						{"content": "Task 2", "status": "completed"},
						{"content": "Current task", "status": "in_progress"},
						{"content": "Task 4", "status": "pending"},
						{"content": "Task 5", "status": "pending"},
					},
				},
			},
			expectedOutput: []string{
				"[TODO] Updated - Completed: 2, In Progress: 1, Pending: 2",
				"[TODO] Current task: Current task",
			},
		},
		{
			name: "Read tool",
			input: map[string]interface{}{
				"tool_name": "Read",
				"tool_input": map[string]interface{}{
					"file_path": "/test/file.go",
				},
			},
			expectedOutput: []string{
				"[READ] Reading file: /test/file.go",
			},
		},
		{
			name: "Bash tool",
			input: map[string]interface{}{
				"tool_name": "Bash",
				"tool_input": map[string]interface{}{
					"command": "go test ./...",
				},
			},
			expectedOutput: []string{
				"[BASH] Executing: go test ./...",
			},
		},
		{
			name: "Grep tool",
			input: map[string]interface{}{
				"tool_name": "Grep",
				"tool_input": map[string]interface{}{
					"pattern": "func Test",
					"path":    "internal/",
				},
			},
			expectedOutput: []string{
				"[GREP] Searching for 'func Test' in internal/",
			},
		},
		{
			name: "Task tool",
			input: map[string]interface{}{
				"tool_name": "Task",
				"tool_input": map[string]interface{}{
					"description": "Find timeout functionality code",
					"prompt":      "Search for code related to timeout waiting for state update",
				},
			},
			expectedOutput: []string{
				"[TASK] Launching agent: Find timeout functionality code",
			},
		},
		{
			name: "Legacy format TodoWrite",
			input: map[string]interface{}{
				"tool": "TodoWrite",
				"args": map[string]interface{}{
					"todos": []map[string]interface{}{
						{"content": "Legacy task", "status": "in_progress"},
					},
				},
			},
			expectedOutput: []string{
				"[TODO] Updated - Completed: 0, In Progress: 1, Pending: 0",
				"[TODO] Current task: Legacy task",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert input to JSON
			inputJSON, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Failed to marshal input: %v", err)
			}

			// Run the script
			cmd := exec.Command(scriptPath)
			cmd.Stdin = bytes.NewReader(inputJSON)

			// Capture stderr (where the output goes)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr

			// Capture stdout as well for debugging
			var stdout bytes.Buffer
			cmd.Stdout = &stdout

			// Run the command
			if err := cmd.Run(); err != nil {
				t.Fatalf("Script execution failed: %v\nStderr: %s\nStdout: %s", err, stderr.String(), stdout.String())
			}

			// Check the output
			output := stderr.String()

			// Verify all expected strings are present
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, output)
				}
			}

			// Verify timestamp format (HH:MM:SS)
			if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
				t.Errorf("Output missing timestamp format, got: %s", output)
			}
		})
	}
}

// TestTodoMonitorInvalidJSON tests that the hook handles invalid JSON gracefully
func TestTodoMonitorInvalidJSON(t *testing.T) {
	// Get the hook script
	script, err := hooks.GetTodoMonitorScript()
	if err != nil {
		t.Fatalf("Failed to get todo monitor script: %v", err)
	}

	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "todo-monitor.rs")

	// Write the script to a file
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	// Test with invalid JSON
	cmd := exec.Command(scriptPath)
	cmd.Stdin = strings.NewReader("not valid json")

	// Should exit gracefully without error
	if err := cmd.Run(); err != nil {
		t.Fatalf("Script should handle invalid JSON gracefully, but got error: %v", err)
	}
}

// TestTodoMonitorFileWrite tests that the hook writes to the todo file
func TestTodoMonitorFileWrite(t *testing.T) {
	// Get the hook script
	script, err := hooks.GetTodoMonitorScript()
	if err != nil {
		t.Fatalf("Failed to get todo monitor script: %v", err)
	}

	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "todo-monitor.rs")
	todoFilePath := filepath.Join(tmpDir, "current-todo.txt")

	// Write the script to a file
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	// Set the environment variable
	_ = os.Setenv("ALPINE_TODO_FILE", todoFilePath)
	defer func() { _ = os.Unsetenv("ALPINE_TODO_FILE") }()

	// Prepare input with an in-progress task
	input := map[string]interface{}{
		"tool_name": "TodoWrite",
		"tool_input": map[string]interface{}{
			"todos": []map[string]interface{}{
				{"content": "Completed task", "status": "completed"},
				{"content": "Current important task", "status": "in_progress"},
				{"content": "Pending task", "status": "pending"},
			},
		},
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Run the script
	cmd := exec.Command(scriptPath)
	cmd.Stdin = bytes.NewReader(inputJSON)
	cmd.Env = append(os.Environ(), "ALPINE_TODO_FILE="+todoFilePath)

	if err := cmd.Run(); err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}

	// Check that the file was created with the correct content
	content, err := os.ReadFile(todoFilePath)
	if err != nil {
		t.Fatalf("Failed to read todo file: %v", err)
	}

	expectedContent := "Current important task"
	if string(content) != expectedContent {
		t.Errorf("Expected todo file to contain %q, but got %q", expectedContent, string(content))
	}
}
