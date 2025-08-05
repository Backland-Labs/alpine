// Package server implements an HTTP server with Server-Sent Events (SSE) support
// and REST API endpoints for Alpine. This server allows frontend applications to
// receive real-time updates about workflow progress and state changes, as well as
// programmatically manage workflows via REST API.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Backland-Labs/alpine/internal/logger"
)

// Constants for server configuration
const (
	// defaultEventBufferSize is the default size for the events channel buffer
	defaultEventBufferSize = 100
)

// Common errors returned by the server
var (
	// ErrServerRunning is returned when attempting to start an already running server
	ErrServerRunning = errors.New("server is already running")
)

// Server represents an HTTP server with Server-Sent Events support.
// It provides real-time updates to connected clients about Alpine's
// workflow progress and state changes.
type Server struct {
	port       int          // Port number to listen on (0 for auto-assignment)
	httpServer *http.Server // Underlying HTTP server instance
	eventsChan chan string  // Channel for broadcasting events to clients
	listener   net.Listener // Network listener for accepting connections
	mu         sync.Mutex   // Protects server state during concurrent access
	running    bool         // Indicates if the server is currently running

	// In-memory storage for REST API
	runs  map[string]*Run  // Storage for workflow runs
	plans map[string]*Plan // Storage for workflow plans

	// Workflow engine integration
	workflowEngine WorkflowEngine // Optional workflow engine for executing workflows

	// Run-specific event filtering
	runEventHub *runSpecificEventHub // Hub for run-specific event subscriptions
}

// NewServer creates a new HTTP server instance configured to run on the specified port.
// The server is initialized but not started - use Start() to begin listening.
func NewServer(port int) *Server {
	logger.WithFields(map[string]interface{}{
		"port":              port,
		"event_buffer_size": defaultEventBufferSize,
	}).Debug("Creating new server")

	mux := http.NewServeMux()

	server := &Server{
		port: port,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf("0.0.0.0:%d", port),
			Handler: mux,
		},
		eventsChan:  make(chan string, defaultEventBufferSize),
		runs:        make(map[string]*Run),
		plans:       make(map[string]*Plan),
		runEventHub: newRunSpecificEventHub(),
	}

	logger.Debugf("Server instance created with address: %s", server.httpServer.Addr)
	return server
}

// NewServerWithConfig creates a new HTTP server instance with custom configuration
func NewServerWithConfig(port int, streamBufferSize int, maxClientsPerRun int) *Server {
	logger.WithFields(map[string]interface{}{
		"port":                port,
		"stream_buffer_size":  streamBufferSize,
		"max_clients_per_run": maxClientsPerRun,
	}).Debug("Creating new server with custom config")

	mux := http.NewServeMux()

	// Use provided buffer size or default
	bufferSize := streamBufferSize
	if bufferSize <= 0 {
		bufferSize = defaultEventBufferSize
		logger.Debugf("Using default buffer size: %d", bufferSize)
	}

	server := &Server{
		port: port,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf("0.0.0.0:%d", port),
			Handler: mux,
		},
		eventsChan:  make(chan string, bufferSize),
		runs:        make(map[string]*Run),
		plans:       make(map[string]*Plan),
		runEventHub: newRunSpecificEventHubWithConfig(bufferSize, maxClientsPerRun),
	}

	logger.Debugf("Server instance created with custom config, address: %s", server.httpServer.Addr)
	return server
}

// Start begins listening for HTTP requests on the configured port.
// The server runs until the provided context is canceled.
// Returns http.ErrServerClosed on graceful shutdown, or any other error if startup fails.
func (s *Server) Start(ctx context.Context) error {
	logger.WithField("port", s.port).Info("Starting HTTP server")

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		logger.Warn("Attempted to start already running server")
		return ErrServerRunning
	}
	s.running = true
	s.mu.Unlock()
	logger.Debug("Server state set to running")

	// Check if context is already canceled
	select {
	case <-ctx.Done():
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		logger.Info("Server start canceled due to context cancellation")
		return ctx.Err()
	default:
	}

	// Create listener with dynamic address for reuse
	addr := s.httpServer.Addr
	if s.port == 0 {
		addr = "localhost:0" // Let OS assign port
		logger.Debug("Using dynamic port assignment")
	}

	logger.Debugf("Creating TCP listener on address: %s", addr)
	var err error
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		logger.WithFields(map[string]interface{}{
			"error":   err.Error(),
			"address": addr,
		}).Error("Failed to create listener")
		return fmt.Errorf("failed to listen: %w", err)
	}

	actualAddr := s.listener.Addr().String()
	logger.WithField("address", actualAddr).Info("Server listening")

	// Create a new HTTP server for each start to avoid reuse issues
	mux := http.NewServeMux()

	// Apply logging middleware to all handlers
	log := logger.GetLogger()
	middleware := logger.HTTPMiddleware(log)

	// Register endpoint handlers with logging
	logger.Debug("Registering HTTP endpoints")

	mux.Handle("/events", logger.SSEMiddleware(log)(http.HandlerFunc(s.sseHandler)))
	mux.Handle("/health", middleware(http.HandlerFunc(s.healthHandler)))
	mux.Handle("/agents/list", middleware(http.HandlerFunc(s.agentsListHandler)))
	mux.Handle("/agents/run", middleware(http.HandlerFunc(s.agentsRunHandler)))
	mux.Handle("/runs", middleware(http.HandlerFunc(s.runsListHandler)))
	mux.Handle("/runs/{id}", middleware(http.HandlerFunc(s.runDetailsHandler)))
	mux.Handle("/runs/{id}/events", logger.SSEMiddleware(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.enhancedRunEventsHandler(w, r, s.runEventHub)
	})))
	mux.Handle("/runs/{id}/cancel", middleware(http.HandlerFunc(s.runCancelHandler)))
	mux.Handle("/plans/{runId}", middleware(http.HandlerFunc(s.planGetHandler)))
	mux.Handle("/plans/{runId}/approve", middleware(http.HandlerFunc(s.planApproveHandler)))
	mux.Handle("/plans/{runId}/feedback", middleware(http.HandlerFunc(s.planFeedbackHandler)))

	logger.Debugf("Registered %d endpoints", 11)

	s.httpServer = &http.Server{
		Handler: mux,
	}

	// Handle shutdown when context is canceled
	go func() {
		<-ctx.Done()
		logger.Info("Server shutdown initiated")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.WithField("error", err.Error()).Error("Error during server shutdown")
		}
	}()

	// Start serving
	logger.Info("Server starting to accept connections")
	err = s.httpServer.Serve(s.listener)

	s.mu.Lock()
	s.running = false
	s.listener = nil
	s.mu.Unlock()

	// http.ErrServerClosed is expected when shutting down gracefully
	if err == http.ErrServerClosed {
		logger.Info("Server shut down gracefully")
		return err
	}

	if err != nil {
		logger.WithField("error", err.Error()).Error("Server error")
	}
	return err
}

// Address returns the actual address the server is listening on.
// Returns empty string if the server is not running.
func (s *Server) Address() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// SetWorkflowEngine sets the workflow engine for the server
func (s *Server) SetWorkflowEngine(engine WorkflowEngine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workflowEngine = engine
	logger.WithField("engine_nil", engine == nil).Debug("Workflow engine set")
}

// BroadcastEvent broadcasts a workflow event to all connected SSE clients
func (s *Server) BroadcastEvent(event WorkflowEvent) {
	// Add panic recovery for robustness
	defer func() {
		if r := recover(); r != nil {
			logger.WithFields(map[string]interface{}{
				"panic":      r,
				"event_type": event.Type,
				"run_id":     event.RunID,
			}).Error("Panic recovered in BroadcastEvent")
		}
	}()

	logger.WithFields(map[string]interface{}{
		"type":      event.Type,
		"run_id":    event.RunID,
		"source":    event.Source,
		"timestamp": event.Timestamp,
	}).Debug("Broadcasting event")

	// Convert event to JSON for SSE
	data, err := json.Marshal(event)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"event_type": event.Type,
		}).Error("Failed to marshal event")
		return
	}

	// Create SSE formatted message
	message := fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(data))

	// Send to global event channel (non-blocking)
	select {
	case s.eventsChan <- message:
		logger.WithFields(map[string]interface{}{
			"event_type":   event.Type,
			"message_size": len(message),
		}).Debug("Event sent to global channel")
	default:
		// Channel full, drop message - this is expected behavior
		// for graceful degradation under load
		logger.WithFields(map[string]interface{}{
			"event_type":   event.Type,
			"run_id":       event.RunID,
			"channel_size": len(s.eventsChan),
			"channel_cap":  cap(s.eventsChan),
		}).Warn("Event channel full, dropping message")
	}

	// Also send to run-specific subscribers
	if s.runEventHub != nil && event.RunID != "" {
		logger.WithField("run_id", event.RunID).Debug("Broadcasting to run-specific subscribers")
		s.runEventHub.broadcast(event)
	}
}

// sseHandler handles Server-Sent Events connections at the /events endpoint.
// It sends an initial "hello world" event upon connection and manages the
// client lifecycle, including proper cleanup on disconnect.
func (s *Server) sseHandler(w http.ResponseWriter, r *http.Request) {
	clientID := r.RemoteAddr
	logger.WithFields(map[string]interface{}{
		"client_id":  clientID,
		"user_agent": r.UserAgent(),
	}).Debug("SSE connection initiated")

	// Set SSE specific headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Ensure buffering is disabled for real-time updates
	flusher, ok := w.(http.Flusher)
	if !ok {
		logger.WithField("client_id", clientID).Error("Streaming unsupported for client")
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial hello world event
	_, _ = fmt.Fprintf(w, "data: hello world\n\n")
	flusher.Flush()
	logger.WithField("client_id", clientID).Debug("Initial SSE event sent")

	// Create keepalive ticker for connection health
	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	var eventCount int64
	startTime := time.Now()

	// Listen for events from the event channel
	for {
		select {
		case event := <-s.eventsChan:
			eventCount++
			// Send event to client
			if _, err := fmt.Fprint(w, event); err != nil {
				// Client write failed, disconnect
				logger.WithFields(map[string]interface{}{
					"client_id":           clientID,
					"error":               err.Error(),
					"events_sent":         eventCount,
					"connection_duration": time.Since(startTime),
				}).Debug("Client disconnected during write")
				return
			}
			flusher.Flush()

			if eventCount%100 == 0 {
				logger.WithFields(map[string]interface{}{
					"client_id":   clientID,
					"events_sent": eventCount,
				}).Debug("SSE event milestone")
			}

		case <-keepaliveTicker.C:
			// Send keepalive comment to maintain connection
			if _, err := fmt.Fprintf(w, ":keepalive\n\n"); err != nil {
				// Keepalive failed, client disconnected
				logger.WithFields(map[string]interface{}{
					"client_id":           clientID,
					"error":               err.Error(),
					"events_sent":         eventCount,
					"connection_duration": time.Since(startTime),
				}).Debug("Client disconnected during keepalive")
				return
			}
			flusher.Flush()

		case <-r.Context().Done():
			// Client disconnected
			logger.WithFields(map[string]interface{}{
				"client_id":           clientID,
				"events_sent":         eventCount,
				"connection_duration": time.Since(startTime),
			}).Info("SSE client disconnected")
			return
		}
	}
}

// Helper methods

// respondWithError sends a JSON error response with the specified status code
func (s *Server) respondWithError(w http.ResponseWriter, statusCode int, message string) {
	logger.WithFields(map[string]interface{}{
		"status_code":   statusCode,
		"error_message": message,
	}).Debug("Sending error response")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := map[string]string{
		"error": message,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.WithFields(map[string]interface{}{
			"error":         err.Error(),
			"status_code":   statusCode,
			"error_message": message,
		}).Error("Failed to encode error response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// updateRunStatus updates a run's status and worktree directory in a thread-safe manner
func (s *Server) updateRunStatus(run *Run, status string, worktreeDir string) {
	previousStatus := run.Status

	s.mu.Lock()
	defer s.mu.Unlock()

	run.Status = status
	run.Updated = time.Now()
	if worktreeDir != "" {
		run.WorktreeDir = worktreeDir
	}

	logger.WithFields(map[string]interface{}{
		"run_id":          run.ID,
		"previous_status": previousStatus,
		"new_status":      status,
		"worktree_dir":    worktreeDir,
		"updated":         run.Updated,
	}).Debug("Run status updated")
}

// UpdateRunStatus updates a run's status (exported for testing)
func (s *Server) UpdateRunStatus(run *Run, status string, worktreeDir string) {
	s.updateRunStatus(run, status, worktreeDir)

	// Ensure run is in the map
	s.mu.Lock()
	if _, exists := s.runs[run.ID]; !exists {
		s.runs[run.ID] = run
	}
	s.mu.Unlock()
}

// countRunsByStatus counts runs with a specific status
func (s *Server) countRunsByStatus(status string) int {
	count := 0
	for _, run := range s.runs {
		if run.Status == status {
			count++
		}
	}
	return count
}
