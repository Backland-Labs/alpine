package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// loadDotEnv reads a .env file and sets variables in the process environment.
// Variables already set in the environment are NOT overwritten, so explicit
// exports always take precedence. This ensures compose passthrough variables
// (e.g., CLAUDE_CODE_OAUTH_TOKEN) pick up values from .env files.
func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip optional "export " prefix.
		line = strings.TrimPrefix(line, "export ")

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		// Remove surrounding quotes.
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		// Only set if not already in environment.
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("failed to set env var %s: %w", key, err)
			}
			slog.Debug("loaded env var from .env", "key", key)
		}
	}
	return nil
}
