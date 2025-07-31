package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// EventCollector collects events sent by the hook
type EventCollector struct {
	mu     sync.Mutex
	events []map[string]interface{}
}

func (ec *EventCollector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var event map[string]interface{}
	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ec.mu.Lock()
	ec.events = append(ec.events, event)
	ec.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

func (ec *EventCollector) GetEvents() []map[string]interface{} {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	result := make([]map[string]interface{}, len(ec.events))
	copy(result, ec.events)
	return result
}

// TestAgUIHookIntegration tests the hook script in a realistic scenario
func TestAgUIHookIntegration(t *testing.T) {
	// Find the hook script
	projectRoot := findProjectRoot(t)
	hookScript := filepath.Join(projectRoot, "hooks", "alpine-ag-ui-emitter.rs")

	// Verify hook script exists
	if _, err := os.Stat(hookScript); os.IsNotExist(err) {
		t.Skip("Hook script not found at", hookScript)
	}

	// Create event collector server
	collector := &EventCollector{}
	server := httptest.NewServer(collector)
	defer server.Close()

	// Test multiple tool invocations
	testCases := []struct {
		name     string
		toolData map[string]interface{}
		wantType string
	}{
		{
			name: "Write tool without output",
			toolData: map[string]interface{}{
				"tool_name": "Write",
				"tool_input": map[string]interface{}{
					"file_path": "/tmp/test.txt",
					"content":   "Hello, Alpine!",
				},
				"event":     "tool_use",
				"timestamp": time.Now().Format(time.RFC3339),
			},
			wantType: "ToolCallStart",
		},
		{
			name: "Bash tool with output",
			toolData: map[string]interface{}{
				"tool_name": "Bash",
				"tool_input": map[string]interface{}{
					"command": "echo 'test'",
				},
				"tool_output": map[string]interface{}{
					"stdout":    "test\n",
					"stderr":    "",
					"exit_code": 0,
				},
				"event":        "tool_use",
				"timestamp":    time.Now().Format(time.RFC3339),
				"tool_call_id": "bash-call-123",
			},
			wantType: "ToolCallEnd", // Should emit both Start and End
		},
		{
			name: "Read tool",
			toolData: map[string]interface{}{
				"tool_name": "Read",
				"tool_input": map[string]interface{}{
					"file_path": "/tmp/test.txt",
				},
				"event":     "tool_use",
				"timestamp": time.Now().Format(time.RFC3339),
			},
			wantType: "ToolCallStart",
		},
	}

	runID := "integration-test-run"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset collector
			collector.mu.Lock()
			collector.events = nil
			collector.mu.Unlock()

			// Execute hook
			toolJSON, _ := json.Marshal(tc.toolData)
			cmd := exec.Command(hookScript)
			cmd.Env = append(os.Environ(),
				"ALPINE_EVENTS_ENDPOINT="+server.URL,
				"ALPINE_RUN_ID="+runID,
			)
			cmd.Stdin = strings.NewReader(string(toolJSON))

			if err := cmd.Run(); err != nil {
				t.Fatalf("Hook script failed: %v", err)
			}

			// Give server time to receive events
			time.Sleep(50 * time.Millisecond)

			// Verify events
			events := collector.GetEvents()
			if len(events) == 0 {
				t.Fatal("No events received")
			}

			// Check first event
			firstEvent := events[0]
			if firstEvent["type"] != "ToolCallStart" {
				t.Errorf("Expected first event type 'ToolCallStart', got %v", firstEvent["type"])
			}

			// Verify data structure
			data, ok := firstEvent["data"].(map[string]interface{})
			if !ok {
				t.Fatal("Event data is not a map")
			}

			if data["toolCallName"] != tc.toolData["tool_name"] {
				t.Errorf("Expected toolCallName %v, got %v", tc.toolData["tool_name"], data["toolCallName"])
			}

			if data["runId"] != runID {
				t.Errorf("Expected runId %v, got %v", runID, data["runId"])
			}

			// If tool has output, verify End event
			if tc.toolData["tool_output"] != nil {
				if len(events) < 2 {
					t.Fatal("Expected ToolCallEnd event for tool with output")
				}

				endEvent := events[1]
				if endEvent["type"] != "ToolCallEnd" {
					t.Errorf("Expected second event type 'ToolCallEnd', got %v", endEvent["type"])
				}
			}
		})
	}
}

// TestAgUIHookPerformance tests hook performance under load
func TestAgUIHookPerformance(t *testing.T) {
	// Find the hook script
	projectRoot := findProjectRoot(t)
	hookScript := filepath.Join(projectRoot, "hooks", "alpine-ag-ui-emitter.rs")

	if _, err := os.Stat(hookScript); os.IsNotExist(err) {
		t.Skip("Hook script not found")
	}

	// Create a server that counts requests
	var requestCount int
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Run multiple hook invocations concurrently
	concurrency := 10
	iterations := 5

	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				toolData := map[string]interface{}{
					"tool_name": "TestTool",
					"tool_input": map[string]interface{}{
						"worker":    workerID,
						"iteration": j,
					},
					"event":     "tool_use",
					"timestamp": time.Now().Format(time.RFC3339),
				}

				toolJSON, _ := json.Marshal(toolData)
				cmd := exec.Command(hookScript)
				cmd.Env = append(os.Environ(),
					"ALPINE_EVENTS_ENDPOINT="+server.URL,
					"ALPINE_RUN_ID=perf-test",
				)
				cmd.Stdin = strings.NewReader(string(toolJSON))

				if err := cmd.Run(); err != nil {
					t.Errorf("Worker %d iteration %d failed: %v", workerID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Verify all events were sent
	expectedEvents := concurrency * iterations
	if requestCount != expectedEvents {
		t.Errorf("Expected %d events, got %d", expectedEvents, requestCount)
	}

	// Check performance
	avgTimePerHook := elapsed / time.Duration(expectedEvents)
	t.Logf("Performance: %d hooks in %v (avg: %v per hook)", expectedEvents, elapsed, avgTimePerHook)

	// Warn if too slow
	if avgTimePerHook > time.Second {
		t.Errorf("Hook is too slow: %v per invocation", avgTimePerHook)
	}
}

// findProjectRoot finds the alpine project root directory
func findProjectRoot(t *testing.T) string {
	t.Helper()

	// Start from test directory and go up
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for {
		// Check if go.mod exists
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		// Go up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find project root")
		}
		dir = parent
	}
}
