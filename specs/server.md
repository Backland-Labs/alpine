# HTTP Server Specification

This document defines the HTTP server functionality for Alpine, providing real-time communication capabilities through Server-Sent Events (SSE).

## 1. Overview

Alpine includes an optional HTTP server that runs concurrently with the main workflow, enabling real-time monitoring and communication with frontend applications. The server provides a Server-Sent Events (SSE) endpoint for streaming updates about workflow progress and AI agent activities.

### Key Features

- Non-blocking HTTP server that runs alongside Alpine workflows
- Server-Sent Events endpoint for real-time updates
- Standalone mode for running the server without workflow execution
- Graceful lifecycle management with proper shutdown handling
- Thread-safe implementation supporting multiple concurrent clients

## 2. Architecture

### 2.1. Design Principles

**Concurrency**
- Server runs in a separate goroutine to avoid blocking the main workflow
- Uses Go's standard `net/http` package for HTTP functionality
- Thread-safe with mutex protection for concurrent client access

**Modularity**
- Server implementation isolated in `internal/server` package
- Clean separation between server logic and workflow execution
- Testable design with dependency injection for port configuration

**Lifecycle Management**
- Context-based lifecycle control for graceful shutdown
- Proper resource cleanup on termination
- Support for both workflow-integrated and standalone modes

### 2.2. Server Components

```
internal/server/
├── server.go         # Core server implementation
└── server_test.go    # Test suite with TDD approach
```

### 2.3. Integration Points

- **CLI Layer**: Flags processed in `internal/cli/root.go`
- **Workflow Layer**: Server lifecycle managed in `internal/cli/workflow.go`
- **Configuration**: Server settings handled via context propagation

## 3. CLI Usage

### 3.1. Flags

**--serve**
- Type: boolean
- Default: false
- Description: Enable HTTP server for real-time updates
- Example: `alpine --serve "Implement feature"`

**--port**
- Type: integer
- Default: 3001
- Description: Port for HTTP server
- Example: `alpine --serve --port 8080`

### 3.2. Command Examples

```bash
# Run workflow with server on default port
alpine --serve "Implement user authentication"

# Run workflow with server on custom port
alpine --serve --port 8080 "Fix bug in payment processing"

# Run server standalone (no workflow)
alpine --serve

# Combine with other flags
alpine --serve --no-plan "Quick fix"
```

## 4. Server Modes

### 4.1. Concurrent Mode (Default)

When a task is provided with `--serve`:
1. Server starts in background goroutine
2. Main workflow executes normally
3. Server provides real-time updates during execution
4. Server shuts down when workflow completes

### 4.2. Standalone Mode

When `--serve` is used without a task:
1. Server starts and blocks (runs in foreground)
2. No workflow execution occurs
3. Server continues until interrupted (Ctrl+C)
4. Useful for development and testing

## 5. SSE Endpoint

### 5.1. Endpoint Details

**URL**: `/events`  
**Method**: GET  
**Content-Type**: `text/event-stream`  
**Connection**: Keep-alive with automatic reconnection support

### 5.2. Response Headers

```http
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
Access-Control-Allow-Origin: *
```

### 5.3. Event Format

Events follow the SSE specification:
```
data: {"type": "hello", "message": "hello world"}

data: {"type": "workflow_update", "step": "Analyzing codebase", "status": "running"}

data: {"type": "tool_call", "tool": "Read", "file": "/path/to/file.go"}

```

### 5.4. Initial Connection

Upon client connection:
1. Server sends immediate "hello world" event
2. Confirms connection is established
3. Client begins receiving subsequent events

### 5.5. Event Types (Current)

- **hello**: Initial connection confirmation
  ```json
  {"type": "hello", "message": "hello world"}
  ```

### 5.6. Event Types (Future)

Planned event types for future implementation:
- **workflow_update**: Workflow state changes
- **tool_call**: AI agent tool usage
- **error**: Error notifications
- **completion**: Workflow completion status

## 6. Implementation Details

### 6.1. Server Structure

```go
type Server struct {
    httpServer *http.Server
    mu         sync.Mutex
    clients    map[chan string]bool
    eventChan  chan string
}
```

### 6.2. Key Methods

**NewServer(port int)**
- Creates new server instance
- Supports dynamic port assignment (port 0)
- Returns configured server ready to start

**Start(ctx context.Context)**
- Starts HTTP server in background
- Returns immediately (non-blocking)
- Returns actual port when using dynamic assignment

**Shutdown(ctx context.Context)**
- Gracefully shuts down server
- Closes all client connections
- Waits for cleanup completion

**SendEvent(event string)**
- Broadcasts event to all connected clients
- Thread-safe with mutex protection
- Non-blocking with buffered channels

### 6.3. Client Management

- Clients tracked in map for efficient management
- Automatic cleanup on disconnect
- Buffered channels prevent slow clients from blocking
- Graceful handling of write errors

### 6.4. Error Handling

- Server startup errors propagated to caller
- Client disconnections handled gracefully
- Write errors result in client removal
- Context cancellation triggers shutdown

## 7. Testing Approach

### 7.1. Test-Driven Development

The server implementation followed strict TDD methodology:
1. **RED**: Write failing tests for requirements
2. **GREEN**: Implement minimal code to pass
3. **REFACTOR**: Improve code quality

### 7.2. Test Coverage

Current test coverage: 81.1%

Key test scenarios:
- Server creation and configuration
- Start and stop lifecycle
- SSE endpoint functionality
- Multiple concurrent clients
- Client disconnect handling
- Dynamic port assignment
- Context-based shutdown

### 7.3. Testing Utilities

```go
// Dynamic port for test isolation
server := NewServer(0)

// Wait for event with timeout
func waitForEvent(t *testing.T, resp *http.Response) string

// Verify server state
func TestServerStartAndStop(t *testing.T)
```

## 8. Future Enhancements

### 8.1. Enhanced Event Types

**Workflow Events**
- Workflow start/stop notifications
- Step progression updates
- Completion status with summary

**Tool Usage Events**
- Real-time tool call notifications
- Tool parameters and results
- Performance metrics

**Error Events**
- Workflow errors and warnings
- Recovery attempts
- Debug information

### 8.2. WebSocket Support

Consider adding WebSocket endpoint for:
- Bidirectional communication
- Lower latency updates
- Reduced overhead for high-frequency events

### 8.3. Authentication

For production use:
- API key authentication
- JWT token support
- Rate limiting per client

### 8.4. Metrics and Monitoring

- Prometheus metrics endpoint
- Client connection statistics
- Event throughput monitoring
- Health check endpoint

### 8.5. Event Persistence

- Optional event history storage
- Replay capability for reconnecting clients
- Event filtering and querying

## 9. Security Considerations

### 9.1. Current State

- CORS enabled with wildcard (development-friendly)
- No authentication required
- Local network exposure only

### 9.2. Production Recommendations

- Restrict CORS to specific origins
- Implement authentication mechanism
- Use HTTPS with proper certificates
- Rate limit connections per IP
- Validate event data before broadcasting

## 10. Performance Considerations

### 10.1. Current Implementation

- Buffered channels for event distribution
- Non-blocking event broadcasting
- Efficient client tracking with maps

### 10.2. Scalability

- Consider event queue for high-volume scenarios
- Implement connection pooling if needed
- Monitor memory usage with many clients
- Profile CPU usage under load

## 11. Integration Examples

### 11.1. JavaScript Client

```javascript
const evtSource = new EventSource('http://localhost:3001/events');

evtSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Received:', data);
};

evtSource.onerror = (err) => {
  console.error('EventSource failed:', err);
};
```

### 11.2. React Hook

```javascript
function useAlpineEvents(port = 3001) {
  const [events, setEvents] = useState([]);
  
  useEffect(() => {
    const evtSource = new EventSource(`http://localhost:${port}/events`);
    
    evtSource.onmessage = (event) => {
      const data = JSON.parse(event.data);
      setEvents(prev => [...prev, data]);
    };
    
    return () => evtSource.close();
  }, [port]);
  
  return events;
}
```

### 11.3. curl Testing

```bash
# Connect to SSE endpoint
curl -N http://localhost:3001/events

# Expected output:
# data: {"type": "hello", "message": "hello world"}
```

## 12. Debugging

### 12.1. Common Issues

**Server not starting**
- Check if port is already in use
- Verify --serve flag is set
- Look for startup errors in logs

**No events received**
- Confirm server is running on expected port
- Check firewall settings
- Verify SSE headers in response

**Connection drops**
- Check for proxy timeout settings
- Verify keep-alive is working
- Monitor for server errors

### 12.2. Debug Commands

```bash
# Check if server is listening
lsof -i :3001

# Test SSE endpoint
curl -v http://localhost:3001/events

# Monitor Alpine logs
ALPINE_LOG_LEVEL=debug alpine --serve "Test task"
```

## 13. REST API Endpoints

Alpine's HTTP server includes a comprehensive REST API for programmatic workflow management. The API enables external tools and services to start workflows, monitor progress, and manage execution lifecycle.

### 13.1. API Overview

**Base URL**: `http://localhost:3001` (configurable via `--port`)  
**Content-Type**: `application/json` (except SSE endpoints)  
**Authentication**: None (MVP implementation)  
**Storage**: In-memory (non-persistent across restarts)

### 13.2. Data Models

#### Agent
Represents a workflow execution agent.

```json
{
  "id": "string",          // Unique identifier
  "name": "string",        // Human-readable name
  "description": "string"  // Agent capabilities description
}
```

**Validation Rules**:
- `id` and `name` are required and cannot be empty

#### Run
Represents a workflow execution instance.

```json
{
  "id": "string",                    // Unique identifier (format: "run-{hex}")
  "agent_id": "string",              // Agent executing this run
  "status": "string",                // Status: running|completed|cancelled|failed
  "issue": "string",                 // GitHub issue URL
  "created": "2025-07-29T12:34:56Z", // ISO 8601 timestamp
  "updated": "2025-07-29T12:34:56Z", // ISO 8601 timestamp
  "worktree_dir": "string"           // Git worktree directory (optional)
}
```

**Status Transitions**:
- Initial state: `running`
- Valid transitions: `running` → `completed|cancelled|failed`
- No transitions allowed from terminal states

#### Plan
Represents a workflow implementation plan.

```json
{
  "run_id": "string",                // Associated run ID
  "content": "string",               // Plan content (markdown)
  "status": "string",                // Status: pending|approved|rejected
  "created": "2025-07-29T12:34:56Z", // ISO 8601 timestamp
  "updated": "2025-07-29T12:34:56Z"  // ISO 8601 timestamp
}
```

**Status Transitions**:
- Initial state: `pending`
- Valid transitions: `pending` → `approved|rejected`
- No transitions allowed from terminal states

### 13.3. Endpoints

#### Health Check
Check server health status.

```http
GET /health
```

**Response**:
```json
{
  "status": "healthy",
  "service": "alpine-server",
  "timestamp": "2025-07-29T12:34:56Z"
}
```

**Status Codes**:
- `200 OK` - Service is healthy
- `405 Method Not Allowed` - Invalid HTTP method

**Example**:
```bash
curl http://localhost:3001/health
```

#### List Agents
Get available workflow agents.

```http
GET /agents/list
```

**Response**:
```json
[
  {
    "id": "alpine-agent",
    "name": "Alpine Workflow Agent",
    "description": "Default agent for running Alpine workflows from GitHub issues"
  }
]
```

**Status Codes**:
- `200 OK` - Success
- `405 Method Not Allowed` - Invalid HTTP method

**Note**: Currently returns hardcoded list (MVP)

**Example**:
```bash
curl http://localhost:3001/agents/list
```

#### Start Workflow Run
Create and start a new workflow run.

```http
POST /agents/run
Content-Type: application/json

{
  "issue_url": "https://github.com/owner/repo/issues/123",
  "agent_id": "alpine-agent"
}
```

**Request Body**:
- `issue_url` (required) - GitHub issue URL to process
- `agent_id` (required) - Agent ID to execute workflow

**Response**:
```json
{
  "id": "run-1234abcd",
  "agent_id": "alpine-agent",
  "status": "running",
  "issue": "https://github.com/owner/repo/issues/123",
  "created": "2025-07-29T12:34:56Z",
  "updated": "2025-07-29T12:34:56Z",
  "worktree_dir": "/path/to/worktree"
}
```

**Status Codes**:
- `201 Created` - Workflow started successfully
- `400 Bad Request` - Invalid request body
- `405 Method Not Allowed` - Invalid HTTP method

**Example**:
```bash
curl -X POST http://localhost:3001/agents/run \
  -H "Content-Type: application/json" \
  -d '{
    "issue_url": "https://github.com/owner/repo/issues/123",
    "agent_id": "alpine-agent"
  }'
```

#### List All Runs
Get all workflow runs.

```http
GET /runs
```

**Response**:
```json
[
  {
    "id": "run-1234abcd",
    "agent_id": "alpine-agent",
    "status": "running",
    "issue": "https://github.com/owner/repo/issues/123",
    "created": "2025-07-29T12:34:56Z",
    "updated": "2025-07-29T12:34:56Z",
    "worktree_dir": "/path/to/worktree"
  }
]
```

**Status Codes**:
- `200 OK` - Success
- `405 Method Not Allowed` - Invalid HTTP method

**Example**:
```bash
curl http://localhost:3001/runs
```

#### Get Run Details
Get details of a specific run.

```http
GET /runs/{id}
```

**Path Parameters**:
- `id` - Run identifier

**Response**:
```json
{
  "id": "run-1234abcd",
  "agent_id": "alpine-agent",
  "status": "running",
  "issue": "https://github.com/owner/repo/issues/123",
  "created": "2025-07-29T12:34:56Z",
  "updated": "2025-07-29T12:34:56Z",
  "worktree_dir": "/path/to/worktree",
  "current_step": "Generating implementation plan"
}
```

**Additional Fields**:
- `current_step` - Current workflow step (when workflow engine is connected)

**Status Codes**:
- `200 OK` - Success
- `400 Bad Request` - Missing run ID
- `404 Not Found` - Run not found
- `405 Method Not Allowed` - Invalid HTTP method

**Example**:
```bash
curl http://localhost:3001/runs/run-1234abcd
```

#### Run Events (SSE)
Subscribe to real-time events for a specific run.

```http
GET /runs/{id}/events
Accept: text/event-stream
```

**Path Parameters**:
- `id` - Run identifier

**Response Headers**:
```http
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
Access-Control-Allow-Origin: *
```

**Event Stream Format**:
```
data: {"type":"connected","runId":"run-1234abcd"}

event: workflow_update
data: {"type":"workflow_update","run_id":"run-1234abcd","timestamp":"2025-07-29T12:34:56Z","data":{...}}

event: workflow_complete
data: {"type":"workflow_complete","run_id":"run-1234abcd","status":"completed"}
```

**Status Codes**:
- `200 OK` - SSE stream established
- `404 Not Found` - Run not found
- `405 Method Not Allowed` - Invalid HTTP method
- `500 Internal Server Error` - Streaming not supported

**Example**:
```bash
curl -N -H "Accept: text/event-stream" \
  http://localhost:3001/runs/run-1234abcd/events
```

#### Cancel Run
Cancel a running workflow.

```http
POST /runs/{id}/cancel
```

**Path Parameters**:
- `id` - Run identifier

**Response**:
```json
{
  "status": "cancelled",
  "runId": "run-1234abcd"
}
```

**Status Codes**:
- `200 OK` - Successfully cancelled
- `400 Bad Request` - Cannot cancel non-running workflow
- `404 Not Found` - Run not found
- `405 Method Not Allowed` - Invalid HTTP method
- `500 Internal Server Error` - Failed to cancel

**Example**:
```bash
curl -X POST http://localhost:3001/runs/run-1234abcd/cancel
```

#### Get Plan
Retrieve the implementation plan for a run.

```http
GET /plans/{runId}
```

**Path Parameters**:
- `runId` - Run identifier

**Response**:
```json
{
  "run_id": "run-1234abcd",
  "content": "# Implementation Plan\n\n## Overview\n...",
  "status": "pending",
  "created": "2025-07-29T12:34:56Z",
  "updated": "2025-07-29T12:34:56Z"
}
```

**Status Codes**:
- `200 OK` - Success
- `404 Not Found` - Plan not found
- `405 Method Not Allowed` - Invalid HTTP method

**Example**:
```bash
curl http://localhost:3001/plans/run-1234abcd
```

#### Approve Plan
Approve a pending implementation plan.

```http
POST /plans/{runId}/approve
```

**Path Parameters**:
- `runId` - Run identifier

**Response**:
```json
{
  "status": "approved",
  "runId": "run-1234abcd"
}
```

**Effects**:
- Plan status changes to `approved`
- Associated run resumes execution
- Workflow engine notified of approval

**Status Codes**:
- `200 OK` - Plan approved
- `404 Not Found` - Plan not found
- `405 Method Not Allowed` - Invalid HTTP method
- `500 Internal Server Error` - Failed to notify workflow

**Example**:
```bash
curl -X POST http://localhost:3001/plans/run-1234abcd/approve
```

#### Send Plan Feedback
Provide feedback on a plan (future enhancement).

```http
POST /plans/{runId}/feedback
Content-Type: application/json

{
  "feedback": "Please add more error handling in step 3"
}
```

**Path Parameters**:
- `runId` - Run identifier

**Request Body**:
- `feedback` - Feedback text

**Response**:
```json
{
  "status": "feedback_received",
  "runId": "run-1234abcd"
}
```

**Status Codes**:
- `200 OK` - Feedback received
- `400 Bad Request` - Invalid request body
- `404 Not Found` - Plan not found
- `405 Method Not Allowed` - Invalid HTTP method

**Note**: Currently only acknowledges receipt; plan regeneration not implemented

**Example**:
```bash
curl -X POST http://localhost:3001/plans/run-1234abcd/feedback \
  -H "Content-Type: application/json" \
  -d '{"feedback": "Add more error handling"}'
```

### 13.4. Error Handling

#### Error Response Format
All error responses use consistent JSON structure:

```json
{
  "error": "Descriptive error message"
}
```

#### Common Error Scenarios

**Invalid HTTP Method**:
```json
{
  "error": "Method not allowed"
}
```

**Resource Not Found**:
```json
{
  "error": "Run not found"
}
```

**Invalid Request Body**:
```json
{
  "error": "Invalid JSON payload"
}
```

**State Transition Error**:
```json
{
  "error": "Cannot cancel workflow in completed state"
}
```

### 13.5. Integration Examples

#### JavaScript/TypeScript
```javascript
// Start a workflow
async function startWorkflow(issueUrl) {
  const response = await fetch('http://localhost:3001/agents/run', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      issue_url: issueUrl,
      agent_id: 'alpine-agent'
    })
  });
  return response.json();
}

// Monitor run progress
function monitorRun(runId) {
  const evtSource = new EventSource(
    `http://localhost:3001/runs/${runId}/events`
  );
  
  evtSource.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('Update:', data);
  };
  
  evtSource.addEventListener('workflow_complete', (event) => {
    const data = JSON.parse(event.data);
    console.log('Completed:', data.status);
    evtSource.close();
  });
}
```

#### Python
```python
import requests
import sseclient

# Start workflow
def start_workflow(issue_url):
    response = requests.post(
        'http://localhost:3001/agents/run',
        json={
            'issue_url': issue_url,
            'agent_id': 'alpine-agent'
        }
    )
    return response.json()

# Monitor events
def monitor_run(run_id):
    url = f'http://localhost:3001/runs/{run_id}/events'
    response = requests.get(url, stream=True)
    client = sseclient.SSEClient(response)
    
    for event in client.events():
        print(f"Event: {event.event}, Data: {event.data}")
```

#### Go
```go
// Start workflow
type StartRunRequest struct {
    IssueURL string `json:"issue_url"`
    AgentID  string `json:"agent_id"`
}

func startWorkflow(issueURL string) (*Run, error) {
    reqBody, _ := json.Marshal(StartRunRequest{
        IssueURL: issueURL,
        AgentID:  "alpine-agent",
    })
    
    resp, err := http.Post(
        "http://localhost:3001/agents/run",
        "application/json",
        bytes.NewBuffer(reqBody),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var run Run
    json.NewDecoder(resp.Body).Decode(&run)
    return &run, nil
}
```

### 13.6. Workflow Integration

The REST API integrates with Alpine's workflow engine through the `WorkflowEngine` interface:

```go
type WorkflowEngine interface {
    // Start a new workflow
    StartWorkflow(ctx context.Context, issueURL string, runID string) (string, error)
    
    // Cancel a running workflow
    CancelWorkflow(ctx context.Context, runID string) error
    
    // Get current workflow state
    GetWorkflowState(ctx context.Context, runID string) (*core.State, error)
    
    // Approve a pending plan
    ApprovePlan(ctx context.Context, runID string) error
    
    // Subscribe to workflow events
    SubscribeToEvents(ctx context.Context, runID string) (<-chan WorkflowEvent, error)
}
```

### 13.7. Limitations and Future Work

#### Current Limitations (MVP)
- **No Authentication**: API is unsecured
- **In-Memory Storage**: Data lost on restart
- **Single Agent**: Only one hardcoded agent type
- **No Pagination**: All results returned at once
- **Limited Filtering**: No query parameters for filtering
- **Basic Plan Workflow**: No iterative plan refinement

#### Planned Enhancements
- JWT-based authentication
- Persistent storage (PostgreSQL/SQLite)
- Multiple agent types with capabilities
- Pagination and filtering for list endpoints
- WebSocket support for bidirectional communication
- Plan revision workflow with feedback loop
- Metrics and monitoring endpoints
- Rate limiting and quotas
- Webhook notifications for status changes
