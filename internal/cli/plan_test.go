package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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

	// No need to create prompt files anymore - they're embedded

	// This test verifies the routing logic defaults to Gemini
	// Full implementation will come after refactoring for testability
	t.Skip("Waiting for command runner injection")
}

// TestPlanCommand_RouteToClaude tests that --cc flag routes to Claude
func TestPlanCommand_RouteToClaude(t *testing.T) {
	// No need to create prompt files anymore - they're embedded

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
		// With embedded prompts, we now expect a different error (e.g., Claude CLI not found)
		// The specific error depends on the execution environment
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
		// Note: Printer adds formatting prefixes like → which may not be captured in pipe
		if !strings.Contains(outputStr, "Generating plan") {
			t.Errorf("expected 'Generating plan' message, got: %s", outputStr)
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
		// With embedded prompts, we expect an error but not specifically about prompt template
		if err == nil {
			t.Error("expected error from generatePlanWithClaude")
		}

		// Close writer and read output
		_ = w.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		outputStr := buf.String()

		// Check that output shows Claude-related messages
		// The printer may format the output differently
		if !strings.Contains(outputStr, "Claude Code") && !strings.Contains(err.Error(), "prompt template") {
			t.Errorf("expected Claude Code message or prompt template error, got: %s", outputStr)
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

	// No need to create prompt files anymore - they're embedded

	t.Run("builds correct prompt with all components", func(t *testing.T) {
		// This test will be implemented once we refactor generatePlan
		// to accept a PlanGenerator interface or similar for testing
		t.Skip("Waiting for generatePlan refactoring")
	})
}

// TestGeneratePlanWithClaude tests the Claude plan generation logic
func TestGeneratePlanWithClaude(t *testing.T) {
	// Test basic functionality and error paths
	t.Run("creates temporary state file", func(t *testing.T) {
		// Prompts are now embedded, so we should test other aspects
		// This test needs proper mocking of the Claude executor
		t.Skip("Requires executor mocking to test state file creation")
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
	t.Run("processes embedded prompt template", func(t *testing.T) {
		// This test will verify that the embedded prompt template is processed correctly
		// It should test that the function:
		// 1. Uses the embedded prompt template
		// 2. Replaces {{TASK}} with the actual task
		// 3. Passes the processed prompt to Claude

		// For now, skip until implementation
		t.Skip("Waiting for generatePlanWithClaude implementation")
	})

	t.Run("handles task replacement in embedded template", func(t *testing.T) {
		// With embedded prompts, this test should verify task replacement works
		// However, we need proper mocking to test this effectively
		t.Skip("Requires executor mocking to test task replacement")
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

// TestValidatePlanFile tests the validatePlanFile function that checks if plan.md exists and has content
func TestValidatePlanFile(t *testing.T) {
	t.Run("returns nil when plan.md exists with content", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalDir) }()
		_ = os.Chdir(tempDir)

		// Create plan.md with content
		content := []byte("# Implementation Plan\n\nThis is a test plan with content.")
		if err := os.WriteFile("plan.md", content, 0644); err != nil {
			t.Fatalf("failed to write plan.md: %v", err)
		}

		// Test validatePlanFile returns nil
		err := validatePlanFile()
		if err != nil {
			t.Errorf("expected nil error for existing plan.md with content, got: %v", err)
		}
	})

	t.Run("returns error when plan.md doesn't exist", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalDir) }()
		_ = os.Chdir(tempDir)

		// Ensure plan.md doesn't exist
		_ = os.Remove("plan.md")

		// Test validatePlanFile returns error
		err := validatePlanFile()
		if err == nil {
			t.Error("expected error when plan.md doesn't exist, got nil")
		}

		// Check error message
		expectedMsg := "plan.md does not exist"
		if err != nil && err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got: %v", expectedMsg, err)
		}
	})

	t.Run("returns error when plan.md is empty (0 bytes)", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalDir) }()
		_ = os.Chdir(tempDir)

		// Create empty plan.md
		if err := os.WriteFile("plan.md", []byte{}, 0644); err != nil {
			t.Fatalf("failed to write empty plan.md: %v", err)
		}

		// Test validatePlanFile returns error
		err := validatePlanFile()
		if err == nil {
			t.Error("expected error when plan.md is empty, got nil")
		}

		// Check error message
		expectedMsg := "plan.md is empty"
		if err != nil && err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got: %v", expectedMsg, err)
		}
	})

	t.Run("handles edge case of single byte file", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalDir) }()
		_ = os.Chdir(tempDir)

		// Create plan.md with single byte
		if err := os.WriteFile("plan.md", []byte{' '}, 0644); err != nil {
			t.Fatalf("failed to write plan.md: %v", err)
		}

		// Test validatePlanFile returns nil (file has content)
		err := validatePlanFile()
		if err != nil {
			t.Errorf("expected nil error for plan.md with single byte, got: %v", err)
		}
	})

	t.Run("handles permission errors gracefully", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalDir) }()
		_ = os.Chdir(tempDir)

		// Create plan.md with no read permissions
		if err := os.WriteFile("plan.md", []byte("content"), 0000); err != nil {
			t.Fatalf("failed to write plan.md: %v", err)
		}
		defer func() { _ = os.Chmod("plan.md", 0644) }() // Restore permissions for cleanup

		// Test validatePlanFile handles permission error
		err := validatePlanFile()

		// On macOS and some systems, os.Stat might still work on files with 0000 permissions
		// The important behavior is that the function doesn't panic and returns a reasonable result
		// Either nil (file exists and has content) or an error is acceptable
		if err != nil && !strings.Contains(err.Error(), "permission denied") && !strings.Contains(err.Error(), "plan.md does not exist") {
			t.Errorf("unexpected error type: %v", err)
		}
	})
}

// TestGeneratePlanWithClaude_ProgressIndicator tests that progress indicator is shown
func TestGeneratePlanWithClaude_ProgressIndicator(t *testing.T) {
	// Test that progress indicator is shown during Claude execution
	t.Run("shows progress indicator during execution", func(t *testing.T) {
		// Create a test environment
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalDir) }()
		_ = os.Chdir(tempDir)

		// No need to create prompt files anymore - they're embedded

		// Capture output to verify progress messages
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		outputChan := make(chan string)
		go func() {
			buf := make([]byte, 1024)
			var output bytes.Buffer
			for {
				n, err := r.Read(buf)
				if err != nil {
					break
				}
				output.Write(buf[:n])
			}
			outputChan <- output.String()
		}()

		// Call generatePlanWithClaude (will fail due to missing Claude CLI)
		_ = generatePlanWithClaude("test task")

		// Restore stdout and get output
		_ = w.Close()
		os.Stdout = oldStdout
		output := <-outputChan

		// Verify initial message is shown
		if !strings.Contains(output, "Generating plan using Claude Code...") {
			t.Error("Expected to see 'Generating plan using Claude Code...' message")
		}

		// Test that progress indicator is shown
		// We should see the "Analyzing codebase" message
		if !strings.Contains(output, "Analyzing codebase and creating plan") {
			t.Error("Expected to see 'Analyzing codebase and creating plan' progress message")
		}

		// We should also see either an error message or completion message
		// In this test, Claude CLI is not installed, so we expect an error
		if !strings.Contains(output, "Claude Code CLI not found") && !strings.Contains(output, "Plan generation completed") {
			t.Error("Expected to see either error or completion message")
		}
	})

	// Test that progress indicator is properly stopped on error
	t.Run("stops progress indicator on error", func(t *testing.T) {
		// Create a test environment
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalDir) }()
		_ = os.Chdir(tempDir)

		// With embedded prompts, this test now focuses on other errors like missing Claude CLI

		// Capture output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		outputChan := make(chan string)
		go func() {
			buf := make([]byte, 1024)
			var output bytes.Buffer
			for {
				n, err := r.Read(buf)
				if err != nil {
					break
				}
				output.Write(buf[:n])
			}
			outputChan <- output.String()
		}()

		// Call generatePlanWithClaude (will fail due to missing prompt template)
		_ = generatePlanWithClaude("test task")

		// Restore stdout and get output
		_ = w.Close()
		os.Stdout = oldStdout
		output := <-outputChan

		// Progress should be stopped before error is shown
		// We should NOT see any spinner characters after the error
		// The test passes if output is clean (no terminal control sequences left)
		if strings.Contains(output, "\r") || strings.Contains(output, "\033[") {
			t.Error("Expected progress indicator to be properly stopped, found terminal control sequences")
		}
	})
}

// TestGhIssueCommand_Exists tests that the gh-issue subcommand exists under plan
// This test verifies that we can find the gh-issue command as a subcommand of plan
func TestGhIssueCommand_Exists(t *testing.T) {
	rootCmd := NewRootCommand()
	planCmd, _, err := rootCmd.Find([]string{"plan"})
	if err != nil {
		t.Fatalf("failed to find plan command: %v", err)
	}

	// Look for gh-issue subcommand
	found := false
	for _, cmd := range planCmd.Commands() {
		if cmd.Use == "gh-issue <url>" {
			found = true
			break
		}
	}

	if !found {
		t.Error("gh-issue subcommand not found under plan command")
	}
}

// TestGhIssueCommand_RequiresOneArgument tests argument validation
// This test ensures the command enforces exactly one argument (the GitHub issue URL)
func TestGhIssueCommand_RequiresOneArgument(t *testing.T) {
	t.Run("error when no arguments provided", func(t *testing.T) {
		rootCmd := NewRootCommand()
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)
		rootCmd.SetArgs([]string{"plan", "gh-issue"})

		err := rootCmd.Execute()
		if err == nil {
			t.Error("expected error when no arguments provided to gh-issue")
		}

		if !strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
			t.Errorf("expected error about requiring 1 argument, got: %v", err)
		}
	})

	t.Run("error when multiple arguments provided", func(t *testing.T) {
		rootCmd := NewRootCommand()
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)
		rootCmd.SetArgs([]string{"plan", "gh-issue", "url1", "url2"})

		err := rootCmd.Execute()
		if err == nil {
			t.Error("expected error when multiple arguments provided to gh-issue")
		}

		if !strings.Contains(err.Error(), "accepts 1 arg(s), received 2") {
			t.Errorf("expected error about accepting only 1 argument, got: %v", err)
		}
	})
}

// TestGhIssueCommand_HelpText tests the help text content
// This test verifies that the command has appropriate help text that explains its purpose
func TestGhIssueCommand_HelpText(t *testing.T) {
	rootCmd := NewRootCommand()
	ghIssueCmd, _, err := rootCmd.Find([]string{"plan", "gh-issue"})
	if err != nil {
		t.Fatalf("failed to find gh-issue command: %v", err)
	}

	// Check Short description
	if ghIssueCmd.Short == "" {
		t.Error("gh-issue command should have a Short description")
	}

	// Check Long description
	if ghIssueCmd.Long == "" {
		t.Error("gh-issue command should have a Long description")
	}

	// Verify help text mentions GitHub and gh CLI
	if !strings.Contains(ghIssueCmd.Long, "GitHub") {
		t.Error("gh-issue Long description should mention GitHub")
	}

	if !strings.Contains(ghIssueCmd.Long, "gh") {
		t.Error("gh-issue Long description should mention gh CLI")
	}
}

// TestFetchGitHubIssue_Success tests successful GitHub issue fetching
// This test verifies that when the gh CLI returns valid JSON, the title and body are correctly extracted
func TestFetchGitHubIssue_Success(t *testing.T) {
	// Skip if gh is not available
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("gh CLI not found, skipping integration test")
	}

	// This is now an integration test that requires gh to be installed
	// For unit testing, we would need to refactor to accept a command executor interface
	t.Skip("Skipping integration test - requires mock implementation")
}

// TestFetchGitHubIssue_GhNotFound tests the scenario where gh CLI is not installed
// This test ensures we provide a helpful error message when the gh command is not found
func TestFetchGitHubIssue_GhNotFound(t *testing.T) {
	// This test would require mocking the command execution
	// For now, we skip it as it requires refactoring fetchGitHubIssue
	t.Skip("Skipping test - requires mock implementation")
}

// TestFetchGitHubIssue_ApiError tests handling of gh CLI errors
// This test verifies that API errors from gh (like authentication issues) are properly handled
func TestFetchGitHubIssue_ApiError(t *testing.T) {
	// This test would require mocking the command execution
	// For now, we skip it as it requires refactoring fetchGitHubIssue
	t.Skip("Skipping test - requires mock implementation")
}

// TestFetchGitHubIssue_InvalidJSON tests handling of invalid JSON from gh
// This test ensures we handle cases where gh returns malformed JSON
func TestFetchGitHubIssue_InvalidJSON(t *testing.T) {
	// This test would require mocking the command execution
	// For now, we skip it as it requires refactoring fetchGitHubIssue
	t.Skip("Skipping test - requires mock implementation")
}

// TestFetchGitHubIssue_EmptyResponse tests handling of empty response
// This test verifies behavior when gh returns empty or minimal JSON
func TestFetchGitHubIssue_EmptyResponse(t *testing.T) {
	// This test would require mocking the command execution
	// For now, we skip it as it requires refactoring fetchGitHubIssue
	t.Skip("Skipping test - requires mock implementation")
}

// TestGhIssueCommand_Integration_Gemini tests the full gh-issue flow with Gemini
// This test verifies that the fetched issue data is correctly passed to generatePlan
func TestGhIssueCommand_Integration_Gemini(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(tmpDir)

	// Initialize git repo for testing
	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create mock gh command
	mockGhScript := `#!/bin/bash
if [[ "$1" == "issue" && "$2" == "view" && "$4" == "--json" && "$5" == "title,body" ]]; then
    echo '{"title":"Test Issue Title","body":"Test issue body content"}'
    exit 0
fi
exit 1
`
	mockGhPath := filepath.Join(tmpDir, "gh")
	if err := os.WriteFile(mockGhPath, []byte(mockGhScript), 0755); err != nil {
		t.Fatalf("Failed to create mock gh: %v", err)
	}

	// Add mock directory to PATH
	oldPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", tmpDir+":"+oldPath)
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	// Also need to mock gemini for the plan generation
	mockGeminiScript := `#!/bin/bash
echo "# Generated Plan"
echo ""
echo "## Task: Test Issue Title"
echo ""
echo "Test issue body content"
`
	mockGeminiPath := filepath.Join(tmpDir, "gemini")
	if err := os.WriteFile(mockGeminiPath, []byte(mockGeminiScript), 0755); err != nil {
		t.Fatalf("Failed to create mock gemini: %v", err)
	}

	// Execute the command
	rootCmd := NewRootCommand()
	rootCmd.SetArgs([]string{"plan", "gh-issue", "https://github.com/test/repo/issues/1"})

	// Capture output
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// Verify plan.md was created with the combined task description
	planContent, err := os.ReadFile("plan.md")
	if err != nil {
		t.Fatalf("Failed to read plan.md: %v", err)
	}

	// Check that the plan contains both title and body
	if !strings.Contains(string(planContent), "Test Issue Title") {
		t.Error("plan.md should contain the issue title")
	}
	if !strings.Contains(string(planContent), "Test issue body content") {
		t.Error("plan.md should contain the issue body")
	}
}

// TestGhIssueCommand_Integration_Claude tests the full gh-issue flow with Claude
// This test verifies that the fetched issue data is correctly passed to generatePlanWithClaude
func TestGhIssueCommand_Integration_Claude(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	_ = os.Chdir(tmpDir)

	// Initialize git repo for testing
	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create mock gh command
	mockGhScript := `#!/bin/bash
if [[ "$1" == "issue" && "$2" == "view" && "$4" == "--json" && "$5" == "title,body" ]]; then
    echo '{"title":"Claude Test Issue","body":"This is a test for Claude integration"}'
    exit 0
fi
exit 1
`
	mockGhPath := filepath.Join(tmpDir, "gh")
	if err := os.WriteFile(mockGhPath, []byte(mockGhScript), 0755); err != nil {
		t.Fatalf("Failed to create mock gh: %v", err)
	}

	// Add mock directory to PATH
	oldPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", tmpDir+":"+oldPath)
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	// Mock claude command
	mockClaudeScript := `#!/bin/bash
# Create a mock plan.md file as Claude would
cat > plan.md << EOF
# Implementation Plan

## Task: Claude Test Issue

This is a test for Claude integration

Generated by Claude
EOF
`
	mockClaudePath := filepath.Join(tmpDir, "claude")
	if err := os.WriteFile(mockClaudePath, []byte(mockClaudeScript), 0755); err != nil {
		t.Fatalf("Failed to create mock claude: %v", err)
	}

	// Execute the command with --cc flag
	rootCmd := NewRootCommand()
	rootCmd.SetArgs([]string{"plan", "--cc", "gh-issue", "https://github.com/test/repo/issues/2"})

	// Capture output
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// Verify plan.md was created with Claude's output
	planContent, err := os.ReadFile("plan.md")
	if err != nil {
		t.Fatalf("Failed to read plan.md: %v", err)
	}

	// Check that the plan contains the expected content
	if !strings.Contains(string(planContent), "Claude Test Issue") {
		t.Error("plan.md should contain the issue title")
	}
	if !strings.Contains(string(planContent), "This is a test for Claude integration") {
		t.Error("plan.md should contain the issue body")
	}
	if !strings.Contains(string(planContent), "Generated by Claude") {
		t.Error("plan.md should indicate it was generated by Claude")
	}
}

// TestGeneratePlan_RetryLoop tests the retry mechanism for plan.md generation
func TestGeneratePlan_RetryLoop(t *testing.T) {
	// Test successful generation on first attempt (no retries)
	t.Run("successful generation on first attempt", func(t *testing.T) {
		// Create a temporary directory for test
		tmpDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(oldDir) }()
		_ = os.Chdir(tmpDir)

		// Set GEMINI_API_KEY
		originalKey := os.Getenv("GEMINI_API_KEY")
		defer func() { _ = os.Setenv("GEMINI_API_KEY", originalKey) }()
		_ = os.Setenv("GEMINI_API_KEY", "test-key")

		// Create mock gemini that creates plan.md on first attempt
		mockGeminiScript := `#!/bin/bash
echo "# Generated Plan" > plan.md
echo "Task implementation plan" >> plan.md
echo "Gemini execution successful"
exit 0
`
		mockGeminiPath := filepath.Join(tmpDir, "gemini")
		if err := os.WriteFile(mockGeminiPath, []byte(mockGeminiScript), 0755); err != nil {
			t.Fatalf("Failed to create mock gemini: %v", err)
		}

		// Add mock directory to PATH
		oldPath := os.Getenv("PATH")
		_ = os.Setenv("PATH", tmpDir+":"+oldPath)
		defer func() { _ = os.Setenv("PATH", oldPath) }()

		// Capture output to verify no retry messages
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		outputChan := make(chan string)
		go func() {
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			outputChan <- buf.String()
		}()

		// Execute generatePlan
		err := generatePlan("test task")

		// Restore stdout and get output
		_ = w.Close()
		os.Stdout = oldStdout
		output := <-outputChan

		// Verify success
		if err != nil {
			t.Errorf("Expected success on first attempt, got error: %v", err)
		}

		// Verify plan.md exists and has content
		info, err := os.Stat("plan.md")
		if err != nil {
			t.Errorf("plan.md should exist after successful generation: %v", err)
		}
		if info.Size() == 0 {
			t.Error("plan.md should have content")
		}

		// Verify output shows "Generating plan..." only once
		if strings.Count(output, "Generating plan...") != 1 {
			t.Errorf("Expected 'Generating plan...' to appear exactly once, but found %d occurrences",
				strings.Count(output, "Generating plan..."))
		}

		// Should show "Attempt 1 of 3..." but no retry messages
		if !strings.Contains(output, "Attempt 1 of 3...") {
			t.Error("Expected to see 'Attempt 1 of 3...' message")
		}
		if strings.Contains(output, "retry") {
			t.Error("Should not show retry messages on first successful attempt")
		}
	})

	// Test successful generation on second attempt (1 retry)
	t.Run("successful generation on second attempt", func(t *testing.T) {
		// Create a temporary directory for test
		tmpDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(oldDir) }()
		_ = os.Chdir(tmpDir)

		// Set GEMINI_API_KEY
		originalKey := os.Getenv("GEMINI_API_KEY")
		defer func() { _ = os.Setenv("GEMINI_API_KEY", originalKey) }()
		_ = os.Setenv("GEMINI_API_KEY", "test-key")

		// Create a counter file to track attempts
		counterFile := filepath.Join(tmpDir, "counter")
		_ = os.WriteFile(counterFile, []byte("1"), 0644)

		// Create mock gemini that fails first time, succeeds second time
		mockGeminiScript := fmt.Sprintf(`#!/bin/bash
# Read the attempt counter
counter=$(cat %s)

if [ "$counter" -eq "1" ]; then
    # First attempt - don't create plan.md
    echo "First attempt - simulating failure"
    echo "2" > %s
    exit 0
else
    # Second attempt - create plan.md
    echo "# Generated Plan" > plan.md
    echo "Task implementation plan - created on second attempt" >> plan.md
    echo "Second attempt - success"
    exit 0
fi
`, counterFile, counterFile)

		mockGeminiPath := filepath.Join(tmpDir, "gemini")
		if err := os.WriteFile(mockGeminiPath, []byte(mockGeminiScript), 0755); err != nil {
			t.Fatalf("Failed to create mock gemini: %v", err)
		}

		// Add mock directory to PATH
		oldPath := os.Getenv("PATH")
		_ = os.Setenv("PATH", tmpDir+":"+oldPath)
		defer func() { _ = os.Setenv("PATH", oldPath) }()

		// Capture output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		outputChan := make(chan string)
		go func() {
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			outputChan <- buf.String()
		}()

		// Execute generatePlan
		err := generatePlan("test task")

		// Restore stdout and get output
		_ = w.Close()
		os.Stdout = oldStdout
		output := <-outputChan

		// Verify success
		if err != nil {
			t.Errorf("Expected success on second attempt, got error: %v", err)
		}

		// Verify plan.md exists and has content
		planContent, err := os.ReadFile("plan.md")
		if err != nil {
			t.Errorf("plan.md should exist after successful generation: %v", err)
		}
		if !strings.Contains(string(planContent), "created on second attempt") {
			t.Error("plan.md should be created on second attempt")
		}

		// Verify output shows retry messages
		if !strings.Contains(output, "Generating plan...") {
			t.Error("Expected to see 'Generating plan...' message for first attempt")
		}
		if !strings.Contains(output, "Attempt 2 of 3") {
			t.Error("Expected to see 'Attempt 2 of 3' message")
		}
		// Should not see third attempt
		if strings.Contains(output, "Attempt 3 of 3") {
			t.Error("Should not see 'Attempt 3 of 3' when successful on second attempt")
		}
	})

	// Test successful generation on third attempt (2 retries)
	t.Run("successful generation on third attempt", func(t *testing.T) {
		// Create a temporary directory for test
		tmpDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(oldDir) }()
		_ = os.Chdir(tmpDir)

		// Set GEMINI_API_KEY
		originalKey := os.Getenv("GEMINI_API_KEY")
		defer func() { _ = os.Setenv("GEMINI_API_KEY", originalKey) }()
		_ = os.Setenv("GEMINI_API_KEY", "test-key")

		// Create a counter file to track attempts
		counterFile := filepath.Join(tmpDir, "counter")
		_ = os.WriteFile(counterFile, []byte("1"), 0644)

		// Create mock gemini that fails twice, succeeds third time
		mockGeminiScript := fmt.Sprintf(`#!/bin/bash
# Read the attempt counter
counter=$(cat %s)

if [ "$counter" -eq "1" ]; then
    # First attempt - don't create plan.md
    echo "First attempt - simulating failure"
    echo "2" > %s
    exit 0
elif [ "$counter" -eq "2" ]; then
    # Second attempt - still don't create plan.md
    echo "Second attempt - simulating failure"
    echo "3" > %s
    exit 0
else
    # Third attempt - create plan.md
    echo "# Generated Plan" > plan.md
    echo "Task implementation plan - created on third attempt" >> plan.md
    echo "Third attempt - success"
    exit 0
fi
`, counterFile, counterFile, counterFile)

		mockGeminiPath := filepath.Join(tmpDir, "gemini")
		if err := os.WriteFile(mockGeminiPath, []byte(mockGeminiScript), 0755); err != nil {
			t.Fatalf("Failed to create mock gemini: %v", err)
		}

		// Add mock directory to PATH
		oldPath := os.Getenv("PATH")
		_ = os.Setenv("PATH", tmpDir+":"+oldPath)
		defer func() { _ = os.Setenv("PATH", oldPath) }()

		// Capture output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		outputChan := make(chan string)
		go func() {
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			outputChan <- buf.String()
		}()

		// Execute generatePlan
		err := generatePlan("test task")

		// Restore stdout and get output
		_ = w.Close()
		os.Stdout = oldStdout
		output := <-outputChan

		// Verify success
		if err != nil {
			t.Errorf("Expected success on third attempt, got error: %v", err)
		}

		// Verify plan.md exists and has content
		planContent, err := os.ReadFile("plan.md")
		if err != nil {
			t.Errorf("plan.md should exist after successful generation: %v", err)
		}
		if !strings.Contains(string(planContent), "created on third attempt") {
			t.Error("plan.md should be created on third attempt")
		}

		// Verify output shows all retry messages
		if !strings.Contains(output, "Generating plan...") {
			t.Error("Expected to see 'Generating plan...' message for first attempt")
		}
		if !strings.Contains(output, "Attempt 2 of 3") {
			t.Error("Expected to see 'Attempt 2 of 3' message")
		}
		if !strings.Contains(output, "Attempt 3 of 3") {
			t.Error("Expected to see 'Attempt 3 of 3' message")
		}
	})

	// Test failure after 3 attempts returns error
	t.Run("failure after 3 attempts returns error", func(t *testing.T) {
		// Create a temporary directory for test
		tmpDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(oldDir) }()
		_ = os.Chdir(tmpDir)

		// Set GEMINI_API_KEY
		originalKey := os.Getenv("GEMINI_API_KEY")
		defer func() { _ = os.Setenv("GEMINI_API_KEY", originalKey) }()
		_ = os.Setenv("GEMINI_API_KEY", "test-key")

		// Create mock gemini that never creates plan.md
		mockGeminiScript := `#!/bin/bash
echo "Simulating Gemini execution that doesn't create plan.md"
exit 0
`
		mockGeminiPath := filepath.Join(tmpDir, "gemini")
		if err := os.WriteFile(mockGeminiPath, []byte(mockGeminiScript), 0755); err != nil {
			t.Fatalf("Failed to create mock gemini: %v", err)
		}

		// Add mock directory to PATH
		oldPath := os.Getenv("PATH")
		_ = os.Setenv("PATH", tmpDir+":"+oldPath)
		defer func() { _ = os.Setenv("PATH", oldPath) }()

		// Capture output (both stdout and stderr)
		oldStdout := os.Stdout
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stdout = w
		os.Stderr = w

		outputChan := make(chan string)
		go func() {
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			outputChan <- buf.String()
		}()

		// Execute generatePlan
		err := generatePlan("test task")

		// Restore stdout/stderr and get output
		_ = w.Close()
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		output := <-outputChan

		// Verify error is returned
		if err == nil {
			t.Error("Expected error after 3 failed attempts, got nil")
		}

		// Verify error message is exactly as specified
		expectedError := "gemini failed to create plan"
		if err != nil && err.Error() != expectedError {
			t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
		}

		// Verify plan.md does not exist
		if _, err := os.Stat("plan.md"); !os.IsNotExist(err) {
			t.Error("plan.md should not exist after all attempts fail")
		}

		// Verify output shows all attempts (first attempt shows "Generating plan...")
		if !strings.Contains(output, "Generating plan...") {
			t.Error("Expected to see 'Generating plan...' message for first attempt")
		}
		if !strings.Contains(output, "Attempt 2 of 3") {
			t.Error("Expected to see 'Attempt 2 of 3' message")
		}
		if !strings.Contains(output, "Attempt 3 of 3") {
			t.Error("Expected to see 'Attempt 3 of 3' message")
		}

		// Verify final error message in output
		if !strings.Contains(output, "Gemini failed to create plan") {
			t.Error("Expected to see 'Gemini failed to create plan' error message in output")
		}
	})
}

// TestGeneratePlan_ProgressMessages tests that progress messages are shown correctly
// according to Task 3 requirements from plan.md
func TestGeneratePlan_ProgressMessages(t *testing.T) {
	// Test that "Attempt 1 of 3..." is shown before first execution
	t.Run("shows Attempt 1 of 3 before first execution", func(t *testing.T) {
		// Create a temporary directory for test
		tmpDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(oldDir) }()
		_ = os.Chdir(tmpDir)

		// Set GEMINI_API_KEY
		originalKey := os.Getenv("GEMINI_API_KEY")
		defer func() { _ = os.Setenv("GEMINI_API_KEY", originalKey) }()
		_ = os.Setenv("GEMINI_API_KEY", "test-key")

		// Create mock gemini that creates plan.md on first attempt
		mockGeminiScript := `#!/bin/bash
echo "# Generated Plan" > plan.md
echo "Task implementation plan" >> plan.md
echo "Gemini execution successful"
exit 0
`
		mockGeminiPath := filepath.Join(tmpDir, "gemini")
		if err := os.WriteFile(mockGeminiPath, []byte(mockGeminiScript), 0755); err != nil {
			t.Fatalf("Failed to create mock gemini: %v", err)
		}

		// Add mock directory to PATH
		oldPath := os.Getenv("PATH")
		_ = os.Setenv("PATH", tmpDir+":"+oldPath)
		defer func() { _ = os.Setenv("PATH", oldPath) }()

		// Capture output to verify progress messages
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		outputChan := make(chan string)
		go func() {
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			outputChan <- buf.String()
		}()

		// Execute generatePlan
		err := generatePlan("test task")

		// Restore stdout and get output
		_ = w.Close()
		os.Stdout = oldStdout
		output := <-outputChan

		// Verify success
		if err != nil {
			t.Errorf("Expected success on first attempt, got error: %v", err)
		}

		// Verify output shows "Attempt 1 of 3..." before first execution
		if !strings.Contains(output, "Attempt 1 of 3...") {
			t.Error("Expected to see 'Attempt 1 of 3...' message before first execution")
		}

		// Verify "Generating plan..." is shown only on first attempt
		if strings.Count(output, "Generating plan...") != 1 {
			t.Errorf("Expected 'Generating plan...' to appear exactly once, but found %d occurrences",
				strings.Count(output, "Generating plan..."))
		}

		// Verify order: "Attempt 1 of 3..." should come before "Generating plan..."
		attemptIdx := strings.Index(output, "Attempt 1 of 3...")
		generatingIdx := strings.Index(output, "Generating plan...")
		if attemptIdx == -1 || generatingIdx == -1 || attemptIdx > generatingIdx {
			t.Error("'Attempt 1 of 3...' should appear before 'Generating plan...'")
		}
	})

	// Test that "Generating plan..." is shown only on first attempt
	t.Run("shows Generating plan only on first attempt", func(t *testing.T) {
		// Create a temporary directory for test
		tmpDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(oldDir) }()
		_ = os.Chdir(tmpDir)

		// Set GEMINI_API_KEY
		originalKey := os.Getenv("GEMINI_API_KEY")
		defer func() { _ = os.Setenv("GEMINI_API_KEY", originalKey) }()
		_ = os.Setenv("GEMINI_API_KEY", "test-key")

		// Create a counter file to track attempts
		counterFile := filepath.Join(tmpDir, "counter")
		_ = os.WriteFile(counterFile, []byte("1"), 0644)

		// Create mock gemini that fails first time, succeeds second time
		mockGeminiScript := fmt.Sprintf(`#!/bin/bash
# Read the attempt counter
counter=$(cat %s)

if [ "$counter" -eq "1" ]; then
    # First attempt - don't create plan.md
    echo "First attempt - simulating failure"
    echo "2" > %s
    exit 0
else
    # Second attempt - create plan.md
    echo "# Generated Plan" > plan.md
    echo "Task implementation plan - created on second attempt" >> plan.md
    echo "Second attempt - success"
    exit 0
fi
`, counterFile, counterFile)

		mockGeminiPath := filepath.Join(tmpDir, "gemini")
		if err := os.WriteFile(mockGeminiPath, []byte(mockGeminiScript), 0755); err != nil {
			t.Fatalf("Failed to create mock gemini: %v", err)
		}

		// Add mock directory to PATH
		oldPath := os.Getenv("PATH")
		_ = os.Setenv("PATH", tmpDir+":"+oldPath)
		defer func() { _ = os.Setenv("PATH", oldPath) }()

		// Capture output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		outputChan := make(chan string)
		go func() {
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			outputChan <- buf.String()
		}()

		// Execute generatePlan
		err := generatePlan("test task")

		// Restore stdout and get output
		_ = w.Close()
		os.Stdout = oldStdout
		output := <-outputChan

		// Verify success
		if err != nil {
			t.Errorf("Expected success on second attempt, got error: %v", err)
		}

		// Verify "Generating plan..." appears only once (on first attempt)
		if strings.Count(output, "Generating plan...") != 1 {
			t.Errorf("Expected 'Generating plan...' to appear exactly once, but found %d occurrences",
				strings.Count(output, "Generating plan..."))
		}

		// Verify correct attempt messages
		if !strings.Contains(output, "Attempt 1 of 3...") {
			t.Error("Expected to see 'Attempt 1 of 3...' message")
		}
		if !strings.Contains(output, "Attempt 2 of 3...") {
			t.Error("Expected to see 'Attempt 2 of 3...' message")
		}

		// Should NOT see duplicate "Generating plan..." on retry
		lines := strings.Split(output, "\n")
		generatingCount := 0
		for _, line := range lines {
			if strings.Contains(line, "Generating plan...") {
				generatingCount++
			}
		}
		if generatingCount > 1 {
			t.Errorf("'Generating plan...' should only appear once, found %d times", generatingCount)
		}
	})

	// Test correct messages shown for each attempt
	t.Run("correct messages shown for each attempt", func(t *testing.T) {
		// Create a temporary directory for test
		tmpDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(oldDir) }()
		_ = os.Chdir(tmpDir)

		// Set GEMINI_API_KEY
		originalKey := os.Getenv("GEMINI_API_KEY")
		defer func() { _ = os.Setenv("GEMINI_API_KEY", originalKey) }()
		_ = os.Setenv("GEMINI_API_KEY", "test-key")

		// Create a counter file to track attempts
		counterFile := filepath.Join(tmpDir, "counter")
		_ = os.WriteFile(counterFile, []byte("1"), 0644)

		// Create mock gemini that fails twice, succeeds third time
		mockGeminiScript := fmt.Sprintf(`#!/bin/bash
# Read the attempt counter
counter=$(cat %s)

if [ "$counter" -eq "1" ]; then
    echo "First attempt - simulating failure"
    echo "2" > %s
    exit 0
elif [ "$counter" -eq "2" ]; then
    echo "Second attempt - simulating failure"
    echo "3" > %s
    exit 0
else
    echo "# Generated Plan" > plan.md
    echo "Task implementation plan - created on third attempt" >> plan.md
    echo "Third attempt - success"
    exit 0
fi
`, counterFile, counterFile, counterFile)

		mockGeminiPath := filepath.Join(tmpDir, "gemini")
		if err := os.WriteFile(mockGeminiPath, []byte(mockGeminiScript), 0755); err != nil {
			t.Fatalf("Failed to create mock gemini: %v", err)
		}

		// Add mock directory to PATH
		oldPath := os.Getenv("PATH")
		_ = os.Setenv("PATH", tmpDir+":"+oldPath)
		defer func() { _ = os.Setenv("PATH", oldPath) }()

		// Capture output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		outputChan := make(chan string)
		go func() {
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			outputChan <- buf.String()
		}()

		// Execute generatePlan
		err := generatePlan("test task")

		// Restore stdout and get output
		_ = w.Close()
		os.Stdout = oldStdout
		output := <-outputChan

		// Verify success
		if err != nil {
			t.Errorf("Expected success on third attempt, got error: %v", err)
		}

		// Verify all attempt messages appear in correct order
		messages := []string{
			"Attempt 1 of 3...",
			"Generating plan...",
			"Attempt 2 of 3...",
			"Attempt 3 of 3...",
		}

		// Check that all messages appear
		for _, msg := range messages {
			if !strings.Contains(output, msg) {
				t.Errorf("Expected to see '%s' in output", msg)
			}
		}

		// Verify order of messages
		indices := make([]int, len(messages))
		for i, msg := range messages {
			indices[i] = strings.Index(output, msg)
			if indices[i] == -1 {
				t.Errorf("Message '%s' not found in output", msg)
			}
		}

		// Check that indices are in increasing order
		for i := 1; i < len(indices); i++ {
			if indices[i] <= indices[i-1] {
				t.Errorf("Message '%s' should appear after '%s'", messages[i], messages[i-1])
			}
		}

		// Verify "Generating plan..." appears only once
		if strings.Count(output, "Generating plan...") != 1 {
			t.Errorf("Expected 'Generating plan...' to appear exactly once, but found %d occurrences",
				strings.Count(output, "Generating plan..."))
		}
	})
}

// TestPlanCommand_WorktreeFlags tests the --worktree and --cleanup flags
func TestPlanCommand_WorktreeFlags(t *testing.T) {
	planCmd := NewPlanCommand()

	t.Run("should have a --worktree flag", func(t *testing.T) {
		worktreeFlag := planCmd.Flags().Lookup("worktree")
		if worktreeFlag == nil {
			t.Fatal("--worktree flag not found on plan command")
		}
		if worktreeFlag.Value.Type() != "bool" {
			t.Errorf("--worktree flag should be a boolean, got: %s", worktreeFlag.Value.Type())
		}
		if worktreeFlag.DefValue != "false" {
			t.Errorf("--worktree flag should default to false, got: %s", worktreeFlag.DefValue)
		}
		if !strings.Contains(worktreeFlag.Usage, "Generate the plan in an isolated git worktree") {
			t.Errorf("--worktree flag usage text is incorrect: %s", worktreeFlag.Usage)
		}
	})

	t.Run("should have a --cleanup flag", func(t *testing.T) {
		cleanupFlag := planCmd.Flags().Lookup("cleanup")
		if cleanupFlag == nil {
			t.Fatal("--cleanup flag not found on plan command")
		}
		if cleanupFlag.Value.Type() != "bool" {
			t.Errorf("--cleanup flag should be a boolean, got: %s", cleanupFlag.Value.Type())
		}
		if cleanupFlag.DefValue != "true" {
			t.Errorf("--cleanup flag should default to true, got: %s", cleanupFlag.DefValue)
		}
		if !strings.Contains(cleanupFlag.Usage, "Automatically clean up (remove) the worktree") {
			t.Errorf("--cleanup flag usage text is incorrect: %s", cleanupFlag.Usage)
		}
	})

	t.Run("parses worktree flag correctly", func(t *testing.T) {
		// Test with --worktree flag set
		err := planCmd.ParseFlags([]string{"--worktree"})
		if err != nil {
			t.Errorf("failed to parse --worktree flag: %v", err)
		}

		worktreeFlag := planCmd.Flag("worktree")
		if worktreeFlag == nil {
			t.Fatal("--worktree flag not found after parsing")
		}

		if worktreeFlag.Value.String() != "true" {
			t.Errorf("--worktree flag should be true when set, got: %s", worktreeFlag.Value.String())
		}
	})

	t.Run("parses cleanup flag correctly", func(t *testing.T) {
		// Reset command for fresh parsing
		planCmd = NewPlanCommand()

		// Test with --cleanup=false
		err := planCmd.ParseFlags([]string{"--cleanup=false"})
		if err != nil {
			t.Errorf("failed to parse --cleanup flag: %v", err)
		}

		cleanupFlag := planCmd.Flag("cleanup")
		if cleanupFlag == nil {
			t.Fatal("--cleanup flag not found after parsing")
		}

		if cleanupFlag.Value.String() != "false" {
			t.Errorf("--cleanup flag should be false when set to false, got: %s", cleanupFlag.Value.String())
		}
	})

	t.Run("gh-issue subcommand inherits worktree flags", func(t *testing.T) {
		// Find the gh-issue subcommand
		var ghIssueCmd *cobra.Command
		for _, cmd := range planCmd.Commands() {
			if cmd.Use == "gh-issue <url>" {
				ghIssueCmd = cmd
				break
			}
		}

		if ghIssueCmd == nil {
			t.Fatal("gh-issue subcommand not found")
		}

		// The flags should be accessible from parent command
		// This test verifies the subcommand can access parent flags
		parentFlags := ghIssueCmd.Parent()
		if parentFlags == nil {
			t.Skip("Cannot test parent flag access without full command setup")
		}
	})
}

// TestPlanExecution_WithWorktree tests the worktree logic integration
func TestPlanExecution_WithWorktree(t *testing.T) {
	// Skip if not in a git repository
	if _, err := exec.Command("git", "status").Output(); err != nil {
		t.Skip("Not in a git repository, skipping worktree tests")
	}

	t.Run("plan generation with worktree", func(t *testing.T) {
		// Create a temporary directory for test
		tempDir := t.TempDir()
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current directory: %v", err)
		}
		defer func() { _ = os.Chdir(originalDir) }()

		// Change to temp directory
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change to temp directory: %v", err)
		}

		// Initialize git repo
		if err := exec.Command("git", "init").Run(); err != nil {
			t.Fatalf("failed to initialize git repo: %v", err)
		}

		// Create initial commit (worktrees require at least one commit)
		if err := os.WriteFile("README.md", []byte("Test repo"), 0644); err != nil {
			t.Fatalf("failed to create README.md: %v", err)
		}
		if err := exec.Command("git", "add", ".").Run(); err != nil {
			t.Fatalf("failed to add files: %v", err)
		}
		if err := exec.Command("git", "commit", "-m", "Initial commit").Run(); err != nil {
			t.Fatalf("failed to create initial commit: %v", err)
		}

		// This test verifies that the worktree flag is recognized
		// Full implementation test will require mocking or actual execution
		rootCmd := NewRootCommand()
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)

		// Set up environment to skip actual plan generation
		_ = os.Setenv("GEMINI_API_KEY", "test-key")
		defer func() { _ = os.Unsetenv("GEMINI_API_KEY") }()

		// The actual worktree implementation will be tested after implementation
		// For now, this test serves as a placeholder for integration testing
	})

	t.Run("plan generation with worktree and no cleanup", func(t *testing.T) {
		// Similar setup as above
		// This test will verify that cleanup=false leaves the worktree intact
		// Full implementation after the worktree logic is added
	})
}

// TestRunPlanInWorktree tests the helper function that handles worktree logic
func TestRunPlanInWorktree(t *testing.T) {
	// This test will be implemented when we add the runPlanInWorktree function
	// It will test:
	// 1. Worktree creation
	// 2. Directory changes
	// 3. Plan generation execution
	// 4. Cleanup behavior
	// 5. Error handling
	t.Skip("Waiting for runPlanInWorktree implementation")
}
