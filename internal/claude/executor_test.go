package claude

import (
	"context"
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
				LinearIssue: "ISSUE-123",
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
				LinearIssue: "ISSUE-123",
				MCPServers:  []string{"playwright", "linear-server"},
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
				LinearIssue: "ISSUE-123",
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
				LinearIssue: "ISSUE-123",
			},
			expectedArgs: []string{
				"--output-format", "text",
				"--mcp-server", "linear-server",
				"--allowedTools",
				"--append-system-prompt",
				"--project", "/tmp",
				"-p", "test prompt",
			},
			expectedEnvSet: map[string]bool{
				"RIVER_STATE_FILE":   true,
				"RIVER_LINEAR_ISSUE": true,
			},
		},
		{
			name: "command with multiple MCP servers",
			config: ExecuteConfig{
				Prompt:      "test prompt",
				StateFile:   "/tmp/state.json",
				MCPServers:  []string{"playwright", "web-mrkdwn"},
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