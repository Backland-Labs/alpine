package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestRunCommand(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		setupMock    func(*mockWorkflowEngine)
		wantErr      bool
		wantInOutput []string
		wantInError  []string
	}{
		{
			name: "valid Linear issue ID executes workflow",
			args: []string{"ABC-123"},
			setupMock: func(m *mockWorkflowEngine) {
				m.runFunc = func(ctx context.Context, issueID string, generatePlan bool) error {
					if issueID != "ABC-123" {
						t.Errorf("unexpected issueID: %s", issueID)
					}
					if !generatePlan {
						t.Error("expected generatePlan to be true")
					}
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "valid Linear issue ID with --no-plan flag",
			args: []string{"ABC-123", "--no-plan"},
			setupMock: func(m *mockWorkflowEngine) {
				m.runFunc = func(ctx context.Context, issueID string, generatePlan bool) error {
					if issueID != "ABC-123" {
						t.Errorf("unexpected issueID: %s", issueID)
					}
					if generatePlan {
						t.Error("expected generatePlan to be false")
					}
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "invalid Linear issue ID format",
			args: []string{"invalid-id"},
			setupMock: func(m *mockWorkflowEngine) {
				// Should not be called
				m.runFunc = func(ctx context.Context, issueID string, generatePlan bool) error {
					t.Error("workflow engine should not be called for invalid ID")
					return nil
				}
			},
			wantErr: true,
			wantInError: []string{
				"invalid Linear issue ID format",
			},
		},
		{
			name: "empty Linear issue ID",
			args: []string{""},
			setupMock: func(m *mockWorkflowEngine) {
				// Should not be called
				m.runFunc = func(ctx context.Context, issueID string, generatePlan bool) error {
					t.Error("workflow engine should not be called for empty ID")
					return nil
				}
			},
			wantErr: true,
			wantInError: []string{
				"invalid Linear issue ID format",
			},
		},
		{
			name: "workflow engine error is propagated",
			args: []string{"ABC-123"},
			setupMock: func(m *mockWorkflowEngine) {
				m.runFunc = func(ctx context.Context, issueID string, generatePlan bool) error {
					return fmt.Errorf("workflow failed: connection timeout")
				}
			},
			wantErr: true,
			wantInError: []string{
				"workflow failed: connection timeout",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock workflow engine
			mockEngine := &mockWorkflowEngine{}
			if tt.setupMock != nil {
				tt.setupMock(mockEngine)
			}

			// Create run command with mock
			cmd := NewRunCommand(mockEngine)

			// Set up output buffers
			bufOut := new(bytes.Buffer)
			bufErr := new(bytes.Buffer)
			cmd.SetOut(bufOut)
			cmd.SetErr(bufErr)
			cmd.SetArgs(tt.args)

			// Execute command
			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check output
			output := bufOut.String()
			for _, want := range tt.wantInOutput {
				if !strings.Contains(output, want) {
					t.Errorf("Execute() output missing %q\nGot: %s", want, output)
				}
			}

			// Check error output
			errorOutput := bufErr.String()
			if err != nil {
				errorOutput += err.Error()
			}
			for _, want := range tt.wantInError {
				if !strings.Contains(errorOutput, want) {
					t.Errorf("Execute() error missing %q\nGot: %s", want, errorOutput)
				}
			}
		})
	}
}

// mockWorkflowEngine is a test double for the workflow engine
type mockWorkflowEngine struct {
	runFunc func(ctx context.Context, issueID string, generatePlan bool) error
}

func (m *mockWorkflowEngine) Run(ctx context.Context, issueID string, generatePlan bool) error {
	if m.runFunc != nil {
		return m.runFunc(ctx, issueID, generatePlan)
	}
	return nil
}

func TestIsValidLinearID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"ABC-123", true},
		{"DEF-1", true},
		{"XYZ-99999", true},
		{"A-1", true},
		{"ABCDEF-123", true},
		{"", false},
		{"ABC", false},
		{"123", false},
		{"ABC-", false},
		{"-123", false},
		{"ABC-ABC", false},
		{"abc-123", false}, // Linear IDs are uppercase
		{"ABC_123", false},
		{"ABC-123-DEF", false},
		{"ABC-0", false}, // ID must be >= 1
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := isValidLinearID(tt.id); got != tt.valid {
				t.Errorf("isValidLinearID(%q) = %v, want %v", tt.id, got, tt.valid)
			}
		})
	}
}
