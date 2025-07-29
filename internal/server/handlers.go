package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// healthHandler responds to health check requests
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status": "healthy",
		"service": "alpine-server",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)
}

// agentsListHandler returns the list of available agents (hardcoded for MVP)
func (s *Server) agentsListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Hardcoded agent list for MVP
	agents := []Agent{
		{
			ID:          "alpine-agent",
			Name:        "Alpine Workflow Agent",
			Description: "Default agent for running Alpine workflows from GitHub issues",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

// agentsRunHandler starts a new workflow run from a GitHub issue
func (s *Server) agentsRunHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var payload struct {
		IssueURL string `json:"issue_url"`
		AgentID  string `json:"agent_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid JSON payload",
		})
		return
	}
	
	// Create new run
	run := &Run{
		ID:      GenerateID("run"),
		AgentID: payload.AgentID,
		Status:  "running",
		Issue:   payload.IssueURL,
		Created: time.Now(),
		Updated: time.Now(),
	}
	
	// Store run
	s.mu.Lock()
	s.runs[run.ID] = run
	s.mu.Unlock()
	
	// TODO: Start actual workflow execution
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(run)
}

// runsListHandler returns all runs from in-memory store
func (s *Server) runsListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	s.mu.Lock()
	runs := make([]Run, 0, len(s.runs))
	for _, run := range s.runs {
		runs = append(runs, *run)
	}
	s.mu.Unlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runs)
}

// runDetailsHandler returns details for a specific run
func (s *Server) runDetailsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	runID := r.PathValue("id")
	
	s.mu.Lock()
	run, exists := s.runs[runID]
	s.mu.Unlock()
	
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Run not found",
		})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

// runEventsHandler provides SSE endpoint for run-specific events
func (s *Server) runEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	runID := r.PathValue("id")
	
	s.mu.Lock()
	_, exists := s.runs[runID]
	s.mu.Unlock()
	
	if !exists {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}
	
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	
	// Send initial connection event
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"runId\":\"%s\"}\n\n", runID)
	flusher.Flush()
	
	// Keep connection open until client disconnects
	<-r.Context().Done()
}

// runCancelHandler cancels a running workflow
func (s *Server) runCancelHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	runID := r.PathValue("id")
	
	s.mu.Lock()
	run, exists := s.runs[runID]
	if exists {
		run.Status = "cancelled"
		run.Updated = time.Now()
	}
	s.mu.Unlock()
	
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Run not found",
		})
		return
	}
	
	// TODO: Cancel actual workflow execution
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "cancelled",
		"runId":  runID,
	})
}

// planGetHandler retrieves plan content for a run
func (s *Server) planGetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	runID := r.PathValue("runId")
	
	s.mu.Lock()
	plan, exists := s.plans[runID]
	s.mu.Unlock()
	
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Plan not found",
		})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plan)
}

// planApproveHandler approves a plan to continue workflow
func (s *Server) planApproveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	runID := r.PathValue("runId")
	
	s.mu.Lock()
	plan, exists := s.plans[runID]
	if exists {
		plan.Status = "approved"
		plan.Updated = time.Now()
	}
	s.mu.Unlock()
	
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Plan not found",
		})
		return
	}
	
	// TODO: Continue workflow execution with approved plan
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "approved",
		"runId":  runID,
	})
}

// planFeedbackHandler handles feedback on a plan
func (s *Server) planFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	runID := r.PathValue("runId")
	
	s.mu.Lock()
	_, exists := s.plans[runID]
	s.mu.Unlock()
	
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Plan not found",
		})
		return
	}
	
	var payload struct {
		Feedback string `json:"feedback"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid JSON payload",
		})
		return
	}
	
	// TODO: Process feedback and regenerate plan
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "feedback_received",
		"runId":  runID,
	})
}