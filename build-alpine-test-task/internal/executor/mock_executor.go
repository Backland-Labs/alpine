package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MockExecutor provides a mock implementation for testing Alpine workflows
type MockExecutor struct {
	simulateInterruption bool
	executionDelay       time.Duration
	expectedCommand      string
	executedCommands     []string
	stateFileConfig      *StateFileConfig
	executionDuration    time.Duration
}

// StateFileConfig holds configuration for simulated state file creation
type StateFileConfig struct {
	FilePath string
	Status   string
	StepDesc string
}

// NewMockExecutor creates a new mock executor instance
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		executedCommands: make([]string, 0),
	}
}

// SimulateStateFile configures the mock to create a state file during execution
func (m *MockExecutor) SimulateStateFile(filePath, status, stepDesc string) {
	m.stateFileConfig = &StateFileConfig{
		FilePath: filePath,
		Status:   status,
		StepDesc: stepDesc,
	}
}

// SimulateInterruption configures the mock to simulate an interrupted execution
func (m *MockExecutor) SimulateInterruption(interrupt bool) {
	m.simulateInterruption = interrupt
}

// Execute simulates command execution with configured behavior
func (m *MockExecutor) Execute(task, workDir string) error {
	start := time.Now()
	defer func() {
		m.executionDuration = time.Since(start)
	}()

	// Simulate execution delay
	if m.executionDelay > 0 {
		time.Sleep(m.executionDelay)
	}

	// Track command execution
	if len(m.expectedCommand) > 0 {
		m.executedCommands = append(m.executedCommands, m.expectedCommand)
	}

	// Create state file if configured
	if m.stateFileConfig != nil {
		stateData := map[string]string{
			"current_step_description": m.stateFileConfig.StepDesc,
			"next_step_prompt":         "Next step prompt",
			"status":                   m.stateFileConfig.Status,
		}

		jsonData, err := json.MarshalIndent(stateData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal state data: %w", err)
		}

		// Ensure directory exists
		dir := filepath.Dir(m.stateFileConfig.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create state directory: %w", err)
		}

		if err := os.WriteFile(m.stateFileConfig.FilePath, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write state file: %w", err)
		}
	}

	// Simulate interruption if configured
	if m.simulateInterruption {
		return fmt.Errorf("execution interrupted")
	}

	return nil
}

// GetExecutionDuration returns the duration of the last execution
func (m *MockExecutor) GetExecutionDuration() time.Duration {
	return m.executionDuration
}

// SetExecutionDelay sets a delay to simulate execution time
func (m *MockExecutor) SetExecutionDelay(delay time.Duration) {
	m.executionDelay = delay
}

// WasCommandExecuted checks if a specific command was executed
func (m *MockExecutor) WasCommandExecuted(cmd string) bool {
	for _, executed := range m.executedCommands {
		if strings.Contains(executed, cmd) {
			return true
		}
	}
	return false
}
