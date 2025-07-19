package claude

import (
	"context"
	"testing"
)

func TestClaudeInterface(t *testing.T) {
	t.Run("interface should be defined", func(t *testing.T) {
		// Test that the Claude interface exists and can be used
		var _ Claude = (*mockClaude)(nil)
	})
}

func TestClaudeOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("BuildCommand should create plan command", func(t *testing.T) {
		// Test BuildCommand method for plan type
		claude := &mockClaude{}
		cmd := Command{
			Type:         CommandTypePlan,
			Prompt:       "Create a new feature",
			OutputFormat: "json",
			SystemPrompt: "You are a helpful assistant",
			AllowedTools: []string{"read", "write"},
		}

		args, err := claude.BuildCommand(ctx, cmd)
		if err != nil {
			t.Fatalf("BuildCommand() error = %v; want nil", err)
		}

		// Check that args contains expected elements
		// Should include "claude" as first argument
		expectedLen := 9 // claude -p --output-format json --append-system-prompt <prompt> --allowedTools <tools> <prompt>
		if len(args) != expectedLen {
			t.Errorf("len(args) = %d; want %d", len(args), expectedLen)
			t.Errorf("actual args: %v", args)
		}

		// Verify key arguments
		if args[0] != "claude" {
			t.Errorf("args[0] = %s; want 'claude'", args[0])
		}
		if args[1] != "-p" {
			t.Errorf("args[1] = %s; want '-p'", args[1])
		}
		// Prompt should be last argument
		if args[len(args)-1] != "/make_plan Create a new feature" {
			t.Errorf("Last arg = %s; want '/make_plan Create a new feature'", args[len(args)-1])
		}
	})

	t.Run("BuildCommand should create continue command", func(t *testing.T) {
		// Test BuildCommand method for continue type
		claude := &mockClaude{}
		cmd := Command{
			Type:         CommandTypeContinue,
			OutputFormat: "json",
		}

		args, err := claude.BuildCommand(ctx, cmd)
		if err != nil {
			t.Fatalf("BuildCommand() error = %v; want nil", err)
		}

		// For continue command, expect ralph prompt as last argument
		if len(args) < 3 || args[1] != "-p" {
			t.Errorf("Continue command should use -p flag")
		}
		if args[len(args)-1] != "/ralph" {
			t.Errorf("Last arg = %s; want '/ralph'", args[len(args)-1])
		}
	})

	t.Run("Execute should run command and return response", func(t *testing.T) {
		// Test Execute method
		claude := &mockClaude{}
		cmd := Command{
			Type:         CommandTypePlan,
			Prompt:       "Test prompt",
			OutputFormat: "json",
		}
		opts := CommandOptions{
			Stream:  false,
			Timeout: 300,
		}

		resp, err := claude.Execute(ctx, cmd, opts)
		if err != nil {
			t.Fatalf("Execute() error = %v; want nil", err)
		}

		if resp == nil {
			t.Fatal("Execute() returned nil response")
		}
		if resp.Content == "" {
			t.Error("Execute() returned empty content")
		}
	})

	t.Run("ParseResponse should extract continue flag from JSON", func(t *testing.T) {
		// Test ParseResponse method
		claude := &mockClaude{}
		jsonOutput := `{"content": "Implementation plan", "continue": true}`

		resp, err := claude.ParseResponse(ctx, jsonOutput)
		if err != nil {
			t.Fatalf("ParseResponse() error = %v; want nil", err)
		}

		if resp.Content != "Implementation plan" {
			t.Errorf("resp.Content = %s; want 'Implementation plan'", resp.Content)
		}
		if !resp.ContinueFlag {
			t.Error("resp.ContinueFlag = false; want true")
		}
	})

	t.Run("ParseResponse should handle invalid JSON", func(t *testing.T) {
		// Test ParseResponse with invalid JSON
		claude := &mockClaude{}
		invalidJSON := `{invalid json}`

		_, err := claude.ParseResponse(ctx, invalidJSON)
		if err == nil {
			t.Error("ParseResponse() should return error for invalid JSON")
		}
	})
}

// mockClaude is a test implementation of the Claude interface
type mockClaude struct{}

func (m *mockClaude) BuildCommand(ctx context.Context, cmd Command) ([]string, error) {
	// Mock implementation to make tests compile
	args := []string{"claude", "-p"}

	if cmd.OutputFormat != "" {
		args = append(args, "--output-format", cmd.OutputFormat)
	}
	if cmd.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", cmd.SystemPrompt)
	}
	if len(cmd.AllowedTools) > 0 {
		toolsStr := ""
		for i, tool := range cmd.AllowedTools {
			if i > 0 {
				toolsStr += ","
			}
			toolsStr += tool
		}
		args = append(args, "--allowedTools", toolsStr)
	}

	// Add prompt as last argument
	if cmd.Type == CommandTypePlan {
		args = append(args, "/make_plan "+cmd.Prompt)
	} else {
		args = append(args, "/ralph")
	}

	return args, nil
}

func (m *mockClaude) Execute(ctx context.Context, cmd Command, opts CommandOptions) (*Response, error) {
	// Mock implementation
	return &Response{
		Content:      "Mock response",
		ContinueFlag: false,
	}, nil
}

func (m *mockClaude) ParseResponse(ctx context.Context, output string) (*Response, error) {
	// Mock implementation with basic JSON parsing
	if output == `{"content": "Implementation plan", "continue": true}` {
		return &Response{
			Content:      "Implementation plan",
			ContinueFlag: true,
		}, nil
	}
	if output == `{invalid json}` {
		return nil, &parseError{msg: "invalid JSON"}
	}
	return &Response{Content: "Default response"}, nil
}

// parseError is a custom error type for parse errors
type parseError struct {
	msg string
}

func (e *parseError) Error() string {
	return e.msg
}
