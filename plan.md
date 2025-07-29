# Implementation Plan: REST API Endpoints for Alpine

## Overview

This plan outlines the implementation of REST API endpoints that build on Alpine's existing HTTP server infrastructure. The goal is to enable programmatic workflow management while maintaining the current SSE functionality and following TDD principles.

### Objectives
- Add 10 REST API endpoints for agent and workflow management
- Integrate with existing workflow engine and state management
- Maintain backwards compatibility with current SSE implementation
- Ensure comprehensive testing and documentation updates

## Previous Implementation (Completed)

### Basic HTTP Server with SSE ✅ **IMPLEMENTED**
- Added `--serve` and `--port` CLI flags ✅
- Created `internal/server` package with HTTP server ✅
- Implemented `/events` SSE endpoint ✅
- Integrated server into CLI workflow ✅
- Achieved 81.1% test coverage ✅

## Current Implementation: REST API Endpoints

### Task 1: Create REST API Data Models (TDD) ✅ **IMPLEMENTED**
**Acceptance Criteria:**
- Define `Agent`, `Run`, and `Plan` structs in `internal/server/models.go` ✅
- Add comprehensive validation and JSON serialization ✅
- Create full test coverage in `internal/server/models_test.go` ✅
- Map data models to Alpine's existing workflow patterns ✅

**Key Data Structures:**
```go
type Agent struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
}

type Run struct {
    ID        string    `json:"id"`
    AgentID   string    `json:"agent_id"`
    Status    string    `json:"status"` // running, completed, cancelled, failed
    Issue     string    `json:"issue"`
    Created   time.Time `json:"created"`
    Updated   time.Time `json:"updated"`
    WorktreeDir string  `json:"worktree_dir,omitempty"`
}

type Plan struct {
    RunID     string    `json:"run_id"`
    Content   string    `json:"content"`
    Status    string    `json:"status"` // pending, approved, rejected
    Created   time.Time `json:"created"`
    Updated   time.Time `json:"updated"`
}
```

**Test Cases:**
```go
// TestAgentValidation - Test agent struct validation and JSON marshaling
// TestRunLifecycle - Test run status transitions and validation
// TestPlanStatus - Test plan approval workflow
// TestModelSerialization - Test JSON serialization/deserialization
```

### Task 2: Implement REST API Handlers (TDD) ✅ **IMPLEMENTED**
**Acceptance Criteria:**
- Add all 10 REST endpoints to `internal/server/server.go`
- Each endpoint has comprehensive unit tests
- Follow existing server patterns and error handling
- Integrate with Alpine's workflow engine

**Endpoints to Implement:**
1. `POST /agents/run` - Start workflow with GitHub issue
2. `GET /agents/list` - Return available agents (hardcoded MVP)
3. `GET /runs` - List all runs from in-memory store
4. `GET /runs/{id}` - Get specific run details
5. `GET /runs/{id}/events` - SSE endpoint for run-specific events
6. `POST /runs/{id}/cancel` - Cancel running workflow
7. `GET /plans/{runId}` - Get plan.md content
8. `POST /plans/{runId}/approve` - Approve plan and continue
9. `POST /plans/{runId}/feedback` - Send feedback on plan
10. `GET /health` - Health check endpoint

**Test Cases:**
```go
// TestHealthEndpoint - Simple health check
// TestAgentsRunEndpoint - Start workflow from GitHub issue
// TestAgentsListEndpoint - Return agent list
// TestRunsListEndpoint - List all runs
// TestRunDetailsEndpoint - Get specific run
// TestRunEventsSSE - Run-specific SSE events
// TestRunCancelEndpoint - Cancel workflow
// TestPlanGetEndpoint - Retrieve plan content
// TestPlanApproveEndpoint - Approve plan workflow
// TestPlanFeedbackEndpoint - Send plan feedback
```

### Task 3: Integrate with Workflow Engine ⏳ **PENDING**
**Acceptance Criteria:**
- Connect REST API to existing workflow execution
- Enable workflow start/stop/cancel from API calls
- Broadcast events to both global and run-specific SSE clients
- Maintain existing state management patterns

**Integration Points:**
- Modify `internal/cli/workflow.go` to accept GitHub issue URLs
- Add event broadcasting hooks for REST API clients
- Connect run lifecycle to existing `agent_state.json` management
- Handle workflow cancellation through context propagation

### Task 4: Update Server Specification ⏳ **PENDING**
**Files to Update:**
- `specs/server.md` - Add complete REST API documentation
- Include OpenAPI-style endpoint specifications
- Add request/response examples for each endpoint
- Document error handling and status codes

### Task 5: Update User Documentation ⏳ **PENDING**
**Files to Update:**
- `CLAUDE.md` - Add REST API usage examples
- `specs/cli-commands.md` - Update server mode documentation
- Include curl examples and integration patterns

### Task 6: Comprehensive Integration Testing ⏳ **PENDING**
**Acceptance Criteria:**
- End-to-end tests for workflow start/stop via API
- Test actual GitHub issue processing
- Verify SSE events work for individual runs
- Test concurrent API usage and server stability
- Achieve >85% test coverage

### Task 7: Verify Test Coverage ⏳ **PENDING**
**Requirements:**
- Run `go test -cover ./...` to verify coverage
- Target >85% test coverage for new REST API code
- Fix any coverage gaps with additional tests

### Task 8: Update Implementation Status ⏳ **PENDING**
**Final Task:**
- Update this plan.md with final implementation status
- Document any deviations from original plan
- Note future enhancement opportunities

## Success Criteria

- **Functionality**: All 10 REST endpoints operational and tested
- **Integration**: REST API properly connects to Alpine workflow engine
- **Testing**: >85% test coverage with comprehensive integration tests
- **Documentation**: Complete API documentation with examples
- **Backwards Compatibility**: Existing SSE and CLI functionality unchanged
- **Code Quality**: Passes linting and follows project conventions

## MVP Constraints

**In-Memory Storage**: Store runs/plans in memory (defer persistence)
**Hardcoded Agents**: Return static agent list initially
**Basic Plan Flow**: Simple approve/reject workflow
**No Authentication**: Security deferred to future iterations

## Technical Architecture

**Data Flow**: REST API → Workflow Engine → State Management → SSE Events
**Storage**: Extend existing in-memory state management
**Concurrency**: Use existing goroutine patterns and context cancellation
**Error Handling**: Follow Alpine's existing error handling patterns

## Implementation Status

- [x] Basic HTTP Server with SSE (Previous work)
- [x] REST API Data Models (Completed - 2025-07-29)
- [x] REST API Handlers Implementation (Completed - 2025-07-29)
- [ ] Workflow Engine Integration
- [ ] Documentation Updates
- [ ] Comprehensive Testing
- [ ] Test Coverage Verification
- [ ] Final Status Update

**Current Phase**: REST API Handlers completed with full test coverage
**Next Phase**: Integrate with Workflow Engine

**Implementation Notes (Task 1)**:
- Added comprehensive validation for all models
- Implemented state machine logic for status transitions
- Created GenerateID utility for unique identifier generation
- Achieved 100% test coverage for models package
- All tests follow TDD methodology (RED-GREEN-REFACTOR)

**Implementation Notes (Task 2)**:
- Created handlers.go file for REST API endpoint implementations
- Added all 10 REST endpoints with proper HTTP method validation
- Implemented in-memory storage for runs and plans
- Thread-safe access with mutex protection
- Comprehensive test coverage (79.4% for server package)
- All tests follow TDD methodology (RED-GREEN-REFACTOR)
- MVP constraints maintained (hardcoded agents, in-memory storage)