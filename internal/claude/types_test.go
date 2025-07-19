package claude

import (
	"testing"
)

func TestCommandType(t *testing.T) {
	t.Run("should create plan command type", func(t *testing.T) {
		// Test that CommandTypePlan constant exists and has correct value
		if CommandTypePlan != "plan" {
			t.Errorf("CommandTypePlan = %s; want 'plan'", CommandTypePlan)
		}
	})

	t.Run("should create continue command type", func(t *testing.T) {
		// Test that CommandTypeContinue constant exists and has correct value
		if CommandTypeContinue != "continue" {
			t.Errorf("CommandTypeContinue = %s; want 'continue'", CommandTypeContinue)
		}
	})
}

func TestCommand(t *testing.T) {
	t.Run("should create a plan command with all required fields", func(t *testing.T) {
		// Test creating a plan command with all necessary fields
		cmd := Command{
			Type:         CommandTypePlan,
			Prompt:       "Create a feature",
			OutputFormat: "json",
			SystemPrompt: "You are a helpful assistant",
			AllowedTools: []string{"read", "write"},
		}

		if cmd.Type != CommandTypePlan {
			t.Errorf("cmd.Type = %s; want %s", cmd.Type, CommandTypePlan)
		}
		if cmd.Prompt != "Create a feature" {
			t.Errorf("cmd.Prompt = %s; want 'Create a feature'", cmd.Prompt)
		}
		if cmd.OutputFormat != "json" {
			t.Errorf("cmd.OutputFormat = %s; want 'json'", cmd.OutputFormat)
		}
		if cmd.SystemPrompt != "You are a helpful assistant" {
			t.Errorf("cmd.SystemPrompt = %s; want 'You are a helpful assistant'", cmd.SystemPrompt)
		}
		if len(cmd.AllowedTools) != 2 {
			t.Errorf("len(cmd.AllowedTools) = %d; want 2", len(cmd.AllowedTools))
		}
	})

	t.Run("should validate command type", func(t *testing.T) {
		// Test the Validate method exists and works correctly
		validCmd := Command{Type: CommandTypePlan}
		if err := validCmd.Validate(); err != nil {
			t.Errorf("Validate() returned error for valid command: %v", err)
		}

		invalidCmd := Command{Type: CommandType("invalid")}
		if err := invalidCmd.Validate(); err == nil {
			t.Error("Validate() should return error for invalid command type")
		}
	})
}

func TestResponse(t *testing.T) {
	t.Run("should parse claude response with continue flag", func(t *testing.T) {
		// Test Response struct with continue flag
		resp := Response{
			Content:      "Implementation plan",
			ContinueFlag: true,
			Error:        "",
		}

		if !resp.ContinueFlag {
			t.Error("resp.ContinueFlag = false; want true")
		}
		if resp.Content != "Implementation plan" {
			t.Errorf("resp.Content = %s; want 'Implementation plan'", resp.Content)
		}
		if resp.Error != "" {
			t.Errorf("resp.Error = %s; want empty string", resp.Error)
		}
	})

	t.Run("should handle error response", func(t *testing.T) {
		// Test Response struct with error
		resp := Response{
			Content:      "",
			ContinueFlag: false,
			Error:        "command failed",
		}

		if resp.HasError() != true {
			t.Error("HasError() = false; want true")
		}
	})
}

func TestIssueID(t *testing.T) {
	t.Run("should create and validate issue ID", func(t *testing.T) {
		// Test custom IssueID type
		id := IssueID("LINEAR-123")
		
		if string(id) != "LINEAR-123" {
			t.Errorf("IssueID = %s; want 'LINEAR-123'", id)
		}

		// Test validation
		if err := id.Validate(); err != nil {
			t.Errorf("Validate() returned error for valid ID: %v", err)
		}

		emptyID := IssueID("")
		if err := emptyID.Validate(); err == nil {
			t.Error("Validate() should return error for empty ID")
		}
	})
}

func TestCommandOptions(t *testing.T) {
	t.Run("should create command options with streaming enabled", func(t *testing.T) {
		// Test CommandOptions struct
		opts := CommandOptions{
			Stream:       true,
			Timeout:      300,
			WorkingDir:   "/tmp/test",
		}

		if !opts.Stream {
			t.Error("opts.Stream = false; want true")
		}
		if opts.Timeout != 300 {
			t.Errorf("opts.Timeout = %d; want 300", opts.Timeout)
		}
		if opts.WorkingDir != "/tmp/test" {
			t.Errorf("opts.WorkingDir = %s; want '/tmp/test'", opts.WorkingDir)
		}
	})
}