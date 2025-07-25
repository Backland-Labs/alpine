//go:build manual
// +build manual

package claude

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/maxmcd/alpine/internal/config"
	"github.com/maxmcd/alpine/internal/output"
)

// TestManualStderrCapture is a manual test to verify stderr capture works
// Run with: go test -tags=manual -run TestManualStderrCapture -v
func TestManualStderrCapture(t *testing.T) {
	// Create a real printer
	printer := output.NewPrinter()

	// Create executor with real components
	executor := &Executor{
		config: &config.Config{
			ShowTodoUpdates: true,
		},
		printer: printer,
	}

	// Test the executeWithStderrCapture method directly
	ctx := context.Background()

	// Create a command that writes to both stdout and stderr
	cmd := exec.CommandContext(ctx, "sh", "-c", `
		echo "Starting process..." >&2
		echo "stdout line 1"
		sleep 0.1
		echo "Tool: Reading file.txt" >&2
		echo "stdout line 2"
		sleep 0.1
		echo "Tool: Writing output.txt" >&2
		echo "Done"
	`)

	fmt.Println("=== Running command with stderr capture ===")
	startTime := time.Now()

	output, err := executor.executeWithStderrCapture(ctx, cmd)
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	fmt.Printf("Duration: %v\n", time.Since(startTime))
	fmt.Printf("Stdout output:\n%s\n", output)
	fmt.Println("=== Command completed ===")

	// The stderr output should have been sent to printer.AddToolLog
	// In a real scenario, these would appear in the tool log display
}
