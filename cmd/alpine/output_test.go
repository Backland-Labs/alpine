package main

import (
	"strings"
	"testing"
)

func TestOutputJSON(t *testing.T) {
	resetFlags(t)
	out := captureStdout(t, func() {
		err := outputJSON(map[string]string{"key": "val"})
		if err != nil {
			t.Fatalf("outputJSON failed: %v", err)
		}
	})
	if !strings.Contains(out, `"key": "val"`) {
		t.Fatalf("expected JSON output, got: %s", out)
	}
}

func TestOutputError(t *testing.T) {
	tests := []struct {
		name     string
		json     bool
		wantJSON bool
	}{
		{name: "json mode", json: true, wantJSON: true},
		{name: "text mode", json: false, wantJSON: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags(t)
			jsonOutput = tt.json

			if tt.wantJSON {
				out := captureStdout(t, func() {
					outputError("test error", 1)
				})
				if !strings.Contains(out, `"error"`) {
					t.Fatalf("expected JSON error output, got: %s", out)
				}
				if !strings.Contains(out, `"exit_code"`) {
					t.Fatalf("expected exit_code in output, got: %s", out)
				}
			} else {
				// Text mode writes to stderr, not stdout.
				out := captureStdout(t, func() {
					outputError("test error", 1)
				})
				if strings.Contains(out, "error") {
					t.Fatalf("text mode should not write to stdout, got: %s", out)
				}
			}
		})
	}
}

func TestOutputCommandErrorJSON(t *testing.T) {
	resetFlags(t)
	jsonOutput = true

	out := captureStdout(t, func() {
		outputCommandError(&commandError{
			exitCode:    2,
			errorCode:   ErrSandboxProvisionFailed,
			cause:       "provision failed",
			nextStep:    "retry",
			retryable:   true,
			operationID: "op-1",
		})
	})

	if !strings.Contains(out, ErrSandboxProvisionFailed) {
		t.Fatalf("expected error_code in output, got: %s", out)
	}
	if !strings.Contains(out, "operation_id") {
		t.Fatalf("expected operation_id in output, got: %s", out)
	}
}

func TestOutputCommandErrorText(t *testing.T) {
	resetFlags(t)
	jsonOutput = false

	stdout, stderr := captureOutputs(t, func() {
		outputCommandError(&commandError{
			exitCode:  1,
			cause:     "boom",
			nextStep:  "retry",
			retryable: false,
		})
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: boom") {
		t.Fatalf("expected error text, got %q", stderr)
	}
	if !strings.Contains(stderr, "Next step: retry") {
		t.Fatalf("expected next step text, got %q", stderr)
	}
}
