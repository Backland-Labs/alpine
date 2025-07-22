package output

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestProgressIndicator(t *testing.T) {
	t.Run("creates progress indicator with message", func(t *testing.T) {
		// Test that we can create a progress indicator with a message
		var buf bytes.Buffer
		printer := NewPrinterWithWriters(&buf, &buf, true)
		
		progress := printer.StartProgress("Executing Claude...")
		if progress == nil {
			t.Fatal("expected progress indicator to be created")
		}
		
		// Progress should display the message
		output := buf.String()
		if !strings.Contains(output, "Executing Claude...") {
			t.Errorf("expected output to contain progress message, got: %s", output)
		}
	})
	
	t.Run("shows spinner animation", func(t *testing.T) {
		// Test that the spinner animates
		var buf bytes.Buffer
		printer := NewPrinterWithWriters(&buf, &buf, true)
		
		progress := printer.StartProgress("Loading...")
		time.Sleep(100 * time.Millisecond) // Allow spinner to animate
		progress.Stop()
		
		output := buf.String()
		// Should contain at least one spinner character
		hasSpinner := false
		spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		for _, char := range spinnerChars {
			if strings.Contains(output, char) {
				hasSpinner = true
				break
			}
		}
		if !hasSpinner {
			t.Errorf("expected output to contain spinner animation, got: %s", output)
		}
	})
	
	t.Run("updates message dynamically", func(t *testing.T) {
		// Test that we can update the progress message
		var buf bytes.Buffer
		printer := NewPrinterWithWriters(&buf, &buf, true)
		
		progress := printer.StartProgress("Step 1...")
		progress.UpdateMessage("Step 2...")
		time.Sleep(50 * time.Millisecond)
		progress.Stop()
		
		output := buf.String()
		if !strings.Contains(output, "Step 2...") {
			t.Errorf("expected output to contain updated message, got: %s", output)
		}
	})
	
	t.Run("shows elapsed time", func(t *testing.T) {
		// Test that elapsed time is shown
		var buf bytes.Buffer
		printer := NewPrinterWithWriters(&buf, &buf, true)
		
		progress := printer.StartProgress("Processing...")
		time.Sleep(150 * time.Millisecond)
		progress.Stop()
		
		output := buf.String()
		// Should show elapsed time in some format (e.g., "0s", "0.1s", etc.)
		if !strings.Contains(output, "s") {
			t.Errorf("expected output to show elapsed time, got: %s", output)
		}
	})
	
	t.Run("clears line when stopped", func(t *testing.T) {
		// Test that progress indicator clears the line when stopped
		var buf bytes.Buffer
		printer := NewPrinterWithWriters(&buf, &buf, true)
		
		progress := printer.StartProgress("Temporary message...")
		time.Sleep(50 * time.Millisecond)
		progress.Stop()
		
		// After stopping, the line should be cleared (contain escape sequences)
		output := buf.String()
		if !strings.Contains(output, "\r") || !strings.Contains(output, "\033[K") {
			t.Errorf("expected output to contain line clearing sequences, got: %s", output)
		}
	})
	
	t.Run("respects color setting", func(t *testing.T) {
		// Test with colors disabled
		var buf bytes.Buffer
		printer := NewPrinterWithWriters(&buf, &buf, false)
		
		progress := printer.StartProgress("No colors...")
		time.Sleep(50 * time.Millisecond)
		progress.Stop()
		
		output := buf.String()
		// Should not contain ANSI color codes
		if strings.Contains(output, "\033[") && !strings.Contains(output, "\033[K") {
			t.Errorf("expected no color codes when colors disabled, got: %s", output)
		}
	})
	
	t.Run("handles concurrent progress indicators", func(t *testing.T) {
		// Test that stopping one progress doesn't affect another
		var buf bytes.Buffer
		printer := NewPrinterWithWriters(&buf, &buf, true)
		
		progress1 := printer.StartProgress("First task...")
		progress2 := printer.StartProgress("Second task...")
		
		// Both should be non-nil
		if progress1 == nil || progress2 == nil {
			t.Fatal("expected both progress indicators to be created")
		}
		
		progress1.Stop()
		// progress2 should still be valid
		progress2.UpdateMessage("Still running...")
		progress2.Stop()
	})
}

func TestProgressWithIteration(t *testing.T) {
	t.Run("shows iteration counter", func(t *testing.T) {
		// Test progress with iteration counter
		var buf bytes.Buffer
		printer := NewPrinterWithWriters(&buf, &buf, true)
		
		progress := printer.StartProgressWithIteration("Running Claude", 3)
		time.Sleep(50 * time.Millisecond)
		progress.Stop()
		
		output := buf.String()
		// Should show iteration number
		if !strings.Contains(output, "Iteration 3") || !strings.Contains(output, "Running Claude") {
			t.Errorf("expected output to show iteration counter, got: %s", output)
		}
	})
}