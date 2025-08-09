package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Backland-Labs/alpine/internal/core"
)

func TestTypes_BasicInstantiation(t *testing.T) {
	// Test ComparisonResult
	compResult := ComparisonResult{
		Match:       true,
		Differences: []Difference{},
	}
	assert.True(t, compResult.Match, "ComparisonResult Match field should be assignable")
	assert.Empty(t, compResult.Differences, "ComparisonResult Differences should be assignable")

	// Test Difference
	diff := Difference{
		Type:        "test_type",
		PythonValue: "python_val",
		GoValue:     "go_val",
		Description: "test description",
	}
	assert.Equal(t, "test_type", diff.Type, "Difference Type field should be assignable")
	assert.Equal(t, "python_val", diff.PythonValue, "Difference PythonValue field should be assignable")

	// Test CommandComponents
	cmdComponents := CommandComponents{
		Executable:    "/usr/bin/claude",
		MCPServers:    map[string]string{"server1": "/path/to/server1"},
		ToolAllowlist: []string{"tool1", "tool2"},
		OutputFormat:  "json",
		SystemPrompt:  "system prompt",
		UserPrompt:    "user prompt",
	}
	assert.Equal(t, "/usr/bin/claude", cmdComponents.Executable, "CommandComponents Executable field should be assignable")
	assert.Len(t, cmdComponents.MCPServers, 1, "CommandComponents MCPServers should be assignable")
	assert.Len(t, cmdComponents.ToolAllowlist, 2, "CommandComponents ToolAllowlist should be assignable")

	// Test OutputMetrics
	metrics := OutputMetrics{
		ErrorCount:        2,
		WarningCount:      1,
		HasErrors:         true,
		CompletionMessage: "completed successfully",
	}
	assert.Equal(t, 2, metrics.ErrorCount, "OutputMetrics ErrorCount field should be assignable")
	assert.True(t, metrics.HasErrors, "OutputMetrics HasErrors field should be assignable")

	// Test ParityConfig
	config := ParityConfig{
		PythonPath:    "/usr/bin/python",
		GoPath:        "/usr/bin/go",
		WorkDir:       "/tmp/test",
		CleanupOnExit: true,
	}
	assert.Equal(t, "/usr/bin/python", config.PythonPath, "ParityConfig PythonPath field should be assignable")
	assert.True(t, config.CleanupOnExit, "ParityConfig CleanupOnExit field should be assignable")

	// Test ParityResults
	results := ParityResults{
		IssueID:            "123",
		Success:            true,
		CommandMatch:       true,
		StateMatch:         true,
		OutputMatch:        true,
		CommandDifferences: []Difference{},
		StateDifferences:   []Difference{},
		OutputDifferences:  []Difference{},
		PythonMetrics:      OutputMetrics{},
		GoMetrics:          OutputMetrics{},
		PythonExecution:    nil,
		GoExecution:        nil,
	}
	assert.Equal(t, "123", results.IssueID, "ParityResults IssueID field should be assignable")
	assert.True(t, results.Success, "ParityResults Success field should be assignable")
	assert.Empty(t, results.CommandDifferences, "ParityResults CommandDifferences should be assignable")

	// Test ExecutionResult
	execResult := ExecutionResult{
		Command:  []string{"alpine", "test"},
		Output:   "test output",
		State:    &core.State{},
		ExitCode: 0,
		Error:    nil,
	}
	assert.Len(t, execResult.Command, 2, "ExecutionResult Command field should be assignable")
	assert.Equal(t, "test output", execResult.Output, "ExecutionResult Output field should be assignable")
	assert.NotNil(t, execResult.State, "ExecutionResult State should be assignable")
	assert.NoError(t, execResult.Error, "ExecutionResult Error field should be assignable")
}
