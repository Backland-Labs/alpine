package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Backland-Labs/alpine/internal/logger"
)

// Common constants for handlers
const (
	contentTypeJSON = "application/json"
	errorFieldName  = "error"
)

// healthHandler responds to health check requests
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Health check requested")

	if r.Method != http.MethodGet {
		logger.WithFields(map[string]interface{}{
			"method":   r.Method,
			"expected": http.MethodGet,
		}).Debug("Invalid method for health check")
		s.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	response := map[string]string{
		"status":    "healthy",
		"service":   "alpine-server",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("Failed to encode health response: %v", err)
	} else {
		logger.WithField("response", response).Debug("Health check response sent")
	}
}

// agentsListHandler returns the list of available agents (hardcoded for MVP)
func (s *Server) agentsListHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Agents list requested")

	if r.Method != http.MethodGet {
		logger.WithFields(map[string]interface{}{
			"method":   r.Method,
			"expected": http.MethodGet,
		}).Debug("Invalid method for agents list")
		s.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
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

	logger.WithField("agent_count", len(agents)).Debug("Returning agents list")
	w.Header().Set("Content-Type", contentTypeJSON)
	if err := json.NewEncoder(w).Encode(agents); err != nil {
		logger.Errorf("Failed to encode agents list: %v", err)
	} else {
		logger.Debug("Agents list sent successfully")
	}
}

// agentsRunHandler starts a new workflow run from a GitHub issue
func (s *Server) agentsRunHandler(w http.ResponseWriter, r *http.Request) {
	logger.WithFields(map[string]interface{}{
		"method":         r.Method,
		"content_length": r.ContentLength,
	}).Debug("Agent run requested")

	if r.Method != http.MethodPost {
		logger.WithFields(map[string]interface{}{
			"method":   r.Method,
			"expected": http.MethodPost,
		}).Debug("Invalid method for agent run")
		s.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var payload struct {
		IssueURL string `json:"issue_url"`
		AgentID  string `json:"agent_id"`
	}

	logger.Debug("Decoding agent run payload")
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		logger.Infof("Invalid JSON payload in agentsRunHandler: %v", err)
		s.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	logger.WithFields(map[string]interface{}{
		"issue_url": payload.IssueURL,
		"agent_id":  payload.AgentID,
	}).Debug("Agent run payload decoded")

	// Validate payload
	logger.Debug("Validating agent run payload")
	if payload.IssueURL == "" {
		logger.Debug("Missing issue_url in payload")
		s.respondWithError(w, http.StatusBadRequest, "issue_url is required")
		return
	}
	if payload.AgentID == "" {
		logger.Debug("Missing agent_id in payload")
		s.respondWithError(w, http.StatusBadRequest, "agent_id is required")
		return
	}

	// Create new run
	runID := GenerateID("run")
	logger.WithFields(map[string]interface{}{
		"run_id":    runID,
		"agent_id":  payload.AgentID,
		"issue_url": payload.IssueURL,
	}).Info("Creating new workflow run")

	run := &Run{
		ID:      runID,
		AgentID: payload.AgentID,
		Status:  "running",
		Issue:   payload.IssueURL,
		Created: time.Now(),
		Updated: time.Now(),
	}

	// Store run
	logger.WithField("run_id", run.ID).Debug("Storing run in memory")
	s.mu.Lock()
	s.runs[run.ID] = run
	runCount := len(s.runs)
	s.mu.Unlock()

	logger.WithFields(map[string]interface{}{
		"run_id":     run.ID,
		"total_runs": runCount,
	}).Debug("Run stored")

	// Start workflow if engine is available
	if s.workflowEngine != nil {
		logger.WithFields(map[string]interface{}{
			"run_id":    run.ID,
			"issue_url": payload.IssueURL,
		}).Debug("Starting workflow execution")

		worktreeDir, err := s.workflowEngine.StartWorkflow(r.Context(), payload.IssueURL, run.ID)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"run_id":    run.ID,
				"error":     err.Error(),
				"issue_url": payload.IssueURL,
			}).Error("Failed to start workflow")
			// Update run status to failed
			s.updateRunStatus(run, "failed", "")
		} else {
			logger.WithFields(map[string]interface{}{
				"run_id":       run.ID,
				"worktree_dir": worktreeDir,
				"issue_url":    payload.IssueURL,
			}).Info("Workflow started successfully")
			// Update run with worktree directory
			s.updateRunStatus(run, run.Status, worktreeDir)
		}
	} else {
		logger.WithFields(map[string]interface{}{
			"run_id": run.ID,
			"action": "run_created_without_execution",
		}).Info("Workflow engine not available")
	}

	logger.WithFields(map[string]interface{}{
		"run_id": run.ID,
		"status": run.Status,
	}).Debug("Sending run response")

	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(run); err != nil {
		logger.Errorf("Failed to encode run response: %v", err)
	} else {
		logger.WithField("run_id", run.ID).Debug("Run response sent successfully")
	}
}

// runsListHandler returns all runs from in-memory store
func (s *Server) runsListHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Runs list requested")

	if r.Method != http.MethodGet {
		logger.WithFields(map[string]interface{}{
			"method":   r.Method,
			"expected": http.MethodGet,
		}).Debug("Invalid method for runs list")
		s.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.mu.Lock()
	runs := make([]Run, 0, len(s.runs))
	for _, run := range s.runs {
		runs = append(runs, *run)
	}
	runCount := len(runs)
	s.mu.Unlock()

	logger.WithFields(map[string]interface{}{
		"run_count":      runCount,
		"active_runs":    s.countRunsByStatus("running"),
		"completed_runs": s.countRunsByStatus("completed"),
		"failed_runs":    s.countRunsByStatus("failed"),
	}).Debug("Returning runs list")

	w.Header().Set("Content-Type", contentTypeJSON)
	if err := json.NewEncoder(w).Encode(runs); err != nil {
		logger.Errorf("Failed to encode runs list: %v", err)
	} else {
		logger.WithField("count", runCount).Debug("Runs list sent successfully")
	}
}

// runDetailsHandler returns details for a specific run
func (s *Server) runDetailsHandler(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	logger.WithFields(map[string]interface{}{
		"run_id": runID,
		"method": r.Method,
	}).Debug("Run details requested")

	if r.Method != http.MethodGet {
		logger.WithFields(map[string]interface{}{
			"method":   r.Method,
			"expected": http.MethodGet,
		}).Debug("Invalid method for run details")
		s.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if runID == "" {
		logger.Debug("Missing run ID in request")
		s.respondWithError(w, http.StatusBadRequest, "Run ID is required")
		return
	}

	s.mu.Lock()
	run, exists := s.runs[runID]
	s.mu.Unlock()

	if !exists {
		logger.WithFields(map[string]interface{}{
			"run_id":     runID,
			"total_runs": len(s.runs),
		}).Debug("Run not found")
		s.respondWithError(w, http.StatusNotFound, "Run not found")
		return
	}

	logger.WithFields(map[string]interface{}{
		"run_id":  run.ID,
		"status":  run.Status,
		"created": run.Created,
		"updated": run.Updated,
	}).Debug("Run found")

	// Create response with run details
	response := map[string]interface{}{
		"id":           run.ID,
		"agent_id":     run.AgentID,
		"status":       run.Status,
		"issue":        run.Issue,
		"created":      run.Created,
		"updated":      run.Updated,
		"worktree_dir": run.WorktreeDir,
	}

	// Add workflow state if available
	if s.workflowEngine != nil {
		logger.WithField("run_id", runID).Debug("Fetching workflow state")
		if state, err := s.workflowEngine.GetWorkflowState(r.Context(), runID); err == nil {
			response["current_step"] = state.CurrentStepDescription
			logger.WithFields(map[string]interface{}{
				"run_id":          runID,
				"workflow_status": state.Status,
				"current_step":    state.CurrentStepDescription,
			}).Debug("Workflow state retrieved")

			// Update run status based on workflow state
			if state.Status == "completed" && run.Status != "completed" {
				logger.WithFields(map[string]interface{}{
					"run_id":          runID,
					"previous_status": run.Status,
				}).Info("Updating run status to completed")
				s.mu.Lock()
				run.Status = "completed"
				run.Updated = time.Now()
				s.mu.Unlock()
			}
		} else {
			logger.WithFields(map[string]interface{}{
				"run_id": runID,
				"error":  err.Error(),
			}).Debug("Failed to get workflow state")
		}
	}

	logger.WithField("run_id", runID).Debug("Sending run details response")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.WithFields(map[string]interface{}{
			"run_id": runID,
			"error":  err.Error(),
		}).Error("Failed to encode run details")
	}
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

	// Subscribe to workflow events if engine is available
	if s.workflowEngine != nil {
		events, err := s.workflowEngine.SubscribeToEvents(r.Context(), runID)
		if err == nil {
			// Forward events to SSE client
			for {
				select {
				case event, ok := <-events:
					if !ok {
						return // Channel closed
					}
					// Send event as SSE
					data, _ := json.Marshal(event)
					fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(data))
					flusher.Flush()
				case <-r.Context().Done():
					return // Client disconnected
				}
			}
		}
	}

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
	if exists && run.Status != "running" {
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Cannot cancel non-running workflow",
		})
		return
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

	// Cancel workflow if engine is available
	if s.workflowEngine != nil {
		if err := s.workflowEngine.CancelWorkflow(r.Context(), runID); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Failed to cancel workflow",
			})
			return
		}
	}

	// Update run status
	s.mu.Lock()
	run.Status = "cancelled"
	run.Updated = time.Now()
	s.mu.Unlock()

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
	run, runExists := s.runs[runID]
	s.mu.Unlock()

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Plan not found",
		})
		return
	}

	// Approve plan in workflow engine
	if s.workflowEngine != nil {
		if err := s.workflowEngine.ApprovePlan(r.Context(), runID); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Failed to approve plan",
			})
			return
		}
	}

	// Update plan status only after successful workflow approval
	s.mu.Lock()
	plan.Status = "approved"
	plan.Updated = time.Now()
	// Update run status
	if runExists {
		run.Status = "running"
		run.Updated = time.Now()
	}
	s.mu.Unlock()

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
