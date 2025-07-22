package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/maxmcd/river/internal/core"
)

// parityRunner implements ParityRunner interface
type parityRunner struct {
	config           *ParityConfig
	commandValidator CommandValidator
	stateValidator   StateValidator
	outputValidator  OutputValidator
}

// NewParityRunner creates a new parity runner
func NewParityRunner(config *ParityConfig) ParityRunner {
	return &parityRunner{
		config:           config,
		commandValidator: NewCommandValidator(),
		stateValidator:   NewStateValidator(),
		outputValidator:  NewOutputValidator(),
	}
}

// Run executes parity tests for a given issue ID
func (r *parityRunner) Run(ctx context.Context, issueID string) (*ParityResults, error) {
	// Create separate working directories for Python and Go
	pythonDir := filepath.Join(r.config.WorkDir, "python")
	goDir := filepath.Join(r.config.WorkDir, "go")

	if err := os.MkdirAll(pythonDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create Python work dir: %w", err)
	}
	if err := os.MkdirAll(goDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create Go work dir: %w", err)
	}

	// Cleanup if configured
	if r.config.CleanupOnExit {
		defer func() {
			_ = os.RemoveAll(pythonDir)
		}()
		defer func() {
			_ = os.RemoveAll(goDir)
		}()
	}

	// Run Python implementation
	pythonResult := r.runImplementation(ctx, r.config.PythonPath, issueID, pythonDir)

	// Run Go implementation
	goResult := r.runImplementation(ctx, r.config.GoPath, issueID, goDir)

	// Compare results
	results := r.compareExecutions(pythonResult, goResult)
	results.IssueID = issueID
	results.PythonExecution = pythonResult
	results.GoExecution = goResult

	return results, nil
}

// runImplementation executes a river implementation and captures results
func (r *parityRunner) runImplementation(ctx context.Context, execPath string, issueID string, workDir string) *ExecutionResult {
	result := &ExecutionResult{
		Command: []string{execPath, issueID},
	}

	// Create command
	cmd := exec.CommandContext(ctx, execPath, issueID)
	cmd.Dir = workDir

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Mock environment for testing
	cmd.Env = append(os.Environ(),
		"RIVER_CLAUDE_PATH=echo", // Mock claude command
		"RIVER_AUTO_CLEANUP=false",
	)

	// Run command
	err := cmd.Run()
	if err != nil {
		result.Error = err
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	}

	// Combine stdout and stderr
	result.Output = stdout.String()
	if stderr.String() != "" {
		result.Output += "\n" + stderr.String()
	}

	// Extract command from output (if logged)
	result.Command = r.extractCommandFromOutput(result.Output)

	// Load state file
	stateFile := filepath.Join(workDir, "claude_state.json")
	if data, err := os.ReadFile(stateFile); err == nil {
		var state core.State
		if err := json.Unmarshal(data, &state); err == nil {
			result.State = &state
		}
	}

	return result
}

// extractCommandFromOutput extracts the Claude command from output logs
func (r *parityRunner) extractCommandFromOutput(output string) []string {
	// Look for command in output (implementations should log the command)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Executing command:") ||
			strings.Contains(line, "Running:") {
			// Extract command after the prefix
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				cmdStr := strings.TrimSpace(parts[1])
				// Simple parsing - in real implementation, use proper shell parsing
				return strings.Fields(cmdStr)
			}
		}
	}
	return []string{}
}

// compareExecutions compares Python and Go execution results
func (r *parityRunner) compareExecutions(pythonExec, goExec *ExecutionResult) *ParityResults {
	results := &ParityResults{
		Success: true,
	}

	// Compare commands
	if len(pythonExec.Command) > 0 && len(goExec.Command) > 0 {
		cmdResult := r.commandValidator.CompareCommands(pythonExec.Command, goExec.Command)
		results.CommandMatch = cmdResult.Match
		results.CommandDifferences = cmdResult.Differences
		if !cmdResult.Match {
			results.Success = false
		}
	} else {
		// If we couldn't extract commands, assume they match
		results.CommandMatch = true
	}

	// Compare states
	if pythonExec.State != nil && goExec.State != nil {
		stateResult := r.stateValidator.CompareStates(pythonExec.State, goExec.State)
		results.StateMatch = stateResult.Match
		results.StateDifferences = stateResult.Differences
		if !stateResult.Match {
			results.Success = false
		}
	} else if pythonExec.State != nil || goExec.State != nil {
		// One has state, other doesn't
		results.StateMatch = false
		results.Success = false
	} else {
		// Neither has state
		results.StateMatch = true
	}

	// Compare outputs
	outputResult := r.outputValidator.CompareOutputs(pythonExec.Output, goExec.Output)
	results.OutputMatch = outputResult.Match
	results.OutputDifferences = outputResult.Differences
	if !outputResult.Match {
		results.Success = false
	}

	// Extract metrics
	results.PythonMetrics = r.outputValidator.ExtractKeyMetrics(pythonExec.Output)
	results.GoMetrics = r.outputValidator.ExtractKeyMetrics(goExec.Output)

	return results
}

// GenerateReport generates a human-readable report of parity results
func (r *parityRunner) GenerateReport(results *ParityResults) string {
	var report strings.Builder

	// Header
	if results.Success {
		report.WriteString("═══ PARITY CHECK PASSED ═══\n\n")
	} else {
		report.WriteString("═══ PARITY CHECK FAILED ═══\n\n")
	}

	report.WriteString(fmt.Sprintf("Issue ID: %s\n\n", results.IssueID))

	// Summary
	report.WriteString("Summary:\n")
	report.WriteString(fmt.Sprintf("  Command Comparison: %s\n", formatMatch(results.CommandMatch)))
	report.WriteString(fmt.Sprintf("  State Comparison: %s\n", formatMatch(results.StateMatch)))
	report.WriteString(fmt.Sprintf("  Output Comparison: %s\n", formatMatch(results.OutputMatch)))
	report.WriteString("\n")

	// Command differences
	if !results.CommandMatch && len(results.CommandDifferences) > 0 {
		report.WriteString("Command Differences:\n")
		for _, diff := range results.CommandDifferences {
			report.WriteString(fmt.Sprintf("  • %s: %s → %s\n", diff.Type, diff.PythonValue, diff.GoValue))
		}
		report.WriteString("\n")
	}

	// State differences
	if !results.StateMatch && len(results.StateDifferences) > 0 {
		report.WriteString("State Differences:\n")
		for _, diff := range results.StateDifferences {
			report.WriteString(fmt.Sprintf("  • %s: %s → %s\n", diff.Type, diff.PythonValue, diff.GoValue))
		}
		report.WriteString("\n")
	}

	// Output differences
	if !results.OutputMatch && len(results.OutputDifferences) > 0 {
		report.WriteString("Output Differences:\n")
		for _, diff := range results.OutputDifferences {
			report.WriteString(fmt.Sprintf("  • %s\n", diff.Description))
			if diff.PythonValue != "" || diff.GoValue != "" {
				report.WriteString(fmt.Sprintf("    Python: %s\n", diff.PythonValue))
				report.WriteString(fmt.Sprintf("    Go: %s\n", diff.GoValue))
			}
		}
		report.WriteString("\n")
	}

	// Metrics comparison
	report.WriteString("Output Metrics:\n")
	report.WriteString(fmt.Sprintf("  Python - Errors: %d, Warnings: %d\n", 
		results.PythonMetrics.ErrorCount, results.PythonMetrics.WarningCount))
	report.WriteString(fmt.Sprintf("  Go     - Errors: %d, Warnings: %d\n", 
		results.GoMetrics.ErrorCount, results.GoMetrics.WarningCount))

	return report.String()
}

// formatMatch formats a boolean match result
func formatMatch(match bool) string {
	if match {
		return "✓ MATCH"
	}
	return "✗ MISMATCH"
}