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
	"path/filepath"
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

func main() {
	// Check circuit breaker first
	if isCircuitBreakerOpen() {
		fmt.Fprintln(os.Stderr, "Circuit breaker is open, skipping hook execution")
		return
	}

	// Read tool data from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
		recordFailure()
		return
	}

	// Parse the JSON data
	var toolData ToolData
	if err = json.Unmarshal(input, &toolData); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse tool data: %v\n", err)
		recordFailure()
		return
	}

	// Log that hook was called
	fmt.Fprintf(os.Stderr, "HOOK CALLED: tool=%s\n", toolData.ToolName)

	// Get environment variables
	endpoint := os.Getenv("ALPINE_EVENTS_ENDPOINT")
	if endpoint == "" {
		fmt.Fprintln(os.Stderr, "ALPINE_EVENTS_ENDPOINT not set, skipping event emission")
		return
	}

	runID := os.Getenv("ALPINE_RUN_ID")
	if runID == "" {
		runID = "unknown"
	}

	batchSize := 10
	if bs := os.Getenv("ALPINE_TOOL_CALL_BATCH_SIZE"); bs != "" {
		fmt.Sscanf(bs, "%d", &batchSize)
	}

	sampleRate := 100
	if sr := os.Getenv("ALPINE_TOOL_CALL_SAMPLE_RATE"); sr != "" {
		fmt.Sscanf(sr, "%d", &sampleRate)
	}

	// Apply sampling - skip event if random number is above sample rate
	if sampleRate < 100 {
		rand.Seed(time.Now().UnixNano())
		randomValue := rand.Intn(100) + 1
		if randomValue > sampleRate {
			fmt.Fprintf(os.Stderr, "Event sampled out (%d%% rate)\n", sampleRate)
			return
		}
	}

	// Generate or use existing tool call ID
	toolCallID := toolData.ToolCallID
	if toolCallID == "" {
		toolCallID = uuid.New().String()
	}

	// Determine event type based on whether we have tool output
	eventType := "ToolCallStart"
	if len(toolData.ToolOutput) > 0 && string(toolData.ToolOutput) != "null" {
		eventType = "ToolCallEnd"
	}

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
		sendErr = addToBatch(&event, batchSize, endpoint)
		if sendErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to add event to batch: %v, trying direct send\n", sendErr)
			sendErr = sendEvent(endpoint, &event)
		}
	} else {
		sendErr = sendEvent(endpoint, &event)
	}

	if sendErr != nil {
		fmt.Fprintf(os.Stderr, "Failed to send event: %v\n", sendErr)
		recordFailure()
	} else {
		recordSuccess()
		fmt.Fprintln(os.Stderr, "Event sent successfully")
	}
}

func addToBatch(event *AgUIEvent, batchSize int, endpoint string) error {
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
	}

	// Add current event to batch
	events = append(events, *event)

	// Check if batch is full
	if len(events) >= batchSize {
		// Send batch
		if err := sendBatch(endpoint, events); err != nil {
			return fmt.Errorf("failed to send batch: %w", err)
		}

		// Clear batch file
		os.Remove(batchFile)
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
	}

	return nil
}

func sendBatch(endpoint string, events []AgUIEvent) error {
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

	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	fmt.Fprintf(os.Stderr, "Sent batch of %d events\n", len(events))
	return nil
}

func sendEvent(endpoint string, event *AgUIEvent) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

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

	if data, err := os.ReadFile(circuitBreakerFile); err == nil {
		json.Unmarshal(data, &state)
	}

	// Reset failure count on success
	if state.FailureCount > 0 {
		state.FailureCount = 0
		state.IsOpen = false
		saveCircuitBreakerState(&state)
	}
}

func saveCircuitBreakerState(state *CircuitBreakerState) {
	data, err := json.Marshal(state)
	if err != nil {
		return
	}

	dir := filepath.Dir(circuitBreakerFile)
	os.MkdirAll(dir, 0755)
	os.WriteFile(circuitBreakerFile, data, 0644)
}
