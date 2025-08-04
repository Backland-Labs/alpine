package logger

import (
	"os"
	"strings"
)

// Config holds logger configuration
type Config struct {
	Level      Level
	Format     string // "console" or "json"
	Caller     bool   // Include caller information
	Stacktrace string // Level at which to include stack traces
	Sampling   bool   // Enable sampling for high-frequency logs
}

// ConfigFromEnv creates a logger configuration from environment variables
func ConfigFromEnv() *Config {
	cfg := &Config{
		Level:      InfoLevel,
		Format:     "console",
		Caller:     false,
		Stacktrace: "panic",
		Sampling:   false,
	}

	// Parse log level
	if levelStr := os.Getenv("ALPINE_LOG_LEVEL"); levelStr != "" {
		cfg.Level = LevelFromString(levelStr)
	}

	// Parse format
	if format := os.Getenv("ALPINE_LOG_FORMAT"); format != "" {
		cfg.Format = strings.ToLower(format)
	}

	// Parse caller flag
	cfg.Caller = os.Getenv("ALPINE_LOG_CALLER") == "true"

	// Parse stacktrace level
	if stacktrace := os.Getenv("ALPINE_LOG_STACKTRACE"); stacktrace != "" {
		cfg.Stacktrace = strings.ToLower(stacktrace)
	}

	// Parse sampling flag
	cfg.Sampling = os.Getenv("ALPINE_LOG_SAMPLING") == "true"

	return cfg
}

// IsDevelopment returns true if the logger is configured for development mode
func (c *Config) IsDevelopment() bool {
	return c.Format == "console"
}
