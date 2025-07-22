// Package output provides colored terminal output functionality for River.
//
// The package offers a simple API for printing colored messages to the terminal
// with automatic color detection and graceful fallback for non-terminal environments.
//
// Features:
//   - Automatic terminal detection
//   - NO_COLOR environment variable support
//   - Different message types (success, error, warning, info, step, detail)
//   - Test-friendly with custom writers
//
// Example usage:
//
//	printer := output.NewPrinter()
//	printer.Success("Operation completed")
//	printer.Error("Failed to process: %v", err)
//	printer.Info("Processing %d items", count)
package output