# Implementation Plan: Alpine HTTP Server with ag-ui Protocol Events

## Overview

This plan details the implementation of an HTTP server mode for Alpine that emits ag-ui protocol events for UI integration. The HTTP server will run alongside Alpine's workflow engine and emit real-time events about task lifecycle, tool usage, and state changes.

The solution prioritizes simplicity: using Claude Code hooks for tool events and direct emission for lifecycle events, with minimal changes to Alpine's core.

## Prioritized Features

### P0: Core HTTP Server and Event Infrastructure

#### Task 1: Create EventEmitter Interface and Mock ✅ [IMPLEMENTED: 2025-07-27]
**TDD Cycle:** Write tests requiring EventEmitter interface before implementation.

**Acceptance Criteria:**
- EventEmitter interface defined in `internal/events/emitter.go` ✓
- Interface has methods for lifecycle events ✓
- Mock implementation for testing ✓
- No-op implementation for CLI mode ✓

**Test Cases:**
```go
// Test mock emitter records method calls
// Test no-op emitter handles calls without side effects
// Test interface can be injected into workflow engine
```

**Implementation:**
1. Create `internal/events/` package
2. Define EventEmitter interface with RunStarted, RunFinished, RunError methods
3. Create mock and no-op implementations

#### Task 2: Add HTTP Server Configuration ✅ [IMPLEMENTED: 2025-07-27]
**TDD Cycle:** Test configuration loading for new HTTP server settings.

**Acceptance Criteria:**
- Add `HTTPEnabled` and `HTTPPort` to config.Config ✓
- Environment variables: `ALPINE_HTTP_ENABLED`, `ALPINE_HTTP_PORT` ✓
- Default: disabled, port 8080 ✓

**Test Cases:**
```go
// Test default values (disabled, port 8080) ✓
// Test environment variable parsing ✓
// Test invalid port handling ✓
```

**Implementation:**
1. Update `internal/config/config.go` ✓
2. Add parsing for new environment variables ✓
3. Update config tests ✓

#### Task 3: Implement Basic HTTP Server ✅ [IMPLEMENTED: 2025-07-27]
**TDD Cycle:** Test HTTP server endpoints respond correctly.

**Acceptance Criteria:**
- HTTP server with /runs endpoint ✓
- Start/stop lifecycle management ✓
- JSON request/response handling ✓
- Graceful shutdown ✓

**Test Cases:**
```go
// Test POST /runs starts a new run ✓
// Test GET /runs/{id}/status returns status ✓
// Test server starts and stops cleanly ✓
```

**Implementation:**
1. Create `internal/server/server.go` ✓
2. Implement basic HTTP handlers ✓
3. Add run tracking with unique IDs ✓

### P1: Event Emission to UI Endpoint

#### Task 4: Implement Event Posting to UI
**TDD Cycle:** Test events are POSTed to configured endpoint.

**Acceptance Criteria:**
- HTTP client posts events to UI endpoint
- Events follow ag-ui protocol format
- Handles connection failures gracefully

**Test Cases:**
```go
// Test event formatting matches ag-ui spec
// Test POST to endpoint with retry
// Test handles endpoint unavailable
```

**Implementation:**
1. Create `internal/events/client.go`
2. Implement PostEvent method
3. Add retry logic with backoff

#### Task 5: Integrate EventEmitter with Workflow Engine
**TDD Cycle:** Test workflow engine calls emitter at lifecycle points.

**Acceptance Criteria:**
- Workflow engine accepts optional EventEmitter
- Emits RunStarted at beginning
- Emits RunFinished on success
- Emits RunError on failure

**Test Cases:**
```go
// Test RunStarted called on workflow start
// Test RunFinished called on completion
// Test RunError called on failure
// Test works with nil emitter
```

**Implementation:**
1. Modify `internal/workflow/workflow.go`
2. Add EventEmitter field to Engine
3. Call emitter at appropriate points

### P2: Claude Code Hook Integration

#### Task 6: Create PostToolUse Hook for Tool Events
**TDD Cycle:** Test hook script generates correct events.

**Acceptance Criteria:**
- Rust hook script captures tool usage
- Formats as ag-ui ToolCallStart/End events
- Posts to configured endpoint

**Test Cases:**
```go
// Test hook parses tool data correctly
// Test generates valid ag-ui events
// Test posts to endpoint from environment
```

**Implementation:**
1. Create `hooks/alpine-ag-ui-emitter.rs`
2. Parse tool data and format events
3. POST to ALPINE_EVENTS_ENDPOINT

#### Task 7: Configure Claude with Hooks
**TDD Cycle:** Test Claude settings include hooks when HTTP mode enabled.

**Acceptance Criteria:**
- Generate .claude/settings.json when HTTP enabled
- Include PostToolUse hook configuration with full absolute path
- Resolve hook script path to absolute path before writing config
- Pass event endpoint via environment

**Test Cases:**
```go
// Test settings.json created correctly
// Test hook path is resolved to absolute path
// Test hook script exists at specified path
// Test environment variables set
```

**Implementation:**
1. Add hook configuration logic to server
2. Resolve hook script to absolute path using filepath.Abs()
3. Write settings.json with full path before Claude execution
4. Set required environment variables

**Example settings.json:**
```json
{
  "hooks": {
    "PostToolUse": [{
      "matcher": ".*",
      "hooks": [{
        "type": "command",
        "command": "/absolute/path/to/alpine/hooks/alpine-ag-ui-emitter.rs"
      }]
    }]
  }
}
```

### P3: State Monitoring and Advanced Events

#### Task 8: Monitor agent_state.json Changes
**TDD Cycle:** Test state changes trigger StateSnapshot events.

**Acceptance Criteria:**
- Watch agent_state.json for changes
- Emit StateSnapshot on updates
- Include full state in event

**Test Cases:**
```go
// Test detects file changes
// Test emits correct event format
// Test handles missing file
```

**Implementation:**
1. Add file watcher to workflow engine
2. Emit StateSnapshot on changes
3. Handle file not found gracefully

## Testing Strategy

### Unit Tests
Each task includes unit tests with mocks to verify component behavior in isolation.

### Integration Tests (`test/integration/`)

#### Test 1: HTTP Server Event Collection
**Purpose:** Verify HTTP server receives and forwards events correctly.

**Test Setup:**
```go
// Start a mock UI server that logs received events
mockUI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Log event to file for verification
    logEvent(r.Body)
}))

// Start Alpine HTTP server pointing to mock UI
server := StartAlpineServer(mockUI.URL)

// Execute a simple workflow
response := POST("/runs", {"task": "echo test", "eventEndpoint": mockUI.URL})
```

**Verification:**
- Check log file contains RunStarted event
- Verify ToolCallStart for echo command
- Confirm RunFinished event received
- Validate event format matches ag-ui spec

#### Test 2: Claude Hook Integration
**Purpose:** Verify PostToolUse hook captures and emits tool events.

**Test Setup:**
```go
// Create test hook script that logs to file
writeTestHook(`
    log_to_file("HOOK_CALLED", $TOOL_NAME, $TOOL_INPUT)
    post_event_to_endpoint($ALPINE_EVENTS_ENDPOINT)
`)

// Run workflow with hook enabled
RunWorkflow(WithHooks(true), WithEventEndpoint(mockUI.URL))
```

**Verification:**
- Check hook log file shows it was called
- Verify tool events in UI server log
- Confirm event ordering is correct

### End-to-End Tests (`test/e2e/`)

#### Test 1: Real Workflow Event Stream E2E
**Purpose:** Verify a UI can track Alpine's progress through events during a realistic workflow.

**Test Scenario:** 
"As a UI developer, I want to receive real-time events when Alpine creates a new function with tests"

**Test Setup:**
```go
// 1. Start mock UI server that simulates a real UI
mockUI := StartMockUI(t)
defer mockUI.Shutdown()

// 2. Start Alpine HTTP server
server := StartAlpineHTTPServer(t, 8080)
defer server.Shutdown()

// 3. Create a realistic workflow request
runRequest := RunRequest{
    Task: "Create a function isPrime with unit tests",
    EventEndpoint: mockUI.EventsEndpoint(),
}

// 4. Execute workflow (uses mock Claude that simulates real tool usage)
runID := server.StartRun(runRequest)

// 5. Wait for completion or timeout
mockUI.WaitForCompletion(30 * time.Second)
```

**Mock Claude Behavior:**
```bash
#!/bin/bash
# Mock claude script that simulates real workflow
case "$*" in
  *"Create a function"*)
    echo "I'll create the isPrime function with tests."
    echo "[1] Write: main.go"
    echo "function isPrime(n) { ... }"
    echo "[2] Write: main_test.go" 
    echo "func TestIsPrime(t *testing.T) { ... }"
    echo "[3] Bash: go test"
    echo "PASS: TestIsPrime"
    echo '{"status":"completed"}' > agent_state.json
    ;;
esac
```

**Verification - What a Real UI Would Need:**
```go
// Verify UI received actionable events
events := mockUI.ReceivedEvents()

// 1. UI knows workflow started
assert.Equal(t, "RunStarted", events[0].Type)
runID := events[0].Data["runId"]

// 2. UI can show which tools are being used
toolEvents := filterByType(events, "ToolCallStart")
assert.Contains(t, toolNames(toolEvents), "Write")  // Creating files
assert.Contains(t, toolNames(toolEvents), "Bash")   // Running tests

// 3. UI can track state changes
stateEvents := filterByType(events, "StateSnapshot")
lastState := stateEvents[len(stateEvents)-1]
assert.Equal(t, "completed", lastState.Data["snapshot"]["status"])

// 4. UI knows workflow completed successfully
finalEvent := events[len(events)-1]
assert.Equal(t, "RunFinished", finalEvent.Type)
assert.Equal(t, runID, finalEvent.Data["runId"])

// 5. Verify event timing and order makes sense
assert.True(t, eventsAreOrdered(events))
assert.True(t, eventsHaveReasonableTimings(events))
```

#### Test 2: UI Progress Tracking E2E
**Purpose:** Verify a UI can display meaningful progress during long-running workflows.

**Test Scenario:**
"As a UI developer, I want to show users what Alpine is currently doing"

**Key Verifications:**
```go
// Mock UI builds a progress view from events
progressView := mockUI.BuildProgressView()

// Verify UI can show:
assert.Contains(t, progressView, "Creating isPrime function")    // From StateSnapshot
assert.Contains(t, progressView, "Writing main.go")              // From ToolCallStart
assert.Contains(t, progressView, "Running tests")                // From ToolCallStart
assert.Contains(t, progressView, "✓ Tests passed")               // From ToolCallResult
assert.Contains(t, progressView, "Workflow completed")           // From RunFinished

// Verify progress is meaningful to end users
assert.True(t, progressView.ShowsCurrentActivity())
assert.True(t, progressView.ShowsOverallProgress())
```

#### Test 3: Critical Feature Validation E2E
**Purpose:** Verify the core feature works end-to-end in realistic conditions.

**Test 1: Happy Path - Complete Workflow with Events**
```go
func TestHTTPServerEmitsEventsE2E(t *testing.T) {
    // 1. Build Alpine HTTP server binary
    alpine := BuildBinary(t, "alpine-server")
    
    // 2. Create the ag-ui event hook script
    hookScript := filepath.Join(t.TempDir(), "alpine-ag-ui-hook.rs")
    WriteFile(t, hookScript, `#!/usr/bin/env rust-script
//! \`\`\`cargo
//! [dependencies]
//! serde_json = "1.0"
//! reqwest = { version = "0.11", features = ["blocking", "json"] }
//! \`\`\`

use std::env;
use std::io::{self, Read};
use serde_json::{json, Value};

fn main() -> io::Result<()> {
    let mut input = String::new();
    io::stdin().read_to_string(&mut input)?;
    let data: Value = serde_json::from_str(&input)?;
    
    // Log to verify hook was called
    eprintln!("HOOK CALLED: tool={}", data["tool_name"]);
    
    // Post event to endpoint
    let endpoint = env::var("ALPINE_EVENTS_ENDPOINT").unwrap();
    let event = json!({
        "type": "ToolCallStart",
        "data": {
            "toolCallId": "test-123",
            "toolCallName": data["tool_name"],
            "runId": env::var("ALPINE_RUN_ID").unwrap()
        }
    });
    
    reqwest::blocking::Client::new()
        .post(&endpoint)
        .json(&event)
        .send()
        .ok();
    
    Ok(())
}`)
    os.Chmod(hookScript, 0755)
    
    // 3. Create Claude settings to use the hook
    settingsDir := filepath.Join(t.TempDir(), ".claude")
    os.MkdirAll(settingsDir, 0755)
    WriteJSON(t, filepath.Join(settingsDir, "settings.json"), map[string]interface{}{
        "hooks": map[string]interface{}{
            "PostToolUse": []map[string]interface{}{
                {
                    "matcher": ".*",
                    "hooks": []map[string]interface{}{
                        {"type": "command", "command": hookScript},
                    },
                },
            },
        },
    })
    
    // 4. Start event collector (simulates UI)
    collector := StartEventCollector(t, 9090)
    defer collector.Stop()
    
    // 5. Start Alpine HTTP server with hook settings
    cmd := exec.Command(alpine)
    cmd.Env = append(os.Environ(), 
        "ALPINE_HTTP_PORT=8080",
        "HOME=" + filepath.Dir(settingsDir), // So Claude finds .claude/settings.json
    )
    StartServer(t, cmd)
    defer cmd.Process.Kill()
    
    // 6. Execute a real workflow
    resp := HTTPPost(t, "http://localhost:8080/runs", map[string]string{
        "task": "Create a hello.txt file with 'Hello Alpine!' content",
        "eventEndpoint": "http://localhost:9090/events",
    })
    runID := resp["runId"]
    
    // 7. Wait for completion (max 30s)
    collector.WaitForEvent("RunFinished", runID, 30*time.Second)
    
    // 8. Verify we got the critical events
    events := collector.GetEventsForRun(runID)
    
    // Must have lifecycle events from Alpine
    assert.True(t, HasEvent(events, "RunStarted"))
    assert.True(t, HasEvent(events, "RunFinished"))
    
    // Must have tool events from Claude hook
    toolEvents := FilterEvents(events, "ToolCallStart")
    assert.Greater(t, len(toolEvents), 0, "Should have tool events from hooks")
    
    // Verify the Write tool was called (creating hello.txt)
    writeEvents := FilterByToolName(toolEvents, "Write")
    assert.Greater(t, len(writeEvents), 0, "Hook should capture Write tool usage")
    
    // Verify file was actually created
    content, _ := os.ReadFile("hello.txt")
    assert.Contains(t, string(content), "Hello Alpine!")
}
```

**Test 2: Error Case - Workflow Failure**
```go
func TestHTTPServerEmitsErrorEventOnFailure(t *testing.T) {
    // Setup same as happy path...
    
    // Execute a workflow that will fail
    resp := HTTPPost(t, "http://localhost:8080/runs", map[string]string{
        "task": "FAIL: This task will cause an error",
        "eventEndpoint": "http://localhost:9090/events",
    })
    runID := resp["runId"]
    
    // Wait for error event
    collector.WaitForEvent("RunError", runID, 10*time.Second)
    
    // Verify error event contains useful information
    events := collector.GetEventsForRun(runID)
    errorEvent := FindEvent(events, "RunError")
    assert.NotNil(t, errorEvent)
    assert.Contains(t, errorEvent.Data["message"], "error")
}
```

**Event Collector Helper (Simple Version):**
```go
type EventCollector struct {
    port   int
    events sync.Map // runID -> []Event
    server *http.Server
}

func StartEventCollector(t *testing.T, port int) *EventCollector {
    c := &EventCollector{port: port}
    
    // Simple HTTP server that logs POSTed events
    mux := http.NewServeMux()
    mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
        var event AgUIEvent
        json.NewDecoder(r.Body).Decode(&event)
        
        runID := event.Data["runId"].(string)
        c.appendEvent(runID, event)
        
        w.WriteHeader(http.StatusOK)
    })
    
    c.server = &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux}
    go c.server.ListenAndServe()
    
    // Wait for server to be ready
    WaitForHTTP(t, fmt.Sprintf("http://localhost:%d", port))
    
    return c
}

func (c *EventCollector) WaitForEvent(eventType, runID string, timeout time.Duration) {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if c.HasEvent(runID, eventType) {
            return
        }
        time.Sleep(100 * time.Millisecond)
    }
    panic(fmt.Sprintf("Timeout waiting for %s event", eventType))
}
```

### Observability Helpers

#### Event Logger Utility
Create `test/utils/event-logger.go`:
```go
// Simple HTTP server that logs all received events
func StartEventLogger(port int, logFile string) *EventLogger {
    // Logs all POST requests to file with timestamp
    // Returns server instance for shutdown
}
```

#### Hook Verification
Create `test/utils/hook-verifier.rs`:
```rust
// Test hook that logs calls and validates data
// Writes to HOOK_VERIFICATION_LOG with details
```

### Test Execution Commands
```bash
# Run all integration tests with event verification
make test-integration-events

# Run E2E tests with full event flow
make test-e2e-events

# View event logs from last test run
cat test/logs/last-events.log | jq .
```

## Success Criteria

- [ ] HTTP server starts with `alpine-server` command
- [ ] POST /runs executes Alpine workflow
- [ ] RunStarted event emitted when workflow begins
- [ ] ToolCallStart/End events emitted via hooks
- [ ] StateSnapshot events on state changes
- [ ] RunFinished/RunError on completion
- [ ] All events POST to UI's endpoint
- [ ] CLI mode unchanged (no events)
- [ ] Minimal dependencies added
- [ ] Comprehensive test coverage
- [ ] Integration tests verify real event emission
- [ ] E2E tests confirm full workflow with observable logs
- [ ] Event logs can be inspected for debugging