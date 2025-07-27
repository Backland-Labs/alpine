package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContinueFlag(t *testing.T) {
	t.Run("continue flag is registered", func(t *testing.T) {
		// Test that the --continue flag exists on the root command
		rootCmd := NewRootCommand()
		continueFlag := rootCmd.Flags().Lookup("continue")
		require.NotNil(t, continueFlag, "--continue flag should be registered")
		assert.Equal(t, "bool", continueFlag.Value.Type(), "--continue flag should be of type bool")
		assert.Equal(t, "Continue from existing state (equivalent to --no-plan --no-worktree)", continueFlag.Usage)
	})

	t.Run("continue flag cannot be used with task argument", func(t *testing.T) {
		// Test that --continue flag conflicts with providing a task
		rootCmd := NewRootCommand()
		rootCmd.SetArgs([]string{"--continue", "some task"})

		// Args validation should fail
		err := rootCmd.ParseFlags([]string{"--continue", "some task"})
		require.NoError(t, err) // Parsing should succeed

		// But validation should fail
		err = rootCmd.Args(rootCmd, []string{"some task"})
		assert.Error(t, err, "should fail when both --continue and task are provided")
		assert.Contains(t, err.Error(), "cannot use --continue with a task description", "error should mention the conflict")
	})

	t.Run("continue flag allows no arguments", func(t *testing.T) {
		// Test that --continue flag allows running without task argument
		rootCmd := NewRootCommand()
		rootCmd.SetArgs([]string{"--continue"})

		// Parse flags
		err := rootCmd.ParseFlags([]string{"--continue"})
		require.NoError(t, err)

		// Args validation should pass with empty args
		err = rootCmd.Args(rootCmd, []string{})
		assert.NoError(t, err, "should allow empty args with --continue")
	})

	t.Run("continue flag requires state file", func(t *testing.T) {
		// Test that workflow checks for state file when --continue is used
		mockFileReader := &mockFileReader{
			err: errors.New("file not found"),
		}

		deps := &Dependencies{
			FileReader:   mockFileReader,
			ConfigLoader: &mockConfigLoader{},
		}

		// Call with --continue flag
		err := runWorkflowWithDependencies(context.Background(), []string{}, true, true, true, deps)

		// Should get error about missing state file
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no existing state file found", "should mention missing state file")
	})

	t.Run("continue flag with existing state file", func(t *testing.T) {
		// Test that workflow proceeds when state file exists
		mockFileReader := &mockFileReader{
			content: `{"status": "running"}`,
		}

		mockEngine := &mockWorkflowEngine{}

		deps := &Dependencies{
			FileReader:     mockFileReader,
			ConfigLoader:   &mockConfigLoader{},
			WorkflowEngine: mockEngine,
		}

		// Call with --continue flag
		err := runWorkflowWithDependencies(context.Background(), []string{}, true, true, true, deps)

		// Should succeed and pass empty task description to engine
		assert.NoError(t, err)
		assert.Equal(t, "", mockEngine.lastTaskDescription, "should pass empty task description")
		assert.False(t, mockEngine.lastGeneratePlan, "should not generate plan")
	})
}
