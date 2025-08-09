package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestObservabilityHealthCheck_Healthy(t *testing.T) {
	server := &Server{
		observabilityMetrics: &ObservabilityMetrics{
			EventCount:    100,
			ErrorCount:    5,
			LastEventTime: "2024-01-01T12:00:00Z",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/health/observability", nil)
	w := httptest.NewRecorder()

	server.observabilityHealthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	if response["event_count"] != float64(100) {
		t.Errorf("Expected event_count 100, got %v", response["event_count"])
	}
}

func TestObservabilityHealthCheck_Degraded(t *testing.T) {
	server := &Server{
		observabilityMetrics: &ObservabilityMetrics{
			EventCount:    100,
			ErrorCount:    25, // High error rate
			LastEventTime: "2024-01-01T12:00:00Z",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/health/observability", nil)
	w := httptest.NewRecorder()

	server.observabilityHealthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "degraded" {
		t.Errorf("Expected status 'degraded', got %v", response["status"])
	}
}

func TestObservabilityHealthCheck_Disabled(t *testing.T) {
	server := &Server{
		observabilityMetrics: nil, // Observability disabled
	}

	req := httptest.NewRequest(http.MethodGet, "/health/observability", nil)
	w := httptest.NewRecorder()

	server.observabilityHealthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "disabled" {
		t.Errorf("Expected status 'disabled', got %v", response["status"])
	}
}
