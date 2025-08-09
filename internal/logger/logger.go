package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Level represents the logging level
type Level int

const (
	// DebugLevel logs everything
	DebugLevel Level = iota
	// InfoLevel logs info, warnings, and errors
	InfoLevel
	// ErrorLevel logs only errors
	ErrorLevel
)

// Logger provides structured logging with timestamps
type Logger struct {
	level  Level
	output io.Writer
	fields map[string]interface{}
	mu     sync.Mutex
	zap    *ZapLogger // New Zap backend
}

var (
	globalLogger *Logger
	globalMu     sync.Mutex
)

func init() {
	// Try to initialize Zap logger from environment
	if zapLogger, err := NewZapLoggerFromEnv(); err == nil {
		globalLogger = &Logger{zap: zapLogger}
	} else {
		// Fall back to legacy logger
		globalLogger = New(InfoLevel)
	}
}

// New creates a new logger with the specified level
func New(level Level) *Logger {
	return &Logger{
		level:  level,
		output: os.Stderr,
		fields: make(map[string]interface{}),
	}
}

// SetOutput sets the output writer for the logger
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// WithField adds a single field to the logger context
func (l *Logger) WithField(key string, value interface{}) *Logger {
	if l.zap != nil {
		return l.zap.WithField(key, value)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		level:  l.level,
		output: l.output,
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	// Add new field
	newLogger.fields[key] = value

	return newLogger
}

// WithFields adds multiple fields to the logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	if l.zap != nil {
		return l.zap.WithFields(fields)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		level:  l.level,
		output: l.output,
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// log is the internal logging function
func (l *Logger) log(level Level, levelStr string, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Format timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// Format message
	message := fmt.Sprintf(format, args...)

	// Build log line
	logLine := fmt.Sprintf("%s %s %s", timestamp, levelStr, message)

	// Add fields if any
	if len(l.fields) > 0 {
		var fieldParts []string
		for k, v := range l.fields {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", k, v))
		}
		logLine += " " + strings.Join(fieldParts, " ")
	}

	// Write to output
	_, _ = fmt.Fprintln(l.output, logLine)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	if l.zap != nil {
		l.zap.Debug(msg)
	} else {
		l.log(DebugLevel, "[DEBUG]", "%s", msg)
	}
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.zap != nil {
		l.zap.Debugf(format, args...)
	} else {
		l.log(DebugLevel, "[DEBUG]", format, args...)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	if l.zap != nil {
		l.zap.Info(msg)
	} else {
		l.log(InfoLevel, "[INFO]", "%s", msg)
	}
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	if l.zap != nil {
		l.zap.Infof(format, args...)
	} else {
		l.log(InfoLevel, "[INFO]", format, args...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	if l.zap != nil {
		l.zap.Warn(msg)
	} else {
		l.log(InfoLevel, "[WARN]", "%s", msg)
	}
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	if l.zap != nil {
		l.zap.Warnf(format, args...)
	} else {
		l.log(InfoLevel, "[WARN]", format, args...)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	if l.zap != nil {
		l.zap.Error(msg)
	} else {
		l.log(ErrorLevel, "[ERROR]", "%s", msg)
	}
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	if l.zap != nil {
		l.zap.Errorf(format, args...)
	} else {
		l.log(ErrorLevel, "[ERROR]", format, args...)
	}
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	globalMu.Lock()
	defer globalMu.Unlock()
	return globalLogger
}

// SetLogger sets the global logger instance
func SetLogger(logger *Logger) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = logger
}

// LevelFromString converts a string to a log level
func LevelFromString(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return DebugLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// NewTestLogger creates a logger suitable for testing with debug level
func NewTestLogger() *Logger {
	return New(DebugLevel)
}
