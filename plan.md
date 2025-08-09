# Implementation Plan

## Overview

This plan addresses GitHub Issue #62 to increase observability in the Alpine project by capturing and emitting Claude's internal tool calls to the SSE endpoint. The implementation focuses on minimal viable observability, extending existing infrastructure rather than creating new systems, and ensuring AG-UI compliance with proper performance considerations.

The approach prioritizes simplicity by extending the existing `alpine-ag-ui-emitter.rs` hook script, leveraging current logging infrastructure where possible, and implementing core observability features (start/end/error events) with built-in performance safeguards including event batching, throttling, and sampling.

## Feature 1: Enhanced AG-UI Event Types and Structures [IMPLEMENTED]

#### Task 1.1: Extend AG-UI Event Types for Tool Calls
- Acceptance Criteria:
  * Add PascalCase event type constants: `ToolCallStart`, `ToolCallEnd`, `ToolCallError`
  * Define internal snake_case event types: `tool_call_started`, `tool_call_finished`, `tool_call_error`
  * Ensure event type validation includes new tool call events
- Test Cases:
  * Test that new event types are recognized as valid AG-UI events and map correctly between internal and external formats
- Integration Points:
  * Update ValidAGUIEventTypes map in agui_types.go
  * Ensure backward compatibility with existing event types
- Files to Modify/Create:
  * internal/events/agui_types.go

#### Task 1.2: Define Complete Tool Call Event Structures with BaseEvent Interface
- Acceptance Criteria:
  * Create BaseEvent interface with common fields (id, timestamp, type, run_id)
  * Implement ToolCallStartEvent, ToolCallEndEvent, and ToolCallErrorEvent structures
  * Include essential fields: tool name, correlation ID, timing, and minimal metadata
  * Follow AG-UI protocol specifications for event structure
- Test Cases:
  * Test event structure serialization, deserialization, and BaseEvent interface compliance
- Integration Points:
  * Integrate with existing WorkflowEvent structure
  * Ensure compatibility with SSE event streaming
- Files to Modify/Create:
  * internal/events/agui_types.go
  * internal/server/interfaces.go

## Feature 2: Extend Existing Hook Infrastructure

#### Task 2.1: Extend alpine-ag-ui-emitter.rs for Tool Call Capture
- Acceptance Criteria:
  * Extend existing `alpine-ag-ui-emitter.rs` to handle PreToolUse and PostToolUse events
  * Add tool call correlation ID generation and tracking
  * Implement event batching with configurable batch size (default: 10 events)
  * Add event sampling for high-frequency tools (configurable rate, default: 100%)
- Test Cases:
  * Test extended hook script with various Claude tools and batching behavior
- Integration Points:
  * Maintain compatibility with existing AG-UI event emission
  * Ensure proper integration with Claude Code hook system
- Files to Modify/Create:
  * hooks/alpine-ag-ui-emitter.rs

#### Task 2.2: Enhance Hook Configuration for Tool Call Events
- Acceptance Criteria:
  * Extend existing hook configuration to capture PreToolUse and PostToolUse events
  * Configure hooks with proper matchers for all tool types
  * Add `ALPINE_TOOL_CALL_EVENTS_ENABLED` environment variable (default: false)
  * Add `ALPINE_TOOL_CALL_BATCH_SIZE` and `ALPINE_TOOL_CALL_SAMPLE_RATE` configuration
- Test Cases:
  * Test hook configuration with feature toggles and performance settings
- Integration Points:
  * Extend existing agui_hooks.go functionality
  * Integrate with centralized configuration system
- Files to Modify/Create:
  * internal/claude/agui_hooks.go
  * internal/config/config.go

## Feature 3: Event Processing and Performance

#### Task 3.1: Implement Event Batching and Throttling System
- Acceptance Criteria:
  * Create event batching system with configurable flush intervals (default: 1 second)
  * Implement backpressure handling to prevent memory overflow
  * Add event throttling with configurable rate limits (default: 100 events/second)
  * Ensure acceptable overhead limits (< 5% CPU, < 10MB memory)
- Test Cases:
  * Test batching behavior under high-frequency tool call scenarios
- Integration Points:
  * Integrate with existing EventEmitter interface
  * Ensure compatibility with SSE broadcasting system
- Files to Modify/Create:
  * internal/events/emitter.go
  * internal/server/server.go

#### Task 3.2: Create POST /events/tool-calls Endpoint with Authentication
- Acceptance Criteria:
  * Implement POST `/events/tool-calls` endpoint for hook event reception
  * Add authentication using existing server authentication mechanisms
  * Validate incoming tool call event data against AG-UI schema
  * Forward validated events to batching system
- Test Cases:
  * Test endpoint authentication, validation, and event processing
- Integration Points:
  * Integrate with existing HTTP server infrastructure and authentication
  * Ensure proper error handling and logging
- Files to Modify/Create:
  * internal/server/handlers.go

## Feature 4: Workflow Integration and Configuration

#### Task 4.1: Integrate Tool Call Hooks with Workflow Execution
- Acceptance Criteria:
  * Enable tool call hooks during workflow execution in server mode
  * Configure hook environment variables with proper event endpoints
  * Ensure hooks are properly cleaned up after workflow completion
- Test Cases:
  * Test end-to-end workflow execution with tool call event emission
- Integration Points:
  * Integrate with existing workflow execution in workflow_integration.go
  * Ensure compatibility with StreamingExecutor interface
- Files to Modify/Create:
  * internal/server/workflow_integration.go
  * internal/claude/executor.go

#### Task 4.2: Centralize Observability Configuration
- Acceptance Criteria:
  * Add observability configuration section to existing config system
  * Implement `ALPINE_*` environment variable pattern for all tool call settings
  * Provide sensible defaults with feature disabled by default
  * Add CLI flags for common configuration options
- Test Cases:
  * Test configuration loading, validation, and environment variable handling
- Integration Points:
  * Integrate with existing configuration management in config.go
  * Ensure proper validation and default values
- Files to Modify/Create:
  * internal/config/config.go
  * internal/cli/root.go

## Feature 5: Enhanced SSE Event Broadcasting

#### Task 5.1: Extend Event Broadcasting for Tool Call Events
- Acceptance Criteria:
  * Extend existing BroadcastEvent functionality to handle batched tool call events
  * Maintain event ordering and correlation with existing workflow events
  * Implement event replay buffer with size limits (default: 1000 events)
- Test Cases:
  * Test tool call event broadcasting and replay functionality
- Integration Points:
  * Integrate with existing SSE connection management
  * Ensure compatibility with run-specific and global SSE endpoints
- Files to Modify/Create:
  * internal/server/server.go
  * internal/server/run_specific_sse.go

#### Task 5.2: Implement Event Correlation and Sequencing
- Acceptance Criteria:
  * Correlate tool call events with workflow steps using run_id and step context
  * Maintain proper event sequencing with timestamps and sequence numbers
  * Handle concurrent tool calls with unique correlation IDs
- Test Cases:
  * Test event correlation with complex workflows and concurrent tool calls
- Integration Points:
  * Integrate with existing workflow state management
  * Ensure proper handling of nested and concurrent operations
- Files to Modify/Create:
  * internal/server/workflow_integration.go

## Feature 6: Error Handling and Resilience

#### Task 6.1: Implement Robust Error Handling
- Acceptance Criteria:
  * Ensure workflow continues even if tool call event emission fails
  * Implement circuit breaker pattern for hook failures (fail after 5 consecutive errors)
  * Provide structured logging for debugging and monitoring
  * Add graceful degradation when event system is overloaded
- Test Cases:
  * Test error scenarios, recovery mechanisms, and circuit breaker behavior
- Integration Points:
  * Integrate with existing error handling and logging systems
  * Ensure backward compatibility and no impact on core functionality
- Files to Modify/Create:
  * internal/claude/agui_hooks.go
  * hooks/alpine-ag-ui-emitter.rs

#### Task 6.2: Implement Feature Toggles and Monitoring
- Acceptance Criteria:
  * Add runtime feature toggles for tool call event emission
  * Implement basic metrics collection (event counts, error rates, processing times)
  * Provide health check endpoint for observability system status
- Test Cases:
  * Test feature toggles, metrics collection, and health monitoring
- Integration Points:
  * Integrate with existing server health check infrastructure
  * Ensure minimal performance impact when disabled
- Files to Modify/Create:
  * internal/server/handlers.go
  * internal/config/config.go

## Feature 7: Testing and Validation

#### Task 7.1: Implement Core Unit Tests
- Acceptance Criteria:
  * Create unit tests for tool call event structures and BaseEvent interface
  * Test event batching, throttling, and sampling logic
  * Test configuration loading and validation
- Test Cases:
  * Test all new event types, batching behavior, and configuration options
- Integration Points:
  * Integrate with existing test infrastructure
  * Ensure tests run in CI/CD pipeline
- Files to Modify/Create:
  * internal/events/agui_types_test.go
  * internal/events/emitter_test.go
  * internal/config/config_test.go

#### Task 7.2: Implement Integration Tests
- Acceptance Criteria:
  * Create integration tests for end-to-end tool call event emission
  * Test hook system integration with real Claude Code execution
  * Validate AG-UI protocol compliance and SSE delivery
- Test Cases:
  * Test complete workflow with tool call events, batching, and SSE streaming
- Integration Points:
  * Integrate with existing integration test framework
  * Ensure tests work with actual Claude Code hook execution
- Files to Modify/Create:
  * test/integration/tool_call_events_test.go
  * internal/claude/agui_hooks_test.go

#### Task 7.3: Implement Performance Testing
- Acceptance Criteria:
  * Create performance tests for high-frequency tool call scenarios
  * Test system behavior under load with batching and throttling
  * Validate memory usage, CPU overhead, and cleanup behavior
- Test Cases:
  * Test system performance with concurrent tool calls, multiple SSE clients, and various batch sizes
- Integration Points:
  * Integrate with existing performance testing infrastructure
  * Ensure tests validate acceptable overhead limits
- Files to Modify/Create:
  * test/performance/tool_call_events_test.go

## Success Criteria

- [ ] Core tool call events (start/end/error) are captured for essential Claude Code tool executions
- [ ] Events follow AG-UI protocol with proper PascalCase naming and BaseEvent interface
- [ ] Event batching and throttling prevent system overload with configurable limits
- [ ] POST /events/tool-calls endpoint handles authenticated hook requests
- [ ] SSE endpoints deliver batched tool call events with proper correlation
- [ ] Feature is disabled by default with ALPINE_* environment variable configuration
- [ ] System maintains < 5% CPU and < 10MB memory overhead when enabled
- [ ] Circuit breaker prevents cascade failures from hook system issues
- [ ] Existing alpine-ag-ui-emitter.rs is extended rather than replaced
- [ ] Configuration is centralized in existing config system with sensible defaults
- [ ] Comprehensive test coverage validates functionality, performance, and error handling
- [ ] Integration works seamlessly with existing StreamingExecutor and EventEmitter interfaces