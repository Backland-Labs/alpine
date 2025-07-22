package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Status constants
const (
	StatusRunning   = "running"
	StatusCompleted = "completed"
)

// Common prompt constants
const (
	PromptMakePlan = "/make_plan"
	PromptRalph    = "/ralph"
	PromptContinue = "/continue"
)

// fileMutex provides global synchronization for state file operations
var fileMutex sync.Mutex

// State represents the current workflow state
type State struct {
	CurrentStepDescription string `json:"current_step_description"`
	NextStepPrompt         string `json:"next_step_prompt"`
	Status                 string `json:"status"`
}

// LoadState loads the state from a JSON file
// If the file doesn't exist, it returns an empty State (not an error)
func LoadState(path string) (*State, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Missing file is not an error - create new empty state
			return &State{}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

// Save writes the state to a JSON file with pretty-printing (2-space indentation)
func (s *State) Save(path string) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Add newline at end for better formatting
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// InitializeState creates a new state for starting a workflow
func InitializeState(issueTitle, issueDescription string, withPlan bool) *State {
	// Create descriptive step description
	description := fmt.Sprintf("Starting work on: %s", issueTitle)
	if issueDescription != "" {
		description = fmt.Sprintf("Starting work on: %s - %s", issueTitle, issueDescription)
	}

	// Set next prompt based on whether planning is requested
	nextPrompt := PromptRalph
	if withPlan {
		nextPrompt = PromptMakePlan
	}

	return &State{
		CurrentStepDescription: description,
		NextStepPrompt:         nextPrompt,
		Status:                 StatusRunning,
	}
}

// Validate checks if the state has valid values
func (s *State) Validate() error {
	if s.CurrentStepDescription == "" {
		return fmt.Errorf("current_step_description cannot be empty")
	}

	if s.Status == "" {
		return fmt.Errorf("status cannot be empty")
	}

	if s.Status != StatusRunning && s.Status != StatusCompleted {
		return fmt.Errorf("status must be 'running' or 'completed'")
	}

	// If status is running, next_step_prompt should not be empty
	if s.Status == StatusRunning && s.NextStepPrompt == "" {
		return fmt.Errorf("next_step_prompt cannot be empty when status is 'running'")
	}

	return nil
}

// IsCompleted returns true if the workflow status is "completed"
func (s *State) IsCompleted() bool {
	return strings.ToLower(s.Status) == StatusCompleted
}
