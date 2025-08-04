package logger

import (
	"github.com/Backland-Labs/alpine/internal/config"
)

// InitializeFromConfig sets up the global logger based on the configuration
func InitializeFromConfig(cfg *config.Config) {
	var level Level

	switch cfg.Verbosity {
	case "debug":
		level = DebugLevel
	case "verbose":
		level = InfoLevel
	default:
		level = ErrorLevel
	}

	// Check if we should use Zap logger based on environment
	if zapLogger, err := NewZapLoggerFromEnv(); err == nil {
		// Successfully created Zap logger from environment
		logger := &Logger{zap: zapLogger}
		SetLogger(logger)
	} else {
		// Fall back to legacy logger with configured level
		logger := New(level)
		SetLogger(logger)
	}
}

// Debug is a convenience function that logs to the global logger
func Debug(msg string) {
	GetLogger().Debug(msg)
}

// Debugf is a convenience function that logs to the global logger
func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

// Info is a convenience function that logs to the global logger
func Info(msg string) {
	GetLogger().Info(msg)
}

// Infof is a convenience function that logs to the global logger
func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

// Warn is a convenience function that logs to the global logger
func Warn(msg string) {
	GetLogger().Warn(msg)
}

// Warnf is a convenience function that logs to the global logger
func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

// Error is a convenience function that logs to the global logger
func Error(msg string) {
	GetLogger().Error(msg)
}

// Errorf is a convenience function that logs to the global logger
func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

// WithField is a convenience function that returns a logger with a field
func WithField(key string, value interface{}) *Logger {
	return GetLogger().WithField(key, value)
}

// WithFields is a convenience function that returns a logger with fields
func WithFields(fields map[string]interface{}) *Logger {
	return GetLogger().WithFields(fields)
}
