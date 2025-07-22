package cli

import (
	"bytes"
	"strings"
	"testing"
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
