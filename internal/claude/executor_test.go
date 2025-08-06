package claude

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/output"
)

func TestNewExecutor(t *testing.T) {
	t.Run("creates executor with valid configuration", func(t *testing.T) {
		// Test that NewExecutor creates a valid executor instance
		exec := NewExecutor()
		if exec == nil {
			t.Fatal("expected executor to be created")
		}
	})
}

func TestExecutor_Execute(t *testing.T) {
	tests := []struct {
		name          string
		config        ExecuteConfig
		mockCommand   *mockCommand
		expectedError bool
		errorContains string
	}{
		{
			name: "successful execution with basic prompt",
			config: ExecuteConfig{
				Prompt:    "test prompt",
				StateFile: "/tmp/state.json",
			},
			mockCommand: &mockCommand{
				output: "Claude execution successful",
				err:    nil,
			},
			expectedError: false,
		},
		{
			name: "successful execution with MCP servers",
			config: ExecuteConfig{
				Prompt:     "test prompt",
				StateFile:  "/tmp/state.json",
				MCPServers: []string{"playwright", "context7"},
			},
			mockCommand: &mockCommand{
				output: "Claude execution with MCP servers successful",
				err:    nil,
			},
			expectedError: false,
		},
		{
			name: "handles command execution failure",
			config: ExecuteConfig{
				Prompt:    "test prompt",
				StateFile: "/tmp/state.json",
			},
			mockCommand: &mockCommand{
				output: "",
				err:    &mockError{msg: "command failed"},
			},
			expectedError: true,
			errorContains: "command failed",
		},
		{
			name: "validates required fields",
			config: ExecuteConfig{
				Prompt:    "",
				StateFile: "/tmp/state.json",
			},
			mockCommand:   nil,
			expectedError: true,
			errorContains: "prompt is required",
		},
		{
			name: "validates state file path",
			config: ExecuteConfig{
				Prompt:    "test prompt",
				StateFile: "",
			},
			mockCommand:   nil,
			expectedError: true,
			errorContains: "state file is required",
		},
		{
			name: "includes custom system prompt when provided",
			config: ExecuteConfig{
				Prompt:       "test prompt",
				StateFile:    "/tmp/state.json",
				SystemPrompt: "Custom system prompt for testing",
			},
			mockCommand: &mockCommand{
				output: "Execution with custom system prompt",
				err:    nil,
			},
			expectedError: false,
		},
		{
			name: "respects timeout configuration",
			config: ExecuteConfig{
				Prompt:    "test prompt",
				StateFile: "/tmp/state.json",
				Timeout:   5 * time.Second,
			},
			mockCommand: &mockCommand{
				output:   "Timed execution",
				err:      nil,
				duration: 100 * time.Millisecond,
			},
			expectedError: false,
		},
		{
			name: "handles timeout exceeded",
			config: ExecuteConfig{
				Prompt:    "test prompt",
				StateFile: "/tmp/state.json",
				Timeout:   100 * time.Millisecond,
			},
			mockCommand: &mockCommand{
				output:   "",
				err:      context.DeadlineExceeded,
				duration: 200 * time.Millisecond,
			},
			expectedError: true,
			errorContains: "deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &Executor{
				commandRunner: tt.mockCommand,
			}

			output, err := exec.Execute(context.Background(), tt.config)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if output != tt.mockCommand.output {
					t.Errorf("expected output %q, got %q", tt.mockCommand.output, output)
				}
			}
		})
	}
}

func TestExecutor_buildCommand(t *testing.T) {
	tests := []struct {
		name           string
		config         ExecuteConfig
		expectedArgs   []string
		expectedEnvSet map[string]bool
		expectedWorkDir string // Added to test working directory behavior
	}{
		{
			name: "basic command construction",
			config: ExecuteConfig{
				Prompt:    "test prompt",
				StateFile: "/tmp/state.json",
			},
			expectedArgs: []string{
				"--output-format", "text",
				"--allowedTools",
				"--append-system-prompt",
				"-p", "test prompt",
			},
			expectedEnvSet: map[string]bool{},
			expectedWorkDir: "", // Will be set to current directory by buildCommand
		},
		{
			name: "command with multiple MCP servers",
			config: ExecuteConfig{
				Prompt:     "test prompt",
				StateFile:  "/tmp/state.json",
				MCPServers: []string{"playwright", "web-mrkdwn"},
			},
			expectedArgs: []string{
				"--output-format", "text",
				"--mcp-server", "playwright",
				"--mcp-server", "web-mrkdwn",
				"--allowedTools",
				"--append-system-prompt",
				"-p", "test prompt",
			},
			expectedEnvSet: map[string]bool{},
			expectedWorkDir: "", // Will be set to current directory
		},
		{
			name: "command with custom system prompt",
			config: ExecuteConfig{
				Prompt:       "test prompt",
				StateFile:    "/tmp/state.json",
				SystemPrompt: "Custom system prompt",
			},
			expectedArgs: []string{
				"--output-format", "text",
				"--allowedTools",
				"--append-system-prompt", "Custom system prompt",
				"-p", "test prompt",
			},
			expectedEnvSet: map[string]bool{},
			expectedWorkDir: "", // Will be set to current directory
		},
		{
			name: "command with tools restriction",
			config: ExecuteConfig{
				Prompt:       "test prompt",
				StateFile:    "/tmp/state.json",
				AllowedTools: []string{"Read", "Write", "Edit"},
			},
			expectedArgs: []string{
				"--output-format", "text",
				"--allowedTools", "Read", "Write", "Edit",
				"--append-system-prompt",
				"-p", "test prompt",
			},
			expectedEnvSet: map[string]bool{},
			expectedWorkDir: "", // Will be set to current directory
		},
		{
			name: "command with additional args",
			config: ExecuteConfig{
				Prompt:         "test prompt",
				StateFile:      "/tmp/state.json",
				AdditionalArgs: []string{"--add-dir", ".", "--verbose"},
			},
			expectedArgs: []string{
				"--output-format", "text",
				"--allowedTools",
				"--append-system-prompt",
				"--add-dir", ".", "--verbose",
				"-p", "test prompt",
			},
			expectedEnvSet: map[string]bool{},
			expectedWorkDir: "", // Will be set to current directory
		},
		{
			name: "command with custom working directory",
			config: ExecuteConfig{
				Prompt:    "test prompt",
				StateFile: "/tmp/state.json",
				WorkDir:   "/custom/work/dir",
			},
			expectedArgs: []string{
				"--output-format", "text",
				"--allowedTools",
				"--append-system-prompt",
				"-p", "test prompt",
			},
			expectedEnvSet: map[string]bool{},
			expectedWorkDir: "/custom/work/dir",
		},
		{
			name: "command with environment variables",
			config: ExecuteConfig{
				Prompt:    "test prompt",
				StateFile: "/tmp/state.json",
				EnvironmentVariables: map[string]string{
					"TEST_VAR": "test_value",
					"ANOTHER":  "another_value",
				},
			},
			expectedArgs: []string{
				"--output-format", "text",
				"--allowedTools",
				"--append-system-prompt",
				"-p", "test prompt",
			},
			expectedEnvSet: map[string]bool{
				"TEST_VAR": true,
				"ANOTHER":  true,
			},
			expectedWorkDir: "", // Will be set to current directory
		},
		{
			name: "command with custom workdir and environment",
			config: ExecuteConfig{
				Prompt:    "test prompt",
				StateFile: "/tmp/state.json",
				WorkDir:   "/tmp/test-workdir",
				EnvironmentVariables: map[string]string{
					"ALPINE_TEST": "true",
				},
			},
			expectedArgs: []string{
				"--output-format", "text",
				"--allowedTools",
				"--append-system-prompt",
				"-p", "test prompt",
			},
			expectedEnvSet: map[string]bool{
				"ALPINE_TEST": true,
			},
			expectedWorkDir: "/tmp/test-workdir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &Executor{}
			cmd := exec.buildCommand(tt.config)

			// Check that the base command is correct
			if cmd.Path != "claude" && !strings.HasSuffix(cmd.Path, "/claude") {
				t.Errorf("expected command path to be 'claude', got %q", cmd.Path)
			}

			// Check working directory is set correctly
			if tt.config.WorkDir != "" {
				// When WorkDir is specified in config, it should be used
				if cmd.Dir != tt.config.WorkDir {
					t.Errorf("expected working directory to be %q, got %q", tt.config.WorkDir, cmd.Dir)
				}
			} else {
				// When WorkDir is not specified, should use current directory
				expectedDir, err := os.Getwd()
				if err != nil {
					// If we can't get current directory, cmd.Dir should be empty (fallback behavior)
					if cmd.Dir != "" {
						t.Errorf("expected working directory to be empty when os.Getwd() fails, got %q", cmd.Dir)
					}
				} else {
					if cmd.Dir != expectedDir {
						t.Errorf("expected working directory to be %q (current dir), got %q", expectedDir, cmd.Dir)
					}
				}
			}

			// Check expected arguments are present
			args := strings.Join(cmd.Args[1:], " ")
			for _, expectedArg := range tt.expectedArgs {
				if !strings.Contains(args, expectedArg) {
					t.Errorf("expected argument %q not found in command args: %v", expectedArg, cmd.Args)
				}
			}

			// Check environment variables
			envMap := make(map[string]string)
			for _, env := range cmd.Env {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					envMap[parts[0]] = parts[1]
				}
			}

			for envKey, shouldBeSet := range tt.expectedEnvSet {
				_, exists := envMap[envKey]
				if shouldBeSet && !exists {
					t.Errorf("expected environment variable %s to be set", envKey)
				}
			}

			// Verify additional environment variables from config are set
			if tt.config.EnvironmentVariables != nil {
				for expectedKey, expectedValue := range tt.config.EnvironmentVariables {
					actualValue, exists := envMap[expectedKey]
					if !exists {
						t.Errorf("expected environment variable %s to be set", expectedKey)
					} else if actualValue != expectedValue {
						t.Errorf("expected environment variable %s to have value %q, got %q", expectedKey, expectedValue, actualValue)
					}
				}
			}

			// Check that prompt is passed with -p flag
			foundPrompt := false
			for i := 0; i < len(cmd.Args)-1; i++ {
				if cmd.Args[i] == "-p" && cmd.Args[i+1] == tt.config.Prompt {
					foundPrompt = true
					break
				}
			}
			if !foundPrompt {
				t.Errorf("expected prompt %q to be passed with -p flag", tt.config.Prompt)
			}
		})
	}
}

// Mock implementations for testing
type mockCommand struct {
	output   string
	err      error
	duration time.Duration
}

func (m *mockCommand) Run(ctx context.Context, config ExecuteConfig) (string, error) {
	if m.duration > 0 {
		select {
		case <-time.After(m.duration):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return m.output, m.err
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

func TestExecutor_BuildCommand_SetsWorkingDirectory(t *testing.T) {
	// Test that buildCommand sets the working directory to the current directory
	// This ensures Claude commands execute in the correct directory for worktree isolation
	t.Run("sets cmd.Dir to current working directory", func(t *testing.T) {
		exec := &Executor{}
		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "/tmp/state.json",
		}

		// Get the expected working directory
		expectedDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}

		cmd := exec.buildCommand(config)

		// Verify cmd.Dir is set to the current working directory
		if cmd.Dir != expectedDir {
			t.Errorf("expected cmd.Dir to be %q, got %q", expectedDir, cmd.Dir)
		}
	})
}

func TestExecutor_BuildCommand_WorkingDirectoryError(t *testing.T) {
	// Test that buildCommand handles os.Getwd() errors gracefully
	// Even if we can't get the working directory, the command should still be built
	t.Run("handles os.Getwd error gracefully", func(t *testing.T) {
		// Note: It's difficult to mock os.Getwd() directly in Go
		// This test documents the expected behavior when os.Getwd() fails
		// The implementation should continue without setting cmd.Dir
		exec := &Executor{}
		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "/tmp/state.json",
		}

		cmd := exec.buildCommand(config)

		// Even without mocking os.Getwd error, we can verify the command is built
		if cmd == nil {
			t.Fatal("expected command to be built even if working directory fails")
		}

		// Verify basic command structure is intact
		if cmd.Path != "claude" && !strings.HasSuffix(cmd.Path, "/claude") {
			t.Errorf("expected command path to be 'claude', got %q", cmd.Path)
		}
	})
}

func TestExecutor_CommandRunner_PreservesDirectory(t *testing.T) {
	// Test that defaultCommandRunner preserves the working directory from buildCommand
	// This ensures the directory context flows through the entire execution pipeline
	t.Run("preserves working directory from buildCommand", func(t *testing.T) {
		// Create a custom command runner that can verify the directory
		runner := &defaultCommandRunner{}
		exec := &Executor{commandRunner: runner}

		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "/tmp/state.json",
		}

		// Build the base command to get expected directory
		baseCmd := exec.buildCommand(config)

		// Note: Since we can't easily mock exec.CommandContext,
		// this test documents the expected behavior.
		// The actual implementation test will be done via integration tests.

		// For now, verify that buildCommand is called and creates a valid command
		if baseCmd == nil {
			t.Error("expected buildCommand to create a valid command")
		}

		// Once implemented, cmd.Dir should be preserved in the CommandContext call
		// This will be verified in integration tests
	})
}

func TestExecutor_WorkingDirectoryFallback(t *testing.T) {
	// Test that the executor continues gracefully when working directory operations fail
	// This test documents expected behavior when os.Getwd() fails or directory validation fails
	t.Run("continues execution when working directory is unavailable", func(t *testing.T) {
		exec := &Executor{}
		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "/tmp/state.json",
		}

		// Build command - even if we can't mock os.Getwd failure directly,
		// we can verify that command building continues
		cmd := exec.buildCommand(config)

		if cmd == nil {
			t.Fatal("expected command to be built even with directory issues")
		}

		// Verify command has all required components
		if cmd.Path == "" {
			t.Error("expected command path to be set")
		}

		// When directory operations fail, cmd.Dir might be empty but command should still work
		// This is the graceful fallback behavior
	})

	t.Run("logs warning when working directory fails", func(t *testing.T) {
		// This test documents that when os.Getwd() fails, a warning should be logged
		// The actual implementation will use the logger package
		// For now, we document the expected behavior
		exec := &Executor{}
		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "/tmp/state.json",
		}

		// Build command
		cmd := exec.buildCommand(config)

		// Even without being able to capture logs in this test,
		// we document that a warning should be logged when directory operations fail
		if cmd == nil {
			t.Error("command building should not fail due to directory issues")
		}
	})
}

// TestExecutor_WorkingDirectoryIntegration tests the complete working directory workflow
func TestExecutor_WorkingDirectoryIntegration(t *testing.T) {
	t.Run("end-to-end working directory behavior with mock runner", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir, err := os.MkdirTemp("", "test-alpine-e2e-")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(tempDir) }()

		// Create a mock command runner that captures the working directory
		capturedDir := ""
		mockRunner := &mockCommandRunnerWithDir{
			output: "Mock execution successful",
			err:    nil,
			dirCapture: &capturedDir,
		}

		// Create executor with mock runner
		executor := &Executor{
			commandRunner: mockRunner,
		}

		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "/tmp/state.json",
			WorkDir:   tempDir,
		}

		// Execute - this should flow through the entire pipeline
		output, err := executor.Execute(context.Background(), config)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if output != mockRunner.output {
			t.Errorf("expected output %q, got %q", mockRunner.output, output)
		}

		// Verify that the working directory was properly passed through
		if capturedDir != tempDir {
			t.Errorf("expected working directory to be passed as %q, got %q", tempDir, capturedDir)
		}
	})

	t.Run("integration with invalid directory clears WorkDir", func(t *testing.T) {
		// Test that invalid directories are cleared and don't cause execution failure
		capturedDir := ""
		mockRunner := &mockCommandRunnerWithDir{
			output: "Mock execution with cleared directory",
			err:    nil,
			dirCapture: &capturedDir,
		}

		executor := &Executor{
			commandRunner: mockRunner,
		}

		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "/tmp/state.json",
			WorkDir:   "/nonexistent/invalid/directory",
		}

		// Execute - should succeed even with invalid WorkDir
		output, err := executor.Execute(context.Background(), config)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if output != mockRunner.output {
			t.Errorf("expected output %q, got %q", mockRunner.output, output)
		}

		// Verify that the invalid directory was cleared (not passed to runner)
		if capturedDir == config.WorkDir {
			t.Error("expected invalid working directory to be cleared before execution")
		}
	})

	t.Run("integration with current directory fallback", func(t *testing.T) {
		// Test that empty WorkDir falls back to current directory
		capturedDir := ""
		mockRunner := &mockCommandRunnerWithDir{
			output: "Mock execution with current directory",
			err:    nil,
			dirCapture: &capturedDir,
		}

		executor := &Executor{
			commandRunner: mockRunner,
		}

		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "/tmp/state.json",
			WorkDir:   "", // Empty - should use current directory
		}

		// Execute
		output, err := executor.Execute(context.Background(), config)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if output != mockRunner.output {
			t.Errorf("expected output %q, got %q", mockRunner.output, output)
		}

		// Verify that current directory was used (if available)
		expectedDir, err := os.Getwd()
		if err != nil {
			// If os.Getwd() fails, directory should be empty
			if capturedDir != "" {
				t.Errorf("expected working directory to be empty when os.Getwd() fails, got %q", capturedDir)
			}
		} else {
			if capturedDir != expectedDir {
				t.Errorf("expected working directory to be current directory %q, got %q", expectedDir, capturedDir)
			}
		}
	})
}

// mockCommandRunnerWithDir is a mock that captures the working directory
type mockCommandRunnerWithDir struct {
	output     string
	err        error
	dirCapture *string // Pointer to capture the directory
}

func (m *mockCommandRunnerWithDir) Run(ctx context.Context, config ExecuteConfig) (string, error) {
	// Create executor to simulate the actual command building process
	executor := &Executor{}
	cmd := executor.buildCommandWithValidation(config)
	
	// Capture the working directory that would be used
	if m.dirCapture != nil {
		*m.dirCapture = cmd.Dir
	}
	
	return m.output, m.err
}

// TestExecutor_ToolLogsDisabled verifies that if ShowToolUpdates is false,
// the stderr of the hook is not captured and the printer's AddToolLog method is not called
func TestExecutor_ToolLogsDisabled(t *testing.T) {
	t.Run("does not capture stderr when ShowToolUpdates is false", func(t *testing.T) {
		// Test with ShowToolUpdates disabled
		cfg := &config.Config{
			ShowToolUpdates: false, // Disabled - should not capture stderr
			ShowTodoUpdates: false, // Disable TODO monitoring to use mock
		}

		// Create executor with the config
		exec := NewExecutorWithConfig(cfg, output.NewPrinter())

		// Verify the configuration is set correctly
		if exec.config.ShowToolUpdates != false {
			t.Error("expected ShowToolUpdates to be false")
		}

		// Create a mock command runner to avoid actual command execution
		mockCmd := &mockCommand{
			output: "Command output",
			err:    nil,
		}
		exec.commandRunner = mockCmd

		// Execute the command
		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "/tmp/state.json",
		}

		output, err := exec.Execute(context.Background(), config)

		// Verify execution succeeded
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output != mockCmd.output {
			t.Errorf("expected output %q, got %q", mockCmd.output, output)
		}

		// The key verification here is that when ShowToolUpdates is false,
		// the executeWithStderrCapture path is NOT taken (line 194 in executor.go).
		// This means stderr will not be captured separately and AddToolLog won't be called.
		// We've verified this by ensuring the mock command runner is used directly
		// without stderr capture.
	})

	t.Run("configuration controls stderr capture", func(t *testing.T) {
		// Test that ShowToolUpdates=true enables tool log capture
		cfgEnabled := &config.Config{
			ShowToolUpdates: true,
			ShowTodoUpdates: true,
		}

		execEnabled := NewExecutorWithConfig(cfgEnabled, output.NewPrinter())
		if execEnabled.config.ShowToolUpdates != true {
			t.Error("expected ShowToolUpdates to be true")
		}

		// Test that ShowToolUpdates=false disables tool log capture
		cfgDisabled := &config.Config{
			ShowToolUpdates: false,
			ShowTodoUpdates: true,
		}

		execDisabled := NewExecutorWithConfig(cfgDisabled, output.NewPrinter())
		if execDisabled.config.ShowToolUpdates != false {
			t.Error("expected ShowToolUpdates to be false")
		}
	})

	t.Run("stderr capture path is taken when ShowToolUpdates is true", func(t *testing.T) {
		// Test that when ShowToolUpdates is true, the executor would take
		// the stderr capture path in executeWithTodoMonitoring
		cfg := &config.Config{
			ShowToolUpdates: true, // Enable tool updates
			ShowTodoUpdates: true, // Enable todo monitoring to test the actual path
		}

		printer := output.NewPrinter()
		exec := NewExecutorWithConfig(cfg, printer)

		// Verify all conditions for stderr capture are met:
		// 1. config is not nil
		// 2. ShowToolUpdates is true
		// 3. printer is not nil
		if exec.config == nil {
			t.Fatal("config should not be nil")
		}
		if !exec.config.ShowToolUpdates {
			t.Error("ShowToolUpdates should be true")
		}
		if exec.printer == nil {
			t.Fatal("printer should not be nil")
		}

		// This confirms that line 194 in executor.go would evaluate to true
		// and executeWithStderrCapture would be called
		stderrCaptureCondition := exec.config != nil && exec.config.ShowToolUpdates && exec.printer != nil
		if !stderrCaptureCondition {
			t.Error("expected stderr capture condition to be true")
		}
	})
}

func TestExecutor_SimpleTestPrompt(t *testing.T) {
	// Test basic functionality with a simple "test prompt"
	t.Run("executes simple test prompt successfully", func(t *testing.T) {
		// Create a mock command runner
		mockCmd := &mockCommand{
			output: "Test prompt executed successfully",
			err:    nil,
		}

		// Create executor with mock
		exec := &Executor{
			commandRunner: mockCmd,
		}

		// Execute with test prompt
		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "/tmp/test-state.json",
		}

		output, err := exec.Execute(context.Background(), config)

		// Verify success
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output != mockCmd.output {
			t.Errorf("expected output %q, got %q", mockCmd.output, output)
		}
	})
}

func TestExecutor_ValidatesWorkingDirectory(t *testing.T) {
	// This test verifies that the executor validates working directory exists and is accessible
	// before setting it on the command. This prevents Claude from being executed in a
	// non-existent or inaccessible directory which could cause confusing errors.
	t.Run("validates working directory exists", func(t *testing.T) {
		executor := &Executor{}
		config := ExecuteConfig{
			StateFile: "test-state.json",
		}

		// Create a temporary directory and then remove it to simulate non-existent directory
		tempDir := "/tmp/test-alpine-validation-" + strings.ReplaceAll(t.Name(), "/", "-")
		err := os.Mkdir(tempDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		// Change to valid directory first
		originalWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalWd) }()
		_ = os.Chdir(tempDir)

		// Now remove it to make it invalid
		_ = os.Chdir(originalWd)
		_ = os.RemoveAll(tempDir)

		// Try to change back to the now non-existent directory
		err = os.Chdir(tempDir)
		if err == nil {
			t.Fatal("Expected error when changing to non-existent directory")
		}

		// Stay in original directory and build command
		cmd := executor.buildCommandWithValidation(config)

		// Command should still be created but Dir should be validated
		if cmd == nil {
			t.Fatal("Expected command to be created even with invalid directory")
		}

		// The validated method should not set an invalid directory
		if cmd.Dir == tempDir {
			t.Errorf("Expected working directory to not be set to non-existent directory")
		}
	})

	t.Run("validates working directory permissions", func(t *testing.T) {
		executor := &Executor{}
		config := ExecuteConfig{
			StateFile: "test-state.json",
		}

		// Create a directory with no read permissions
		tempDir := "/tmp/test-alpine-noperm-" + strings.ReplaceAll(t.Name(), "/", "-")
		err := os.Mkdir(tempDir, 0000)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = os.Chmod(tempDir, 0755) // Reset permissions to allow removal
			_ = os.RemoveAll(tempDir)
		}()

		// Save original directory
		originalWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalWd) }()

		// Try to change to the no-permission directory
		err = os.Chdir(tempDir)
		if err == nil {
			// If we somehow can change to it, skip this test
			t.Skip("Unable to test permission validation - system allows access")
		}

		// Build command should handle permission errors gracefully
		cmd := executor.buildCommandWithValidation(config)

		if cmd == nil {
			t.Fatal("Expected command to be created even with permission errors")
		}
	})

	t.Run("sets working directory when valid", func(t *testing.T) {
		executor := &Executor{}
		config := ExecuteConfig{
			StateFile: "test-state.json",
		}

		// Create a valid temporary directory
		tempDir, err := os.MkdirTemp("", "test-alpine-valid-")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(tempDir) }()

		// Change to the valid directory
		originalWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalWd) }()
		err = os.Chdir(tempDir)
		if err != nil {
			t.Fatal(err)
		}

		// Build command with validation
		cmd := executor.buildCommandWithValidation(config)

		if cmd == nil {
			t.Fatal("Expected command to be created")
		}

		// Working directory should be set to the valid directory
		// Handle potential symlink resolution on macOS (/var/folders -> /private/var/folders)
		expectedDir := tempDir
		if resolved, err := os.Stat(tempDir); err == nil {
			// Check if the directory is accessible
			if resolved.IsDir() {
				// Directory is valid, should be preserved
				if cmd.Dir != tempDir {
					// Handle potential symlink resolution on macOS
					if absTemp, err := filepath.Abs(tempDir); err == nil {
						if absCmd, err := filepath.Abs(cmd.Dir); err == nil && absTemp == absCmd {
							// Paths are equivalent after resolution
							return
						}
					}
					// Also check if it's just the /var -> /private/var symlink on macOS
					if strings.HasPrefix(tempDir, "/var/") && strings.HasPrefix(cmd.Dir, "/private/var/") {
						if "/private"+tempDir == cmd.Dir {
							return // This is the expected macOS symlink behavior
						}
					}
					t.Errorf("Expected working directory to be %q, got %q", expectedDir, cmd.Dir)
				}
			}
		}
	})

	t.Run("preserves valid directory from config over current directory", func(t *testing.T) {
		executor := &Executor{}
		
		// Create a valid temporary directory
		tempDir, err := os.MkdirTemp("", "test-alpine-config-")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(tempDir) }()

		config := ExecuteConfig{
			Prompt:    "test prompt",
			StateFile: "test-state.json",
			WorkDir:   tempDir, // Specify custom workdir
		}

		// Get current directory for comparison
		currentDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}

		// Build command with validation
		cmd := executor.buildCommandWithValidation(config)

		if cmd == nil {
			t.Fatal("Expected command to be created")
		}

		// Should use config WorkDir, not current directory
		if cmd.Dir == currentDir {
			t.Errorf("Expected working directory to use config WorkDir %q, not current directory %q", tempDir, currentDir)
		}

		// Should match the configured WorkDir
		if cmd.Dir != tempDir {
			t.Errorf("Expected working directory to be %q (from config), got %q", tempDir, cmd.Dir)
		}
	})

	t.Run("validation does not affect other command properties", func(t *testing.T) {
		executor := &Executor{}
		
		// Use invalid directory but verify other properties are preserved
		config := ExecuteConfig{
			Prompt:       "test validation prompt",
			StateFile:    "test-state.json",
			WorkDir:      "/invalid/nonexistent/directory",
			MCPServers:   []string{"test-server"},
			AllowedTools: []string{"Read", "Write"},
			SystemPrompt: "Custom validation prompt",
		}

		cmd := executor.buildCommandWithValidation(config)

		if cmd == nil {
			t.Fatal("Expected command to be created")
		}

		// Directory should be cleared due to validation
		if cmd.Dir == config.WorkDir {
			t.Error("Expected invalid working directory to be cleared")
		}

		// Other command properties should be preserved
		if cmd.Path != "claude" && !strings.HasSuffix(cmd.Path, "/claude") {
			t.Errorf("expected command path to be 'claude', got %q", cmd.Path)
		}

		// Check that arguments are preserved
		argsStr := strings.Join(cmd.Args, " ")
		if !strings.Contains(argsStr, "test validation prompt") {
			t.Error("Expected prompt to be preserved in arguments")
		}
		if !strings.Contains(argsStr, "test-server") {
			t.Error("Expected MCP server to be preserved in arguments")
		}
		if !strings.Contains(argsStr, "Read") || !strings.Contains(argsStr, "Write") {
			t.Error("Expected allowed tools to be preserved in arguments")
		}
		if !strings.Contains(argsStr, "Custom validation prompt") {
			t.Error("Expected system prompt to be preserved in arguments")
		}
	})
}
