# Implementation Plan: HTTP Server with Server-Sent Events

## Overview

This plan outlines the implementation of a basic HTTP server with a Server-Sent Events (SSE) endpoint. This will allow frontend applications to receive real-time updates from Alpine's AI workflows. The implementation will follow a Test-Driven Development (TDD) approach.

### Objectives
-   Implement a non-blocking HTTP server that runs concurrently with the main workflow.
-   Provide an SSE endpoint at `/events` for real-time communication.
-   Add a `--serve` flag to the CLI to enable the server.
-   Ensure the implementation is modular and well-tested.

## Prioritized Features

### P0: Core Server Implementation

#### Task 1: Add CLI Flags ✅ **IMPLEMENTED**
**Acceptance Criteria:**
-   A `--serve` boolean flag is added to the root command. ✅
-   A `--port` integer flag is added, with a default value of `3001`. ✅
-   The flags are correctly parsed and accessible in the CLI logic. ✅

**Test Cases:**
```go
// Test that 'alpine --serve' is a valid command. ✅
// Test that 'alpine --port 8080' is a valid command. ✅
// Test that the default port is 3001 when --port is not specified. ✅
```

**Implementation:**
-   Modify `internal/cli/root.go` to add the `--serve` and `--port` flags using Cobra. ✅
-   Update the configuration logic in `internal/config/config.go` to handle the new server-related settings. ✅

**Implementation Date:** 2025-07-27
**Notes:** 
- Added --serve and --port flags to root command
- Flags are stored in context for access by workflow logic
- Added ServerConfig struct to config package for future use
- Full test coverage with TDD approach (RED-GREEN-REFACTOR)

#### Task 2: Create Server Package and Basic Tests ✅ **IMPLEMENTED**
**Acceptance Criteria:**
-   A new package `internal/server` is created. ✅
-   A `server.go` file is created with a `Server` struct. ✅
-   A `server_test.go` file is created with a basic test for server creation. ✅

**Test Cases:**
```go
// TestNewServer validates that a new server instance can be created. ✅
// TestServerStartAndStop verifies that the server can be started and stopped gracefully. ✅
```

**Implementation:**
-   Create the `internal/server` directory. ✅
-   Create `server.go` and `server_test.go`. ✅
-   Define the `Server` struct with fields for the HTTP server and a channel for events. ✅

**Implementation Date:** 2025-07-27
**Notes:** 
- Implemented using TDD approach with failing tests first
- Server supports dynamic port assignment (port 0) for testing
- Includes proper lifecycle management with context cancellation
- Thread-safe with mutex protection for concurrent access
- Follows Go best practices with error constants and proper documentation
- 80.5% test coverage achieved

#### Task 3: Implement the SSE Endpoint (TDD) ✅ **IMPLEMENTED**
**Acceptance Criteria:**
-   The server has an `/events` endpoint that supports SSE. ✅
-   When a client connects, it receives a "hello world" event. ✅
-   The server handles multiple concurrent client connections. ✅
-   The server gracefully handles client disconnections. ✅

**Test Cases:**
```go
// TestSSEHelloWorldEvent verifies that a client receives the initial event. ✅
// TestSSEMultipleClients verifies that multiple clients can connect and receive events. ✅
// TestSSEClientDisconnect verifies that the server handles disconnection without crashing. ✅
```

**Implementation:**
-   Add an HTTP handler for the `/events` endpoint in `server.go`. ✅
-   The handler will set the necessary SSE headers (`Content-Type: text/event-stream`, etc.). ✅
-   Implement the logic to send the "hello world" event upon connection. ✅
-   Use channels to manage client connections and event broadcasting. ✅

**Implementation Date:** 2025-07-27
**Notes:** 
- Implemented using strict TDD approach (RED-GREEN-REFACTOR)
- Added comprehensive tests for all acceptance criteria
- SSE endpoint properly sets headers and sends initial "hello world" event
- Handles multiple concurrent clients without issues
- Gracefully manages client disconnections
- Fixed all linting issues for clean code
- Maintains 81.1% test coverage

### P1: Integration with Main Workflow

#### Task 4: Integrate Server into CLI ✅ **IMPLEMENTED**
**Acceptance Criteria:**
-   When the `alpine --serve` command is run, the HTTP server starts in the background. ✅
-   The main Alpine workflow (task execution) proceeds as usual while the server is running. ✅
-   The server is gracefully shut down when the main workflow completes or is interrupted. ✅

**Test Cases:**
```go
// TestServeFlagStartsServer verifies that the server is started when --serve is used. ✅
// TestWorkflowRunsConcurrentlyWithServer verifies that the main task is executed while the server is active. ✅
// TestServerShutdownOnWorkflowComplete verifies graceful shutdown on context cancellation. ✅
```

**Implementation:**
-   In `internal/cli/workflow.go`, check if the `--serve` flag is present. ✅
-   If it is, create and start the server in a separate goroutine. ✅
-   Use a context to manage the lifecycle of the server, ensuring it's shut down when the main context is canceled. ✅

**Implementation Date:** 2025-07-27
**Notes:** 
- Implemented server integration in workflow.go using TDD approach
- Server starts concurrently without blocking main workflow
- Proper context-based lifecycle management for graceful shutdown
- Extracted server startup logic into dedicated function for better organization
- Added proper error handling and logging
- Fixed port configuration to support dynamic port assignment (port 0)
- Test coverage includes server startup, concurrent execution, and shutdown scenarios

## Success Criteria

-   [ ] All new code is covered by unit and integration tests.
-   [ ] The `alpine --serve` command starts the server without blocking the main task.
-   [ ] A simple JavaScript client can connect to `http://localhost:3001/events` and receive the "hello world" message.
-   [ ] Existing CLI functionality remains unchanged and fully functional.
-   [ ] The implementation follows the project's coding conventions and quality standards.
