// Package server implements an HTTP server with Server-Sent Events (SSE) support
// for Alpine. This server allows frontend applications to receive real-time
// updates about workflow progress and state changes.
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
}

// NewServer creates a new HTTP server instance configured to run on the specified port.
// The server is initialized but not started - use Start() to begin listening.
func NewServer(port int) *Server {
	mux := http.NewServeMux()

	return &Server{
		port: port,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf("localhost:%d", port),
			Handler: mux,
		},
		eventsChan: make(chan string, defaultEventBufferSize),
		runs:       make(map[string]*Run),
		plans:      make(map[string]*Plan),
	}
}

// Start begins listening for HTTP requests on the configured port.
// The server runs until the provided context is canceled.
// Returns http.ErrServerClosed on graceful shutdown, or any other error if startup fails.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ErrServerRunning
	}
	s.running = true
	s.mu.Unlock()

	// Check if context is already canceled
	select {
	case <-ctx.Done():
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return ctx.Err()
	default:
	}

	// Create listener with dynamic address for reuse
	addr := s.httpServer.Addr
	if s.port == 0 {
		addr = "localhost:0" // Let OS assign port
	}

	var err error
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Create a new HTTP server for each start to avoid reuse issues
	mux := http.NewServeMux()

	// Register endpoint handlers
	mux.HandleFunc("/events", s.sseHandler)
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/agents/list", s.agentsListHandler)
	mux.HandleFunc("/agents/run", s.agentsRunHandler)
	mux.HandleFunc("/runs", s.runsListHandler)
	mux.HandleFunc("/runs/{id}", s.runDetailsHandler)
	mux.HandleFunc("/runs/{id}/events", s.runEventsHandler)
	mux.HandleFunc("/runs/{id}/cancel", s.runCancelHandler)
	mux.HandleFunc("/plans/{runId}", s.planGetHandler)
	mux.HandleFunc("/plans/{runId}/approve", s.planApproveHandler)
	mux.HandleFunc("/plans/{runId}/feedback", s.planFeedbackHandler)

	s.httpServer = &http.Server{
		Handler: mux,
	}

	// Handle shutdown when context is canceled
	go func() {
		<-ctx.Done()
		_ = s.httpServer.Shutdown(context.Background())
	}()

	// Start serving
	err = s.httpServer.Serve(s.listener)

	s.mu.Lock()
	s.running = false
	s.listener = nil
	s.mu.Unlock()

	// http.ErrServerClosed is expected when shutting down gracefully
	if err == http.ErrServerClosed {
		return err
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
}

// BroadcastEvent broadcasts a workflow event to all connected SSE clients
func (s *Server) BroadcastEvent(event WorkflowEvent) {
	// Convert event to JSON for SSE
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	
	// Create SSE formatted message
	message := fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(data))
	
	// Send to event channel (non-blocking)
	select {
	case s.eventsChan <- message:
	default:
		// Channel full, drop message
	}
}

// sseHandler handles Server-Sent Events connections at the /events endpoint.
// It sends an initial "hello world" event upon connection and manages the
// client lifecycle, including proper cleanup on disconnect.
func (s *Server) sseHandler(w http.ResponseWriter, r *http.Request) {
	// Set SSE specific headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Ensure buffering is disabled for real-time updates
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial hello world event
	_, _ = fmt.Fprintf(w, "data: hello world\n\n")
	flusher.Flush()

	// Listen for events from the event channel
	for {
		select {
		case event := <-s.eventsChan:
			// Send event to client
			_, _ = fmt.Fprint(w, event)
			flusher.Flush()
		case <-r.Context().Done():
			// Client disconnected
			return
		}
	}
}

// Helper methods

// respondWithError sends a JSON error response with the specified status code
func (s *Server) respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := map[string]string{
		"error": message,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log the error but don't attempt to write more to the response
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// updateRunStatus updates a run's status and worktree directory in a thread-safe manner
func (s *Server) updateRunStatus(run *Run, status string, worktreeDir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	run.Status = status
	run.Updated = time.Now()
	if worktreeDir != "" {
		run.WorktreeDir = worktreeDir
	}
}
