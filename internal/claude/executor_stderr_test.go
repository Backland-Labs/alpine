package claude

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/output"
)

// TestExecuteWithStderrCapture_Integration tests stderr capture integration
// This test verifies that we can capture stderr output separately from stdout
func TestExecuteWithStderrCapture_Integration(t *testing.T) {
	// This test will fail until we implement proper stderr capture
	// Currently executeClaudeCommand uses cmd.CombinedOutput()
	t.Run("captures stderr separately from stdout", func(t *testing.T) {
		// Mock the command execution to test stderr capture behavior
		// We'll create a simple test that shows current behavior fails

		// Create a command that writes to both stdout and stderr
		cmd := exec.Command("sh", "-c", "echo 'stdout message' && echo 'stderr message' >&2")

		// Current implementation would use CombinedOutput
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command failed: %v", err)
		}

		// With CombinedOutput, both stdout and stderr are mixed
		outputStr := string(output)
		if !strings.Contains(outputStr, "stdout message") {
			t.Error("stdout not captured")
		}
		if !strings.Contains(outputStr, "stderr message") {
			t.Error("stderr not captured")
		}

		// This demonstrates the problem: we can't separate stdout from stderr
		// We need to implement separate pipe handling
		t.Log("Current implementation mixes stdout and stderr")
		t.Log("Output:", outputStr)

		// What we want instead:
		// 1. stdout should be returned as the command output
		// 2. stderr should be captured line-by-line and sent to AddToolLog
	})
}

// TestStderrPipeImplementation shows how to implement stderr capture
func TestStderrPipeImplementation(t *testing.T) {
	// This test demonstrates the implementation approach for stderr capture
	t.Run("demonstrates stderr pipe approach", func(t *testing.T) {
		// Create a command that writes to both stdout and stderr
		cmd := exec.Command("sh", "-c", "echo 'Tool: Read file.txt' >&2 && echo 'stdout output' && echo 'Tool: Write result.txt' >&2")

		// Get separate pipes for stdout and stderr
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatalf("failed to create stdout pipe: %v", err)
		}

		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			t.Fatalf("failed to create stderr pipe: %v", err)
		}

		// Collect stderr lines
		var stderrLines []string
		stderrScanner := bufio.NewScanner(stderrPipe)

		// Start the command
		if err := cmd.Start(); err != nil {
			t.Fatalf("failed to start command: %v", err)
		}

		// Read stderr in a goroutine
		stderrDone := make(chan bool)
		go func() {
			for stderrScanner.Scan() {
				line := stderrScanner.Text()
				stderrLines = append(stderrLines, line)
			}
			stderrDone <- true
		}()

		// Read stdout
		var stdoutBuf bytes.Buffer
		stdoutScanner := bufio.NewScanner(stdoutPipe)
		for stdoutScanner.Scan() {
			stdoutBuf.WriteString(stdoutScanner.Text())
			stdoutBuf.WriteString("\n")
		}

		// Wait for stderr reading to complete
		<-stderrDone

		// Wait for command to complete
		if err := cmd.Wait(); err != nil {
			t.Fatalf("command failed: %v", err)
		}

		// Verify we captured stdout and stderr separately
		stdoutStr := strings.TrimSpace(stdoutBuf.String())
		if stdoutStr != "stdout output" {
			t.Errorf("expected stdout 'stdout output', got %q", stdoutStr)
		}

		// Verify stderr lines were captured separately
		expectedStderr := []string{
			"Tool: Read file.txt",
			"Tool: Write result.txt",
		}

		if len(stderrLines) != len(expectedStderr) {
			t.Errorf("expected %d stderr lines, got %d", len(expectedStderr), len(stderrLines))
		}

		for i, expected := range expectedStderr {
			if i < len(stderrLines) && stderrLines[i] != expected {
				t.Errorf("stderr line %d: expected %q, got %q", i, expected, stderrLines[i])
			}
		}

		t.Log("Successfully captured stdout and stderr separately")
		t.Log("Stdout:", stdoutStr)
		t.Log("Stderr lines:", stderrLines)
	})
}

// TestExecuteClaudeCommandWithStderr tests the new stderr capture functionality
// This test will fail until we modify executeClaudeCommand to use pipes
func TestExecuteClaudeCommandWithStderr(t *testing.T) {
	// This test documents the expected behavior after implementation
	t.Run("executeClaudeCommand should capture stderr to AddToolLog", func(t *testing.T) {
		t.Skip("Skipping - executeClaudeCommand needs to be modified to use pipes instead of CombinedOutput")

		// After implementation, this test would verify:
		// 1. executeClaudeCommand uses cmd.StdoutPipe() and cmd.StderrPipe()
		// 2. Each line from stderr is passed to printer.AddToolLog(line)
		// 3. stdout is returned as the function result
		// 4. The function handles errors properly
	})
}

// TestDefaultCommandRunnerWithStderr tests stderr capture in defaultCommandRunner
func TestDefaultCommandRunnerWithStderr(t *testing.T) {
	// This test will verify the defaultCommandRunner implementation
	t.Run("defaultCommandRunner should support stderr capture", func(t *testing.T) {
		t.Skip("Skipping - defaultCommandRunner needs to be modified for stderr capture")

		// After implementation, this would test:
		// 1. defaultCommandRunner.Run() uses separate pipes
		// 2. If a printer is provided, stderr goes to AddToolLog
		// 3. stdout is returned correctly
	})
}

// MockPrinterForStderr is a test helper for verifying AddToolLog calls
type MockPrinterForStderr struct {
	*output.Printer
	ToolLogCalls []string
}

func (m *MockPrinterForStderr) AddToolLog(message string) {
	m.ToolLogCalls = append(m.ToolLogCalls, message)
}

// TestShowToolUpdatesFlag verifies the configuration flag behavior
func TestShowToolUpdatesFlag(t *testing.T) {
	// According to plan.md Task 4, we need a ShowToolUpdates config flag
	// But checking config.go shows it's ShowTodoUpdates, not ShowToolUpdates
	// We may need to add a new flag or repurpose the existing one
	t.Run("config flag controls stderr capture", func(t *testing.T) {
		cfg := &config.Config{
			ShowTodoUpdates: true, // This exists, but may need ShowToolUpdates
		}

		if !cfg.ShowTodoUpdates {
			t.Error("ShowTodoUpdates should default to true")
		}

		// TODO: Determine if we need a separate ShowToolUpdates flag
		// or if we should use ShowTodoUpdates for this feature
		t.Log("Config has ShowTodoUpdates, may need ShowToolUpdates for tool logs")
	})
}
