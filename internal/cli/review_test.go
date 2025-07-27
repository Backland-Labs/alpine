package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReviewCommandExists tests that the review command is registered in the root command
func TestReviewCommandExists(t *testing.T) {
	rootCmd := NewRootCommand()
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "review <plan-file>" {
			found = true
			break
		}
	}
	assert.True(t, found, "review command not found in rootCmd")
}

// TestReviewCommand_RequiresOneArgument tests that the review command returns an error
// if no arguments or more than one argument are provided.
func TestReviewCommand_RequiresOneArgument(t *testing.T) {
	rootCmd := NewRootCommand()
	output := &bytes.Buffer{}
	rootCmd.SetOut(output)
	rootCmd.SetErr(output)

	// Test with no arguments
	rootCmd.SetArgs([]string{"review"})
	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s), received 0")

	// Test with two arguments
	rootCmd.SetArgs([]string{"review", "plan1.md", "plan2.md"})
	err = rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s), received 2")
}

// TestReviewCommand_FileDoesNotExist tests that the command returns an error
// if the provided path to plan.md does not exist.
func TestReviewCommand_FileDoesNotExist(t *testing.T) {
	rootCmd := NewRootCommand()
	output := &bytes.Buffer{}
	rootCmd.SetOut(output)
	rootCmd.SetErr(output)

	// Test with a non-existent file
	rootCmd.SetArgs([]string{"review", "non-existent-plan.md"})
	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plan file not found: non-existent-plan.md")
}

// TestReviewCommand_Success tests the command with a valid, existing file.
func TestReviewCommand_Success(t *testing.T) {
	// Create a temporary plan file
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.md")
	planContent := `# Test Plan

## Task 1: Implement Feature A ✅ IMPLEMENTED
- Status: Complete

## Task 2: Implement Feature B
- Status: Pending
`
	err := os.WriteFile(planFile, []byte(planContent), 0644)
	require.NoError(t, err)

	rootCmd := NewRootCommand()
	output := &bytes.Buffer{}
	rootCmd.SetOut(output)
	rootCmd.SetErr(output)

	// Test with the existing file
	rootCmd.SetArgs([]string{"review", planFile})
	err = rootCmd.Execute()
	assert.NoError(t, err)

	// Check that output contains review information
	outputStr := output.String()
	assert.Contains(t, outputStr, "Reviewing plan file:")
	assert.Contains(t, outputStr, "Tasks found: 2")
	assert.Contains(t, outputStr, "Implemented: 1")
	assert.Contains(t, outputStr, "Pending: 1")
}
