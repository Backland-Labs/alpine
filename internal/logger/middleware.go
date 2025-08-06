package logger

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// responseWriter wraps http.ResponseWriter to capture status code and size
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}

// Hijack implements the http.Hijacker interface
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, errors.New("responseWriter doesn't support hijacking")
}

// Flush implements the http.Flusher interface
func (w *responseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// HTTPMiddleware creates a logging middleware for HTTP requests
func HTTPMiddleware(logger *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a logger with request context
			reqLogger := logger
			if logger.zap != nil {
				reqLogger = &Logger{zap: logger.zap.WithHTTPRequest(r)}
			} else {
				reqLogger = logger.WithFields(map[string]interface{}{
					"method":      r.Method,
					"path":        r.URL.Path,
					"remote_addr": r.RemoteAddr,
				})
			}

			// Log request received at debug level
			reqLogger.Debug("Request received")

			// Log detailed request info at debug level
			if logger.zap != nil && logger.zap.Logger.Core().Enabled(zap.DebugLevel) {
				reqLogger.Debugf("Request details - Headers: %v, ContentLength: %d",
					r.Header, r.ContentLength)
			}

			// Wrap response writer
			wrapped := &responseWriter{
				ResponseWriter: w,
				status:         200, // default status
			}

			// Process request
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start)

			// Add response fields
			if logger.zap != nil {
				zapLogger := reqLogger.zap.WithDuration(duration).With(
					zap.Int("status", wrapped.status),
					zap.Int("size", wrapped.size),
				)
				respLogger := &Logger{
					zap: &ZapLogger{
						Logger: zapLogger,
						sugar:  zapLogger.Sugar(),
					},
				}
				reqLogger = respLogger
			} else {
				reqLogger = reqLogger.WithFields(map[string]interface{}{
					"status":      wrapped.status,
					"size":        wrapped.size,
					"duration_ms": float64(duration.Nanoseconds()) / 1e6,
				})
			}

			// Log based on status code
			switch {
			case wrapped.status >= 500:
				reqLogger.Error("Request failed with server error")
			case wrapped.status >= 400:
				reqLogger.Warn("Request failed with client error")
			case wrapped.status >= 300:
				reqLogger.Info("Request redirected")
			default:
				reqLogger.Info("Request completed")
			}

			// Log slow requests at warning level
			if duration > 1*time.Second {
				reqLogger.Warnf("Slow request detected: %v", duration)
			}
		})
	}
}

// SSEMiddleware creates a logging middleware specifically for SSE endpoints
func SSEMiddleware(logger *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a logger with SSE context
			sseLogger := logger
			if logger.zap != nil {
				zapLogger := logger.zap.WithHTTPRequest(r).With(
					zap.String("connection_type", "sse"),
				)
				sseLogger = &Logger{
					zap: &ZapLogger{
						Logger: zapLogger,
						sugar:  zapLogger.Sugar(),
					},
				}
			} else {
				sseLogger = logger.WithFields(map[string]interface{}{
					"method":          r.Method,
					"path":            r.URL.Path,
					"remote_addr":     r.RemoteAddr,
					"connection_type": "sse",
				})
			}

			// Log SSE connection attempt
			sseLogger.Info("SSE connection initiated")

			// Wrap the handler to log disconnection
			wrapped := w
			if _, ok := w.(http.Hijacker); ok {
				wrapped = &responseWriter{
					ResponseWriter: w,
					status:         200,
				}
			}

			// Process request
			next.ServeHTTP(wrapped, r)

			// Log connection closed
			duration := time.Since(start)
			if logger.zap != nil {
				sseLogger = &Logger{zap: sseLogger.zap.WithDuration(duration)}
			} else {
				sseLogger = sseLogger.WithField("duration_ms", float64(duration.Nanoseconds())/1e6)
			}

			sseLogger.Info("SSE connection closed")
		})
	}
}
