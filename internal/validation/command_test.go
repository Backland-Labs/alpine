package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareCommands(t *testing.T) {
	tests := []struct {
		name           string
		pythonCmd      []string
		goCmd          []string
		expectedResult ComparisonResult
	}{
		{
			name: "identical commands match",
			pythonCmd: []string{
				"claude",
				"--mcp-server", "context7",
				"--tool-allowlist", "mcp__context7__*",
				"--tool-allowlist", "Read",
				"--tool-allowlist", "Write",
				"--output", "text",
				"--system", "You are an AI assistant helping with a task.",
				"Fix the bug in the login system",
			},
			goCmd: []string{
				"claude",
				"--mcp-server", "context7",
				"--tool-allowlist", "mcp__context7__*",
				"--tool-allowlist", "Read",
				"--tool-allowlist", "Write",
				"--output", "text",
				"--system", "You are an AI assistant helping with a task.",
				"Fix the bug in the login system",
			},
			expectedResult: ComparisonResult{
				Match: true,
			},
		},
		{
			name: "different tool allowlists",
			pythonCmd: []string{
				"claude",
				"--tool-allowlist", "Read",
				"--tool-allowlist", "Write",
			},
			goCmd: []string{
				"claude",
				"--tool-allowlist", "Read",
				"--tool-allowlist", "Edit",
			},
			expectedResult: ComparisonResult{
				Match: false,
				Differences: []Difference{
					{
						Type:        "tool_allowlist",
						PythonValue: "Write",
						GoValue:     "Edit",
						Description: "Tool allowlist mismatch",
					},
				},
			},
		},
		{
			name: "different system prompts",
			pythonCmd: []string{
				"claude",
				"--system", "You are an AI assistant.",
			},
			goCmd: []string{
				"claude",
				"--system", "You are a helpful AI assistant.",
			},
			expectedResult: ComparisonResult{
				Match: false,
				Differences: []Difference{
					{
						Type:        "system_prompt",
						PythonValue: "You are an AI assistant.",
						GoValue:     "You are a helpful AI assistant.",
						Description: "System prompt mismatch",
					},
				},
			},
		},
		{
			name: "different user prompts",
			pythonCmd: []string{
				"claude",
				"Fix the bug",
			},
			goCmd: []string{
				"claude",
				"Resolve the issue",
			},
			expectedResult: ComparisonResult{
				Match: false,
				Differences: []Difference{
					{
						Type:        "user_prompt",
						PythonValue: "Fix the bug",
						GoValue:     "Resolve the issue",
						Description: "User prompt mismatch",
					},
				},
			},
		},
		{
			name: "order independent for tool allowlists",
			pythonCmd: []string{
				"claude",
				"--tool-allowlist", "tool_a",
				"--tool-allowlist", "tool_b",
				"--tool-allowlist", "tool_c",
			},
			goCmd: []string{
				"claude",
				"--tool-allowlist", "tool_c",
				"--tool-allowlist", "tool_a",
				"--tool-allowlist", "tool_b",
			},
			expectedResult: ComparisonResult{
				Match: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCommandValidator()
			result := validator.CompareCommands(tt.pythonCmd, tt.goCmd)

			assert.Equal(t, tt.expectedResult.Match, result.Match)
			if !tt.expectedResult.Match {
				require.Equal(t, len(tt.expectedResult.Differences), len(result.Differences))
				for i, expected := range tt.expectedResult.Differences {
					assert.Equal(t, expected.Type, result.Differences[i].Type)
					assert.Equal(t, expected.PythonValue, result.Differences[i].PythonValue)
					assert.Equal(t, expected.GoValue, result.Differences[i].GoValue)
				}
			}
		})
	}
}

func TestExtractCommandComponents(t *testing.T) {
	tests := []struct {
		name     string
		cmd      []string
		expected CommandComponents
	}{
		{
			name: "extract all components",
			cmd: []string{
				"claude",
				"--mcp-server", "context7",
				"--tool-allowlist", "tool1",
				"--tool-allowlist", "tool2",
				"--output", "text",
				"--system", "System prompt here",
				"User prompt here",
			},
			expected: CommandComponents{
				Executable:    "claude",
				MCPServers:    map[string]string{"": "context7"},
				ToolAllowlist: []string{"tool1", "tool2"},
				OutputFormat:  "text",
				SystemPrompt:  "System prompt here",
				UserPrompt:    "User prompt here",
			},
		},
		{
			name: "multiple mcp servers",
			cmd: []string{
				"claude",
				"--mcp-server", "context7",
				"--mcp-server", "filesystem",
				"Do something",
			},
			expected: CommandComponents{
				Executable: "claude",
				MCPServers: map[string]string{
					"": "context7,filesystem",
				},
				ToolAllowlist: []string{},
				UserPrompt:    "Do something",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCommandValidator()
			components := validator.ExtractComponents(tt.cmd)

			assert.Equal(t, tt.expected.Executable, components.Executable)
			assert.Equal(t, tt.expected.MCPServers, components.MCPServers)
			assert.ElementsMatch(t, tt.expected.ToolAllowlist, components.ToolAllowlist)
			assert.Equal(t, tt.expected.OutputFormat, components.OutputFormat)
			assert.Equal(t, tt.expected.SystemPrompt, components.SystemPrompt)
			assert.Equal(t, tt.expected.UserPrompt, components.UserPrompt)
		})
	}
}