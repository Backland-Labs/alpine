# Server Module

This module implements the REST API server for Alpine, providing HTTP endpoints and Server-Sent Events (SSE) for real-time workflow monitoring.

## Module Overview

The server package provides:
- REST API endpoints for workflow management
- SSE streaming for real-time progress updates
- AG-UI protocol compliance for frontend integration
- GitHub issue integration for workflow initiation
- In-memory run storage and management

## Key Components

### Server (`server.go`)
- HTTP server with configurable port (default: 3001)
- CORS enabled with wildcard origin for development
- Graceful shutdown handling
- Request logging and error middleware

### Handlers (`handlers.go`)
- `/agents/list` - List available agents
- `/agents/run` - Start new workflow from GitHub issue
- `/runs` - List all workflow runs
- `/runs/{id}` - Get specific run details
- `/runs/{id}/events` - SSE stream for run events
- `/runs/{id}/cancel` - Cancel running workflow
- `/plans/{id}` - Get plan content
- `/plans/{id}/approve` - Approve plan for execution
- `/plans/{id}/feedback` - Submit plan feedback

### Models (`models.go`)
- `Run` - Workflow execution state
- `Agent` - Available agent definitions
- `RunRequest` - GitHub issue workflow request
- `PlanFeedback` - Plan modification request
- AG-UI protocol event types

### Workflow Integration (`workflow_integration.go`)
- Bridges server with Alpine workflow engine
- Manages workflow lifecycle
- Handles plan generation and approval flow
- Emits events during execution

### SSE Streaming (`streamer.go`, `run_specific_sse.go`)
- Real-time event streaming to clients
- Client connection management
- Heartbeat/keepalive for connection health
- Buffered event delivery

## AG-UI Protocol Compliance

The server implements the AG-UI protocol for frontend compatibility:

### Event Types
```go
type AGUIEvent struct {
    Type      string      `json:"type"`
    Timestamp time.Time   `json:"timestamp"`
    Data      interface{} `json:"data"`
}
```

### Standard Events
- `workflow.started` - Workflow execution begun
- `workflow.completed` - Workflow finished successfully
- `workflow.failed` - Workflow encountered error
- `plan.generated` - Plan creation complete
- `plan.approved` - Plan approved for execution
- `step.started` - Individual step begun
- `step.completed` - Step finished
- `output.stdout` - Standard output from Claude
- `output.stderr` - Error output from Claude

## Docker Container Context

**IMPORTANT**: The REST API server runs exclusively in Docker containers:
- Assume isolated filesystem environment
- Use environment variables for configuration
- No persistent storage (in-memory only)
- Container networking considerations
- Health check endpoint required

## Implementation Guidelines

### Error Handling
```go
// Use error response helpers
func respondWithError(w http.ResponseWriter, code int, message string) {
    respondWithJSON(w, code, map[string]string{"error": message})
}

// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to start workflow: %w", err)
}
```

### SSE Implementation
```go
// Set proper headers for SSE
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("Connection", "keep-alive")

// Send events with retry
fmt.Fprintf(w, "retry: 1000\n")
fmt.Fprintf(w, "data: %s\n\n", jsonData)
w.(http.Flusher).Flush()
```

### Concurrent Request Handling
- All handlers must be thread-safe
- Use mutexes for shared state access
- Avoid blocking operations in handlers
- Stream events asynchronously

## Testing Requirements

### Unit Tests
- Handler logic isolation
- Model validation
- Error response verification
- SSE formatting correctness

### Integration Tests
- Full request/response cycles
- SSE connection and streaming
- Workflow lifecycle management
- Concurrent request handling

### AG-UI Compliance Tests (`agui_compliance_test.go`)
- Protocol format validation
- Event type correctness
- Timestamp formatting
- Data structure compliance

## Configuration

Environment variables:
- `ALPINE_SERVER_PORT` - Server port (default: 3001)
- `ALPINE_LOG_LEVEL` - Logging verbosity
- `GEMINI_API_KEY` - For plan generation
- `GITHUB_TOKEN` - For issue fetching

## Performance Considerations

- In-memory storage limits scalability
- SSE connections consume resources
- Consider connection limits
- Implement request rate limiting
- Monitor memory usage in container

## Security Notes

- CORS currently allows all origins (development only)
- No authentication/authorization implemented
- Input validation required for all endpoints
- Sanitize GitHub issue URLs
- Prevent command injection in workflow execution

## Common Patterns

### Starting Server
```go
server := NewServer(ServerConfig{
    Port: 3001,
    WorkflowEngine: engine,
})
server.Start(ctx)
```

### Handling SSE Clients
```go
clients := make(map[string]chan Event)
// Add client
clients[clientID] = make(chan Event, 100)
// Send event
clients[clientID] <- event
// Remove on disconnect
delete(clients, clientID)
```

## Dependencies

- Standard library HTTP server
- No external web frameworks
- Alpine workflow engine
- GitHub API client (internal/github)

## Future Considerations

- WebSocket support for bidirectional communication
- Persistent storage (database integration)
- Authentication and authorization
- Rate limiting and quotas
- Metrics and monitoring endpoints
- OpenAPI/Swagger documentation