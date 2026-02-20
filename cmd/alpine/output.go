package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// outputJSON marshals v to JSON and writes it to stdout. Used by all commands
// when the --json flag is set.
func outputJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// outputError writes a structured error to stderr. When --json is set, it
// outputs a JSON object with "error" and "exit_code" fields. Otherwise it
// prints a plain text error message.
func outputError(msg string, exitCode int) {
	outputErrorWithReason(msg, exitCode, "", false)
}

func outputErrorWithReason(msg string, exitCode int, reasonCode string, retryable bool) {
	if jsonOutput {
		payload := map[string]interface{}{
			"error":     msg,
			"exit_code": exitCode,
		}
		if reasonCode != "" {
			payload["reason_code"] = reasonCode
		}
		if retryable {
			payload["retryable"] = true
		}
		_ = outputJSON(payload)
		return
	}
	fmt.Fprintln(os.Stderr, "Error: "+msg)
}
