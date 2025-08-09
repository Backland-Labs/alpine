# Implementation Plan

## Overview
This plan adds support for an optional `plan` boolean field in the REST API `/agents/run` endpoint. When `plan` is set to `false`, the workflow will execute without generating a plan.md file, directly proceeding to implementation. This addresses the need for flexibility in workflow execution modes via the REST API.

#### Task 1.1: Update WorkflowEngine Interface ✓
- Acceptance Criteria:
  * The `StartWorkflow` method signature includes a `plan bool` parameter
  * Documentation comments reflect the new parameter's purpose
- Test Cases:
  * Verify compilation with updated interface signature
- Integration Points:
  * All implementations of WorkflowEngine interface must be updated
- Files to Modify/Create:
  * /internal/server/interfaces.go

#### Task 1.2: Update AlpineWorkflowEngine Implementation ✓
- Acceptance Criteria:
  * `StartWorkflow` method accepts and uses the `plan` parameter
  * The `plan` parameter is passed to `workflow.Engine.Run` method
  * AG-UI event metadata contains dynamic `planMode` value based on the parameter
- Test Cases:
  * Test workflow execution with `plan=true` creates plan.md
- Integration Points:
  * Workflow engine's Run method call
  * Event emission with correct metadata
- Files to Modify/Create:
  * /internal/server/workflow_integration.go

#### Task 1.3: Update REST API Handler
- Acceptance Criteria:
  * Handler accepts optional `plan` field in JSON payload
  * Uses pointer type for optional boolean field
  * Defaults to `true` when field is omitted
  * Passes plan value to WorkflowEngine.StartWorkflow
- Test Cases:
  * Test request with `plan: false` passes false to workflow engine
- Integration Points:
  * JSON decoding of request payload
  * WorkflowEngine.StartWorkflow method call
- Files to Modify/Create:
  * /internal/server/handlers.go

#### Task 1.4: Add Validation for Plan Field
- Acceptance Criteria:
  * Non-boolean values for `plan` field return 400 Bad Request
  * Error message clearly indicates invalid field type
  * Follows error handling patterns from specs/error-handling.md
- Test Cases:
  * Test request with `plan: "invalid"` returns validation error
- Integration Points:
  * JSON unmarshaling and type checking
  * Error response handling
- Files to Modify/Create:
  * /internal/server/handlers.go

#### Task 1.5: Update MockWorkflowEngine in server package
- Acceptance Criteria:
  * Mock implementation matches new interface signature
  * StartWorkflowFunc accepts plan parameter
- Test Cases:
  * Verify mock compiles with new signature
- Integration Points:
  * Test files using MockWorkflowEngine
- Files to Modify/Create:
  * /internal/server/workflow_integration_test.go

#### Task 1.6: Update mockWorkflowEngine in cli package
- Acceptance Criteria:
  * Mock implementation in cli package updated for any workflow engine interface changes
  * Maintains compatibility with existing tests
- Test Cases:
  * Verify existing cli tests still pass
- Integration Points:
  * CLI workflow tests
- Files to Modify/Create:
  * /internal/cli/workflow_test.go
  * /internal/cli/workflow_server_test.go
  * /internal/cli/run_test.go
  * /internal/cli/coverage_test.go
  * /internal/cli/worktree_test.go

#### Task 1.7: Add API Handler Tests
- Acceptance Criteria:
  * Test coverage for plan field with true, false, and omitted values
  * Test validation error for non-boolean plan values
  * Test that plan parameter flows correctly to workflow engine
- Test Cases:
  * Test `plan: true` starts workflow with plan generation
- Integration Points:
  * Server test helpers and mocks
- Files to Modify/Create:
  * /internal/server/handlers_test.go

#### Task 1.8: Add Integration Tests
- Acceptance Criteria:
  * End-to-end test verifies `plan=false` skips plan.md creation
  * Test verifies AG-UI events contain correct planMode value
  * Test confirms workflow executes correctly in both modes
- Test Cases:
  * Integration test for workflow without plan generation
- Integration Points:
  * Test workflow engine and event streaming
- Files to Modify/Create:
  * /test/integration/rest_api_integration_test.go
  * /internal/server/workflow_integration_test.go

#### Task 1.9: Update Error Handling Tests
- Acceptance Criteria:
  * Error handling tests include validation scenarios for plan field
  * Tests follow patterns from existing error handling tests
- Test Cases:
  * Test error response structure for invalid plan values
- Integration Points:
  * Error handling middleware and response formatting
- Files to Modify/Create:
  * /internal/server/error_handling_test.go

## Success Criteria
- [ ] WorkflowEngine interface updated with plan parameter
- [ ] AlpineWorkflowEngine passes plan parameter to workflow.Engine.Run
- [ ] REST API accepts optional plan field with proper validation
- [ ] AG-UI events contain dynamic planMode value
- [ ] All mock implementations updated to match new interface
- [ ] Test coverage for plan field functionality
- [ ] Error handling for invalid plan values
- [ ] Integration tests verify end-to-end behavior