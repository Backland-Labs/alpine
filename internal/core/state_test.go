package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestState_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		state     State
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid state with running status",
			state: State{
				CurrentStepDescription: "Implementing feature X",
				NextStepPrompt:         "/run_implementation_loop",
				Status:                 "running",
			},
			wantError: false,
		},
		{
			name: "valid state with completed status",
			state: State{
				CurrentStepDescription: "Feature X implemented",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
			wantError: false,
		},
		{
			name: "empty current step description",
			state: State{
				CurrentStepDescription: "",
				NextStepPrompt:         "/run_implementation_loop",
				Status:                 "running",
			},
			wantError: true,
			errorMsg:  "current_step_description cannot be empty",
		},
		{
			name: "empty status",
			state: State{
				CurrentStepDescription: "Doing something",
				NextStepPrompt:         "/run_implementation_loop",
				Status:                 "",
			},
			wantError: true,
			errorMsg:  "status cannot be empty",
		},
		{
			name: "invalid status value",
			state: State{
				CurrentStepDescription: "Doing something",
				NextStepPrompt:         "/run_implementation_loop",
				Status:                 "invalid",
			},
			wantError: true,
			errorMsg:  "status must be 'running' or 'completed'",
		},
		{
			name: "running status with empty next step",
			state: State{
				CurrentStepDescription: "Doing something",
				NextStepPrompt:         "",
				Status:                 "running",
			},
			wantError: true,
			errorMsg:  "next_step_prompt cannot be empty when status is 'running'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.Validate()
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestState_IsCompleted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		state    State
		expected bool
	}{
		{
			name: "completed status",
			state: State{
				Status: "completed",
			},
			expected: true,
		},
		{
			name: "running status",
			state: State{
				Status: "running",
			},
			expected: false,
		},
		{
			name: "empty status",
			state: State{
				Status: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.IsCompleted()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		wantError   bool
		errorMsg    string
		checkResult func(t *testing.T, state *State)
	}{
		{
			name: "load valid state file",
			setupFunc: func(t *testing.T) string {
				tmpFile := filepath.Join(t.TempDir(), "state.json")
				content := `{
					"current_step_description": "Testing load",
					"next_step_prompt": "/continue",
					"status": "running"
				}`
				err := os.WriteFile(tmpFile, []byte(content), 0644)
				require.NoError(t, err)
				return tmpFile
			},
			wantError: false,
			checkResult: func(t *testing.T, state *State) {
				assert.Equal(t, "Testing load", state.CurrentStepDescription)
				assert.Equal(t, "/continue", state.NextStepPrompt)
				assert.Equal(t, "running", state.Status)
			},
		},
		{
			name: "missing file creates new empty state",
			setupFunc: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.json")
			},
			wantError: false,
			checkResult: func(t *testing.T, state *State) {
				assert.Empty(t, state.CurrentStepDescription)
				assert.Empty(t, state.NextStepPrompt)
				assert.Empty(t, state.Status)
			},
		},
		{
			name: "invalid JSON",
			setupFunc: func(t *testing.T) string {
				tmpFile := filepath.Join(t.TempDir(), "invalid.json")
				err := os.WriteFile(tmpFile, []byte("invalid json"), 0644)
				require.NoError(t, err)
				return tmpFile
			},
			wantError: true,
			errorMsg:  "failed to parse state file",
		},
		{
			name: "missing required field",
			setupFunc: func(t *testing.T) string {
				tmpFile := filepath.Join(t.TempDir(), "incomplete.json")
				content := `{
					"current_step_description": "Testing",
					"status": "running"
				}`
				err := os.WriteFile(tmpFile, []byte(content), 0644)
				require.NoError(t, err)
				return tmpFile
			},
			wantError: false,
			checkResult: func(t *testing.T, state *State) {
				assert.Equal(t, "Testing", state.CurrentStepDescription)
				assert.Empty(t, state.NextStepPrompt)
				assert.Equal(t, "running", state.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setupFunc(t)
			state, err := LoadState(path)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, state)
				if tt.checkResult != nil {
					tt.checkResult(t, state)
				}
			}
		})
	}
}

func TestState_Save(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		state     State
		wantError bool
		errorMsg  string
	}{
		{
			name: "save valid state",
			state: State{
				CurrentStepDescription: "Implementing feature Y",
				NextStepPrompt:         "/run_implementation_loop",
				Status:                 "running",
			},
			wantError: false,
		},
		{
			name: "save completed state",
			state: State{
				CurrentStepDescription: "All done",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "save_test.json")
			err := tt.state.Save(tmpFile)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)

				// Verify file was created with correct content
				content, err := os.ReadFile(tmpFile)
				require.NoError(t, err)

				// Check JSON is valid and properly formatted
				var loaded State
				err = json.Unmarshal(content, &loaded)
				require.NoError(t, err)
				assert.Equal(t, tt.state, loaded)

				// Check pretty-printing (2-space indentation)
				assert.Contains(t, string(content), "  \"current_step_description\"")
			}
		})
	}
}

func TestInitializeState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		issueTitle       string
		issueDescription string
		withPlan         bool
		checkResult      func(t *testing.T, state *State)
	}{
		{
			name:             "initialize with plan",
			issueTitle:       "Add authentication",
			issueDescription: "Implement JWT authentication for the API",
			withPlan:         true,
			checkResult: func(t *testing.T, state *State) {
				assert.Contains(t, state.CurrentStepDescription, "authentication")
				assert.Equal(t, "/make_plan", state.NextStepPrompt)
				assert.Equal(t, "running", state.Status)
			},
		},
		{
			name:             "initialize without plan",
			issueTitle:       "Fix bug in parser",
			issueDescription: "Parser crashes on empty input",
			withPlan:         false,
			checkResult: func(t *testing.T, state *State) {
				assert.Contains(t, state.CurrentStepDescription, "parser")
				assert.Equal(t, "/run_implementation_loop", state.NextStepPrompt)
				assert.Equal(t, "running", state.Status)
			},
		},
		{
			name:             "empty issue description",
			issueTitle:       "Update dependencies",
			issueDescription: "",
			withPlan:         true,
			checkResult: func(t *testing.T, state *State) {
				assert.Contains(t, state.CurrentStepDescription, "Update dependencies")
				assert.Equal(t, "/make_plan", state.NextStepPrompt)
				assert.Equal(t, "running", state.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := InitializeState(tt.issueTitle, tt.issueDescription, tt.withPlan)
			assert.NotNil(t, state)
			tt.checkResult(t, state)
		})
	}
}

func TestConcurrentStateAccess(t *testing.T) {
	// This test simulates concurrent access to the state file
	// It tests that our implementation handles file locking properly
	tmpFile := filepath.Join(t.TempDir(), "concurrent.json")

	// Create initial state
	initial := State{
		CurrentStepDescription: "Initial state",
		NextStepPrompt:         "/run_implementation_loop",
		Status:                 "running",
	}
	err := initial.Save(tmpFile)
	require.NoError(t, err)

	// Run multiple goroutines trying to read and write
	done := make(chan bool, 10)
	for i := 0; i < 5; i++ {
		// Writers
		go func(id int) {
			state := State{
				CurrentStepDescription: fmt.Sprintf("Writer %d", id),
				NextStepPrompt:         "/continue",
				Status:                 "running",
			}
			err := state.Save(tmpFile)
			assert.NoError(t, err)
			done <- true
		}(i)

		// Readers
		go func() {
			state, err := LoadState(tmpFile)
			assert.NoError(t, err)
			assert.NotNil(t, state)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Final state should be valid
	final, err := LoadState(tmpFile)
	assert.NoError(t, err)
	assert.NotNil(t, final)
	assert.NotEmpty(t, final.CurrentStepDescription)
}
