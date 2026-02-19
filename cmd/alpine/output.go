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

// outputError writes command errors. In JSON mode it writes one JSON object to
// stdout. In text mode it writes a human-readable message to stderr.
func outputError(msg string, exitCode int) {
	if jsonOutput {
		_ = outputJSON(map[string]interface{}{
			"error":     msg,
			"exit_code": exitCode,
		})
		return
	}
	fmt.Fprintln(os.Stderr, "Error: "+msg)
}

func outputCommandError(err *commandError) {
	if jsonOutput {
		payload := map[string]interface{}{
			"error_code": err.errorCode,
			"cause":      err.cause,
			"next_step":  err.nextStep,
			"retryable":  err.retryable,
			"exit_code":  err.exitCode,
		}
		if err.operationID != "" {
			payload["operation_id"] = err.operationID
		}
		_ = outputJSON(payload)
		return
	}
	fmt.Fprintln(os.Stderr, "Error: "+err.cause)
	if err.nextStep != "" {
		fmt.Fprintln(os.Stderr, "Next step: "+err.nextStep)
	}
}
