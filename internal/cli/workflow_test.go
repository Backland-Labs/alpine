package cli

import (
	"context"
	"testing"

	"github.com/maxmcd/river/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractTaskDescription_BareMode(t *testing.T) {
	// Test that empty task description is accepted when both flags are set
	t.Run("bare mode allows empty task description", func(t *testing.T) {
		deps := &Dependencies{
			FileReader:     &mockFileReader{},
			ConfigLoader:   &mockConfigLoader{},
			WorkflowEngine: &mockWorkflowEngine{},
		}

		// Call with empty args, but both flags set to true
		err := runWorkflowWithDependencies(context.Background(), []string{}, true, true, "", deps)

		// In bare mode (both flags set), empty task should be allowed
		// This test should PASS after implementation
		assert.NoError(t, err, "bare mode should allow empty task description")
	})

	t.Run("empty task description rejected with only no-plan flag", func(t *testing.T) {
		deps := &Dependencies{
			FileReader:   &mockFileReader{},
			ConfigLoader: &mockConfigLoader{},
		}

		// Call with empty args and only no-plan flag
		err := runWorkflowWithDependencies(context.Background(), []string{}, true, false, "", deps)

		// Should fail when only one flag is set
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task description")
	})

	t.Run("empty task description rejected with only no-worktree flag", func(t *testing.T) {
		deps := &Dependencies{
			FileReader:   &mockFileReader{},
			ConfigLoader: &mockConfigLoader{},
		}

		// Call with empty args and only no-worktree flag
		err := runWorkflowWithDependencies(context.Background(), []string{}, false, true, "", deps)

		// Should fail when only one flag is set
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task description")
	})

	t.Run("whitespace-only task description treated as empty in bare mode", func(t *testing.T) {
		deps := &Dependencies{
			FileReader:     &mockFileReader{},
			ConfigLoader:   &mockConfigLoader{},
			WorkflowEngine: &mockWorkflowEngine{},
		}

		// Call with whitespace-only task and both flags
		err := runWorkflowWithDependencies(context.Background(), []string{"   \n\t  "}, true, true, "", deps)

		// In bare mode, whitespace-only should be treated as empty and allowed
		assert.NoError(t, err, "bare mode should allow whitespace-only task description")
	})

	t.Run("bare mode passes empty string to workflow engine", func(t *testing.T) {
		mockEngine := &mockWorkflowEngine{}
		deps := &Dependencies{
			FileReader:     &mockFileReader{},
			ConfigLoader:   &mockConfigLoader{},
			WorkflowEngine: mockEngine,
		}

		// Call with no args in bare mode
		err := runWorkflowWithDependencies(context.Background(), []string{}, true, true, "", deps)

		require.NoError(t, err)
		// Verify that empty string was passed to the engine
		assert.Equal(t, "", mockEngine.lastTaskDescription)
		assert.False(t, mockEngine.lastGeneratePlan) // no-plan flag should be respected
	})
}

// Mock implementations for testing

type mockFileReader struct {
	content string
	err     error
}

func (m *mockFileReader) ReadFile(filename string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []byte(m.content), nil
}

type mockConfigLoader struct {
	cfg *config.Config
	err error
}

func (m *mockConfigLoader) Load() (*config.Config, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.cfg != nil {
		return m.cfg, nil
	}
	// Return a default config
	return &config.Config{
		Git: config.GitConfig{
			WorktreeEnabled: true,
			BaseBranch:      "main",
		},
		Verbosity: config.VerbosityNormal,
	}, nil
}

type mockWorkflowEngine struct {
	lastTaskDescription string
	lastGeneratePlan    bool
	err                 error
}

func (m *mockWorkflowEngine) Run(ctx context.Context, taskDescription string, generatePlan bool) error {
	m.lastTaskDescription = taskDescription
	m.lastGeneratePlan = generatePlan
	if m.err != nil {
		return m.err
	}
	// Don't validate empty task description in mock - that's handled by real engine in Task 3
	return nil
}
