# AG-UI Protocol Integration

## Overview

This specification defines how Alpine integrates with the AG-UI (Agent-User Interaction) protocol to enable real-time streaming of Claude Code execution events to frontend applications. AG-UI is an event-based protocol that standardizes how AI agents connect to user-facing applications through Server-Sent Events (SSE) and structured JSON messaging.

## 1. Protocol Overview

### 1.1. What is AG-UI

AG-UI is a lightweight, event-based protocol that enables:
- Real-time streaming of agent execution events
- Bidirectional communication between agents and frontend applications
- Standardized event formats for consistent integration
- Transport-agnostic communication (SSE, WebSockets, HTTP)

### 1.2. Alpine Integration Goals

Alpine's AG-UI integration enables:
- Real-time streaming of Claude Code `stdout` and execution status
- Frontend applications to monitor workflow progress in real-time
- Standardized event formats for consistent agent-UI communication
- Human-in-the-loop interactions during workflow execution

## 2. Event Types and Schema

### 2.1. Supported Event Types

Alpine implements the following AG-UI event types:

#### Lifecycle Events
```go
// RunStarted - Signals the beginning of a workflow execution
type RunStartedEvent struct {
    Type      string    `json:"type"`             // "run_started"
    RunID     string    `json:"runId"`
    Timestamp time.Time `json:"timestamp"`
    Metadata  struct {
        Task        string `json:"task"`
        WorktreeDir string `json:"worktreeDir,omitempty"`
        PlanMode    bool   `json:"planMode"`
    } `json:"metadata"`
}

// RunFinished - Signals successful completion of workflow
type RunFinishedEvent struct {
    Type      string    `json:"type"`             // "run_finished"
    RunID     string    `json:"runId"`
    Timestamp time.Time `json:"timestamp"`
    Result    struct {
        Status     string `json:"status"`         // "completed"
        StepCount  int    `json:"stepCount"`
        Duration   string `json:"duration"`
    } `json:"result"`
}

// RunError - Signals workflow termination due to error
type RunErrorEvent struct {
    Type      string    `json:"type"`             // "run_error"
    RunID     string    `json:"runId"`
    Timestamp time.Time `json:"timestamp"`
    Error     struct {
        Message string `json:"message"`
        Code    string `json:"code,omitempty"`
        Step    int    `json:"step,omitempty"`
    } `json:"error"`
}
```

#### Step Events
```go
// StepStarted - Signals the beginning of a workflow step
type StepStartedEvent struct {
    Type        string    `json:"type"`           // "step_started"
    RunID       string    `json:"runId"`
    StepID      string    `json:"stepId"`
    Timestamp   time.Time `json:"timestamp"`
    Description string    `json:"description"`
    StepNumber  int       `json:"stepNumber"`
}

// StepFinished - Signals completion of a workflow step
type StepFinishedEvent struct {
    Type       string    `json:"type"`           // "step_finished"
    RunID      string    `json:"runId"`
    StepID     string    `json:"stepId"`
    Timestamp  time.Time `json:"timestamp"`
    Status     string    `json:"status"`         // "completed" | "failed"
    Duration   string    `json:"duration"`
}
```

#### Text Message Events
```go
// TextMessageStart - Begins streaming a text message
type TextMessageStartEvent struct {
    Type      string    `json:"type"`             // "text_message_start"
    RunID     string    `json:"runId"`
    MessageID string    `json:"messageId"`
    Timestamp time.Time `json:"timestamp"`
    Source    string    `json:"source"`           // "claude" | "alpine"
}

// TextMessageContent - Streams chunks of message content
type TextMessageContentEvent struct {
    Type      string    `json:"type"`             // "text_message_content"
    RunID     string    `json:"runId"`
    MessageID string    `json:"messageId"`
    Timestamp time.Time `json:"timestamp"`
    Content   string    `json:"content"`
    Delta     bool      `json:"delta"`            // true for incremental content
}

// TextMessageEnd - Completes a text message
type TextMessageEndEvent struct {
    Type      string    `json:"type"`             // "text_message_end"
    RunID     string    `json:"runId"`
    MessageID string    `json:"messageId"`
    Timestamp time.Time `json:"timestamp"`
    Complete  bool      `json:"complete"`
}
```

#### State Management Events
```go
// StateSnapshot - Complete workflow state
type StateSnapshotEvent struct {
    Type      string    `json:"type"`             // "state_snapshot"
    RunID     string    `json:"runId"`
    Timestamp time.Time `json:"timestamp"`
    State     struct {
        Status            string `json:"status"`
        CurrentStep       int    `json:"currentStep"`
        StepDescription   string `json:"stepDescription"`
        NextStepPrompt    string `json:"nextStepPrompt"`
        WorkingDirectory  string `json:"workingDirectory"`
    } `json:"state"`
}

// StateDelta - Incremental state changes
type StateDeltaEvent struct {
    Type      string    `json:"type"`             // "state_delta"
    RunID     string    `json:"runId"`
    Timestamp time.Time `json:"timestamp"`
    Delta     []struct {
        Op    string      `json:"op"`               // "replace" | "add" | "remove"
        Path  string      `json:"path"`             // JSON Pointer path
        Value interface{} `json:"value,omitempty"`
    } `json:"delta"`
}
```

### 2.2. Event Base Structure

All events implement a common interface:

```go
type BaseEvent interface {
    GetType() string
    GetRunID() string
    GetTimestamp() time.Time
    Validate() error
}

type EventEmitter interface {
    EmitEvent(event BaseEvent) error
    Subscribe(runID string) (<-chan BaseEvent, error)
    Unsubscribe(runID string) error
}
```

## 3. Transport Layer

### 3.1. Server-Sent Events (SSE)

Alpine's HTTP server implements SSE endpoints for AG-UI event streaming:

```go
// SSE endpoint for real-time event streaming
GET /runs/{runId}/events
Accept: text/event-stream
Cache-Control: no-cache

// Response format
event: run_started
data: {"type":"run_started","runId":"run_abc123","timestamp":"2025-07-30T10:00:00Z",...}

event: text_message_content
data: {"type":"text_message_content","runId":"run_abc123","content":"Analyzing codebase...",...}
```

### 3.2. HTTP API Integration

AG-UI events are integrated into existing REST endpoints:

```go
// Start workflow with AG-UI event streaming
POST /agents/run
Content-Type: application/json
{
    "github_issue_url": "https://github.com/owner/repo/issues/123",
    "enable_ag_ui": true,
    "stream_events": true
}

// Response includes AG-UI event stream URL
{
    "run_id": "run_abc123",
    "status": "running",
    "ag_ui_stream": "/runs/run_abc123/events"
}
```

### 3.3. Event Streaming Architecture

```go
type AGUIEventStreamer struct {
    subscribers map[string][]chan BaseEvent
    mutex       sync.RWMutex
    buffer      map[string][]BaseEvent  // Event replay buffer
}

func (s *AGUIEventStreamer) EmitEvent(event BaseEvent) error {
    s.mutex.RLock()
    defer s.mutex.RUnlock()
    
    runID := event.GetRunID()
    if channels, exists := s.subscribers[runID]; exists {
        for _, ch := range channels {
            select {
            case ch <- event:
            default:
                // Handle slow consumer
            }
        }
    }
    
    // Buffer for replay
    s.buffer[runID] = append(s.buffer[runID], event)
    return nil
}
```

## 4. Integration with Alpine Components

### 4.1. Workflow Engine Integration

The workflow engine emits AG-UI events at key execution points:

```go
type WorkflowEngine struct {
    eventStreamer *AGUIEventStreamer
    // ... existing fields
}

func (w *WorkflowEngine) ExecuteStep(step WorkflowStep) error {
    // Emit step started event
    w.eventStreamer.EmitEvent(&StepStartedEvent{
        Type:        "step_started",
        RunID:       w.runID,
        StepID:      step.ID,
        Timestamp:   time.Now(),
        Description: step.Description,
        StepNumber:  step.Number,
    })
    
    // Execute step logic
    err := step.Execute()
    
    // Emit step completion event
    status := "completed"
    if err != nil {
        status = "failed"
    }
    
    w.eventStreamer.EmitEvent(&StepFinishedEvent{
        Type:      "step_finished",
        RunID:     w.runID,
        StepID:    step.ID,
        Timestamp: time.Now(),
        Status:    status,
        Duration:  step.Duration.String(),
    })
    
    return err
}
```

### 4.2. Claude Executor Integration

The Claude executor streams stdout content as AG-UI text message events:

```go
type ClaudeExecutor struct {
    eventStreamer *AGUIEventStreamer
    // ... existing fields
}

func (e *ClaudeExecutor) Execute(prompt string) (string, error) {
    messageID := generateMessageID()
    
    // Start message event
    e.eventStreamer.EmitEvent(&TextMessageStartEvent{
        Type:      "text_message_start",
        RunID:     e.runID,
        MessageID: messageID,
        Timestamp: time.Now(),
        Source:    "claude",
    })
    
    // Stream stdout content in chunks
    scanner := bufio.NewScanner(stdout)
    for scanner.Scan() {
        content := scanner.Text()
        e.eventStreamer.EmitEvent(&TextMessageContentEvent{
            Type:      "text_message_content",
            RunID:     e.runID,
            MessageID: messageID,
            Timestamp: time.Now(),
            Content:   content + "\n",
            Delta:     true,
        })
    }
    
    // End message event
    e.eventStreamer.EmitEvent(&TextMessageEndEvent{
        Type:      "text_message_end",
        RunID:     e.runID,
        MessageID: messageID,
        Timestamp: time.Now(),
        Complete:  true,
    })
    
    return fullOutput, nil
}
```

### 4.3. State Manager Integration

The state manager emits state change events:

```go
func (s *StateManager) UpdateState(updates map[string]interface{}) error {
    // Create JSON Patch deltas
    var deltas []StateDelta
    for path, value := range updates {
        deltas = append(deltas, StateDelta{
            Op:    "replace",
            Path:  "/" + path,
            Value: value,
        })
    }
    
    // Emit state delta event
    s.eventStreamer.EmitEvent(&StateDeltaEvent{
        Type:      "state_delta",
        RunID:     s.runID,
        Timestamp: time.Now(),
        Delta:     deltas,
    })
    
    return s.applyUpdates(updates)
}
```

## 5. Configuration

### 5.1. Environment Variables

```bash
# Enable AG-UI protocol support
ALPINE_AG_UI_ENABLED=true

# Configure event buffer size for replay
ALPINE_AG_UI_BUFFER_SIZE=1000

# Set event streaming timeout
ALPINE_AG_UI_STREAM_TIMEOUT=30s

# Enable event validation
ALPINE_AG_UI_VALIDATE_EVENTS=true
```

### 5.2. CLI Flags

```bash
# Enable AG-UI event streaming
./alpine --serve --ag-ui "Implement feature"

# Configure AG-UI buffer size
./alpine --serve --ag-ui-buffer-size 2000 "Debug issue"

# Disable AG-UI for lightweight execution
./alpine --serve --no-ag-ui "Quick task"
```

## 6. Frontend Integration Examples

### 6.1. JavaScript Client

```javascript
// Connect to AG-UI event stream
const eventSource = new EventSource('/runs/run_abc123/events');

// Handle different event types
eventSource.addEventListener('run_started', (event) => {
    const data = JSON.parse(event.data);
    console.log('Workflow started:', data.metadata.task);
});

eventSource.addEventListener('text_message_content', (event) => {
    const data = JSON.parse(event.data);
    appendToOutput(data.content);
});

eventSource.addEventListener('run_finished', (event) => {
    const data = JSON.parse(event.data);
    console.log('Workflow completed in:', data.result.duration);
    eventSource.close();
});
```

### 6.2. React Component

```jsx
function AlpineWorkflowMonitor({ runId }) {
    const [events, setEvents] = useState([]);
    const [status, setStatus] = useState('connecting');
    
    useEffect(() => {
        const eventSource = new EventSource(`/runs/${runId}/events`);
        
        const handleEvent = (event) => {
            const data = JSON.parse(event.data);
            setEvents(prev => [...prev, data]);
            
            if (data.type === 'run_started') {
                setStatus('running');
            } else if (data.type === 'run_finished') {
                setStatus('completed');
            } else if (data.type === 'run_error') {
                setStatus('error');
            }
        };
        
        eventSource.onmessage = handleEvent;
        eventSource.onerror = () => setStatus('disconnected');
        
        return () => eventSource.close();
    }, [runId]);
    
    return (
        <div className="workflow-monitor">
            <div className="status">Status: {status}</div>
            <div className="events">
                {events.map((event, i) => (
                    <EventDisplay key={i} event={event} />
                ))}
            </div>
        </div>
    );
}
```

## 7. Error Handling and Resilience

### 7.1. Event Validation

```go
func (e *RunStartedEvent) Validate() error {
    if e.Type != "run_started" {
        return fmt.Errorf("invalid event type: %s", e.Type)
    }
    if e.RunID == "" {
        return fmt.Errorf("runId is required")
    }
    if e.Timestamp.IsZero() {
        return fmt.Errorf("timestamp is required")
    }
    return nil
}
```

### 7.2. Connection Management

```go
type ConnectionManager struct {
    connections map[string]*SSEConnection
    mutex       sync.RWMutex
    heartbeat   time.Duration
}

func (c *ConnectionManager) HandleDisconnection(runID string) {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    if conn, exists := c.connections[runID]; exists {
        conn.Close()
        delete(c.connections, runID)
        log.Printf("Client disconnected from run %s", runID)
    }
}

func (c *ConnectionManager) SendHeartbeat() {
    ticker := time.NewTicker(c.heartbeat)
    for range ticker.C {
        c.mutex.RLock()
        for runID, conn := range c.connections {
            if err := conn.Ping(); err != nil {
                c.HandleDisconnection(runID)
            }
        }
        c.mutex.RUnlock()
    }
}
```

## 8. Testing Strategy

### 8.1. Event Testing

```go
func TestEventEmission(t *testing.T) {
    streamer := NewAGUIEventStreamer()
    
    // Subscribe to events
    events, err := streamer.Subscribe("test_run")
    require.NoError(t, err)
    
    // Emit test event
    testEvent := &RunStartedEvent{
        Type:      "run_started",
        RunID:     "test_run",
        Timestamp: time.Now(),
    }
    
    err = streamer.EmitEvent(testEvent)
    require.NoError(t, err)
    
    // Verify event received
    select {
    case receivedEvent := <-events:
        assert.Equal(t, testEvent.Type, receivedEvent.GetType())
    case <-time.After(time.Second):
        t.Fatal("Event not received within timeout")
    }
}
```

### 8.2. Integration Testing

```go
func TestWorkflowEventIntegration(t *testing.T) {
    // Setup test environment
    server := setupTestServer()
    streamer := NewAGUIEventStreamer()
    
    // Create test workflow
    workflow := &WorkflowEngine{
        eventStreamer: streamer,
        runID:        "test_workflow",
    }
    
    // Subscribe to events
    events, _ := streamer.Subscribe("test_workflow")
    
    // Execute workflow
    go workflow.Execute("Test task")
    
    // Verify event sequence
    expectedEvents := []string{
        "run_started",
        "step_started",
        "text_message_start",
        "text_message_content",
        "text_message_end",
        "step_finished",
        "run_finished",
    }
    
    for _, expectedType := range expectedEvents {
        select {
        case event := <-events:
            assert.Equal(t, expectedType, event.GetType())
        case <-time.After(5 * time.Second):
            t.Fatalf("Expected event %s not received", expectedType)
        }
    }
}
```

## 9. Performance Considerations

### 9.1. Event Buffering

- Buffer events for late-joining clients
- Implement circular buffer to prevent memory leaks
- Configurable buffer size based on use case
- Automatic cleanup of old event buffers

### 9.2. Connection Scaling

- Use connection pooling for multiple clients
- Implement backpressure handling for slow consumers
- Monitor connection health and automatically reconnect
- Rate limiting to prevent event flooding

## 10. Future Enhancements

### 10.1. Tool Call Events

Future versions may support AG-UI tool call events:

```go
type ToolCallStartEvent struct {
    Type       string    `json:"type"`           // "tool_call_start"
    RunID      string    `json:"runId"`
    ToolCallID string    `json:"toolCallId"`
    Timestamp  time.Time `json:"timestamp"`
    ToolName   string    `json:"toolName"`
}
```

### 10.2. Human-in-the-Loop Integration

- Interactive approval events
- User input request events
- Feedback collection events
- Plan modification events

### 10.3. Multi-Agent Support

When Alpine supports multiple concurrent agents:
- Agent-specific event namespacing
- Cross-agent communication events
- Resource coordination events

## 11. Migration and Compatibility

### 11.1. Backward Compatibility

- AG-UI events are additive to existing API
- Existing clients continue to work without changes
- AG-UI can be disabled via configuration
- Graceful degradation when AG-UI is unavailable

### 11.2. Version Management

- Event schema versioning
- Client capability negotiation
- Progressive enhancement based on client support
- Migration paths for schema changes