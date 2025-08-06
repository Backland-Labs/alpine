package logger

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger wraps zap.Logger to provide our logging interface
type ZapLogger struct {
	*zap.Logger
	sugar *zap.SugaredLogger
}

// NewZapLogger creates a new ZapLogger with the specified configuration
func NewZapLogger(level Level, development bool) (*ZapLogger, error) {
	var config zap.Config

	if development {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Set log level
	switch level {
	case DebugLevel:
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case InfoLevel:
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case ErrorLevel:
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	}

	// Build the logger
	logger, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, fmt.Errorf("failed to create zap logger: %w", err)
	}

	return &ZapLogger{
		Logger: logger,
		sugar:  logger.Sugar(),
	}, nil
}

// NewZapLoggerFromEnv creates a logger configured from environment variables
func NewZapLoggerFromEnv() (*ZapLogger, error) {
	// Check for explicit log level first
	levelStr := os.Getenv("ALPINE_LOG_LEVEL")
	if levelStr == "" {
		// Fall back to ALPINE_VERBOSITY if ALPINE_LOG_LEVEL is not set
		verbosity := os.Getenv("ALPINE_VERBOSITY")
		switch verbosity {
		case "debug":
			levelStr = "debug"
		case "verbose":
			levelStr = "info"
		default:
			levelStr = "info" // Default to info for normal verbosity
		}
	}

	level := LevelFromString(levelStr)
	development := os.Getenv("ALPINE_LOG_FORMAT") != "json"

	logger, err := NewZapLogger(level, development)
	if err != nil {
		return nil, err
	}

	// Add caller info if requested
	if os.Getenv("ALPINE_LOG_CALLER") == "true" {
		logger.Logger = logger.WithOptions(zap.AddCaller())
	}

	// Configure stack traces
	stacktraceLevel := os.Getenv("ALPINE_LOG_STACKTRACE")
	if stacktraceLevel != "" {
		var zapLevel zapcore.Level
		switch strings.ToLower(stacktraceLevel) {
		case "error":
			zapLevel = zap.ErrorLevel
		case "panic":
			zapLevel = zap.PanicLevel
		default:
			zapLevel = zap.FatalLevel
		}
		logger.Logger = logger.WithOptions(zap.AddStacktrace(zapLevel))
	}

	return logger, nil
}

// Helper methods for common fields

// WithHTTPRequest adds HTTP request context to the logger
func (l *ZapLogger) WithHTTPRequest(r *http.Request) *ZapLogger {
	fields := []zap.Field{
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("remote_addr", r.RemoteAddr),
		zap.String("user_agent", r.UserAgent()),
		zap.Int64("content_length", r.ContentLength),
	}

	// Add query parameters if present
	if r.URL.RawQuery != "" {
		fields = append(fields, zap.String("query", r.URL.RawQuery))
	}

	// Add important headers
	if contentType := r.Header.Get("Content-Type"); contentType != "" {
		fields = append(fields, zap.String("content_type", contentType))
	}

	return &ZapLogger{
		Logger: l.With(fields...),
		sugar:  l.Logger.With(fields...).Sugar(),
	}
}

// WithWorkflow adds workflow context to the logger
func (l *ZapLogger) WithWorkflow(runID, workflowID string) *ZapLogger {
	return &ZapLogger{
		Logger: l.With(
			zap.String("run_id", runID),
			zap.String("workflow_id", workflowID),
		),
		sugar: l.Logger.With(
			zap.String("run_id", runID),
			zap.String("workflow_id", workflowID),
		).Sugar(),
	}
}

// WithDuration adds a duration field to the logger
func (l *ZapLogger) WithDuration(duration time.Duration) *ZapLogger {
	return &ZapLogger{
		Logger: l.With(
			zap.Duration("duration", duration),
			zap.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
		),
		sugar: l.Logger.With(
			zap.Duration("duration", duration),
			zap.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
		).Sugar(),
	}
}

// WithError adds error context to the logger
func (l *ZapLogger) WithError(err error) *ZapLogger {
	if err == nil {
		return l
	}

	return &ZapLogger{
		Logger: l.With(
			zap.Error(err),
			zap.String("error_type", fmt.Sprintf("%T", err)),
		),
		sugar: l.Logger.With(
			zap.Error(err),
			zap.String("error_type", fmt.Sprintf("%T", err)),
		).Sugar(),
	}
}

// WithField adds a single field to the logger context
func (l *ZapLogger) WithField(key string, value interface{}) *Logger {
	newZapLogger := &ZapLogger{
		Logger: l.With(zap.Any(key, value)),
		sugar:  l.Logger.With(zap.Any(key, value)).Sugar(),
	}
	return &Logger{zap: newZapLogger}
}

// WithFields adds multiple fields to the logger context
func (l *ZapLogger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	newZapLogger := &ZapLogger{
		Logger: l.With(zapFields...),
		sugar:  l.Logger.With(zapFields...).Sugar(),
	}
	return &Logger{zap: newZapLogger}
}

// Timed creates a timed logger for measuring operation duration
func (l *ZapLogger) Timed(operation string) *TimedLogger {
	l.Logger.Debug("Operation started", zap.String("operation", operation))
	return &TimedLogger{
		logger: l,
		start:  time.Now(),
		op:     operation,
	}
}

// TimedLogger tracks the duration of an operation
type TimedLogger struct {
	logger *ZapLogger
	start  time.Time
	op     string
}

// Done logs the completion of the timed operation
func (t *TimedLogger) Done() {
	duration := time.Since(t.start)
	t.logger.Logger.Debug("Operation completed",
		zap.String("operation", t.op),
		zap.Duration("duration", duration),
		zap.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
	)
}

// DoneWithError logs the completion of the timed operation with an error
func (t *TimedLogger) DoneWithError(err error) {
	duration := time.Since(t.start)
	if err != nil {
		t.logger.Logger.Error("Operation failed",
			zap.String("operation", t.op),
			zap.Error(err),
			zap.Duration("duration", duration),
			zap.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
		)
	} else {
		t.Done()
	}
}

// Legacy interface compatibility

func (l *ZapLogger) Debug(msg string) {
	l.Logger.Debug(msg)
}

func (l *ZapLogger) Debugf(format string, args ...interface{}) {
	l.sugar.Debugf(format, args...)
}

func (l *ZapLogger) Info(msg string) {
	l.Logger.Info(msg)
}

func (l *ZapLogger) Infof(format string, args ...interface{}) {
	l.sugar.Infof(format, args...)
}

func (l *ZapLogger) Warn(msg string) {
	l.Logger.Warn(msg)
}

func (l *ZapLogger) Warnf(format string, args ...interface{}) {
	l.sugar.Warnf(format, args...)
}

func (l *ZapLogger) Error(msg string) {
	l.Logger.Error(msg)
}

func (l *ZapLogger) Errorf(format string, args ...interface{}) {
	l.sugar.Errorf(format, args...)
}

// Sync flushes any buffered log entries
func (l *ZapLogger) Sync() error {
	return l.Logger.Sync()
}
