package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ToolData struct {
	ToolName   string          `json:"tool_name"`
	ToolInput  json.RawMessage `json:"tool_input"`
	ToolOutput json.RawMessage `json:"tool_output"`
	Event      string          `json:"event"`
	Timestamp  string          `json:"timestamp"`
	ToolCallID string          `json:"tool_call_id"`
}

type AgUIEvent struct {
	EventType string    `json:"type"`
	Data      EventData `json:"data"`
}

type EventData struct {
	ToolCallID   string          `json:"toolCallId"`
	ToolCallName string          `json:"toolCallName"`
	RunID        string          `json:"runId"`
	ToolInput    json.RawMessage `json:"toolInput,omitempty"`
	ToolOutput   json.RawMessage `json:"toolOutput,omitempty"`
}

const (
	circuitBreakerFile = "/tmp/alpine_circuit_breaker.json"
	batchFile          = "/tmp/alpine_event_batch.json"
)

type CircuitBreakerState struct {
	FailureCount int       `json:"failure_count"`
	LastFailure  time.Time `json:"last_failure"`
	IsOpen       bool      `json:"is_open"`
}

// Logger provides structured logging for the hook
type Logger struct {
	runID   string
	verbose bool
}

func newLogger(runID string) *Logger {
	verbose := os.Getenv("ALPINE_HOOK_VERBOSE") == "true"
	return &Logger{
		runID:   runID,
		verbose: verbose,
	}
}

func (l *Logger) logJSON(level string, message string, data map[string]interface{}) {
	logEntry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"component": "ag-ui-emitter",
		"run_id":    l.runID,
		"message":   message,
	}

	// Add additional data
	for k, v := range data {
		logEntry[k] = v
	}

	jsonBytes, _ := json.Marshal(logEntry)
	fmt.Fprintln(os.Stderr, string(jsonBytes))
}

func (l *Logger) Info(message string, data map[string]interface{}) {
	l.logJSON("INFO", message, data)
}

func (l *Logger) Debug(message string, data map[string]interface{}) {
	if l.verbose {
		l.logJSON("DEBUG", message, data)
	}
}

func (l *Logger) Error(message string, data map[string]interface{}) {
	l.logJSON("ERROR", message, data)
}

func (l *Logger) Warn(message string, data map[string]interface{}) {
	l.logJSON("WARN", message, data)
}

func main() {
	startTime := time.Now()

	// Initialize logger (will get run ID later)
	logger := newLogger("unknown")

	logger.Info("Hook execution started", map[string]interface{}{
		"hook_type": "ag-ui-emitter",
		"pid":       os.Getpid(),
	})

	// Check circuit breaker first
	if isCircuitBreakerOpen() {
		logger.Warn("Circuit breaker is open, skipping hook execution", map[string]interface{}{
			"circuit_breaker_file": circuitBreakerFile,
		})
		return
	}

	// Read tool data from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		logger.Error("Failed to read input from stdin", map[string]interface{}{
			"error": err.Error(),
		})
		recordFailure()
		return
	}

	logger.Debug("Raw input received", map[string]interface{}{
		"input_size":    len(input),
		"input_preview": string(input)[:min(len(input), 200)] + "...",
	})

	// Parse the JSON data
	var toolData ToolData
	if err = json.Unmarshal(input, &toolData); err != nil {
		logger.Error("Failed to parse tool data JSON", map[string]interface{}{
			"error": err.Error(),
			"input": string(input),
		})
		recordFailure()
		return
	}

	// Get environment variables
	endpoint := os.Getenv("ALPINE_EVENTS_ENDPOINT")
	runID := os.Getenv("ALPINE_RUN_ID")
	if runID == "" {
		runID = "unknown"
	}

	// Update logger with actual run ID
	logger.runID = runID

	// Log detailed tool call information
	logger.Info("Tool call hook triggered", map[string]interface{}{
		"tool_name":    toolData.ToolName,
		"event_type":   toolData.Event,
		"tool_call_id": toolData.ToolCallID,
		"timestamp":    toolData.Timestamp,
		"has_input":    len(toolData.ToolInput) > 0,
		"has_output":   len(toolData.ToolOutput) > 0,
		"input_size":   len(toolData.ToolInput),
		"output_size":  len(toolData.ToolOutput),
	})

	// Log tool input/output (sanitized)
	if len(toolData.ToolInput) > 0 {
		logger.Debug("Tool input data", map[string]interface{}{
			"tool_input": sanitizeToolData(toolData.ToolInput),
		})
	}

	if len(toolData.ToolOutput) > 0 {
		logger.Debug("Tool output data", map[string]interface{}{
			"tool_output": sanitizeToolData(toolData.ToolOutput),
		})
	}

	if endpoint == "" {
		logger.Warn("ALPINE_EVENTS_ENDPOINT not set, skipping event emission", map[string]interface{}{
			"available_env_vars": getAvailableEnvVars(),
		})
		return
	}

	batchSize := 10
	if bs := os.Getenv("ALPINE_TOOL_CALL_BATCH_SIZE"); bs != "" {
		fmt.Sscanf(bs, "%d", &batchSize)
	}

	sampleRate := 100
	if sr := os.Getenv("ALPINE_TOOL_CALL_SAMPLE_RATE"); sr != "" {
		fmt.Sscanf(sr, "%d", &sampleRate)
	}

	logger.Info("Hook configuration loaded", map[string]interface{}{
		"endpoint":    endpoint,
		"batch_size":  batchSize,
		"sample_rate": sampleRate,
	})

	// Apply sampling - skip event if random number is above sample rate
	if sampleRate < 100 {
		rand.Seed(time.Now().UnixNano())
		randomValue := rand.Intn(100) + 1
		if randomValue > sampleRate {
			logger.Info("Event sampled out", map[string]interface{}{
				"sample_rate":  sampleRate,
				"random_value": randomValue,
				"tool_name":    toolData.ToolName,
			})
			return
		}
		logger.Debug("Event passed sampling", map[string]interface{}{
			"sample_rate":  sampleRate,
			"random_value": randomValue,
		})
	}

	// Generate or use existing tool call ID
	toolCallID := toolData.ToolCallID
	if toolCallID == "" {
		toolCallID = uuid.New().String()
		logger.Debug("Generated new tool call ID", map[string]interface{}{
			"tool_call_id": toolCallID,
		})
	}

	// Determine event type based on whether we have tool output
	eventType := "ToolCallStart"
	if len(toolData.ToolOutput) > 0 && string(toolData.ToolOutput) != "null" {
		eventType = "ToolCallEnd"
	}

	logger.Info("Creating AG-UI event", map[string]interface{}{
		"event_type":   eventType,
		"tool_call_id": toolCallID,
		"tool_name":    toolData.ToolName,
	})

	// Create event
	event := AgUIEvent{
		EventType: eventType,
		Data: EventData{
			ToolCallID:   toolCallID,
			ToolCallName: toolData.ToolName,
			RunID:        runID,
		},
	}

	if eventType == "ToolCallStart" && len(toolData.ToolInput) > 0 {
		event.Data.ToolInput = toolData.ToolInput
	}
	if eventType == "ToolCallEnd" && len(toolData.ToolOutput) > 0 {
		event.Data.ToolOutput = toolData.ToolOutput
	}

	// Handle batching with error handling
	var sendErr error
	if batchSize > 1 {
		logger.Debug("Attempting to add event to batch", map[string]interface{}{
			"batch_size": batchSize,
			"batch_file": batchFile,
		})
		sendErr = addToBatch(&event, batchSize, endpoint, logger)
		if sendErr != nil {
			logger.Warn("Failed to add event to batch, trying direct send", map[string]interface{}{
				"error": sendErr.Error(),
			})
			sendErr = sendEvent(endpoint, &event, logger)
		}
	} else {
		logger.Debug("Sending event directly (batch size = 1)", nil)
		sendErr = sendEvent(endpoint, &event, logger)
	}

	duration := time.Since(startTime)

	if sendErr != nil {
		logger.Error("Failed to send event", map[string]interface{}{
			"error":    sendErr.Error(),
			"duration": duration.String(),
		})
		recordFailure()
	} else {
		recordSuccess()
		logger.Info("Hook execution completed successfully", map[string]interface{}{
			"duration":     duration.String(),
			"event_type":   eventType,
			"tool_name":    toolData.ToolName,
			"tool_call_id": toolCallID,
		})
	}
}

func addToBatch(event *AgUIEvent, batchSize int, endpoint string, logger *Logger) error {
	// Read existing batch or create new one
	var events []AgUIEvent

	if data, err := os.ReadFile(batchFile); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			var e AgUIEvent
			if err := json.Unmarshal(scanner.Bytes(), &e); err == nil {
				events = append(events, e)
			}
		}
		logger.Debug("Loaded existing batch", map[string]interface{}{
			"existing_events": len(events),
		})
	}

	// Add current event to batch
	events = append(events, *event)
	logger.Debug("Added event to batch", map[string]interface{}{
		"batch_size":     len(events),
		"max_batch_size": batchSize,
	})

	// Check if batch is full
	if len(events) >= batchSize {
		logger.Info("Batch is full, sending batch", map[string]interface{}{
			"batch_size": len(events),
			"endpoint":   endpoint,
		})

		// Send batch
		if err := sendBatch(endpoint, events, logger); err != nil {
			return fmt.Errorf("failed to send batch: %w", err)
		}

		// Clear batch file
		os.Remove(batchFile)
		logger.Debug("Cleared batch file after successful send", nil)
	} else {
		// Write updated batch back to file
		file, err := os.Create(batchFile)
		if err != nil {
			return fmt.Errorf("failed to create batch file: %w", err)
		}
		defer file.Close()

		for _, e := range events {
			data, err := json.Marshal(e)
			if err != nil {
				continue
			}
			fmt.Fprintln(file, string(data))
		}

		logger.Debug("Updated batch file", map[string]interface{}{
			"events_in_batch": len(events),
			"batch_file":      batchFile,
		})
	}

	return nil
}

func sendBatch(endpoint string, events []AgUIEvent, logger *Logger) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	batchPayload := map[string]interface{}{
		"events": events,
	}

	data, err := json.Marshal(batchPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	logger.Debug("Sending batch HTTP request", map[string]interface{}{
		"endpoint":     endpoint,
		"event_count":  len(events),
		"payload_size": len(data),
	})

	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		logger.Error("HTTP request failed", map[string]interface{}{
			"error":    err.Error(),
			"endpoint": endpoint,
		})
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		logger.Error("HTTP error response", map[string]interface{}{
			"status_code": resp.StatusCode,
			"status":      resp.Status,
			"endpoint":    endpoint,
		})
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	logger.Info("Batch sent successfully", map[string]interface{}{
		"event_count": len(events),
		"status_code": resp.StatusCode,
		"endpoint":    endpoint,
	})
	return nil
}

func sendEvent(endpoint string, event *AgUIEvent, logger *Logger) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	logger.Debug("Sending single event HTTP request", map[string]interface{}{
		"endpoint":     endpoint,
		"event_type":   event.EventType,
		"payload_size": len(data),
	})

	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		logger.Error("HTTP request failed", map[string]interface{}{
			"error":    err.Error(),
			"endpoint": endpoint,
		})
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		logger.Error("HTTP error response", map[string]interface{}{
			"status_code": resp.StatusCode,
			"status":      resp.Status,
			"endpoint":    endpoint,
		})
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	logger.Info("Event sent successfully", map[string]interface{}{
		"event_type":  event.EventType,
		"status_code": resp.StatusCode,
		"endpoint":    endpoint,
	})
	return nil
}

func isCircuitBreakerOpen() bool {
	data, err := os.ReadFile(circuitBreakerFile)
	if err != nil {
		return false // No circuit breaker file means it's closed
	}

	var state CircuitBreakerState
	if err := json.Unmarshal(data, &state); err != nil {
		return false
	}

	// Reset circuit breaker after 30 seconds
	if state.IsOpen && time.Since(state.LastFailure) > 30*time.Second {
		state.IsOpen = false
		state.FailureCount = 0
		saveCircuitBreakerState(&state)
		return false
	}

	return state.IsOpen
}

func recordFailure() {
	var state CircuitBreakerState

	// Load existing state
	if data, err := os.ReadFile(circuitBreakerFile); err == nil {
		json.Unmarshal(data, &state)
	}

	state.FailureCount++
	state.LastFailure = time.Now()

	// Open circuit breaker after 5 failures
	if state.FailureCount >= 5 {
		state.IsOpen = true
	}

	saveCircuitBreakerState(&state)
}

func recordSuccess() {
	var state CircuitBreakerState

	// Load existing state
	if data, err := os.ReadFile(circuitBreakerFile); err == nil {
		json.Unmarshal(data, &state)
	}

	// Reset failure count on success
	state.FailureCount = 0
	state.IsOpen = false

	saveCircuitBreakerState(&state)
}

func saveCircuitBreakerState(state *CircuitBreakerState) {
	data, err := json.Marshal(state)
	if err != nil {
		return
	}

	os.WriteFile(circuitBreakerFile, data, 0644)
}

// sanitizeToolData removes sensitive information from tool data for logging
func sanitizeToolData(data json.RawMessage) interface{} {
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return string(data)[:min(len(data), 100)] + "..."
	}

	// If it's a map, sanitize sensitive fields
	if m, ok := parsed.(map[string]interface{}); ok {
		sanitized := make(map[string]interface{})
		for k, v := range m {
			key := strings.ToLower(k)
			if strings.Contains(key, "password") ||
				strings.Contains(key, "token") ||
				strings.Contains(key, "secret") ||
				strings.Contains(key, "key") {
				sanitized[k] = "[REDACTED]"
			} else {
				sanitized[k] = v
			}
		}
		return sanitized
	}

	return parsed
}

// getAvailableEnvVars returns a list of available ALPINE_* environment variables
func getAvailableEnvVars() []string {
	var alpineVars []string
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "ALPINE_") {
			alpineVars = append(alpineVars, env)
		}
	}
	return alpineVars
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
