package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPlanCommand tests the plan command definition and structure
func TestPlanCommand(t *testing.T) {
	// Test that plan command exists as a subcommand of rootCmd
	t.Run("plan command exists", func(t *testing.T) {
		rootCmd := NewRootCommand()
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == "plan <task-description>" {
				found = true
				break
			}
		}
		if !found {
			t.Error("plan command not found in rootCmd")
		}
	})

	// Test that plan command requires exactly one argument
	t.Run("plan command requires one argument", func(t *testing.T) {
		rootCmd := NewRootCommand()
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)
		rootCmd.SetArgs([]string{"plan"})

		err := rootCmd.Execute()
		if err == nil {
			t.Error("expected error when no arguments provided")
		}

		if !strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
			t.Errorf("expected error about requiring 1 argument, got: %v", err)
		}
	})

	// Test that plan command accepts one argument
	t.Run("plan command accepts one argument", func(t *testing.T) {
		rootCmd := NewRootCommand()
		// This will fail until we implement generatePlan
		// But we're testing the command structure, not the implementation
		cmd, _, err := rootCmd.Find([]string{"plan", "test task"})
		if err != nil {
			t.Errorf("failed to find plan command: %v", err)
		}

		if cmd == nil || cmd.Use != "plan <task-description>" {
			t.Error("plan command not properly configured")
		}

		// Verify Args validation
		if cmd.Args == nil {
			t.Error("plan command Args validator not set")
		}
	})
}

// TestGeneratePlan tests the generatePlan function
func TestGeneratePlan(t *testing.T) {
	// Test that generatePlan returns error if GEMINI_API_KEY is not set
	t.Run("error when GEMINI_API_KEY not set", func(t *testing.T) {
		// Save original env var and restore after test
		originalKey := os.Getenv("GEMINI_API_KEY")
		defer func() {
			_ = os.Setenv("GEMINI_API_KEY", originalKey)
		}()

		// Unset the API key
		_ = os.Unsetenv("GEMINI_API_KEY")

		err := generatePlan("test task")
		if err == nil {
			t.Error("expected error when GEMINI_API_KEY not set")
		}

		if !strings.Contains(err.Error(), "GEMINI_API_KEY not set") {
			t.Errorf("expected error about GEMINI_API_KEY not set, got: %v", err)
		}
	})

	// Test that generatePlan returns error if prompt template cannot be read
	t.Run("error when prompt template not found", func(t *testing.T) {
		// Set a dummy API key for this test
		originalKey := os.Getenv("GEMINI_API_KEY")
		defer func() {
			_ = os.Setenv("GEMINI_API_KEY", originalKey)
		}()
		_ = os.Setenv("GEMINI_API_KEY", "test-key")

		// We'll need to mock the file system access or work in a temp directory
		// For now, this test will verify the error when the file doesn't exist
		// We'll need to modify generatePlan to accept a working directory parameter

		// This test will be properly implemented when we refactor generatePlan
		// to accept dependencies for testing
		t.Skip("Skipping until generatePlan is refactored for testability")
	})

	// Test that generatePlan returns error if no spec files are found
	t.Run("error when no spec files found", func(t *testing.T) {
		// Set a dummy API key for this test
		originalKey := os.Getenv("GEMINI_API_KEY")
		defer func() {
			_ = os.Setenv("GEMINI_API_KEY", originalKey)
		}()
		_ = os.Setenv("GEMINI_API_KEY", "test-key")

		// This test will be properly implemented when we refactor generatePlan
		// to accept dependencies for testing
		t.Skip("Skipping until generatePlan is refactored for testability")
	})

	// Test successful prompt construction
	t.Run("successful prompt construction", func(t *testing.T) {
		// Set a dummy API key for this test
		originalKey := os.Getenv("GEMINI_API_KEY")
		defer func() {
			_ = os.Setenv("GEMINI_API_KEY", originalKey)
		}()
		_ = os.Setenv("GEMINI_API_KEY", "test-key")

		// This test will verify that the prompt is correctly constructed
		// with all spec files and the task description
		t.Skip("Skipping until generatePlan is refactored for testability")
	})

	// Test successful output writing
	t.Run("successful output writing", func(t *testing.T) {
		// Set a dummy API key for this test
		originalKey := os.Getenv("GEMINI_API_KEY")
		defer func() {
			_ = os.Setenv("GEMINI_API_KEY", originalKey)
		}()
		_ = os.Setenv("GEMINI_API_KEY", "test-key")

		// This test will verify that the output from Gemini is correctly
		// written to plan.md
		t.Skip("Skipping until generatePlan is refactored for testability")
	})
}

// TestGeneratePlanRefactored tests the refactored generatePlan function with dependencies
func TestGeneratePlanRefactored(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create test spec files
	specsDir := filepath.Join(tempDir, "specs")
	if err := os.Mkdir(specsDir, 0755); err != nil {
		t.Fatalf("failed to create specs dir: %v", err)
	}

	specFiles := []string{"architecture.md", "cli-commands.md", "configuration.md"}
	for _, file := range specFiles {
		content := []byte("# " + file + "\nTest spec content for " + file)
		if err := os.WriteFile(filepath.Join(specsDir, file), content, 0644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}
	}

	// Create test prompt template
	promptsDir := filepath.Join(tempDir, "prompts")
	if err := os.Mkdir(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}
	promptTemplate := `# Plan Generation Template

Task: {{TASK}}

Please generate an implementation plan based on the following specifications:
{{SPECS}}

Generate a detailed plan in markdown format.`
	if err := os.WriteFile(filepath.Join(promptsDir, "prompt-plan.md"), []byte(promptTemplate), 0644); err != nil {
		t.Fatalf("failed to write prompt template: %v", err)
	}

	t.Run("builds correct prompt with all components", func(t *testing.T) {
		// This test will be implemented once we refactor generatePlan
		// to accept a PlanGenerator interface or similar for testing
		t.Skip("Waiting for generatePlan refactoring")
	})
}
