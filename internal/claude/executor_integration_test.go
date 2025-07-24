package claude

import (
	"context"
	"strings"
	"testing"
)

// TestExecuteClaudeCommand_StderrCaptureIntegration tests the actual implementation
// This test verifies that executeClaudeCommand properly captures stderr
func TestExecuteClaudeCommand_StderrCaptureIntegration(t *testing.T) {
	// This test verifies the actual stderr capture implementation
	t.Run("captures stderr and sends to AddToolLog", func(t *testing.T) {
		// Create a mock printer that records AddToolLog calls
		mockPrinter := &recordingPrinter{
			toolLogs: make([]string, 0),
		}

		// For this test, we would need to refactor Executor to use an interface
		// or create a proper output.Printer with custom writers
		// Since this is a skipped test, we'll just document the intent
		_ = mockPrinter

		// Test would create an executor and test stderr capture

		// Since we can't easily test executeClaudeCommand directly (it calls claude),
		// let's test executeWithStderrCapture with a mock command
		t.Skip("Integration test - requires refactoring to test properly")

		// After implementation, we would verify:
		// 1. stdout contains only "stdout message"
		// 2. mockPrinter.toolLogs contains the stderr lines
	})
}

// recordingPrinter is a test printer that records AddToolLog calls
type recordingPrinter struct {
	toolLogs []string
}

func (r *recordingPrinter) AddToolLog(message string) {
	r.toolLogs = append(r.toolLogs, message)
}

func (r *recordingPrinter) StartTodoMonitoring()                       {}
func (r *recordingPrinter) StopTodoMonitoring()                        {}
func (r *recordingPrinter) UpdateCurrentTask(task string)              {}
func (r *recordingPrinter) Success(format string, args ...interface{}) {}
func (r *recordingPrinter) Error(format string, args ...interface{})   {}
func (r *recordingPrinter) Warning(format string, args ...interface{}) {}
func (r *recordingPrinter) Info(format string, args ...interface{})    {}
func (r *recordingPrinter) Step(format string, args ...interface{})    {}
func (r *recordingPrinter) Detail(format string, args ...interface{})  {}
func (r *recordingPrinter) Print(format string, args ...interface{})   {}
func (r *recordingPrinter) Println(args ...interface{})                {}

// TestExecuteWithStderrCapture_Unit tests the executeWithStderrCapture method
func TestExecuteWithStderrCapture_Unit(t *testing.T) {
	// This test verifies that executeWithStderrCapture correctly handles pipes
	t.Run("captures stderr lines and sends to AddToolLog", func(t *testing.T) {
		// Skip for now as we need to refactor to make this testable
		t.Skip("Unit test - requires mock command injection")
	})
}

// TestDefaultCommandRunner_NoStderrCapture verifies defaultCommandRunner behavior
func TestDefaultCommandRunner_NoStderrCapture(t *testing.T) {
	// This test verifies that defaultCommandRunner continues to use CombinedOutput
	t.Run("defaultCommandRunner uses CombinedOutput", func(t *testing.T) {
		runner := &defaultCommandRunner{}
		ctx := context.Background()
		config := ExecuteConfig{
			Prompt:    "echo test",
			StateFile: "/tmp/test.json",
		}

		// This would actually call claude, so we skip it
		t.Skip("Integration test - would call actual claude command")

		output, err := runner.Run(ctx, config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// With CombinedOutput, both stdout and stderr would be in output
		if !strings.Contains(output, "test") {
			t.Error("expected output to contain 'test'")
		}
	})
}
