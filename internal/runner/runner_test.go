package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/maxkrieger/river/internal/claude"
)

// MockClaude implements the claude.Claude interface for testing
type MockClaude struct {
	ExecuteFunc func(ctx context.Context, cmd claude.Command, opts claude.CommandOptions) (*claude.Response, error)
	calls       []struct {
		cmd  claude.Command
		opts claude.CommandOptions
	}
}

func (m *MockClaude) BuildCommand(ctx context.Context, cmd claude.Command) ([]string, error) {
	// Not used in our tests, but needed to implement the interface
	return []string{"claude"}, nil
}

func (m *MockClaude) Execute(ctx context.Context, cmd claude.Command, opts claude.CommandOptions) (*claude.Response, error) {
	m.calls = append(m.calls, struct {
		cmd  claude.Command
		opts claude.CommandOptions
	}{cmd, opts})

	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, cmd, opts)
	}
	return &claude.Response{}, nil
}

func (m *MockClaude) ParseResponse(ctx context.Context, output string) (*claude.Response, error) {
	// Not used in our tests, but needed to implement the interface
	return &claude.Response{}, nil
}

func TestRunnerWithClaude(t *testing.T) {
	// Test that runner executes Claude commands directly without shell script
	tests := []struct {
		name         string
		issueID      string
		workingDir   string
		mockResponse *claude.Response
		mockError    error
		wantError    bool
		wantCalls    int
	}{
		{
			name:       "successful execution",
			issueID:    "LINEAR-123",
			workingDir: "/tmp/test-worktree",
			mockResponse: &claude.Response{
				Content:      "Task completed successfully",
				ContinueFlag: false,
			},
			wantError: false,
			wantCalls: 1,
		},
		{
			name:       "execution with error",
			issueID:    "LINEAR-456",
			workingDir: "/tmp/test-worktree",
			mockError:  errors.New("claude execution failed"),
			wantError:  true,
			wantCalls:  1,
		},
		{
			name:       "empty issue ID",
			issueID:    "",
			workingDir: "/tmp/test-worktree",
			wantError:  true,
			wantCalls:  0,
		},
		{
			name:       "empty working directory",
			issueID:    "LINEAR-789",
			workingDir: "",
			wantError:  true,
			wantCalls:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock Claude executor
			mock := &MockClaude{
				ExecuteFunc: func(ctx context.Context, cmd claude.Command, opts claude.CommandOptions) (*claude.Response, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			// Create runner with mock
			r := NewRunner(mock)

			// Execute the runner
			err := r.Run(context.Background(), tt.issueID, tt.workingDir)

			// Check error expectation
			if (err != nil) != tt.wantError {
				t.Errorf("Run() error = %v, wantError %v", err, tt.wantError)
			}

			// Check number of Claude calls
			if len(mock.calls) != tt.wantCalls {
				t.Errorf("Expected %d Claude calls, got %d", tt.wantCalls, len(mock.calls))
			}

			// Verify the command was properly configured if calls were made
			if len(mock.calls) > 0 {
				call := mock.calls[0]
				if call.opts.WorkingDir != tt.workingDir {
					t.Errorf("Expected working dir %s, got %s", tt.workingDir, call.opts.WorkingDir)
				}
			}
		})
	}
}

func TestRunnerErrorPropagation(t *testing.T) {
	// Test that errors are properly wrapped and returned
	tests := []struct {
		name          string
		claudeError   error
		expectedInErr string
	}{
		{
			name:          "API error",
			claudeError:   errors.New("Linear API error: rate limit exceeded"),
			expectedInErr: "Linear API error",
		},
		{
			name:          "network error",
			claudeError:   errors.New("connection refused"),
			expectedInErr: "connection refused",
		},
		{
			name:          "timeout error",
			claudeError:   errors.New("context deadline exceeded"),
			expectedInErr: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock that returns specific error
			mock := &MockClaude{
				ExecuteFunc: func(ctx context.Context, cmd claude.Command, opts claude.CommandOptions) (*claude.Response, error) {
					return nil, tt.claudeError
				},
			}

			r := NewRunner(mock)
			err := r.Run(context.Background(), "LINEAR-123", "/tmp/test")

			// Check that error is returned
			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			// Check that error contains expected text
			if !errors.Is(err, tt.claudeError) && err.Error() != tt.claudeError.Error() {
				// Check if it's wrapped
				if !contains(err.Error(), tt.expectedInErr) {
					t.Errorf("Error should contain '%s', got: %v", tt.expectedInErr, err)
				}
			}
		})
	}
}

func TestRunnerCommandConfiguration(t *testing.T) {
	// Test that the runner properly configures Claude commands
	var capturedCommand claude.Command
	var capturedOptions claude.CommandOptions

	mock := &MockClaude{
		ExecuteFunc: func(ctx context.Context, cmd claude.Command, opts claude.CommandOptions) (*claude.Response, error) {
			capturedCommand = cmd
			capturedOptions = opts
			return &claude.Response{ContinueFlag: false}, nil
		},
	}

	r := NewRunner(mock)
	err := r.Run(context.Background(), "LINEAR-999", "/work/dir")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify command configuration
	if capturedCommand.Type != claude.CommandTypePlan {
		t.Errorf("Expected command type %s, got %s", claude.CommandTypePlan, capturedCommand.Type)
	}

	if capturedCommand.OutputFormat != "json" {
		t.Errorf("Expected output format 'json', got %s", capturedCommand.OutputFormat)
	}

	if !contains(capturedCommand.Content, "LINEAR-999") {
		t.Errorf("Command content should contain issue ID, got: %s", capturedCommand.Content)
	}

	if capturedOptions.WorkingDir != "/work/dir" {
		t.Errorf("Expected working dir '/work/dir', got %s", capturedOptions.WorkingDir)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || len(s) > len(substr) && containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 1; i < len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
