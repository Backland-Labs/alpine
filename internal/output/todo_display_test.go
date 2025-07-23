package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrinter_TodoMonitoring(t *testing.T) {
	// Test todo monitoring display functions
	t.Run("StartTodoMonitoring displays correct message", func(t *testing.T) {
		var buf bytes.Buffer
		printer := &Printer{
			useColor: true,
			out:      &buf,
		}

		printer.StartTodoMonitoring()

		output := buf.String()
		if !strings.Contains(output, "⚡") {
			t.Error("Expected lightning emoji in output")
		}
		if !strings.Contains(output, "Monitoring Claude's progress...") {
			t.Error("Expected monitoring message in output")
		}
	})

	t.Run("UpdateCurrentTask displays task with clear line", func(t *testing.T) {
		var buf bytes.Buffer
		printer := &Printer{
			useColor: true,
			out:      &buf,
		}

		printer.UpdateCurrentTask("Implementing user authentication")

		output := buf.String()
		// Check for clear line ANSI code
		if !strings.Contains(output, "\r\033[K") {
			t.Error("Expected clear line ANSI code")
		}
		if !strings.Contains(output, "🔄") {
			t.Error("Expected refresh emoji in output")
		}
		if !strings.Contains(output, "Working on: Implementing user authentication") {
			t.Error("Expected task description in output")
		}
	})

	t.Run("UpdateCurrentTask ignores empty tasks", func(t *testing.T) {
		var buf bytes.Buffer
		printer := &Printer{
			useColor: true,
			out:      &buf,
		}

		printer.UpdateCurrentTask("")

		output := buf.String()
		if output != "" {
			t.Error("Expected no output for empty task")
		}
	})

	t.Run("StopTodoMonitoring displays completion message", func(t *testing.T) {
		var buf bytes.Buffer
		printer := &Printer{
			useColor: true,
			out:      &buf,
		}

		printer.StopTodoMonitoring()

		output := buf.String()
		// Check for clear line ANSI code
		if !strings.Contains(output, "\r\033[K") {
			t.Error("Expected clear line ANSI code")
		}
		if !strings.Contains(output, "✓") {
			t.Error("Expected checkmark in output")
		}
		if !strings.Contains(output, "Task completed") {
			t.Error("Expected completion message in output")
		}
	})

	t.Run("respects color settings", func(t *testing.T) {
		var buf bytes.Buffer
		printer := &Printer{
			useColor: false,
			out:      &buf,
		}

		printer.StartTodoMonitoring()

		output := buf.String()
		// Should still have emoji and message but no color codes
		if !strings.Contains(output, "⚡") {
			t.Error("Expected lightning emoji in output")
		}
		if !strings.Contains(output, "Monitoring Claude's progress...") {
			t.Error("Expected monitoring message in output")
		}
		// Check that there are no ANSI color codes (beyond clear line)
		if strings.Contains(output, "\033[3") || strings.Contains(output, "\033[9") {
			t.Error("Found color codes when color is disabled")
		}
	})
}
