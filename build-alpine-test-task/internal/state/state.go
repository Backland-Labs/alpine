package state

import (
	"encoding/json"
	"fmt"
	"os"
)

type AgentState struct {
	CurrentStepDescription string `json:"current_step_description"`
	NextStepPrompt         string `json:"next_step_prompt"`
	Status                 string `json:"status"`
}

func LoadState(filename string) (*AgentState, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state AgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

func (s *AgentState) Save(filename string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}
