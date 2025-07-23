package validation

import (
	"testing"

	"github.com/maxmcd/river/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareStates(t *testing.T) {
	tests := []struct {
		name           string
		pythonState    *core.State
		goState        *core.State
		expectedResult ComparisonResult
	}{
		{
			name: "identical states match",
			pythonState: &core.State{
				CurrentStepDescription: "Implementing feature X",
				NextStepPrompt:         "/run_implementation_loop",
				Status:                 "running",
			},
			goState: &core.State{
				CurrentStepDescription: "Implementing feature X",
				NextStepPrompt:         "/run_implementation_loop",
				Status:                 "running",
			},
			expectedResult: ComparisonResult{
				Match: true,
			},
		},
		{
			name: "different current step descriptions",
			pythonState: &core.State{
				CurrentStepDescription: "Implementing feature X",
				NextStepPrompt:         "/run_implementation_loop",
				Status:                 "running",
			},
			goState: &core.State{
				CurrentStepDescription: "Working on feature X",
				NextStepPrompt:         "/run_implementation_loop",
				Status:                 "running",
			},
			expectedResult: ComparisonResult{
				Match: false,
				Differences: []Difference{
					{
						Type:        "current_step_description",
						PythonValue: "Implementing feature X",
						GoValue:     "Working on feature X",
						Description: "Current step description mismatch",
					},
				},
			},
		},
		{
			name: "different next step prompts",
			pythonState: &core.State{
				CurrentStepDescription: "Completed implementation",
				NextStepPrompt:         "/verify_plan",
				Status:                 "running",
			},
			goState: &core.State{
				CurrentStepDescription: "Completed implementation",
				NextStepPrompt:         "/run_implementation_loop",
				Status:                 "running",
			},
			expectedResult: ComparisonResult{
				Match: false,
				Differences: []Difference{
					{
						Type:        "next_step_prompt",
						PythonValue: "/verify_plan",
						GoValue:     "/run_implementation_loop",
						Description: "Next step prompt mismatch",
					},
				},
			},
		},
		{
			name: "different status values",
			pythonState: &core.State{
				CurrentStepDescription: "All done",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
			goState: &core.State{
				CurrentStepDescription: "All done",
				NextStepPrompt:         "",
				Status:                 "running",
			},
			expectedResult: ComparisonResult{
				Match: false,
				Differences: []Difference{
					{
						Type:        "status",
						PythonValue: "completed",
						GoValue:     "running",
						Description: "Status mismatch",
					},
				},
			},
		},
		{
			name: "multiple differences",
			pythonState: &core.State{
				CurrentStepDescription: "Step 1",
				NextStepPrompt:         "/run_implementation_loop",
				Status:                 "running",
			},
			goState: &core.State{
				CurrentStepDescription: "Step 2",
				NextStepPrompt:         "/verify_plan",
				Status:                 "completed",
			},
			expectedResult: ComparisonResult{
				Match: false,
				Differences: []Difference{
					{
						Type:        "current_step_description",
						PythonValue: "Step 1",
						GoValue:     "Step 2",
						Description: "Current step description mismatch",
					},
					{
						Type:        "next_step_prompt",
						PythonValue: "/run_implementation_loop",
						GoValue:     "/verify_plan",
						Description: "Next step prompt mismatch",
					},
					{
						Type:        "status",
						PythonValue: "running",
						GoValue:     "completed",
						Description: "Status mismatch",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewStateValidator()
			result := validator.CompareStates(tt.pythonState, tt.goState)

			assert.Equal(t, tt.expectedResult.Match, result.Match)
			if !tt.expectedResult.Match {
				require.Equal(t, len(tt.expectedResult.Differences), len(result.Differences))
				for i, expected := range tt.expectedResult.Differences {
					assert.Equal(t, expected.Type, result.Differences[i].Type)
					assert.Equal(t, expected.PythonValue, result.Differences[i].PythonValue)
					assert.Equal(t, expected.GoValue, result.Differences[i].GoValue)
					assert.Equal(t, expected.Description, result.Differences[i].Description)
				}
			}
		})
	}
}

func TestNormalizeState(t *testing.T) {
	tests := []struct {
		name     string
		input    *core.State
		expected *core.State
	}{
		{
			name: "trim whitespace from all fields",
			input: &core.State{
				CurrentStepDescription: "  Implementing feature  ",
				NextStepPrompt:         " /run_implementation_loop ",
				Status:                 " running ",
			},
			expected: &core.State{
				CurrentStepDescription: "Implementing feature",
				NextStepPrompt:         "/run_implementation_loop",
				Status:                 "running",
			},
		},
		{
			name: "normalize line endings",
			input: &core.State{
				CurrentStepDescription: "Line 1\r\nLine 2\r\nLine 3",
				NextStepPrompt:         "/run_implementation_loop",
				Status:                 "running",
			},
			expected: &core.State{
				CurrentStepDescription: "Line 1\nLine 2\nLine 3",
				NextStepPrompt:         "/run_implementation_loop",
				Status:                 "running",
			},
		},
		{
			name: "handle empty fields",
			input: &core.State{
				CurrentStepDescription: "",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
			expected: &core.State{
				CurrentStepDescription: "",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewStateValidator()
			normalized := validator.NormalizeState(tt.input)

			assert.Equal(t, tt.expected.CurrentStepDescription, normalized.CurrentStepDescription)
			assert.Equal(t, tt.expected.NextStepPrompt, normalized.NextStepPrompt)
			assert.Equal(t, tt.expected.Status, normalized.Status)
		})
	}
}
