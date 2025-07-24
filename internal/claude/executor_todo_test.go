package claude

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxmcd/river/internal/config"
	"github.com/maxmcd/river/internal/output"
)

func TestExecutor_executeWithTodoMonitoring(t *testing.T) {
	// Test the executeWithTodoMonitoring functionality
	t.Run("falls back to executeWithoutMonitoring on hook setup failure", func(t *testing.T) {
		// Create a test directory where we can't write files
		readOnlyDir := t.TempDir()
		// Change to that directory to make setupTodoHook fail when creating .claude dir
		oldWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get working directory: %v", err)
		}
		defer func() { _ = os.Chdir(oldWd) }()

		if err := os.Chdir(readOnlyDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		// Make directory read-only to cause hook setup failure
		if err := os.Chmod(readOnlyDir, 0555); err != nil {
			t.Fatalf("Failed to make directory read-only: %v", err)
		}
		defer func() { _ = os.Chmod(readOnlyDir, 0755) }()

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

	t.Run("sets up todo monitoring infrastructure", func(t *testing.T) {
		// Change to temp directory for test isolation
		tmpDir := t.TempDir()
		oldWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get working directory: %v", err)
		}
		defer func() { _ = os.Chdir(oldWd) }()

		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		executor := &Executor{
			config: &config.Config{
				ShowTodoUpdates: true,
			},
			printer: output.NewPrinterWithWriters(os.Stdout, os.Stderr, false),
		}

		// Test setupTodoHook directly to verify it creates the necessary files
		todoFile, cleanup, err := executor.setupTodoHook()
		if err != nil {
			t.Fatalf("setupTodoHook failed: %v", err)
		}
		defer cleanup()

		// Verify todo file was created
		if _, err := os.Stat(todoFile); os.IsNotExist(err) {
			t.Error("Todo file was not created")
		}

		// Check if .claude directory was created
		claudeDir := ".claude"
		if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
			t.Error(".claude directory was not created")
		}

		// Check if settings.local.json was created
		settingsFile := filepath.Join(claudeDir, "settings.local.json")
		if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
			t.Error("settings.local.json was not created")
		}

		// Check if hook script was created
		hookScript := filepath.Join(claudeDir, "todo-monitor.rs")
		if _, err := os.Stat(hookScript); os.IsNotExist(err) {
			t.Error("todo-monitor.rs was not created")
		}

		// Verify the hook script is executable
		if info, err := os.Stat(hookScript); err == nil {
			if info.Mode()&0111 == 0 {
				t.Error("Hook script is not executable")
			}
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
		_ = os.Setenv("HOME", tmpDir)
		defer func() { _ = os.Setenv("HOME", oldHome) }()

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
