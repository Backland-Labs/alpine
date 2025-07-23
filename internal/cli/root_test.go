package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantErr        bool
		wantInOutput   []string
		wantExactMatch string
	}{
		{
			name:    "help flag shows usage",
			args:    []string{"--help"},
			wantErr: false,
			wantInOutput: []string{
				"River",
				"CLI orchestrator for Claude Code",
				"Usage:",
				"river <task-description>",
				"Flags:",
				"--help",
				"--version",
			},
		},
		{
			name:    "short help flag shows usage",
			args:    []string{"-h"},
			wantErr: false,
			wantInOutput: []string{
				"River",
				"CLI orchestrator for Claude Code",
			},
		},
		{
			name:           "version flag shows version",
			args:           []string{"--version"},
			wantErr:        false,
			wantExactMatch: "river version 0.2.0\n",
		},
		{
			name:           "short version flag shows version",
			args:           []string{"-v"},
			wantErr:        false,
			wantExactMatch: "river version 0.2.0\n",
		},
		{
			name:    "no arguments shows help",
			args:    []string{},
			wantErr: true,
			wantInOutput: []string{
				"Error: requires a task description (use quotes for multi-word descriptions)",
				"Usage:",
			},
		},
		{
			name:    "invalid flag shows error",
			args:    []string{"--invalid"},
			wantErr: true,
			wantInOutput: []string{
				"unknown flag: --invalid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := NewRootCommand()
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			output := buf.String()

			if tt.wantExactMatch != "" {
				if output != tt.wantExactMatch {
					t.Errorf("Execute() output = %q, want %q", output, tt.wantExactMatch)
				}
			} else {
				for _, want := range tt.wantInOutput {
					if !strings.Contains(output, want) {
						t.Errorf("Execute() output missing %q\nGot: %s", want, output)
					}
				}
			}
		})
	}
}

func TestExecute(t *testing.T) {
	// Test that Execute function exists and returns an error type
	// This is mainly a compilation test
	err := Execute()
	_ = err // Use the variable to avoid compiler complaints
}

// TestCLIWorktreeFlags tests that the --no-worktree flag is parsed correctly
func TestCLIWorktreeFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantErr        bool
		wantInOutput   []string
		checkWorktree  func(t *testing.T, cmd *cobra.Command)
	}{
		{
			name:    "--no-worktree flag in help",
			args:    []string{"--help"},
			wantErr: false,
			wantInOutput: []string{
				"--no-worktree",
				"Disable git worktree creation",
			},
		},
		{
			name:    "valid task with --no-worktree flag",
			args:    []string{"Implement feature", "--no-worktree"},
			wantErr: false,
			checkWorktree: func(t *testing.T, cmd *cobra.Command) {
				// Check that the flag was parsed correctly
				noWorktree, err := cmd.Flags().GetBool("no-worktree")
				if err != nil {
					t.Errorf("Failed to get no-worktree flag: %v", err)
				}
				if !noWorktree {
					t.Errorf("Expected no-worktree flag to be true, got false")
				}
			},
		},
		{
			name:    "valid task without --no-worktree flag",
			args:    []string{"Implement feature"},
			wantErr: false,
			checkWorktree: func(t *testing.T, cmd *cobra.Command) {
				// Check that the flag defaults to false
				noWorktree, err := cmd.Flags().GetBool("no-worktree")
				if err != nil {
					t.Errorf("Failed to get no-worktree flag: %v", err)
				}
				if noWorktree {
					t.Errorf("Expected no-worktree flag to be false, got true")
				}
			},
		},
		{
			name:    "--no-worktree with --no-plan flag",
			args:    []string{"Fix bug", "--no-plan", "--no-worktree"},
			wantErr: false,
			checkWorktree: func(t *testing.T, cmd *cobra.Command) {
				// Check both flags are parsed correctly
				noWorktree, err := cmd.Flags().GetBool("no-worktree")
				if err != nil {
					t.Errorf("Failed to get no-worktree flag: %v", err)
				}
				if !noWorktree {
					t.Errorf("Expected no-worktree flag to be true, got false")
				}
				
				noPlan, err := cmd.Flags().GetBool("no-plan")
				if err != nil {
					t.Errorf("Failed to get no-plan flag: %v", err)
				}
				if !noPlan {
					t.Errorf("Expected no-plan flag to be true, got false")
				}
			},
		},
		{
			name:    "--no-worktree with file input",
			args:    []string{"--file", "task.md", "--no-worktree"},
			wantErr: false,
			checkWorktree: func(t *testing.T, cmd *cobra.Command) {
				// Check flag is parsed with file input
				noWorktree, err := cmd.Flags().GetBool("no-worktree")
				if err != nil {
					t.Errorf("Failed to get no-worktree flag: %v", err)
				}
				if !noWorktree {
					t.Errorf("Expected no-worktree flag to be true, got false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := NewRootCommand()
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs(tt.args)

			// For test purposes, override the RunE to avoid actual execution
			if tt.checkWorktree != nil && !contains(tt.args, "--help") {
				rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
					tt.checkWorktree(t, cmd)
					return nil
				}
			}

			err := rootCmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			output := buf.String()
			for _, want := range tt.wantInOutput {
				if !strings.Contains(output, want) {
					t.Errorf("Execute() output missing %q\nGot: %s", want, output)
				}
			}
		})
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestRootCmd_BareMode_AcceptsNoArgs tests that bare execution mode accepts no arguments
// when both --no-plan and --no-worktree flags are set
func TestRootCmd_BareMode_AcceptsNoArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		wantMsg string
	}{
		{
			name:    "bare mode with both flags accepts no args",
			args:    []string{"--no-plan", "--no-worktree"},
			wantErr: false,
			wantMsg: "",
		},
		{
			name:    "bare mode flags order reversed also works",
			args:    []string{"--no-worktree", "--no-plan"},
			wantErr: false,
			wantMsg: "",
		},
		{
			name:    "bare mode with task description still works",
			args:    []string{"--no-plan", "--no-worktree", "Some task"},
			wantErr: false,
			wantMsg: "",
		},
		{
			name:    "bare mode with file flag works",
			args:    []string{"--no-plan", "--no-worktree", "--file", "task.md"},
			wantErr: false,
			wantMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := NewRootCommand()
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs(tt.args)

			// Override RunE to avoid actual workflow execution
			rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
				return nil
			}

			err := rootCmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantMsg != "" && !strings.Contains(buf.String(), tt.wantMsg) {
				t.Errorf("Expected output to contain %q, got %q", tt.wantMsg, buf.String())
			}
		})
	}
}

// TestRootCmd_RequiresArgs_WithSingleFlag tests that single flags still require arguments
func TestRootCmd_RequiresArgs_WithSingleFlag(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		wantMsg string
	}{
		{
			name:    "only --no-plan flag requires args",
			args:    []string{"--no-plan"},
			wantErr: true,
			wantMsg: "requires a task description",
		},
		{
			name:    "only --no-worktree flag requires args",
			args:    []string{"--no-worktree"},
			wantErr: true,
			wantMsg: "requires a task description",
		},
		{
			name:    "no flags requires args",
			args:    []string{},
			wantErr: true,
			wantMsg: "requires a task description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := NewRootCommand()
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			output := buf.String()
			if !strings.Contains(output, tt.wantMsg) {
				t.Errorf("Expected output to contain %q, got %q", tt.wantMsg, output)
			}
		})
	}
}
