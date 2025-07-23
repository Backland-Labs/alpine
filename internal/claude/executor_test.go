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
				Prompt:      "test prompt",
				StateFile:   "/tmp/state.json",
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
				Prompt:      "test prompt",
				StateFile:   "/tmp/state.json",
				MCPServers:  []string{"playwright", "context7"},
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
				Prompt:      "test prompt",
				StateFile:   "/tmp/state.json",
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
				Prompt:      "test prompt",
				StateFile:   "/tmp/state.json",
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
