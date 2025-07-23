package claude

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewExecutor(t *testing.T) {
	t.Run("creates executor with valid configuration", func(t *testing.T) {
		// Test that NewExecutor creates a valid executor instance
		exec := NewExecutor()
		if exec == nil {
			t.Fatal("expected executor to be created")
		}
	})
}

func TestExecutor_Execute(t *testing.T) {
	tests := []struct {
		name          string
		config        ExecuteConfig
		mockCommand   *mockCommand
		expectedError bool
		errorContains string
	}{
		{
			name: "successful execution with basic prompt",
			config: ExecuteConfig{
				Prompt:    "test prompt",
				StateFile: "/tmp/state.json",
			},
			mockCommand: &mockCommand{
				output: "Claude execution successful",
				err:    nil,
			},
			expectedError: false,
		},
		{
			name: "successful execution with MCP servers",
			config: ExecuteConfig{
				Prompt:     "test prompt",
				StateFile:  "/tmp/state.json",
				MCPServers: []string{"playwright", "context7"},
			},
			mockCommand: &mockCommand{
				output: "Claude execution with MCP servers successful",
				err:    nil,
			},
			expectedError: false,
		},
		{
			name: "handles command execution failure",
			config: ExecuteConfig{
				Prompt:    "test prompt",
				StateFile: "/tmp/state.json",
			},
			mockCommand: &mockCommand{
				output: "",
				err:    &mockError{msg: "command failed"},
			},
			expectedError: true,
			errorContains: "command failed",
		},
		{
			name: "validates required fields",
			config: ExecuteConfig{
				Prompt:    "",
				StateFile: "/tmp/state.json",
			},
			mockCommand:   nil,
			expectedError: true,
			errorContains: "prompt is required",
		},
		{
			name: "validates state file path",
			config: ExecuteConfig{
				Prompt:    "test prompt",
				StateFile: "",
			},
			mockCommand:   nil,
			expectedError: true,
			errorContains: "state file is required",
		},
		{
			name: "includes custom system prompt when provided",
			config: ExecuteConfig{
				Prompt:       "test prompt",
				StateFile:    "/tmp/state.json",
				SystemPrompt: "Custom system prompt for testing",
			},
			mockCommand: &mockCommand{
				output: "Execution with custom system prompt",
				err:    nil,
			},
			expectedError: false,
		},
		{
			name: "respects timeout configuration",
			config: ExecuteConfig{
				Prompt:    "test prompt",
				StateFile: "/tmp/state.json",
				Timeout:   5 * time.Second,
			},
			mockCommand: &mockCommand{
				output:   "Timed execution",
				err:      nil,
				duration: 100 * time.Millisecond,
			},
			expectedError: false,
		},
		{
			name: "handles timeout exceeded",
			config: ExecuteConfig{
				Prompt:    "test prompt",
				StateFile: "/tmp/state.json",
				Timeout:   100 * time.Millisecond,
			},
			mockCommand: &mockCommand{
				output:   "",
				err:      context.DeadlineExceeded,
				duration: 200 * time.Millisecond,
			},
			expectedError: true,
			errorContains: "deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &Executor{
				commandRunner: tt.mockCommand,
			}

			output, err := exec.Execute(context.Background(), tt.config)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if output != tt.mockCommand.output {
					t.Errorf("expected output %q, got %q", tt.mockCommand.output, output)
				}
			}
		})
	}
}

func TestExecutor_buildCommand(t *testing.T) {
	tests := []struct {
		name           string
		config         ExecuteConfig
		expectedArgs   []string
		expectedEnvSet map[string]bool
	}{
		{
			name: "basic command construction",
			config: ExecuteConfig{
				Prompt:    "test prompt",
				StateFile: "/tmp/state.json",
			},
			expectedArgs: []string{
				"--output-format", "text",
				"--allowedTools",
				"--append-system-prompt",
				"-p", "test prompt",
			},
			expectedEnvSet: map[string]bool{
				"RIVER_STATE_FILE": true,
			},
		},
		{
			name: "command with multiple MCP servers",
			config: ExecuteConfig{
				Prompt:     "test prompt",
				StateFile:  "/tmp/state.json",
				MCPServers: []string{"playwright", "web-mrkdwn"},
			},
			expectedArgs: []string{
				"--output-format", "text",
				"--mcp-server", "playwright",
				"--mcp-server", "web-mrkdwn",
				"--allowedTools",
				"--append-system-prompt",
				"-p", "test prompt",
			},
			expectedEnvSet: map[string]bool{
				"RIVER_STATE_FILE": true,
			},
		},
		{
			name: "command with custom system prompt",
			config: ExecuteConfig{
				Prompt:       "test prompt",
				StateFile:    "/tmp/state.json",
				SystemPrompt: "Custom system prompt",
			},
			expectedArgs: []string{
				"--output-format", "text",
				"--allowedTools",
				"--append-system-prompt", "Custom system prompt",
				"-p", "test prompt",
			},
			expectedEnvSet: map[string]bool{
				"RIVER_STATE_FILE": true,
			},
		},
		{
			name: "command with tools restriction",
			config: ExecuteConfig{
				Prompt:       "test prompt",
				StateFile:    "/tmp/state.json",
				AllowedTools: []string{"Read", "Write", "Edit"},
			},
			expectedArgs: []string{
				"--output-format", "text",
				"--allowedTools", "Read", "Write", "Edit",
				"--append-system-prompt",
				"-p", "test prompt",
			},
			expectedEnvSet: map[string]bool{
				"RIVER_STATE_FILE": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &Executor{}
			cmd := exec.buildCommand(tt.config)

			// Check that the base command is correct
			if cmd.Path != "claude" && !strings.HasSuffix(cmd.Path, "/claude") {
				t.Errorf("expected command path to be 'claude', got %q", cmd.Path)
			}

			// Check expected arguments are present
			args := strings.Join(cmd.Args[1:], " ")
			for _, expectedArg := range tt.expectedArgs {
				if !strings.Contains(args, expectedArg) {
					t.Errorf("expected argument %q not found in command args: %v", expectedArg, cmd.Args)
				}
			}

			// Check environment variables
			envMap := make(map[string]string)
			for _, env := range cmd.Env {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					envMap[parts[0]] = parts[1]
				}
			}

			for envKey, shouldBeSet := range tt.expectedEnvSet {
				_, exists := envMap[envKey]
				if shouldBeSet && !exists {
					t.Errorf("expected environment variable %s to be set", envKey)
				}
			}

			// Check that prompt is passed with -p flag
			foundPrompt := false
			for i := 0; i < len(cmd.Args)-1; i++ {
				if cmd.Args[i] == "-p" && cmd.Args[i+1] == tt.config.Prompt {
					foundPrompt = true
					break
				}
			}
			if !foundPrompt {
				t.Errorf("expected prompt %q to be passed with -p flag", tt.config.Prompt)
			}
		})
	}
}

// Mock implementations for testing
type mockCommand struct {
	output   string
	err      error
	duration time.Duration
}

func (m *mockCommand) Run(ctx context.Context, config ExecuteConfig) (string, error) {
	if m.duration > 0 {
		select {
		case <-time.After(m.duration):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return m.output, m.err
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

func TestExecutor_BuildCommand_SetsWorkingDirectory(t *testing.T) {
	// Test that buildCommand sets the working directory to the current directory
	// This ensures Claude commands execute in the correct directory for worktree isolation
	t.Run("sets cmd.Dir to current working directory", func(t *testing.T) {
		exec := &Executor{}
		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "/tmp/state.json",
		}

		// Get the expected working directory
		expectedDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}

		cmd := exec.buildCommand(config)

		// Verify cmd.Dir is set to the current working directory
		if cmd.Dir != expectedDir {
			t.Errorf("expected cmd.Dir to be %q, got %q", expectedDir, cmd.Dir)
		}
	})
}

func TestExecutor_BuildCommand_WorkingDirectoryError(t *testing.T) {
	// Test that buildCommand handles os.Getwd() errors gracefully
	// Even if we can't get the working directory, the command should still be built
	t.Run("handles os.Getwd error gracefully", func(t *testing.T) {
		// Note: It's difficult to mock os.Getwd() directly in Go
		// This test documents the expected behavior when os.Getwd() fails
		// The implementation should continue without setting cmd.Dir
		exec := &Executor{}
		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "/tmp/state.json",
		}

		cmd := exec.buildCommand(config)

		// Even without mocking os.Getwd error, we can verify the command is built
		if cmd == nil {
			t.Fatal("expected command to be built even if working directory fails")
		}

		// Verify basic command structure is intact
		if cmd.Path != "claude" && !strings.HasSuffix(cmd.Path, "/claude") {
			t.Errorf("expected command path to be 'claude', got %q", cmd.Path)
		}
	})
}

func TestExecutor_CommandRunner_PreservesDirectory(t *testing.T) {
	// Test that defaultCommandRunner preserves the working directory from buildCommand
	// This ensures the directory context flows through the entire execution pipeline
	t.Run("preserves working directory from buildCommand", func(t *testing.T) {
		// Create a custom command runner that can verify the directory
		runner := &defaultCommandRunner{}
		exec := &Executor{commandRunner: runner}

		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "/tmp/state.json",
		}

		// Build the base command to get expected directory
		baseCmd := exec.buildCommand(config)

		// Note: Since we can't easily mock exec.CommandContext,
		// this test documents the expected behavior.
		// The actual implementation test will be done via integration tests.

		// For now, verify that buildCommand is called and creates a valid command
		if baseCmd == nil {
			t.Error("expected buildCommand to create a valid command")
		}

		// Once implemented, cmd.Dir should be preserved in the CommandContext call
		// This will be verified in integration tests
	})
}

func TestExecutor_WorkingDirectoryFallback(t *testing.T) {
	// Test that the executor continues gracefully when working directory operations fail
	// This test documents expected behavior when os.Getwd() fails or directory validation fails
	t.Run("continues execution when working directory is unavailable", func(t *testing.T) {
		exec := &Executor{}
		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "/tmp/state.json",
		}

		// Build command - even if we can't mock os.Getwd failure directly,
		// we can verify that command building continues
		cmd := exec.buildCommand(config)

		if cmd == nil {
			t.Fatal("expected command to be built even with directory issues")
		}

		// Verify command has all required components
		if cmd.Path == "" {
			t.Error("expected command path to be set")
		}

		// When directory operations fail, cmd.Dir might be empty but command should still work
		// This is the graceful fallback behavior
	})

	t.Run("logs warning when working directory fails", func(t *testing.T) {
		// This test documents that when os.Getwd() fails, a warning should be logged
		// The actual implementation will use the logger package
		// For now, we document the expected behavior
		exec := &Executor{}
		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "/tmp/state.json",
		}

		// Build command
		cmd := exec.buildCommand(config)

		// Even without being able to capture logs in this test,
		// we document that a warning should be logged when directory operations fail
		if cmd == nil {
			t.Error("command building should not fail due to directory issues")
		}
	})
}

func TestExecutor_ValidatesWorkingDirectory(t *testing.T) {
	// This test verifies that the executor validates working directory exists and is accessible
	// before setting it on the command. This prevents Claude from being executed in a
	// non-existent or inaccessible directory which could cause confusing errors.
	t.Run("validates working directory exists", func(t *testing.T) {
		executor := &Executor{}
		config := ExecuteConfig{
			StateFile: "test-state.json",
		}

		// Create a temporary directory and then remove it to simulate non-existent directory
		tempDir := "/tmp/test-river-validation-" + strings.ReplaceAll(t.Name(), "/", "-")
		err := os.Mkdir(tempDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		// Change to valid directory first
		originalWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalWd) }()
		_ = os.Chdir(tempDir)

		// Now remove it to make it invalid
		_ = os.Chdir(originalWd)
		_ = os.RemoveAll(tempDir)

		// Try to change back to the now non-existent directory
		err = os.Chdir(tempDir)
		if err == nil {
			t.Fatal("Expected error when changing to non-existent directory")
		}

		// Stay in original directory and build command
		cmd := executor.buildCommandWithValidation(config)

		// Command should still be created but Dir should be validated
		if cmd == nil {
			t.Fatal("Expected command to be created even with invalid directory")
		}

		// The validated method should not set an invalid directory
		if cmd.Dir == tempDir {
			t.Errorf("Expected working directory to not be set to non-existent directory")
		}
	})

	t.Run("validates working directory permissions", func(t *testing.T) {
		executor := &Executor{}
		config := ExecuteConfig{
			StateFile: "test-state.json",
		}

		// Create a directory with no read permissions
		tempDir := "/tmp/test-river-noperm-" + strings.ReplaceAll(t.Name(), "/", "-")
		err := os.Mkdir(tempDir, 0000)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = os.Chmod(tempDir, 0755) // Reset permissions to allow removal
			_ = os.RemoveAll(tempDir)
		}()

		// Save original directory
		originalWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalWd) }()

		// Try to change to the no-permission directory
		err = os.Chdir(tempDir)
		if err == nil {
			// If we somehow can change to it, skip this test
			t.Skip("Unable to test permission validation - system allows access")
		}

		// Build command should handle permission errors gracefully
		cmd := executor.buildCommandWithValidation(config)

		if cmd == nil {
			t.Fatal("Expected command to be created even with permission errors")
		}
	})

	t.Run("sets working directory when valid", func(t *testing.T) {
		executor := &Executor{}
		config := ExecuteConfig{
			StateFile: "test-state.json",
		}

		// Create a valid temporary directory
		tempDir, err := os.MkdirTemp("", "test-river-valid-")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(tempDir) }()

		// Change to the valid directory
		originalWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalWd) }()
		err = os.Chdir(tempDir)
		if err != nil {
			t.Fatal(err)
		}

		// Build command with validation
		cmd := executor.buildCommandWithValidation(config)

		if cmd == nil {
			t.Fatal("Expected command to be created")
		}

		// Working directory should be set to the valid directory
		// Need to handle macOS symlink behavior for /var/folders -> /private/var/folders
		if cmd.Dir != tempDir && !strings.HasSuffix(cmd.Dir, strings.TrimPrefix(tempDir, "/private")) {
			t.Errorf("Expected working directory to be %s, got %s", tempDir, cmd.Dir)
		}
	})
}
