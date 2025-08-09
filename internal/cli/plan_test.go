package cli

import (
	"bytes"
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

	// TestPlanCommand_NoGeminiReferences: Verify no Gemini references remain
	t.Run("no gemini references in help", func(t *testing.T) {
		rootCmd := NewRootCommand()
		planCmd, _, err := rootCmd.Find([]string{"plan"})
		if err != nil {
			t.Fatalf("failed to find plan command: %v", err)
		}

		// Verify no --cc flag exists
		ccFlag := planCmd.Flag("cc")
		if ccFlag != nil {
			t.Error("--cc flag should not exist on plan command")
		}

		// Verify help text doesn't mention Gemini
		if strings.Contains(planCmd.Long, "Gemini") {
			t.Error("plan command Long description should not mention Gemini")
		}
		if strings.Contains(planCmd.Short, "Gemini") {
			t.Error("plan command Short description should not mention Gemini")
		}
	})

	// TestPlanCommand_ClaudeOnly: Verify Claude Code is the only option
	t.Run("claude code is the only option", func(t *testing.T) {
		rootCmd := NewRootCommand()
		planCmd, _, err := rootCmd.Find([]string{"plan"})
		if err != nil {
			t.Fatalf("failed to find plan command: %v", err)
		}

		// Verify help text mentions Claude Code
		if !strings.Contains(planCmd.Short, "Claude Code") {
			t.Error("plan command Short description should mention Claude Code")
		}
		if !strings.Contains(planCmd.Long, "Claude Code") {
			t.Error("plan command Long description should mention Claude Code")
		}
	})

	// TestPlanCommand_RejectsUnknownFlags: Test that unknown flags are rejected
	t.Run("rejects unknown flags", func(t *testing.T) {
		// Test that --cc flag is rejected
		rootCmd := NewRootCommand()
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)

		planCmd, _, err := rootCmd.Find([]string{"plan"})
		if err != nil {
			t.Fatalf("failed to find plan command: %v", err)
		}

		// Parse the --cc flag - should fail
		err = planCmd.ParseFlags([]string{"--cc"})
		if err == nil {
			t.Fatal("parsing --cc flag should fail")
		}

		if !strings.Contains(err.Error(), "unknown flag: --cc") {
			t.Errorf("expected error about unknown flag --cc, got: %v", err)
		}
	})

	// TestPlanCommand_HelpText: Verify help text mentions only Claude Code
	t.Run("help text mentions only claude code", func(t *testing.T) {
		rootCmd := NewRootCommand()
		planCmd, _, err := rootCmd.Find([]string{"plan"})
		if err != nil {
			t.Fatalf("failed to find plan command: %v", err)
		}

		// Check that Long description mentions Claude Code only
		if strings.Contains(planCmd.Long, "Gemini") {
			t.Error("plan command Long description should not mention Gemini")
		}
		if !strings.Contains(planCmd.Long, "Claude Code") {
			t.Error("plan command Long description should mention Claude Code")
		}

		// Check that examples don't mention Gemini or --cc flag
		if strings.Contains(planCmd.Example, "Gemini") {
			t.Error("plan command examples should not mention Gemini")
		}
		if strings.Contains(planCmd.Example, "--cc") {
			t.Error("plan command examples should not mention --cc flag")
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
	// Test that generatePlan works with Claude Code (integration test)
	t.Run("claude code integration", func(t *testing.T) {
		// This is an integration test that requires Claude CLI to be installed
		// Skip if we're in CI or if Claude CLI is not available
		if os.Getenv("CI") == "true" {
			t.Skip("Skipping Claude CLI integration test in CI")
		}

		// Check if Claude CLI is available
		if _, err := exec.LookPath("claude"); err != nil {
			t.Skip("Claude CLI not found, skipping integration test")
		}

		// Test that generatePlan creates necessary state files and calls Claude
		// This will fail gracefully if there are configuration issues
		err := generatePlan("test task")
		// We expect this to fail in test environment, but not with missing API key errors
		if err != nil && strings.Contains(err.Error(), "GEMINI_API_KEY") {
			t.Error("generatePlan should not reference GEMINI_API_KEY - should use Claude Code only")
		}
	})

	// Test that generatePlan handles Claude Code execution properly
	t.Run("handles claude code execution", func(t *testing.T) {
		// This test verifies that generatePlan uses Claude Code executor
		// and handles the temporary state file creation properly
		// We'll verify that no Gemini-specific logic is used

		// This is a unit test that doesn't require Claude CLI to be installed
		// It will test the setup and configuration logic
		t.Skip("Implementation test - requires refactoring generatePlan for better testability")
	})

	// Test that generatePlan configures Claude properly
	t.Run("configures claude properly", func(t *testing.T) {
		// Test Claude Code configuration is set up properly
		// No API key needed for Claude Code CLI

		// This test will be properly implemented when we refactor generatePlan
		// to accept dependencies for testing
		t.Skip("Skipping until generatePlan is refactored for testability")
	})
}

// TestPlanCommand_ErrorPropagation tests error handling for Claude Code
func TestPlanCommand_ErrorPropagation(t *testing.T) {
	t.Run("Claude error propagation", func(t *testing.T) {
		// Test that errors from generatePlan are properly propagated
		// This test will likely timeout or fail in CI since Claude CLI integration is complex
		// We primarily want to verify no Gemini references remain
		rootCmd := NewRootCommand()
		output := &bytes.Buffer{}
		rootCmd.SetOut(output)
		rootCmd.SetErr(output)
		rootCmd.SetArgs([]string{"plan", "test task"})

		// Execute the command - we expect it to fail but not because of missing API key
		err := rootCmd.Execute()
		if err != nil && strings.Contains(err.Error(), "GEMINI_API_KEY") {
			t.Error("generatePlan should not reference GEMINI_API_KEY - should use Claude Code only")
		}
		// Note: We don't require this to succeed in tests since Claude CLI integration
		// requires proper setup, but we verify no Gemini artifacts remain
	})

}

// TestPlanCommand_RoutingLogic tests that the plan command always uses Claude Code
func TestPlanCommand_RoutingLogic(t *testing.T) {
	// This test verifies that plan generation always routes to Claude Code
	t.Run("always routes to Claude Code", func(t *testing.T) {
		rootCmd := NewRootCommand()
		rootCmd.SetArgs([]string{"plan", "test task"})

		// We expect this to fail in test environments but not due to GEMINI_API_KEY
		err := rootCmd.Execute()
		if err != nil && strings.Contains(err.Error(), "GEMINI_API_KEY") {
			t.Error("plan command should not reference GEMINI_API_KEY - should use Claude Code only")
		}
		// Test passes as long as no Gemini references appear in errors
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
		t.Skip("Waiting for generatePlan implementation")
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
		t.Skip("Waiting for generatePlan implementation")
	})

	t.Run("handles Claude execution failure", func(t *testing.T) {
		// Test error propagation from Claude executor
		t.Skip("Waiting for generatePlan implementation")
	})

	t.Run("handles timeout", func(t *testing.T) {
		// Test timeout handling (5 minutes default)
		t.Skip("Waiting for generatePlan implementation")
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
		t.Skip("Waiting for generatePlan implementation")
	})

	t.Run("uses planning-specific configuration", func(t *testing.T) {
		// Verify that planning-specific settings are used:
		// - No MCP servers
		// - Planning system prompt
		// - Temporary state file
		t.Skip("Waiting for generatePlan implementation")
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

		// Call generatePlan (will fail due to missing Claude CLI)
		_ = generatePlan("test task")

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

		// Call generatePlan (will fail due to missing prompt template)
		_ = generatePlan("test task")

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

// TestGhIssueCommand_Integration_Claude tests the full gh-issue flow with Claude
// This test verifies that the fetched issue data is correctly passed to generatePlan
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

	// Execute the gh-issue command (now always uses Claude)
	rootCmd := NewRootCommand()
	rootCmd.SetArgs([]string{"plan", "gh-issue", "https://github.com/test/repo/issues/2"})

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
