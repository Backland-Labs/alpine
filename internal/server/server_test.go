package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestNewServer verifies that a new server can be created with proper configuration
func TestNewServer(t *testing.T) {
	s := NewServer(8080)
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
	if s.port != 8080 {
		t.Errorf("Expected port 8080, got %d", s.port)
	}
}

// TestServerStartStop verifies that the server starts and stops cleanly
func TestServerStartStop(t *testing.T) {
	s := NewServer(0) // Use port 0 for automatic port assignment
	
	// Start the server
	err := s.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	
	// Verify server is running
	if s.GetPort() == 0 {
		t.Error("Server port should not be 0 after starting")
	}
	
	// Stop the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = s.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}
}

// TestPostRunsEndpoint verifies that POST /runs starts a new run
func TestPostRunsEndpoint(t *testing.T) {
	s := NewServer(0)
	
	// Create request body
	reqBody := RunRequest{
		Task:          "Test task",
		EventEndpoint: "http://localhost:9090/events",
	}
	body, _ := json.Marshal(reqBody)
	
	// Create test request
	req := httptest.NewRequest("POST", "/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Handle request
	s.router.ServeHTTP(w, req)
	
	// Check response status
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Errorf("Expected status 200 or 201, got %d", w.Code)
	}
	
	// Parse response
	var resp RunResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Verify response contains run ID
	if resp.RunID == "" {
		t.Error("Response should contain a run ID")
	}
	
	// Verify run was tracked
	run := s.GetRun(resp.RunID)
	if run == nil {
		t.Error("Run should be tracked by server")
		return
	}
	if run.Task != "Test task" {
		t.Errorf("Expected task 'Test task', got '%s'", run.Task)
	}
}

// TestGetRunStatusEndpoint verifies that GET /runs/{id}/status returns run status
func TestGetRunStatusEndpoint(t *testing.T) {
	s := NewServer(0)
	
	// Create a run first
	runID := "test-run-123"
	s.runs[runID] = &Run{
		ID:     runID,
		Task:   "Test task",
		Status: "running",
	}
	
	// Create test request
	req := httptest.NewRequest("GET", "/runs/"+runID+"/status", nil)
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Handle request
	s.router.ServeHTTP(w, req)
	
	// Check response status
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	// Parse response
	var resp RunStatusResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Verify response
	if resp.RunID != runID {
		t.Errorf("Expected run ID %s, got %s", runID, resp.RunID)
	}
	if resp.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", resp.Status)
	}
}

// TestGetRunStatusNotFound verifies that GET /runs/{id}/status returns 404 for unknown runs
func TestGetRunStatusNotFound(t *testing.T) {
	s := NewServer(0)
	
	// Create test request for non-existent run
	req := httptest.NewRequest("GET", "/runs/unknown-run/status", nil)
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Handle request
	s.router.ServeHTTP(w, req)
	
	// Check response status
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestJSONRequestValidation verifies that the server validates JSON requests properly
func TestJSONRequestValidation(t *testing.T) {
	s := NewServer(0)
	
	tests := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
	}{
		{
			name:        "Invalid JSON",
			body:        `{"invalid json`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Empty body",
			body:        "",
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Missing task field",
			body:        `{"eventEndpoint": "http://localhost:9090"}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/runs", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", tt.contentType)
			
			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, req)
			
			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

// TestConcurrentRunManagement verifies thread-safe run management
func TestConcurrentRunManagement(t *testing.T) {
	s := NewServer(0)
	
	// Start multiple goroutines creating runs
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			reqBody := RunRequest{
				Task:          "Concurrent task",
				EventEndpoint: "http://localhost:9090/events",
			}
			body, _ := json.Marshal(reqBody)
			
			req := httptest.NewRequest("POST", "/runs", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, req)
			
			if w.Code != http.StatusOK && w.Code != http.StatusCreated {
				t.Errorf("Request %d failed with status %d", id, w.Code)
			}
			
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify all runs were tracked
	runCount := len(s.runs)
	if runCount != 10 {
		t.Errorf("Expected 10 runs, got %d", runCount)
	}
}