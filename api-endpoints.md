# Alpine REST API Endpoints

This document provides a detailed overview of the client-facing REST API endpoints for the Alpine CLI server. The server is activated by running Alpine with the `--serve` flag.

## Health Check

Provides a simple health check to verify that the Alpine server is running and responsive.

- **Endpoint**: `GET /health`
- **Description**: Checks the health status of the server.
- **Request**: No parameters.
- **Example Request**:
  ```bash
  curl http://localhost:3001/health
  ```
- **Success Response** (200 OK):
  ```json
  {
    "status": "healthy",
    "service": "alpine-server",
    "timestamp": "2025-08-08T10:00:00Z"
  }
  ```

## Agent Management

### List Available Agents

Retrieves a list of all available agents that can be used to start workflows.

- **Endpoint**: `GET /agents/list`
- **Description**: Returns a list of available agents.
- **Request**: No parameters.
- **Example Request**:
  ```bash
  curl http://localhost:3001/agents/list
  ```
- **Success Response** (200 OK):
  ```json
  [
    {
      "id": "alpine-agent",
      "name": "Alpine Workflow Agent",
      "description": "Default agent for running Alpine workflows from GitHub issues"
    }
  ]
  ```

### Start Workflow

Starts a new workflow from a given GitHub issue URL.

- **Endpoint**: `POST /agents/run`
- **Description**: Initiates a new workflow run.
- **Request Body**:
  ```json
  {
    "github_issue_url": "https://github.com/owner/repo/issues/123",
    "plan": true
  }
  ```
  - `github_issue_url` (string, required): The URL of the GitHub issue to process.
  - `plan` (boolean, optional): Whether to generate a `plan.md` file. Defaults to `true`.
- **Example Request**:
  ```bash
  curl -X POST http://localhost:3001/agents/run \
    -H "Content-Type: application/json" \
    -d '{"github_issue_url": "https://github.com/owner/repo/issues/123"}'
  ```
- **Success Response** (201 Created):
  ```json
  {
    "run_id": "run_abc123",
    "status": "running",
    "message": "Workflow started successfully"
  }
  ```

## Workflow Run Management

### List All Runs

Retrieves a list of all workflow runs, including their current status and details.

- **Endpoint**: `GET /runs`
- **Description**: Returns a list of all workflow runs.
- **Request**: No parameters.
- **Example Request**:
  ```bash
  curl http://localhost:3001/runs
  ```
- **Success Response** (200 OK):
  ```json
  [
    {
      "id": "run_abc123",
      "agent_id": "alpine-agent",
      "status": "running",
      "issue": "https://github.com/owner/repo/issues/123",
      "created": "2025-08-08T10:00:00Z",
      "updated": "2025-08-08T10:01:00Z",
      "worktree_dir": "/path/to/worktree"
    }
  ]
  ```

### Get Run Details

Retrieves detailed information about a specific workflow run.

- **Endpoint**: `GET /runs/{run-id}`
- **Description**: Returns details for a specific run.
- **Path Parameters**:
  - `run-id` (string, required): The ID of the workflow run.
- **Example Request**:
  ```bash
  curl http://localhost:3001/runs/run_abc123
  ```
- **Success Response** (200 OK):
  ```json
  {
    "id": "run_abc123",
    "agent_id": "alpine-agent",
    "status": "running",
    "issue": "https://github.com/owner/repo/issues/123",
    "created": "2025-08-08T10:00:00Z",
    "updated": "2025-08-08T10:01:00Z",
    "worktree_dir": "/path/to/worktree"
  }
  ```

### Monitor Run Events (SSE)

Subscribes to a real-time stream of events for a specific workflow run using Server-Sent Events (SSE).

- **Endpoint**: `GET /runs/{run-id}/events`
- **Description**: Streams real-time events for a run.
- **Path Parameters**:
  - `run-id` (string, required): The ID of the workflow run.
- **Example Request**:
  ```bash
  curl -N http://localhost:3001/runs/run_abc123/events
  ```
- **Event Stream**:
  ```
  event: run_started
  data: {"type":"run_started", "runId":"run_abc123", ...}

  event: text_message_content
  data: {"type":"text_message_content", "runId":"run_abc123", "content":"...", ...}

  event: run_finished
  data: {"type":"run_finished", "runId":"run_abc123", ...}
  ```

### Cancel Workflow

Cancels a currently running workflow.

- **Endpoint**: `POST /runs/{run-id}/cancel`
- **Description**: Cancels a running workflow.
- **Path Parameters**:
  - `run-id` (string, required): The ID of the workflow run.
- **Example Request**:
  ```bash
  curl -X POST http://localhost:3001/runs/run_abc123/cancel
  ```
- **Success Response** (200 OK):
  ```json
  {
    "status": "cancelled",
    "runId": "run_abc123"
  }
  ```

## Plan Management

### Get Plan Content

Retrieves the content of the implementation plan for a specific run.

- **Endpoint**: `GET /plans/{run-id}`
- **Description**: Returns the content of a plan.
- **Path Parameters**:
  - `run-id` (string, required): The ID of the workflow run.
- **Example Request**:
  ```bash
  curl http://localhost:3001/plans/run_abc123
  ```
- **Success Response** (200 OK):
  ```json
  {
    "run_id": "run_abc123",
    "content": "# Implementation Plan\n\n...",
    "status": "pending",
    "created": "2025-08-08T10:00:00Z",
    "updated": "2025-08-08T10:01:00Z"
  }
  ```

### Approve Plan

Approves a pending plan, allowing the workflow to proceed with implementation.

- **Endpoint**: `POST /plans/{run-id}/approve`
- **Description**: Approves a plan for execution.
- **Path Parameters**:
  - `run-id` (string, required): The ID of the workflow run.
- **Example Request**:
  ```bash
  curl -X POST http://localhost:3001/plans/run_abc123/approve
  ```
- **Success Response** (200 OK):
  ```json
  {
    "status": "approved",
    "runId": "run_abc123"
  }
  ```

### Send Plan Feedback

Submits feedback on a generated plan, which can be used to request revisions.

- **Endpoint**: `POST /plans/{run-id}/feedback`
- **Description**: Submits feedback on a plan.
- **Path Parameters**:
  - `run-id` (string, required): The ID of the workflow run.
- **Request Body**:
  ```json
  {
    "feedback": "Please add more details to the testing phase."
  }
  ```
  - `feedback` (string, required): The feedback text.
- **Example Request**:
  ```bash
  curl -X POST http://localhost:3001/plans/run_abc123/feedback \
    -H "Content-Type: application/json" \
    -d '{"feedback": "Please add error handling for edge cases"}'
  ```
- **Success Response** (200 OK):
  ```json
  {
    "status": "feedback_received",
    "runId": "run_abc123"
  }
  ```
