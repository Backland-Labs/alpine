package claude

import (
	"context"
	"testing"
)

// TestBuildCommand_Plan tests building a plan command with various options
func TestBuildCommand_Plan(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		cmd      Command
		expected []string
	}{
		{
			name: "basic plan command",
			cmd: Command{
				Type:         CommandTypePlan,
				Prompt:       "Implement feature X",
				OutputFormat: "json",
			},
			expected: []string{
				"claude",
				"-p",
				"--output-format",
				"json",
				"/make_plan Implement feature X",
			},
		},
		{
			name: "plan command with system prompt",
			cmd: Command{
				Type:         CommandTypePlan,
				Prompt:       "Fix critical bug",
				OutputFormat: "json",
				SystemPrompt: "You are a helpful assistant",
			},
			expected: []string{
				"claude",
				"-p",
				"--output-format",
				"json",
				"--append-system-prompt",
				"You are a helpful assistant",
				"/make_plan Fix critical bug",
			},
		},
		{
			name: "plan command with allowed tools",
			cmd: Command{
				Type:         CommandTypePlan,
				Prompt:       "Refactor module",
				OutputFormat: "json",
				AllowedTools: []string{"read", "write", "exec"},
			},
			expected: []string{
				"claude",
				"-p",
				"--output-format",
				"json",
				"--allowedTools",
				"read,write,exec",
				"/make_plan Refactor module",
			},
		},
		{
			name: "plan command with all options",
			cmd: Command{
				Type:         CommandTypePlan,
				Prompt:       "Complex task with \"quotes\" and special chars",
				OutputFormat: "json",
				SystemPrompt: "Be careful with quotes",
				AllowedTools: []string{"read", "write"},
			},
			expected: []string{
				"claude",
				"-p",
				"--output-format",
				"json",
				"--append-system-prompt",
				"Be careful with quotes",
				"--allowedTools",
				"read,write",
				"/make_plan Complex task with \"quotes\" and special chars",
			},
		},
	}

	builder := &commandBuilder{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := builder.BuildCommand(ctx, tt.cmd)
			if err != nil {
				t.Fatalf("BuildCommand() returned unexpected error: %v", err)
			}
			if len(result) != len(tt.expected) {
				t.Errorf("BuildCommand() returned %d args, expected %d", len(result), len(tt.expected))
				t.Errorf("Got: %v", result)
				t.Errorf("Expected: %v", tt.expected)
				return
			}
			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("BuildCommand() arg[%d] = %q, expected %q", i, arg, tt.expected[i])
				}
			}
		})
	}
}

// TestBuildCommand_Continue tests building a continue command
func TestBuildCommand_Continue(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		cmd      Command
		expected []string
	}{
		{
			name: "basic continue command",
			cmd: Command{
				Type:         CommandTypeContinue,
				Prompt:       "Continue implementation",
				OutputFormat: "json",
			},
			expected: []string{
				"claude",
				"-p",
				"--output-format",
				"json",
				"/ralph Continue implementation",
			},
		},
		{
			name: "continue command with system prompt and tools",
			cmd: Command{
				Type:         CommandTypeContinue,
				Prompt:       "Keep going with fixes",
				OutputFormat: "json",
				SystemPrompt: "Focus on error handling",
				AllowedTools: []string{"read", "write", "exec", "test"},
			},
			expected: []string{
				"claude",
				"-p",
				"--output-format",
				"json",
				"--append-system-prompt",
				"Focus on error handling",
				"--allowedTools",
				"read,write,exec,test",
				"/ralph Keep going with fixes",
			},
		},
	}

	builder := &commandBuilder{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := builder.BuildCommand(ctx, tt.cmd)
			if err != nil {
				t.Fatalf("BuildCommand() returned unexpected error: %v", err)
			}
			if len(result) != len(tt.expected) {
				t.Errorf("BuildCommand() returned %d args, expected %d", len(result), len(tt.expected))
				t.Errorf("Got: %v", result)
				t.Errorf("Expected: %v", tt.expected)
				return
			}
			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("BuildCommand() arg[%d] = %q, expected %q", i, arg, tt.expected[i])
				}
			}
		})
	}
}

// TestBuildCommand_EdgeCases tests edge cases and error handling
func TestBuildCommand_EdgeCases(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		cmd         Command
		expectError bool
	}{
		{
			name: "empty prompt",
			cmd: Command{
				Type:         CommandTypePlan,
				Prompt:       "",
				OutputFormat: "json",
			},
			expectError: true,
		},
		{
			name: "whitespace-only prompt",
			cmd: Command{
				Type:         CommandTypePlan,
				Prompt:       "   \t\n  ",
				OutputFormat: "json",
			},
			expectError: true,
		},
		{
			name: "invalid command type",
			cmd: Command{
				Type:         "invalid",
				Prompt:       "Test",
				OutputFormat: "json",
			},
			expectError: true,
		},
		{
			name: "prompt with newlines",
			cmd: Command{
				Type:         CommandTypeContinue,
				Prompt:       "Multi\nline\nprompt",
				OutputFormat: "json",
			},
			expectError: false,
		},
		{
			name: "prompt with special shell characters",
			cmd: Command{
				Type:         CommandTypePlan,
				Prompt:       "Task with $VAR and `backticks` and |pipes|",
				OutputFormat: "json",
			},
			expectError: false,
		},
		{
			name: "empty output format defaults to text",
			cmd: Command{
				Type:         CommandTypePlan,
				Prompt:       "Test prompt",
				OutputFormat: "",
			},
			expectError: false,
		},
	}

	builder := &commandBuilder{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := builder.BuildCommand(ctx, tt.cmd)
			if tt.expectError {
				if err == nil {
					t.Errorf("BuildCommand() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("BuildCommand() returned unexpected error: %v", err)
				}
				if len(result) < 3 {
					t.Errorf("BuildCommand() returned too few arguments: %v", result)
				}
				if result[0] != "claude" {
					t.Errorf("BuildCommand() first arg should be 'claude', got %q", result[0])
				}
			}
		})
	}
}

// TestNewCommandBuilder tests the constructor
func TestNewCommandBuilder(t *testing.T) {
	builder := NewCommandBuilder()
	if builder == nil {
		t.Fatal("NewCommandBuilder() returned nil")
	}

	// Verify it implements the Claude interface
	var _ Claude = builder
}
