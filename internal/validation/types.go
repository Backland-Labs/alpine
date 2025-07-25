package validation

import (
	"context"

	"github.com/maxmcd/alpine/internal/core"
)

// ComparisonResult represents the result of comparing Python and Go implementations
type ComparisonResult struct {
	Match       bool
	Differences []Difference
}

// Difference represents a specific difference found during comparison
type Difference struct {
	Type        string // e.g., "tool_allowlist", "system_prompt", "user_prompt"
	PythonValue string
	GoValue     string
	Description string
}

// CommandComponents represents the parsed components of a Claude command
type CommandComponents struct {
	Executable    string
	MCPServers    map[string]string // server name -> server path
	ToolAllowlist []string
	OutputFormat  string
	SystemPrompt  string
	UserPrompt    string
}

// CommandValidator validates command-line arguments between Python and Go
type CommandValidator interface {
	CompareCommands(pythonCmd, goCmd []string) ComparisonResult
	ExtractComponents(cmd []string) CommandComponents
}

// StateValidator validates state files between Python and Go
type StateValidator interface {
	CompareStates(pythonState, goState *core.State) ComparisonResult
	NormalizeState(state *core.State) *core.State
}

// OutputValidator validates output between Python and Go
type OutputValidator interface {
	CompareOutputs(pythonOutput, goOutput string) ComparisonResult
	ExtractKeyMetrics(output string) OutputMetrics
	NormalizeOutput(output string) string
}

// OutputMetrics represents key metrics extracted from output
type OutputMetrics struct {
	ErrorCount        int
	WarningCount      int
	HasErrors         bool
	CompletionMessage string
}

// ParityRunner runs parity tests between Python and Go implementations
type ParityRunner interface {
	Run(ctx context.Context, issueID string) (*ParityResults, error)
	GenerateReport(results *ParityResults) string
}

// ParityConfig configures the parity runner
type ParityConfig struct {
	PythonPath    string // Path to Python alpine script
	GoPath        string // Path to Go alpine binary
	WorkDir       string // Working directory for test runs
	CleanupOnExit bool   // Whether to cleanup temp files
}

// ParityResults contains the results of a parity check
type ParityResults struct {
	IssueID            string
	Success            bool
	CommandMatch       bool
	StateMatch         bool
	OutputMatch        bool
	CommandDifferences []Difference
	StateDifferences   []Difference
	OutputDifferences  []Difference
	PythonMetrics      OutputMetrics
	GoMetrics          OutputMetrics
	PythonExecution    *ExecutionResult
	GoExecution        *ExecutionResult
}

// ExecutionResult represents the result of running alpine
type ExecutionResult struct {
	Command  []string
	Output   string
	State    *core.State
	ExitCode int
	Error    error
}
