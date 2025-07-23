package claude

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maxmcd/river/internal/config"
	"github.com/maxmcd/river/internal/output"
)

func TestExecutor_executeWithTodoMonitoring(t *testing.T) {
	// Test the executeWithTodoMonitoring functionality
	t.Run("falls back to executeWithoutMonitoring on hook setup failure", func(t *testing.T) {
		// Unset HOME to cause hook setup failure
		oldHome := os.Getenv("HOME")
		os.Unsetenv("HOME")
		defer os.Setenv("HOME", oldHome)

		executor := &Executor{
			config: &config.Config{
				ShowTodoUpdates: true,
			},
			printer: output.NewPrinterWithWriters(os.Stdout, os.Stderr, false),
		}

		// Mock executeWithoutMonitoring to track if it was called
		withoutMonitoringCalled := false
		executor.commandRunner = &mockCommandRunner{
			runFunc: func(ctx context.Context, config ExecuteConfig) (string, error) {
				withoutMonitoringCalled = true
				return "success", nil
			},
		}

		cfg := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "test_state.json",
		}

		ctx := context.Background()
		result, err := executor.executeWithTodoMonitoring(ctx, cfg)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != "success" {
			t.Errorf("Expected 'success', got %s", result)
		}

		// Since hook setup fails, it should fall back to executeWithoutMonitoring
		// which uses the command runner
		if !withoutMonitoringCalled {
			t.Error("Expected fallback to executeWithoutMonitoring")
		}
	})

	t.Run("sets RIVER_TODO_FILE environment variable", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", oldHome)

		executor := &Executor{
			config: &config.Config{
				ShowTodoUpdates: true,
			},
			printer: output.NewPrinterWithWriters(os.Stdout, os.Stderr, false),
		}

		// Mock command runner
		executor.commandRunner = &mockCommandRunner{
			runFunc: func(ctx context.Context, config ExecuteConfig) (string, error) {
				// This won't be called since executeClaudeCommand is used in monitoring mode
				return "", nil
			},
		}

		cfg := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "test_state.json",
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Run in goroutine since it would block
		done := make(chan struct{})
		go func() {
			// We expect this to hang until cancelled since executeClaudeCommand
			// needs a real Claude binary. We're just testing the setup phase.
			executor.executeWithTodoMonitoring(ctx, cfg)
			close(done)
		}()

		// Give it time to set up
		time.Sleep(100 * time.Millisecond)

		// Check if .claude directory was created
		claudeDir := os.ExpandEnv("$HOME/.claude")
		if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
			t.Error(".claude directory was not created")
		}

		// Cancel and clean up
		cancel()

		select {
		case <-done:
			// Expected
		case <-time.After(1 * time.Second):
			t.Error("executeWithTodoMonitoring did not complete after cancellation")
		}
	})
}

func TestExecutor_Execute_TodoMonitoring(t *testing.T) {
	// Test the Execute method with todo monitoring
	t.Run("uses todo monitoring when ShowTodoUpdates is true", func(t *testing.T) {
		executor := &Executor{
			config: &config.Config{
				ShowTodoUpdates: true,
			},
			printer: output.NewPrinterWithWriters(os.Stdout, os.Stderr, false),
		}

		// Track which method was called
		var methodCalled string
		executor.commandRunner = &mockCommandRunner{
			runFunc: func(ctx context.Context, config ExecuteConfig) (string, error) {
				methodCalled = "commandRunner"
				return "success", nil
			},
		}

		// Set HOME to ensure hook setup would succeed
		tmpDir := t.TempDir()
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", oldHome)

		cfg := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "test_state.json",
		}

		ctx, cancel := context.WithCancel(context.Background())
		// Cancel immediately to avoid hanging on real Claude execution
		cancel()

		_, err := executor.Execute(ctx, cfg)

		// We expect a context cancelled error since we cancelled immediately
		if err == nil || !strings.Contains(err.Error(), "context canceled") {
			t.Errorf("Expected context cancelled error, got %v", err)
		}

		// The command runner should not have been called since todo monitoring was enabled
		if methodCalled == "commandRunner" {
			t.Error("Expected todo monitoring path, but command runner was called")
		}
	})

	t.Run("uses regular execution when ShowTodoUpdates is false", func(t *testing.T) {
		executor := &Executor{
			config: &config.Config{
				ShowTodoUpdates: false,
			},
			printer: output.NewPrinterWithWriters(os.Stdout, os.Stderr, false),
		}

		// Track which method was called
		var methodCalled string
		executor.commandRunner = &mockCommandRunner{
			runFunc: func(ctx context.Context, config ExecuteConfig) (string, error) {
				methodCalled = "commandRunner"
				return "success", nil
			},
		}

		cfg := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "test_state.json",
		}

		ctx := context.Background()
		result, err := executor.Execute(ctx, cfg)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != "success" {
			t.Errorf("Expected 'success', got %s", result)
		}

		if methodCalled != "commandRunner" {
			t.Error("Expected command runner to be called when ShowTodoUpdates is false")
		}
	})
}

// mockCommandRunner implements CommandRunner for testing
type mockCommandRunner struct {
	runFunc func(ctx context.Context, config ExecuteConfig) (string, error)
}

func (m *mockCommandRunner) Run(ctx context.Context, config ExecuteConfig) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, config)
	}
	return "", nil
}
