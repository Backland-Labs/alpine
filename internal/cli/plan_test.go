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

	// TestPlanCommand_CCFlagExists: Verify the flag is registered on the command
	t.Run("cc flag exists", func(t *testing.T) {
		rootCmd := NewRootCommand()
		planCmd, _, err := rootCmd.Find([]string{"plan"})
		if err != nil {
			t.Fatalf("failed to find plan command: %v", err)
		}

		ccFlag := planCmd.Flag("cc")
		if ccFlag == nil {
			t.Error("--cc flag not found on plan command")
			return
		}

		if ccFlag.Usage != "Use Claude Code instead of Gemini for plan generation" {
			t.Errorf("incorrect usage text for --cc flag: %s", ccFlag.Usage)
		}
	})

	// TestPlanCommand_CCFlagDefault: Verify flag defaults to false
	t.Run("cc flag defaults to false", func(t *testing.T) {
		rootCmd := NewRootCommand()
		planCmd, _, err := rootCmd.Find([]string{"plan"})
		if err != nil {
			t.Fatalf("failed to find plan command: %v", err)
		}

		ccFlag := planCmd.Flag("cc")
		if ccFlag == nil {
			t.Fatal("--cc flag not found on plan command")
		}

		if ccFlag.DefValue != "false" {
			t.Errorf("--cc flag should default to false, got: %s", ccFlag.DefValue)
		}
	})

	// TestPlanCommand_ParsesCCFlag: Test that the flag value is correctly parsed
	t.Run("parses cc flag correctly", func(t *testing.T) {
		// Test with --cc flag set to true
		rootCmd := NewRootCommand()
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)

		// We can't test the actual execution without implementing the feature
		// but we can verify the flag is parsed by the command
		planCmd, _, err := rootCmd.Find([]string{"plan"})
		if err != nil {
			t.Fatalf("failed to find plan command: %v", err)
		}

		// Parse the flag
		err = planCmd.ParseFlags([]string{"--cc"})
		if err != nil {
			t.Errorf("failed to parse --cc flag: %v", err)
		}

		ccFlag := planCmd.Flag("cc")
		if ccFlag == nil {
			t.Fatal("--cc flag not found after parsing")
		}

		if ccFlag.Value.String() != "true" {
			t.Errorf("--cc flag should be true when set, got: %s", ccFlag.Value.String())
		}
	})

	// TestPlanCommand_HelpText: Verify help text includes both Gemini and Claude options
	t.Run("help text mentions both engines", func(t *testing.T) {
		rootCmd := NewRootCommand()
		planCmd, _, err := rootCmd.Find([]string{"plan"})
		if err != nil {
			t.Fatalf("failed to find plan command: %v", err)
		}

		// Check that the Long description mentions both Gemini and Claude
		if !strings.Contains(planCmd.Long, "Gemini") {
			t.Error("plan command Long description should mention Gemini")
		}

		if !strings.Contains(planCmd.Long, "Claude") {
			t.Error("plan command Long description should mention Claude Code option")
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

	// Test successful execution
	t.Run("successful execution", func(t *testing.T) {
		// Set a dummy API key for this test
		originalKey := os.Getenv("GEMINI_API_KEY")
		defer func() {
			_ = os.Setenv("GEMINI_API_KEY", originalKey)
		}()
		_ = os.Setenv("GEMINI_API_KEY", "test-key")

		// This test will verify that Gemini is executed correctly
		// and allowed to handle file creation directly
		t.Skip("Skipping until generatePlan is refactored for testability")
	})
}

// TestPlanCommand_RouteToGeminiByDefault tests that default behavior calls Gemini
func TestPlanCommand_RouteToGeminiByDefault(t *testing.T) {
	// Save and restore environment
	originalKey := os.Getenv("GEMINI_API_KEY")
	defer func() {
		_ = os.Setenv("GEMINI_API_KEY", originalKey)
	}()
	_ = os.Setenv("GEMINI_API_KEY", "test-key")

	// Create a test prompt template
	tempDir := t.TempDir()
	promptsDir := filepath.Join(tempDir, "prompts")
	if err := os.Mkdir(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}
	promptTemplate := `Task: {{TASK}}`
	if err := os.WriteFile(filepath.Join(promptsDir, "prompt-plan.md"), []byte(promptTemplate), 0644); err != nil {
		t.Fatalf("failed to write prompt template: %v", err)
	}

	// Change to temp directory for the test
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	_ = os.Chdir(tempDir)

	// This test verifies the routing logic defaults to Gemini
	// Full implementation will come after refactoring for testability
	t.Skip("Waiting for command runner injection")
}

// TestPlanCommand_RouteToClaude tests that --cc flag routes to Claude
func TestPlanCommand_RouteToClaude(t *testing.T) {
	// Create a test prompt template
	tempDir := t.TempDir()
	promptsDir := filepath.Join(tempDir, "prompts")
	if err := os.Mkdir(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}
	promptTemplate := `Task: {{TASK}}`
	if err := os.WriteFile(filepath.Join(promptsDir, "prompt-plan.md"), []byte(promptTemplate), 0644); err != nil {
		t.Fatalf("failed to write prompt template: %v", err)
	}

	// Change to temp directory for the test
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	_ = os.Chdir(tempDir)

	// This test verifies the routing logic routes to Claude when --cc is used
	// Full implementation will come after implementing generatePlanWithClaude
	t.Skip("Waiting for generatePlanWithClaude implementation")
}

// TestPlanCommand_ErrorPropagation tests error handling from both paths
func TestPlanCommand_ErrorPropagation(t *testing.T) {
	t.Run("Gemini error propagation", func(t *testing.T) {
		// Test that errors from generatePlan are properly propagated
		rootCmd := NewRootCommand()
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)
		rootCmd.SetArgs([]string{"plan", "test task"})

		// Without GEMINI_API_KEY, it should error
		originalKey := os.Getenv("GEMINI_API_KEY")
		defer func() {
			_ = os.Setenv("GEMINI_API_KEY", originalKey)
		}()
		_ = os.Unsetenv("GEMINI_API_KEY")

		err := rootCmd.Execute()
		if err == nil {
			t.Error("expected error when GEMINI_API_KEY not set")
		}
		if !strings.Contains(err.Error(), "GEMINI_API_KEY not set") {
			t.Errorf("expected GEMINI_API_KEY error, got: %v", err)
		}
	})

	t.Run("Claude error propagation", func(t *testing.T) {
		// Test that errors from generatePlanWithClaude are properly propagated
		rootCmd := NewRootCommand()
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)
		rootCmd.SetArgs([]string{"plan", "--cc", "test task"})

		err := rootCmd.Execute()
		if err == nil {
			t.Error("expected error from generatePlanWithClaude")
		}
		// Since we're not in a directory with prompts/prompt-plan.md, we expect a prompt template error
		if !strings.Contains(err.Error(), "failed to read prompt template") {
			t.Errorf("expected prompt template error, got: %v", err)
		}
	})
}

// TestPlanCommand_RoutingLogic tests that the correct function is called based on flag
func TestPlanCommand_RoutingLogic(t *testing.T) {
	// Capture stdout to verify the routing messages
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	t.Run("default routes to Gemini", func(t *testing.T) {
		// Create a pipe to capture stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		defer func() { os.Stdout = oldStdout }()

		rootCmd := NewRootCommand()
		rootCmd.SetArgs([]string{"plan", "test task"})

		// Without GEMINI_API_KEY, we expect the Gemini-specific error
		originalKey := os.Getenv("GEMINI_API_KEY")
		defer func() {
			_ = os.Setenv("GEMINI_API_KEY", originalKey)
		}()
		_ = os.Unsetenv("GEMINI_API_KEY")

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "GEMINI_API_KEY not set") {
			t.Errorf("expected GEMINI_API_KEY error (indicating Gemini path), got: %v", err)
		}

		// Close writer and read output
		_ = w.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		outputStr := buf.String()

		// Check that output shows "Generating plan..." (Gemini message)
		if !strings.Contains(outputStr, "Generating plan...") {
			t.Errorf("expected 'Generating plan...' message, got: %s", outputStr)
		}
	})

	t.Run("--cc flag routes to Claude", func(t *testing.T) {
		// Create a pipe to capture stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		defer func() { os.Stdout = oldStdout }()

		rootCmd := NewRootCommand()
		rootCmd.SetArgs([]string{"plan", "--cc", "test task"})

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failed to read prompt template") {
			t.Errorf("expected prompt template error (indicating Claude path), got: %v", err)
		}

		// Close writer and read output
		_ = w.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		outputStr := buf.String()

		// Check that output shows "Generating plan using Claude Code..." (Claude message)
		if !strings.Contains(outputStr, "Generating plan using Claude Code...") {
			t.Errorf("expected 'Generating plan using Claude Code...' message, got: %s", outputStr)
		}
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

// TestGeneratePlanWithClaude tests the Claude plan generation logic
func TestGeneratePlanWithClaude(t *testing.T) {
	// Test basic functionality and error paths
	t.Run("handles missing prompt template", func(t *testing.T) {
		// Create a temporary directory without prompt template
		tempDir := t.TempDir()

		// Change to temp directory for the test
		originalDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalDir) }()
		_ = os.Chdir(tempDir)

		// The function should fail to read the prompt template
		err := generatePlanWithClaude("test task")

		if err == nil || !strings.Contains(err.Error(), "failed to read prompt template") {
			t.Errorf("expected prompt template error, got: %v", err)
		}
	})

	t.Run("replaces task placeholder correctly", func(t *testing.T) {
		// This test verifies that {{TASK}} is replaced in the prompt
		// We can't easily test this without mocking the executor
		// So we'll skip this for now
		t.Skip("Requires executor mocking to test prompt replacement")
	})
}

// TestGeneratePlanWithClaude_PromptTemplate verifies correct prompt template usage
func TestGeneratePlanWithClaude_PromptTemplate(t *testing.T) {
	t.Run("reads and processes prompt template", func(t *testing.T) {
		// This test will verify that the prompt template is read and {{TASK}} is replaced
		// It should test that the function:
		// 1. Reads prompts/prompt-plan.md
		// 2. Replaces {{TASK}} with the actual task
		// 3. Passes the processed prompt to Claude

		// For now, skip until implementation
		t.Skip("Waiting for generatePlanWithClaude implementation")
	})

	t.Run("handles missing prompt template", func(t *testing.T) {
		// Create a temporary directory without prompt template
		tempDir := t.TempDir()

		// Change to temp directory for the test
		originalDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalDir) }()
		_ = os.Chdir(tempDir)

		// The function should fail to read the prompt template
		err := generatePlanWithClaude("test task")

		if err == nil || !strings.Contains(err.Error(), "failed to read prompt template") {
			t.Errorf("expected prompt template error, got: %v", err)
		}
	})
}

// TestGeneratePlanWithClaude_ErrorHandling tests various error scenarios
func TestGeneratePlanWithClaude_ErrorHandling(t *testing.T) {
	t.Run("handles missing Claude CLI", func(t *testing.T) {
		// Test that appropriate error is returned when Claude CLI is not found
		t.Skip("Waiting for generatePlanWithClaude implementation")
	})

	t.Run("handles Claude execution failure", func(t *testing.T) {
		// Test error propagation from Claude executor
		t.Skip("Waiting for generatePlanWithClaude implementation")
	})

	t.Run("handles timeout", func(t *testing.T) {
		// Test timeout handling (5 minutes default)
		t.Skip("Waiting for generatePlanWithClaude implementation")
	})
}

// TestGeneratePlanWithClaude_MockExecution tests with mock executor
func TestGeneratePlanWithClaude_MockExecution(t *testing.T) {
	t.Run("executes Claude with correct configuration", func(t *testing.T) {
		// This test will use a mock executor to verify:
		// 1. ExecuteConfig is properly configured
		// 2. Temporary state file is created
		// 3. Output is streamed correctly
		// 4. Cleanup happens after execution
		t.Skip("Waiting for generatePlanWithClaude implementation")
	})

	t.Run("uses planning-specific configuration", func(t *testing.T) {
		// Verify that planning-specific settings are used:
		// - No MCP servers
		// - Planning system prompt
		// - Temporary state file
		t.Skip("Waiting for generatePlanWithClaude implementation")
	})
}
