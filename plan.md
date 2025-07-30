# Real-Time Claude Code Output Streaming Implementation Plan

## Overview

Implement real-time streaming of Claude Code's stdout output to frontend applications via Server-Sent Events (SSE) using Alpine's existing infrastructure. The implementation extends the current WorkflowEvent system to support AG-UI protocol-compliant streaming while maintaining backward compatibility.

## Key Objectives

- Stream Claude Code output in real-time to connected frontend clients
- Maintain full backward compatibility with existing CLI workflows  
- Leverage existing SSE infrastructure and event broadcasting system
- Follow TDD principles with comprehensive test coverage
- Support AG-UI protocol event format for frontend integration

## Feature Priority Breakdown

### P0: Core Streaming Infrastructure

Essential functionality for basic streaming capability.

#### Feature 1: Streamer Interface and Server Implementation ✅ IMPLEMENTED

**Acceptance Criteria:**
- ✅ Streamer interface defines clean contract for streaming operations
- ✅ Server implements Streamer interface using existing BroadcastEvent infrastructure
- ✅ No-op implementation available for non-streaming mode
- ✅ Thread-safe streaming to multiple concurrent clients

**Implementation Date**: 2025-07-30

**TDD Cycle:**

*Test Cases:*
```go
// internal/events/streamer_test.go
func TestStreamerInterface(t *testing.T) {
    // Verify interface methods are called correctly
    // Test streaming lifecycle (Start -> Content -> End)
    // Validate error handling scenarios
}

// internal/server/streamer_integration_test.go  
func TestServerStreamerImplementation(t *testing.T) {
    // Test server broadcasts streaming events via SSE
    // Verify WorkflowEvent format matches streaming requirements
    // Test concurrent client streaming
}
```

*Implementation Steps:*
1. Create `internal/events/streamer.go` with interface definition
2. Create `internal/server/streamer.go` with server implementation
3. Add NoOpStreamer for backward compatibility
4. Integrate with existing BroadcastEvent method

*Integration Points:*
- Extends existing `internal/server/server.go` BroadcastEvent functionality
- Uses existing WorkflowEvent structure with new event types

#### Feature 2: Claude Executor Streaming Integration ✅ IMPLEMENTED

**Acceptance Criteria:**
- ✅ Claude executor accepts Streamer interface via dependency injection
- ✅ stdout content is streamed in real-time during execution
- ✅ Complete stdout still returned as string for CLI compatibility
- ✅ Streaming errors are handled gracefully without failing execution

**Implementation Date**: 2025-07-30

**TDD Cycle:**

*Test Cases:*
```go
// internal/claude/executor_stream_test.go
func TestExecutorStreaming(t *testing.T) {
    // Test stdout streaming with mock streamer
    // Verify backward compatibility (returns complete stdout)
    // Test streaming disabled when no streamer provided
    // Test error handling when streaming fails
}
```

*Implementation Steps:*
1. Add Streamer field to claude.Executor struct
2. Modify executeWithStderrCapture to use io.MultiWriter
3. Create StreamWriter helper to bridge Streamer interface
4. Add streaming lifecycle calls (Start -> Content chunks -> End)

*Integration Points:*
- Modifies existing `internal/claude/executor.go` executeWithStderrCapture method
- Uses existing stdout/stderr capture infrastructure

#### Feature 3: Workflow Integration and Dependency Passing ✅ IMPLEMENTED

**Acceptance Criteria:**
- ✅ Server instance passed as Streamer through workflow chain
- ✅ Workflow engine propagates Streamer to Claude executor
- ✅ Run ID tracking for stream correlation
- ✅ Non-server mode continues to work without streaming

**Implementation Date**: 2025-07-30

**TDD Cycle:**

*Test Cases:*
```go
// internal/workflow/workflow_stream_test.go
func TestWorkflowStreaming(t *testing.T) {
    // Test streamer passed correctly to executor
    // Verify run ID correlation
    // Test workflow without streamer (backward compatibility)
}

// internal/cli/workflow_integration_test.go
func TestCLIStreamerIntegration(t *testing.T) {
    // Test server passed as streamer in serve mode
    // Test no streamer in CLI-only mode
}
```

*Implementation Steps:*
1. Update workflow.Engine constructor to accept Streamer
2. Modify CLI layer to pass server as Streamer when --serve flag used
3. Update executor creation to include Streamer and run ID
4. Add proper nil checks for backward compatibility

*Integration Points:*
- Updates `internal/workflow/workflow.go` engine creation
- Modifies `internal/cli/workflow.go` server integration
- Uses existing dependency injection patterns

### P1: Enhanced Event Support

Improved streaming capabilities and event formatting.

#### Feature 4: AG-UI Protocol Compliance ✅ IMPLEMENTED

**Acceptance Criteria:**
- ✅ WorkflowEvent structure supports all required AG-UI event types and fields
- ✅ Strict event sequencing: RunStarted → TextMessage* → RunFinished
- ✅ Proper camelCase field naming (runId, messageId) per AG-UI spec
- ✅ Complete event lifecycle management with proper ID correlation
- ✅ JSON schema validation for all AG-UI event types

**Implementation Date**: 2025-07-30

**AG-UI Event Types Required:**
```go
// Lifecycle Events
"run_started"     // First event when Alpine workflow begins
"run_finished"    // Successful workflow completion
"run_error"       // Workflow failure

// Text Streaming Events (Claude output)
"text_message_start"    // Begin Claude stdout stream
"text_message_content"  // Claude stdout chunks (delta=true)
"text_message_end"      // Complete Claude stdout stream
```

**Required WorkflowEvent Structure:**
```go
type WorkflowEvent struct {
    Type      string      `json:"type"`                    // AG-UI event type
    RunID     string      `json:"runId"`                   // camelCase per spec
    MessageID string      `json:"messageId,omitempty"`     // For text correlation
    Timestamp time.Time   `json:"timestamp"`               // ISO 8601 format
    
    // AG-UI streaming fields
    Content   string      `json:"content,omitempty"`       // Text chunks
    Delta     bool        `json:"delta,omitempty"`         // Incremental flag
    Source    string      `json:"source,omitempty"`        // "claude"
    Complete  bool        `json:"complete,omitempty"`      // End marker
    
    // Flexible event data
    Metadata  interface{} `json:"metadata,omitempty"`      // RunStarted data
    Result    interface{} `json:"result,omitempty"`        // RunFinished data
    Error     interface{} `json:"error,omitempty"`         // RunError data
}
```

**TDD Cycle:**

*Test Cases:*
```go
// internal/server/agui_compliance_test.go
func TestAGUIEventSequencing(t *testing.T) {
    // Test mandatory RunStarted → RunFinished sequence
    // Verify TextMessage Start → Content* → End lifecycle
    // Test messageId correlation across events
}

func TestAGUIFieldValidation(t *testing.T) {
    // Test camelCase field naming (runId not run_id)
    // Verify required fields present per event type
    // Test timestamp ISO 8601 format
    // Validate source="claude" for Claude output
}

func TestAGUIJSONSerialization(t *testing.T) {
    // Test JSON marshaling matches AG-UI schema
    // Verify optional fields omitted when empty
    // Test event type strings exact match
}
```

*Implementation Steps:*
1. Update `internal/server/interfaces.go` WorkflowEvent with AG-UI fields
2. Create AG-UI event type constants in `internal/events/agui_types.go`
3. Add message ID generation using existing `GenerateID("msg")` pattern
4. Implement strict event sequencing validation
5. Update all event emission points for AG-UI compliance
6. Add JSON schema validation tests

**Event Sequencing Implementation:**
```go
// When workflow starts
BroadcastEvent(WorkflowEvent{
    Type: "run_started",
    RunID: runID,
    Timestamp: time.Now(),
    Metadata: map[string]interface{}{
        "task": taskDescription,
        "worktreeDir": worktreePath,
    },
})

// When Claude execution begins
messageID := GenerateID("msg")
BroadcastEvent(WorkflowEvent{
    Type: "text_message_start",
    RunID: runID,
    MessageID: messageID,
    Timestamp: time.Now(),
    Source: "claude",
})

// For each Claude stdout chunk
BroadcastEvent(WorkflowEvent{
    Type: "text_message_content",
    RunID: runID,
    MessageID: messageID,
    Timestamp: time.Now(),
    Content: chunk,
    Delta: true,
    Source: "claude",
})

// When Claude completes
BroadcastEvent(WorkflowEvent{
    Type: "text_message_end",
    RunID: runID,
    MessageID: messageID,
    Timestamp: time.Now(),
    Complete: true,
    Source: "claude",
})
```

*Integration Points:*
- Extends existing `internal/server/interfaces.go` WorkflowEvent
- Uses existing `GenerateID()` from `internal/server/models.go`
- Updates workflow event emission in `internal/server/workflow_integration.go`
- Maintains SSE compatibility with existing `BroadcastEvent()` method

### P2: Production Readiness

Performance, reliability, and operational improvements.

#### Feature 5: Error Handling and Performance Optimization ✅ IMPLEMENTED

**Acceptance Criteria:**
- ✅ Streaming failures don't crash Claude execution
- ✅ Proper resource cleanup on client disconnects
- ✅ Configurable streaming buffer sizes
- ✅ Memory usage bounded per streaming client

**Implementation Date**: 2025-07-30

**TDD Cycle:**

*Test Cases:*
```go
// internal/server/streaming_performance_test.go
func TestStreamingPerformance(t *testing.T) {
    // Test concurrent client streaming
    // Verify memory usage bounds
    // Test large stdout streaming
}
```

*Implementation Steps:*
1. Add error recovery in streaming pipeline
2. Implement client disconnect detection
3. Add configuration for streaming buffer sizes
4. Memory usage monitoring and bounds

*Integration Points:*
- Enhances existing server client management
- Uses existing configuration system

## Success Criteria Checklist

### Functional Requirements
- [x] Real-time stdout streaming during Claude execution
- [x] SSE delivery to connected frontend clients via `/runs/{id}/events`
- [x] Complete backward compatibility with CLI-only usage
- [x] Proper streaming lifecycle management (start/content/end events)
- [x] Run ID correlation between streams and REST API

### Technical Requirements  
- [x] Streamer interface properly abstracts streaming concerns
- [x] Server implements streaming via existing BroadcastEvent infrastructure
- [x] Claude executor streams without breaking existing stdout return
- [x] AG-UI protocol compliant event formatting
- [x] Thread-safe concurrent streaming operations

### Quality Requirements
- [ ] 100% backward compatibility - existing CLI workflows unchanged
- [ ] Test coverage >80% for all new streaming components
- [ ] No memory leaks during long-running streaming sessions
- [ ] Graceful degradation when streaming fails
- [ ] Clean separation of concerns with existing architecture

## AG-UI Protocol Compliance Guide

### AG-UI Documentation Resources

**Primary AG-UI Protocol Repository:**
- **Official Specification**: https://github.com/ag-ui-protocol/ag-ui
- **Event Schema Documentation**: https://github.com/ag-ui-protocol/ag-ui/blob/main/README.md
- **TypeScript Type Definitions**: https://github.com/ag-ui-protocol/ag-ui/blob/main/src/types.ts

**Relevant Alpine Infrastructure:**
- **Server SSE Implementation**: [specs/server.md](specs/server.md) - Existing SSE infrastructure
- **Event Broadcasting**: [internal/server/server.go](internal/server/server.go) - Current BroadcastEvent method
- **WorkflowEvent Structure**: [internal/server/interfaces.go](internal/server/interfaces.go) - Current event format

### Critical Compliance Requirements

**Event Type Naming**: Must use exact strings per AG-UI spec
- ✅ `"run_started"` not `"workflow_started"` 
- ✅ `"text_message_content"` not `"claude_output"`
- ✅ `"run_finished"` not `"workflow_completed"`

**Field Naming Convention**: camelCase per AG-UI JSON schema
- ✅ `"runId"` not `"run_id"`
- ✅ `"messageId"` not `"message_id"`
- ✅ `"timestamp"` not `"created_at"`

**Event Sequencing Rules**: Strict ordering required
1. **RunStarted** must be absolute first event
2. **TextMessageStart** begins each Claude output stream
3. **TextMessageContent** events use same messageId as Start
4. **TextMessageEnd** completes stream with same messageId
5. **RunFinished/RunError** must be absolute last event

**Message ID Correlation**: Critical for frontend state management
```go
// Generate once per Claude execution
messageID := GenerateID("msg") // "msg-a1b2c3d4"

// Use same ID for entire message lifecycle
StreamStart(runID, messageID)    // text_message_start
StreamContent(runID, messageID, chunk1) // text_message_content
StreamContent(runID, messageID, chunk2) // text_message_content  
StreamEnd(runID, messageID)      // text_message_end
```

**Source Attribution**: Required for multi-agent environments
- All Claude output events must include `"source": "claude"`
- Enables frontend to distinguish between different agent outputs
- Future-proofs for multi-agent Alpine workflows

### JSON Schema Validation

**Event Format Examples**:
```json
// RunStarted - Workflow begins
{
  "type": "run_started",
  "runId": "run-abc123",
  "timestamp": "2024-01-01T12:00:00Z",
  "metadata": {
    "task": "Implement user authentication",
    "worktreeDir": "/path/to/alpine_agent_state",
    "planMode": false
  }
}

// TextMessageStart - Claude output begins  
{
  "type": "text_message_start",
  "runId": "run-abc123",
  "messageId": "msg-def456",
  "timestamp": "2024-01-01T12:01:00Z",
  "source": "claude"
}

// TextMessageContent - Claude stdout chunk
{
  "type": "text_message_content",
  "runId": "run-abc123", 
  "messageId": "msg-def456",
  "timestamp": "2024-01-01T12:01:01Z",
  "content": "I'll help you implement user authentication. Let me start by...",
  "delta": true,
  "source": "claude"
}

// TextMessageEnd - Claude output complete
{
  "type": "text_message_end",
  "runId": "run-abc123",
  "messageId": "msg-def456", 
  "timestamp": "2024-01-01T12:01:30Z",
  "complete": true,
  "source": "claude"
}

// RunFinished - Workflow success
{
  "type": "run_finished",
  "runId": "run-abc123",
  "timestamp": "2024-01-01T12:05:00Z",
  "result": {
    "status": "completed",
    "stepCount": 3,
    "duration": "4m30s"
  }
}
```

### SSE Transport Format

AG-UI events must use proper SSE formatting:
```
event: run_started
data: {"type":"run_started","runId":"run-abc123",...}

event: text_message_start  
data: {"type":"text_message_start","runId":"run-abc123","messageId":"msg-def456",...}

event: text_message_content
data: {"type":"text_message_content","runId":"run-abc123","messageId":"msg-def456","content":"Claude output...","delta":true,"source":"claude"}

```

**Critical Transport Details**:
- Event type in SSE `event:` field matches JSON `type` field
- JSON data must be single-line (no newlines in data section)
- Empty line required between events (`\n\n`)
- Proper Content-Type: `text/event-stream`

### Implementation Checklist

**WorkflowEvent Structure**:
- [x] Add `MessageID string` field with `messageId` JSON tag
- [x] Add `Content string` field for text chunks
- [x] Add `Delta bool` field for incremental content flag
- [x] Add `Source string` field for agent attribution
- [x] Add `Complete bool` field for stream completion
- [x] Change `RunID` JSON tag from `run_id` to `runId`

**Event Emission Points**:
- [x] Emit `run_started` when workflow begins in `StartWorkflow()`
- [x] Emit `text_message_start` when Claude execution begins
- [x] Emit `text_message_content` for each stdout chunk from Claude
- [x] Emit `text_message_end` when Claude execution completes
- [x] Emit `run_finished` when workflow completes successfully
- [x] Emit `run_error` when workflow fails

**ID Management**:
- [x] Generate runId once per workflow using `GenerateID("run")`
- [x] Generate messageId once per Claude execution using `GenerateID("msg")`  
- [x] Use same messageId for Start → Content* → End sequence
- [x] Generate new messageId for each separate Claude invocation

## Implementation Notes

### Key Design Decisions
- **Extend WorkflowEvent**: Reuse existing event infrastructure rather than create parallel system
- **Interface Injection**: Use Streamer interface to decouple executor from server implementation  
- **Backward Compatibility**: NoOpStreamer ensures non-streaming mode continues to work
- **Failed Fast**: Streaming errors logged but don't fail Claude execution

### Technical Constraints
- Must maintain existing CLI output behavior exactly
- Cannot modify existing REST API endpoint signatures
- Must work with existing SSE infrastructure unchanged
- Run ID generation and management through existing workflow engine

### Testing Strategy
- **Unit Tests**: Mock-based tests for each component in isolation
- **Integration Tests**: End-to-end streaming with real SSE connections
- **Performance Tests**: Memory usage and concurrent client capacity
- **Compatibility Tests**: Verify existing workflows remain unchanged

## Required Functional Verification Test

### End-to-End Streaming Validation Test

**Test Objective**: Verify complete AG-UI compliant streaming implementation from Alpine workflow execution to frontend SSE delivery.

**Test Procedure**:
```bash
# Terminal 1: Start Alpine with streaming
./alpine --serve "Create a simple Python calculator"

# Terminal 2: Monitor SSE stream
curl -N -H "Accept: text/event-stream" http://localhost:3001/runs/{run-id}/events
```

**Expected Event Sequence**:
```
event: run_started
data: {"type":"run_started","runId":"run-abc123","timestamp":"2024-01-01T12:00:00Z","metadata":{"task":"Create a simple Python calculator","worktreeDir":"/path/to/worktree","planMode":false}}

event: text_message_start  
data: {"type":"text_message_start","runId":"run-abc123","messageId":"msg-def456","timestamp":"2024-01-01T12:01:00Z","source":"claude"}

event: text_message_content
data: {"type":"text_message_content","runId":"run-abc123","messageId":"msg-def456","timestamp":"2024-01-01T12:01:01Z","content":"I'll help you create a Python calculator...","delta":true,"source":"claude"}

event: text_message_content
data: {"type":"text_message_content","runId":"run-abc123","messageId":"msg-def456","timestamp":"2024-01-01T12:01:02Z","content":"Let me start by creating the basic structure...","delta":true,"source":"claude"}

event: text_message_end
data: {"type":"text_message_end","runId":"run-abc123","messageId":"msg-def456","timestamp":"2024-01-01T12:01:30Z","complete":true,"source":"claude"}

event: run_finished
data: {"type":"run_finished","runId":"run-abc123","timestamp":"2024-01-01T12:05:00Z","result":{"status":"completed","stepCount":3,"duration":"4m30s"}}
```

**Pass Criteria**:
- [ ] All 5 AG-UI event types appear in exact sequence: `run_started` → `text_message_start` → `text_message_content`(s) → `text_message_end` → `run_finished`
- [ ] Same `runId` appears across all events for correlation
- [ ] Same `messageId` used for `text_message_start`, all `text_message_content`, and `text_message_end` events
- [ ] JSON format matches AG-UI specification exactly (camelCase fields: `runId`, `messageId`)
- [ ] All text message events include `"source": "claude"`
- [ ] Content events have `"delta": true` flag
- [ ] End event has `"complete": true` flag
- [ ] Timestamps are valid ISO 8601 format
- [ ] SSE transport format correct: proper `event:` and `data:` lines with empty line separators
- [ ] Events stream in real-time (not batched at end of execution)

**Failure Scenarios to Test**:
- [ ] Missing events in sequence
- [ ] Incorrect event type names (e.g., `workflow_started` instead of `run_started`)
- [ ] Wrong field naming (e.g., `run_id` instead of `runId`)
- [ ] Mismatched IDs between related events
- [ ] Missing required fields (`source`, `delta`, `complete`)
- [ ] Invalid JSON format or SSE transport format

**Test Implementation Location**: `test/integration/end_to_end_streaming_test.go`

This test must pass before the implementation is considered complete and ready for production use.