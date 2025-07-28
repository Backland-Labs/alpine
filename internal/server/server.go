// Package server provides HTTP server functionality for Alpine,
// enabling workflow execution via HTTP API and event emission to UI endpoints.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Server represents the Alpine HTTP server that provides REST API endpoints
// for managing workflow runs and emitting events to UI endpoints.
type Server struct {
	port     int
	server   *http.Server
	router   http.ServeMux
	runs     map[string]*Run
	runsMux  sync.RWMutex
	listener net.Listener
}

// NewServer creates a new HTTP server instance
func NewServer(port int) *Server {
	s := &Server{
		port: port,
		runs: make(map[string]*Run),
	}
	
	// Set up routes
	s.router.HandleFunc("/runs", s.handleRuns)
	s.router.HandleFunc("/runs/", s.handleRunStatus)
	
	// Create HTTP server
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: &s.router,
	}
	
	return s
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Create listener
	listener, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	s.listener = listener
	
	// Update port if it was 0 (auto-assigned)
	if s.port == 0 {
		if addr, ok := listener.Addr().(*net.TCPAddr); ok {
			s.port = addr.Port
		}
	}
	
	log.Printf("Starting Alpine HTTP server on port %d", s.port)
	
	// Start server in background
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	
	return nil
}

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) error {
	log.Printf("Shutting down Alpine HTTP server")
	return s.server.Shutdown(ctx)
}

// GetPort returns the actual port the server is listening on
func (s *Server) GetPort() int {
	return s.port
}

// GetRun retrieves a run by ID
func (s *Server) GetRun(runID string) *Run {
	s.runsMux.RLock()
	defer s.runsMux.RUnlock()
	return s.runs[runID]
}

// handleRuns handles POST /runs requests to create new workflow runs
func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse request
	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		s.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Validate request
	if req.Task == "" {
		s.writeError(w, "Task field is required", http.StatusBadRequest)
		return
	}
	
	// Create new run
	runID := uuid.New().String()
	run := &Run{
		ID:            runID,
		Task:          req.Task,
		Status:        "running",
		EventEndpoint: req.EventEndpoint,
		StartTime:     time.Now(),
	}
	
	// Store run
	s.runsMux.Lock()
	s.runs[runID] = run
	s.runsMux.Unlock()
	
	log.Printf("Created new run %s for task: %s", runID, req.Task)
	
	// TODO: Actually execute the workflow here
	// For now, just store the run
	
	// Send response
	resp := RunResponse{
		RunID: runID,
	}
	
	s.writeJSON(w, resp, http.StatusCreated)
}

// handleRunStatus handles GET /runs/{id}/status requests to query run status
func (s *Server) handleRunStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Extract run ID from path
	path := strings.TrimPrefix(r.URL.Path, "/runs/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "status" {
		s.writeError(w, "Not found", http.StatusNotFound)
		return
	}
	runID := parts[0]
	
	// Get run
	s.runsMux.RLock()
	run, exists := s.runs[runID]
	s.runsMux.RUnlock()
	
	if !exists {
		s.writeError(w, "Run not found", http.StatusNotFound)
		return
	}
	
	// Send response
	resp := RunStatusResponse{
		RunID:  run.ID,
		Status: run.Status,
		Task:   run.Task,
	}
	
	s.writeJSON(w, resp, http.StatusOK)
}

// writeJSON writes a JSON response with the given status code
func (s *Server) writeJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// writeError writes a JSON error response
func (s *Server) writeError(w http.ResponseWriter, message string, statusCode int) {
	errorResp := map[string]string{"error": message}
	s.writeJSON(w, errorResp, statusCode)
}