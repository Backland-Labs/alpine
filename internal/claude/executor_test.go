package claude

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestExecutor_Execute tests the Execute method of the Claude executor
func TestExecutor_Execute(t *testing.T) {
	tests := []struct {
		name          string
		command       Command
		opts          CommandOptions
		setupMock     func(t *testing.T) (cleanup func())
		expectedResp  *Response
		expectedError string
		skipOnCI      bool
	}{
		{
			name: "Execute plan command successfully",
			command: Command{
				Type:         CommandTypePlan,
				Prompt:       "Create a TODO application",
				OutputFormat: "json",
			},
			opts: CommandOptions{
				Stream:  false,
				Timeout: 30,
			},
			setupMock: func(t *testing.T) func() {
				// Create a mock claude executable that returns success
				return createMockClaude(t, 0, `{"content": "I'll create a TODO app", "continue": false}`, "")
			},
			expectedResp: &Response{
				Content:      "I'll create a TODO app",
				ContinueFlag: false,
			},
		},
		{
			name: "Execute continue command successfully",
			command: Command{
				Type:         CommandTypeContinue,
				Prompt:       "Continue",
				OutputFormat: "json",
			},
			opts: CommandOptions{
				Stream:  false,
				Timeout: 30,
			},
			setupMock: func(t *testing.T) func() {
				return createMockClaude(t, 0, `{"content": "Continuing the task...", "continue": true}`, "")
			},
			expectedResp: &Response{
				Content:      "Continuing the task...",
				ContinueFlag: true,
			},
		},
		{
			name: "Handle command not found error",
			command: Command{
				Type:         CommandTypePlan,
				Prompt:       "Test",
				OutputFormat: "json",
			},
			opts: CommandOptions{
				Stream:  false,
				Timeout: 30,
			},
			setupMock: func(t *testing.T) func() {
				// Remove claude from PATH to simulate command not found
				oldPath := os.Getenv("PATH")
				os.Setenv("PATH", "")
				return func() {
					os.Setenv("PATH", oldPath)
				}
			},
			expectedError: "claude command not found",
			skipOnCI:      true, // Skip on CI as it might have different PATH handling
		},
		{
			name: "Handle command timeout",
			command: Command{
				Type:         CommandTypePlan,
				Prompt:       "Long running task",
				OutputFormat: "json",
			},
			opts: CommandOptions{
				Stream:  false,
				Timeout: 1, // 1 second timeout
			},
			setupMock: func(t *testing.T) func() {
				// Create a mock that sleeps longer than timeout
				return createMockClaude(t, 0, "", "sleep 3")
			},
			expectedError: "command timed out",
		},
		{
			name: "Handle non-zero exit code",
			command: Command{
				Type:         CommandTypePlan,
				Prompt:       "Invalid request",
				OutputFormat: "json",
			},
			opts: CommandOptions{
				Stream:  false,
				Timeout: 30,
			},
			setupMock: func(t *testing.T) func() {
				return createMockClaude(t, 1, "", "Error: Invalid API key")
			},
			expectedError: "command failed with exit code 1",
		},
		{
			name: "Execute with custom working directory",
			command: Command{
				Type:         CommandTypePlan,
				Prompt:       "Test in different directory",
				OutputFormat: "json",
			},
			opts: CommandOptions{
				Stream:     false,
				Timeout:    30,
				WorkingDir: os.TempDir(),
			},
			setupMock: func(t *testing.T) func() {
				return createMockClaude(t, 0, `{"content": "Working in temp dir", "continue": false}`, "")
			},
			expectedResp: &Response{
				Content:      "Working in temp dir",
				ContinueFlag: false,
			},
		},
		{
			name: "Handle invalid working directory",
			command: Command{
				Type:         CommandTypePlan,
				Prompt:       "Test",
				OutputFormat: "json",
			},
			opts: CommandOptions{
				Stream:     false,
				Timeout:    30,
				WorkingDir: "/nonexistent/directory",
			},
			setupMock: func(t *testing.T) func() {
				return func() {} // No mock needed
			},
			expectedError: "invalid working directory",
		},
		{
			name: "Handle malformed JSON output",
			command: Command{
				Type:         CommandTypePlan,
				Prompt:       "Test malformed output",
				OutputFormat: "json",
			},
			opts: CommandOptions{
				Stream:  false,
				Timeout: 30,
			},
			setupMock: func(t *testing.T) func() {
				return createMockClaude(t, 0, "This is not JSON", "")
			},
			expectedError: "failed to parse response",
		},
		{
			name: "Handle empty output",
			command: Command{
				Type:         CommandTypePlan,
				Prompt:       "Test empty output",
				OutputFormat: "json",
			},
			opts: CommandOptions{
				Stream:  false,
				Timeout: 30,
			},
			setupMock: func(t *testing.T) func() {
				return createMockClaude(t, 0, "", "")
			},
			expectedResp: &Response{
				Content:      "",
				ContinueFlag: false,
			},
		},
		{
			name: "Context cancellation is respected",
			command: Command{
				Type:         CommandTypePlan,
				Prompt:       "Test context cancellation",
				OutputFormat: "json",
			},
			opts: CommandOptions{
				Stream:  false,
				Timeout: 30,
			},
			setupMock: func(t *testing.T) func() {
				return createMockClaude(t, 0, "", "sleep 10")
			},
			expectedError: "context canceled",
		},
		{
			name: "Handle empty prompt",
			command: Command{
				Type:         CommandTypePlan,
				Prompt:       "",
				OutputFormat: "json",
			},
			opts: CommandOptions{
				Stream:  false,
				Timeout: 30,
			},
			setupMock: func(t *testing.T) func() {
				return createMockClaude(t, 0, `{"content": "Empty prompt handled", "continue": false}`, "")
			},
			expectedError: "prompt cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnCI && os.Getenv("CI") == "true" {
				t.Skip("Skipping test on CI")
			}

			cleanup := tt.setupMock(t)
			defer cleanup()

			claude := New()
			ctx := context.Background()

			// Special handling for context cancellation test
			if tt.name == "Context cancellation is respected" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				go func() {
					time.Sleep(100 * time.Millisecond)
					cancel()
				}()
			}

			resp, err := claude.Execute(ctx, tt.command, tt.opts)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', but got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if resp == nil {
					t.Error("Expected response, but got nil")
				} else {
					if resp.Content != tt.expectedResp.Content {
						t.Errorf("Expected content '%s', but got '%s'", tt.expectedResp.Content, resp.Content)
					}
					if resp.ContinueFlag != tt.expectedResp.ContinueFlag {
						t.Errorf("Expected continue flag %v, but got %v", tt.expectedResp.ContinueFlag, resp.ContinueFlag)
					}
				}
			}
		})
	}
}

// TestExecutor_StreamingOutput tests streaming functionality
func TestExecutor_StreamingOutput(t *testing.T) {
	// This test verifies that streaming option is properly handled
	// For now, we'll just verify the option is passed correctly
	// Actual streaming implementation can be tested with integration tests

	cleanup := createMockClaude(t, 0, `{"content": "Streaming response", "continue": false}`, "")
	defer cleanup()

	claude := New()
	ctx := context.Background()

	resp, err := claude.Execute(ctx, Command{
		Type:         CommandTypePlan,
		Prompt:       "Test streaming",
		OutputFormat: "json",
	}, CommandOptions{
		Stream:  true,
		Timeout: 30,
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp == nil {
		t.Error("Expected response, but got nil")
	}
}

// createMockClaude creates a temporary executable that mimics claude behavior
func createMockClaude(t *testing.T, exitCode int, stdout, stderr string) func() {
	t.Helper()

	// Create a temporary directory for our mock
	tmpDir, err := os.MkdirTemp("", "claude-mock-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create the mock script
	var mockScript string
	var mockPath string

	if runtime.GOOS == "windows" {
		mockPath = filepath.Join(tmpDir, "claude.bat")
		mockScript = "@echo off\n"
		if stderr != "" {
			if strings.HasPrefix(stderr, "sleep ") {
				sleepTime := strings.TrimPrefix(stderr, "sleep ")
				mockScript += "timeout /t " + sleepTime + " /nobreak >nul\n"
			} else {
				mockScript += "echo " + stderr + " 1>&2\n"
			}
		}
		if stdout != "" {
			mockScript += "echo " + strings.ReplaceAll(stdout, "\"", "\"\"") + "\n"
		}
		mockScript += "exit /b " + string(rune(exitCode)) + "\n"
	} else {
		mockPath = filepath.Join(tmpDir, "claude")
		mockScript = "#!/bin/sh\n"
		if stderr != "" {
			if strings.HasPrefix(stderr, "sleep ") {
				mockScript += stderr + "\n"
			} else {
				mockScript += "echo '" + stderr + "' >&2\n"
			}
		}
		if stdout != "" {
			// Escape single quotes in stdout for shell
			escapedStdout := strings.ReplaceAll(stdout, "'", "'\"'\"'")
			mockScript += "echo '" + escapedStdout + "'\n"
		}
		mockScript += fmt.Sprintf("exit %d\n", exitCode)
	}

	err = os.WriteFile(mockPath, []byte(mockScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock script: %v", err)
	}

	// Add the temp directory to PATH
	oldPath := os.Getenv("PATH")
	newPath := tmpDir + string(os.PathListSeparator) + oldPath
	os.Setenv("PATH", newPath)

	// Return cleanup function
	return func() {
		os.Setenv("PATH", oldPath)
		os.RemoveAll(tmpDir)
	}
}

// TestExecutor_ParseResponse tests the response parsing functionality
func TestExecutor_ParseResponse(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		expectedResp *Response
		expectedErr  bool
	}{
		{
			name:   "Parse valid JSON response",
			output: `{"content": "Task completed", "continue": false}`,
			expectedResp: &Response{
				Content:      "Task completed",
				ContinueFlag: false,
			},
			expectedErr: false,
		},
		{
			name:   "Parse response with continue flag true",
			output: `{"content": "Still working...", "continue": true}`,
			expectedResp: &Response{
				Content:      "Still working...",
				ContinueFlag: true,
			},
			expectedErr: false,
		},
		{
			name:         "Parse invalid JSON",
			output:       "Not a JSON response",
			expectedResp: nil,
			expectedErr:  true,
		},
		{
			name:   "Parse empty JSON object",
			output: "{}",
			expectedResp: &Response{
				Content:      "",
				ContinueFlag: false,
			},
			expectedErr: false,
		},
		{
			name:   "Parse response with extra fields",
			output: `{"content": "Done", "continue": false, "extra": "ignored"}`,
			expectedResp: &Response{
				Content:      "Done",
				ContinueFlag: false,
			},
			expectedErr: false,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claude := New()
			resp, err := claude.ParseResponse(ctx, tt.output)

			if tt.expectedErr {
				if err == nil {
					t.Error("Expected error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if resp == nil {
					t.Error("Expected response, but got nil")
				} else {
					if resp.Content != tt.expectedResp.Content {
						t.Errorf("Expected content '%s', but got '%s'", tt.expectedResp.Content, resp.Content)
					}
					if resp.ContinueFlag != tt.expectedResp.ContinueFlag {
						t.Errorf("Expected continue flag %v, but got %v", tt.expectedResp.ContinueFlag, resp.ContinueFlag)
					}
				}
			}
		})
	}
}
