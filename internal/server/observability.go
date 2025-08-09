package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/Backland-Labs/alpine/internal/logger"
)

// ObservabilityMetrics holds metrics for the observability system
type ObservabilityMetrics struct {
	mu            sync.RWMutex
	EventCount    int64     `json:"event_count"`
	ErrorCount    int64     `json:"error_count"`
	LastEventTime string    `json:"last_event_time"`
	StartTime     time.Time `json:"start_time"`
}

// NewObservabilityMetrics creates a new metrics instance
func NewObservabilityMetrics() *ObservabilityMetrics {
	return &ObservabilityMetrics{
		StartTime: time.Now(),
	}
}

// IncrementEventCount increments the event count
func (om *ObservabilityMetrics) IncrementEventCount() {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.EventCount++
	om.LastEventTime = time.Now().Format(time.RFC3339)
}

// IncrementErrorCount increments the error count
func (om *ObservabilityMetrics) IncrementErrorCount() {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.ErrorCount++
}

// GetSnapshot returns a snapshot of current metrics
func (om *ObservabilityMetrics) GetSnapshot() ObservabilityMetrics {
	om.mu.RLock()
	defer om.mu.RUnlock()
	return ObservabilityMetrics{
		EventCount:    om.EventCount,
		ErrorCount:    om.ErrorCount,
		LastEventTime: om.LastEventTime,
		StartTime:     om.StartTime,
	}
}

// GetErrorRate returns the error rate as a percentage
func (om *ObservabilityMetrics) GetErrorRate() float64 {
	om.mu.RLock()
	defer om.mu.RUnlock()
	if om.EventCount == 0 {
		return 0.0
	}
	return float64(om.ErrorCount) / float64(om.EventCount) * 100.0
}

// observabilityHealthHandler provides health check for observability system
func (s *Server) observabilityHealthHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Observability health check requested")

	if r.Method != http.MethodGet {
		logger.WithFields(map[string]interface{}{
			"method":   r.Method,
			"expected": http.MethodGet,
		}).Debug("Invalid method for observability health check")
		s.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Check if observability is enabled
	if s.observabilityMetrics == nil {
		response := map[string]interface{}{
			"status":    "disabled",
			"message":   "Observability system is disabled",
			"timestamp": time.Now().Format(time.RFC3339),
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Errorf("Failed to encode observability health response: %v", err)
		}
		return
	}

	// Get metrics snapshot
	metrics := s.observabilityMetrics.GetSnapshot()
	errorRate := s.observabilityMetrics.GetErrorRate()

	// Determine health status
	status := "healthy"
	if errorRate > 20.0 { // More than 20% error rate is degraded
		status = "degraded"
	}

	response := map[string]interface{}{
		"status":         status,
		"event_count":    metrics.EventCount,
		"error_count":    metrics.ErrorCount,
		"error_rate":     errorRate,
		"last_event":     metrics.LastEventTime,
		"uptime_seconds": time.Since(metrics.StartTime).Seconds(),
		"timestamp":      time.Now().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("Failed to encode observability health response: %v", err)
	} else {
		logger.WithFields(map[string]interface{}{
			"status":      status,
			"event_count": metrics.EventCount,
			"error_rate":  errorRate,
		}).Debug("Observability health check response sent")
	}
}
