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