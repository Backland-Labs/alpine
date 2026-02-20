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

func TestOutputErrorWithReason(t *testing.T) {
	resetFlags(t)
	jsonOutput = true
	out := captureStdout(t, func() {
		outputErrorWithReason("retry later", 2, "transient_failure", true)
	})
	for _, want := range []string{"\"reason_code\": \"transient_failure\"", "\"retryable\": true"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}
