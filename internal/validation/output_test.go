package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareOutputs(t *testing.T) {
	tests := []struct {
		name           string
		pythonOutput   string
		goOutput       string
		expectedResult ComparisonResult
	}{
		{
			name:         "identical outputs match",
			pythonOutput: "Task completed successfully\nAll tests passing",
			goOutput:     "Task completed successfully\nAll tests passing",
			expectedResult: ComparisonResult{
				Match: true,
			},
		},
		{
			name:         "whitespace normalized outputs match",
			pythonOutput: "Task completed successfully\n\nAll tests passing",
			goOutput:     "Task completed successfully\n\n\nAll tests passing",
			expectedResult: ComparisonResult{
				Match: true,
			},
		},
		{
			name:         "different content",
			pythonOutput: "Task completed successfully",
			goOutput:     "Task failed with errors",
			expectedResult: ComparisonResult{
				Match: false,
				Differences: []Difference{
					{
						Type:        "output_content",
						PythonValue: "Task completed successfully",
						GoValue:     "Task failed with errors",
						Description: "Output content mismatch",
					},
				},
			},
		},
		{
			name:         "line ending differences ignored",
			pythonOutput: "Line1\r\nLine2\r\nLine3",
			goOutput:     "Line1\nLine2\nLine3",
			expectedResult: ComparisonResult{
				Match: true,
			},
		},
		{
			name:         "trailing whitespace ignored",
			pythonOutput: "Output line   \nAnother line  ",
			goOutput:     "Output line\nAnother line",
			expectedResult: ComparisonResult{
				Match: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewOutputValidator()
			result := validator.CompareOutputs(tt.pythonOutput, tt.goOutput)

			assert.Equal(t, tt.expectedResult.Match, result.Match)
			if !tt.expectedResult.Match {
				require.Equal(t, len(tt.expectedResult.Differences), len(result.Differences))
				for i, expected := range tt.expectedResult.Differences {
					assert.Equal(t, expected.Type, result.Differences[i].Type)
					assert.Contains(t, result.Differences[i].Description, expected.Description)
				}
			}
		})
	}
}

func TestExtractKeyMetrics(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected OutputMetrics
	}{
		{
			name: "extract error count",
			output: `Running tests...
ERROR: Test failed in test_module.py
ERROR: Another test failed
WARNING: Deprecated function used
All done.`,
			expected: OutputMetrics{
				ErrorCount:   2,
				WarningCount: 1,
				HasErrors:    true,
			},
		},
		{
			name: "extract completion status",
			output: `Starting workflow...
Processing task...
Task completed successfully
Status: completed`,
			expected: OutputMetrics{
				ErrorCount:        0,
				WarningCount:      0,
				HasErrors:         false,
				CompletionMessage: "Task completed successfully",
			},
		},
		{
			name: "no errors or warnings",
			output: `Running workflow...
Everything is fine.
Done.`,
			expected: OutputMetrics{
				ErrorCount:   0,
				WarningCount: 0,
				HasErrors:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewOutputValidator()
			metrics := validator.ExtractKeyMetrics(tt.output)

			assert.Equal(t, tt.expected.ErrorCount, metrics.ErrorCount)
			assert.Equal(t, tt.expected.WarningCount, metrics.WarningCount)
			assert.Equal(t, tt.expected.HasErrors, metrics.HasErrors)
			if tt.expected.CompletionMessage != "" {
				assert.Equal(t, tt.expected.CompletionMessage, metrics.CompletionMessage)
			}
		})
	}
}

func TestNormalizeOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normalize line endings",
			input:    "Line1\r\nLine2\r\nLine3",
			expected: "Line1\nLine2\nLine3",
		},
		{
			name:     "trim trailing whitespace per line",
			input:    "Line1   \nLine2  \nLine3 ",
			expected: "Line1\nLine2\nLine3",
		},
		{
			name:     "collapse multiple blank lines",
			input:    "Line1\n\n\n\nLine2\n\n\nLine3",
			expected: "Line1\n\nLine2\n\nLine3",
		},
		{
			name:     "trim leading and trailing whitespace",
			input:    "\n\n  Content here  \n\n",
			expected: "Content here",
		},
		{
			name:     "handle empty output",
			input:    "",
			expected: "",
		},
		{
			name:     "handle whitespace only",
			input:    "   \n\n   \n   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewOutputValidator()
			normalized := validator.NormalizeOutput(tt.input)
			assert.Equal(t, tt.expected, normalized)
		})
	}
}