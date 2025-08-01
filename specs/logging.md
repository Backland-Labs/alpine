# Logging Specification

## Overview

Alpine uses a dual-logger architecture that provides both a simple, lightweight logger and an advanced structured logging solution through Uber's Zap library. This approach ensures maximum flexibility, performance, and compatibility across different deployment scenarios.

## Philosophy

### Core Principles

1. **Progressive Enhancement**: Start with a simple logger and automatically upgrade to Zap when available
2. **Zero Configuration**: Sensible defaults that work out of the box
3. **Environment-Driven**: All configuration through environment variables
4. **Performance First**: Minimal overhead in production, rich debugging in development
5. **Structured Logging**: Support for contextual fields and structured output
6. **Backward Compatibility**: Legacy interface maintained for smooth transitions

### Design Goals

- **Development Experience**: Human-readable output with colors and clear formatting
- **Production Readiness**: JSON output for log aggregation systems
- **Debugging Support**: Caller information and stack traces when needed
- **Performance**: Sampling and buffering for high-throughput scenarios
- **Context Propagation**: Request IDs, workflow IDs, and other correlation data

## Architecture

### Logger Hierarchy

```
logger.Logger (interface)
├── Simple Logger (default fallback)
│   ├── Text output to stderr
│   ├── Basic timestamp formatting
│   └── Field support via string concatenation
└── Zap Logger (auto-enabled via environment)
    ├── Structured JSON/Console output
    ├── High-performance buffering
    ├── Advanced field handling
    └── Sampling and rotation support
```

### Initialization Flow

1. Check for Zap configuration via environment variables
2. If Zap config exists, initialize Zap logger
3. Otherwise, fall back to simple logger
4. Apply verbosity settings from config/environment

## Configuration

### Environment Variables

#### Primary Configuration

- `ALPINE_LOG_LEVEL`: Set the logging level (debug, info, error)
  - Takes precedence over `ALPINE_VERBOSITY`
  - Default: "info"

- `ALPINE_VERBOSITY`: Legacy verbosity control
  - "debug" → Debug level
  - "verbose" → Info level  
  - "normal" → Error level
  - Default: "normal"

#### Zap-Specific Configuration

- `ALPINE_LOG_FORMAT`: Output format
  - "json": Structured JSON output (production)
  - Any other value: Console output (development)
  - Default: Console output

- `ALPINE_LOG_CALLER`: Include caller information
  - "true": Add file:line to log entries
  - Default: false

- `ALPINE_LOG_STACKTRACE`: Stack trace threshold
  - "error": Stack traces for errors and above
  - "panic": Stack traces for panics only
  - "fatal": Stack traces for fatal errors only
  - Default: No stack traces

- `ALPINE_LOG_SAMPLING`: Enable log sampling
  - "true": Enable sampling for high-volume scenarios
  - Default: false

### Configuration Examples

```bash
# Development mode with maximum verbosity
export ALPINE_LOG_LEVEL=debug
export ALPINE_LOG_FORMAT=console
export ALPINE_LOG_CALLER=true
export ALPINE_LOG_STACKTRACE=error

# Production mode with JSON output
export ALPINE_LOG_LEVEL=info
export ALPINE_LOG_FORMAT=json
export ALPINE_LOG_SAMPLING=true

# Debugging specific issues
export ALPINE_LOG_LEVEL=debug
export ALPINE_LOG_CALLER=true
export ALPINE_LOG_STACKTRACE=error
```

## Usage Patterns

### Basic Logging

```go
// Simple messages
logger.Info("Server started")
logger.Error("Failed to connect to database")

// Formatted messages
logger.Infof("Processing %d items", count)
logger.Errorf("Failed to parse config: %v", err)
```

### Structured Logging

```go
// Add contextual fields
log := logger.WithField("user_id", userID)
log.Info("User logged in")

// Multiple fields
log := logger.WithFields(map[string]interface{}{
    "request_id": requestID,
    "method": "POST",
    "path": "/api/users",
})
log.Info("Request received")
```

### Context-Specific Logging

```go
// HTTP request context (Zap only)
log := zapLogger.WithHTTPRequest(r)
log.Info("Processing request")

// Workflow context (Zap only)
log := zapLogger.WithWorkflow(runID, workflowID)
log.Info("Workflow started")

// Error context (Zap only)
log := zapLogger.WithError(err)
log.Error("Operation failed")

// Duration tracking (Zap only)
timer := zapLogger.Timed("database_query")
// ... perform operation ...
timer.Done() // Logs with duration
```

### HTTP Middleware

```go
// Standard HTTP logging
handler := logger.HTTPMiddleware(log)(yourHandler)

// Server-Sent Events logging
sseHandler := logger.SSEMiddleware(log)(yourSSEHandler)
```

## Log Levels

### Level Hierarchy

1. **Debug**: Detailed information for debugging
   - Function entry/exit
   - Variable values
   - Detailed request/response data

2. **Info**: General informational messages
   - Service startup/shutdown
   - Request completion
   - State transitions

3. **Warn**: Warning conditions
   - Deprecated API usage
   - Slow operations
   - Recoverable errors

4. **Error**: Error conditions
   - Failed operations
   - Unhandled exceptions
   - Service degradation

## Performance Considerations

### Zap Optimizations

1. **Zero-allocation logging**: Zap uses a zero-allocation JSON encoder
2. **Buffered writing**: Reduces syscall overhead
3. **Sampling**: Prevents log flooding in high-throughput scenarios
4. **Lazy evaluation**: Fields are only serialized when needed

### Best Practices

1. **Use structured fields** instead of string formatting when possible
2. **Avoid logging in hot paths** unless using debug level
3. **Enable sampling** in production for high-volume services
4. **Use appropriate log levels** to control verbosity
5. **Include correlation IDs** for distributed tracing

## Integration Points

### Claude Executor

The Claude executor uses contextual logging to track command execution:

```go
log := logger.WithFields(map[string]interface{}{
    "command": "claude",
    "working_dir": workDir,
})
log.Debug("Executing Claude command")
```

### Workflow Engine

Workflows maintain execution context through logging:

```go
log := logger.WithWorkflow(runID, workflowID)
log.Info("Starting workflow execution")
```

### HTTP Server

All HTTP endpoints use middleware for consistent request logging:

```go
router.Use(logger.HTTPMiddleware(log))
```

## Testing

### Test Logger

Use `NewTestLogger()` for tests with debug output:

```go
func TestSomething(t *testing.T) {
    log := logger.NewTestLogger()
    // Test code with full debug logging
}
```

### Log Verification

For testing log output:

```go
var buf bytes.Buffer
log := logger.New(logger.DebugLevel)
log.SetOutput(&buf)
// ... code that logs ...
assert.Contains(t, buf.String(), "expected message")
```

## Migration Guide

### From Simple to Zap

1. Existing `logger.Info()` calls work unchanged
2. To use Zap features, set environment variables:
   ```bash
   export ALPINE_LOG_FORMAT=json
   export ALPINE_LOG_LEVEL=debug
   ```
3. Access Zap-specific features through type assertion:
   ```go
   if zapLogger, ok := log.(*logger.ZapLogger); ok {
       zapLogger.WithHTTPRequest(r).Info("Request")
   }
   ```

### Adding Context

Transform simple logs to structured logs:

```go
// Before
logger.Infof("User %s logged in from %s", userID, ipAddr)

// After
logger.WithFields(map[string]interface{}{
    "user_id": userID,
    "ip_address": ipAddr,
    "event": "login",
}).Info("User logged in")
```

## Future Enhancements

### Planned Features

1. **Log Rotation**: Built-in file rotation support
2. **Remote Logging**: Direct integration with log aggregation services
3. **Metrics Integration**: Automatic metric extraction from logs
4. **Dynamic Level Changes**: Runtime log level adjustment
5. **Custom Encoders**: Plugin support for custom output formats

### Compatibility Commitments

- The basic `logger.Logger` interface will remain stable
- Environment variable names will not change
- New features will be opt-in via environment variables
- Legacy behavior will be preserved by default