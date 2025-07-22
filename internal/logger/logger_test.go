package logger

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// TestNewLogger tests the creation of a new logger instance
func TestNewLogger(t *testing.T) {
	tests := []struct {
		name      string
		level     Level
		wantLevel Level
	}{
		{
			name:      "create debug logger",
			level:     DebugLevel,
			wantLevel: DebugLevel,
		},
		{
			name:      "create info logger",
			level:     InfoLevel,
			wantLevel: InfoLevel,
		},
		{
			name:      "create error logger",
			level:     ErrorLevel,
			wantLevel: ErrorLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.level)
			if logger.level != tt.wantLevel {
				t.Errorf("New() level = %v, want %v", logger.level, tt.wantLevel)
			}
		})
	}
}

// TestLoggerDebug tests debug level logging with timestamps
func TestLoggerDebug(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  DebugLevel,
		output: &buf,
	}

	// Test that debug messages are logged at debug level
	logger.Debug("test debug message")
	output := buf.String()

	// Check timestamp format (YYYY-MM-DD HH:MM:SS)
	if !strings.Contains(output, time.Now().Format("2006-01-02")) {
		t.Error("Debug log should contain date in YYYY-MM-DD format")
	}

	// Check log level indicator
	if !strings.Contains(output, "[DEBUG]") {
		t.Error("Debug log should contain [DEBUG] level indicator")
	}

	// Check message
	if !strings.Contains(output, "test debug message") {
		t.Error("Debug log should contain the message")
	}

	// Test that debug messages are not logged at info level
	buf.Reset()
	logger.level = InfoLevel
	logger.Debug("should not appear")
	if buf.Len() > 0 {
		t.Error("Debug messages should not be logged when level is Info")
	}
}

// TestLoggerInfo tests info level logging
func TestLoggerInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  InfoLevel,
		output: &buf,
	}

	logger.Info("test info message")
	output := buf.String()

	// Check timestamp
	if !strings.Contains(output, time.Now().Format("2006-01-02")) {
		t.Error("Info log should contain timestamp")
	}

	// Check level
	if !strings.Contains(output, "[INFO]") {
		t.Error("Info log should contain [INFO] level indicator")
	}

	// Check message
	if !strings.Contains(output, "test info message") {
		t.Error("Info log should contain the message")
	}

	// Test that info messages are not logged at error level
	buf.Reset()
	logger.level = ErrorLevel
	logger.Info("should not appear")
	if buf.Len() > 0 {
		t.Error("Info messages should not be logged when level is Error")
	}
}

// TestLoggerError tests error level logging
func TestLoggerError(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  ErrorLevel,
		output: &buf,
	}

	logger.Error("test error message")
	output := buf.String()

	// Check timestamp
	if !strings.Contains(output, time.Now().Format("2006-01-02")) {
		t.Error("Error log should contain timestamp")
	}

	// Check level
	if !strings.Contains(output, "[ERROR]") {
		t.Error("Error log should contain [ERROR] level indicator")
	}

	// Check message
	if !strings.Contains(output, "test error message") {
		t.Error("Error log should contain the message")
	}
}

// TestLoggerFormatting tests printf-style formatting
func TestLoggerFormatting(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  DebugLevel,
		output: &buf,
	}

	logger.Debugf("formatted %s with number %d", "string", 42)
	output := buf.String()

	if !strings.Contains(output, "formatted string with number 42") {
		t.Error("Debugf should support printf-style formatting")
	}

	buf.Reset()
	logger.Infof("info %v", true)
	output = buf.String()

	if !strings.Contains(output, "info true") {
		t.Error("Infof should support printf-style formatting")
	}

	buf.Reset()
	logger.Errorf("error %x", 255)
	output = buf.String()

	if !strings.Contains(output, "error ff") {
		t.Error("Errorf should support printf-style formatting")
	}
}

// TestLoggerWithContext tests logging with contextual information
func TestLoggerWithContext(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:  DebugLevel,
		output: &buf,
	}

	// Test WithField
	contextLogger := logger.WithField("issue_id", "LINEAR-123")
	contextLogger.Info("processing issue")
	output := buf.String()

	if !strings.Contains(output, "issue_id=LINEAR-123") {
		t.Error("WithField should add field to log output")
	}

	// Test WithFields
	buf.Reset()
	multiFieldLogger := logger.WithFields(map[string]interface{}{
		"user":   "john",
		"action": "create",
		"count":  5,
	})
	multiFieldLogger.Debug("multiple fields test")
	output = buf.String()

	if !strings.Contains(output, "user=john") {
		t.Error("WithFields should add user field")
	}
	if !strings.Contains(output, "action=create") {
		t.Error("WithFields should add action field")
	}
	if !strings.Contains(output, "count=5") {
		t.Error("WithFields should add count field")
	}
}

// TestGlobalLogger tests the global logger instance
func TestGlobalLogger(t *testing.T) {
	// Test default logger
	if GetLogger() == nil {
		t.Error("GetLogger should return a non-nil logger")
	}

	// Test setting custom logger
	var buf bytes.Buffer
	customLogger := &Logger{
		level:  DebugLevel,
		output: &buf,
	}

	SetLogger(customLogger)
	GetLogger().Debug("global logger test")

	if !strings.Contains(buf.String(), "global logger test") {
		t.Error("SetLogger should update the global logger")
	}
}

// TestLogLevelFromString tests parsing log levels from strings
func TestLogLevelFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", DebugLevel},
		{"DEBUG", DebugLevel},
		{"info", InfoLevel},
		{"INFO", InfoLevel},
		{"error", ErrorLevel},
		{"ERROR", ErrorLevel},
		{"invalid", InfoLevel}, // default
		{"", InfoLevel},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := LevelFromString(tt.input)
			if level != tt.expected {
				t.Errorf("LevelFromString(%q) = %v, want %v", tt.input, level, tt.expected)
			}
		})
	}
}