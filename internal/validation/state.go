package validation

import (
	"strings"

	"github.com/Backland-Labs/alpine/internal/core"
)

// stateValidator implements StateValidator interface
type stateValidator struct{}

// NewStateValidator creates a new state validator
func NewStateValidator() StateValidator {
	return &stateValidator{}
}

// CompareStates compares Python and Go state objects
func (v *stateValidator) CompareStates(pythonState, goState *core.State) ComparisonResult {
	// Normalize states before comparison
	normalizedPython := v.NormalizeState(pythonState)
	normalizedGo := v.NormalizeState(goState)

	result := ComparisonResult{
		Match:       true,
		Differences: []Difference{},
	}

	// Compare CurrentStepDescription
	if normalizedPython.CurrentStepDescription != normalizedGo.CurrentStepDescription {
		result.Match = false
		result.Differences = append(result.Differences, Difference{
			Type:        "current_step_description",
			PythonValue: normalizedPython.CurrentStepDescription,
			GoValue:     normalizedGo.CurrentStepDescription,
			Description: "Current step description mismatch",
		})
	}

	// Compare NextStepPrompt
	if normalizedPython.NextStepPrompt != normalizedGo.NextStepPrompt {
		result.Match = false
		result.Differences = append(result.Differences, Difference{
			Type:        "next_step_prompt",
			PythonValue: normalizedPython.NextStepPrompt,
			GoValue:     normalizedGo.NextStepPrompt,
			Description: "Next step prompt mismatch",
		})
	}

	// Compare Status
	if normalizedPython.Status != normalizedGo.Status {
		result.Match = false
		result.Differences = append(result.Differences, Difference{
			Type:        "status",
			PythonValue: normalizedPython.Status,
			GoValue:     normalizedGo.Status,
			Description: "Status mismatch",
		})
	}

	return result
}

// NormalizeState normalizes a state object for comparison
func (v *stateValidator) NormalizeState(state *core.State) *core.State {
	if state == nil {
		return nil
	}

	normalized := &core.State{
		CurrentStepDescription: normalizeString(state.CurrentStepDescription),
		NextStepPrompt:         normalizeString(state.NextStepPrompt),
		Status:                 normalizeString(state.Status),
	}

	return normalized
}

// normalizeString trims whitespace and normalizes line endings
func normalizeString(s string) string {
	// Trim whitespace
	s = strings.TrimSpace(s)

	// Normalize line endings (CRLF -> LF)
	s = strings.ReplaceAll(s, "\r\n", "\n")

	return s
}
