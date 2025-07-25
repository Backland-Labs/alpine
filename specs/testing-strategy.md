# Testing Strategy Specification

This document outlines the testing philosophy and strategies for the Alpine project to ensure code quality, reliability, and maintainability.

## 1. Guiding Principles

- **Test for Confidence:** Tests should provide confidence that the system works as expected and that new changes do not break existing functionality.
- **Clarity and Readability:** Tests are documentation. They should be easy to read and understand.
- **Automation:** All tests should be fully automated and runnable with a single command.
- **Speed:** Fast feedback is crucial. Unit tests should be fast, while longer-running tests are reserved for integration and end-to-end suites.

## 2. Types of Tests

The project uses a multi-layered testing approach, including unit, integration, and end-to-end tests.

### 2.1. Unit Tests (`*_test.go`)

- **Purpose:** To test the smallest units of code (functions or methods) in isolation. They are fast and form the majority of the tests.
- **Location:** In the same package as the code they are testing.
- **Characteristics:**
  - No external dependencies (no network, no filesystem, no git).
  - Dependencies are replaced with mocks or stubs.
  - Focus on business logic, edge cases, and error conditions.
- **How to Run:** `go test ./...` or `make test-unit`.

### 2.2. Integration Tests (`test/integration/*_test.go`)

- **Purpose:** To test the interaction between different components of the system. For example, testing the CLI command's interaction with the workflow engine.
- **Location:** In the `test/integration/` directory.
- **Characteristics:**
  - May involve the filesystem (e.g., creating temporary state files).
  - External services like the Claude CLI are mocked.
  - Slower than unit tests.
- **How to Run:** `go test ./test/integration/...` or `make test-integration`.

### 2.3. End-to-End (E2E) Tests (`test/e2e/*_test.go`)

- **Purpose:** To test the entire application from the user's perspective, simulating real-world scenarios.
- **Location:** In the `test/e2e/` directory, marked with the `e2e` build tag.
- **Characteristics:**
  - Involve real external dependencies like `git`.
  - The `alpine` binary is built and executed as a subprocess.
  - The Claude CLI is replaced with a mock script to control its behavior.
  - These are the slowest and most complex tests.
- **How to Run:** `go test -tags=e2e ./test/e2e/...` or `make test-e2e`.

## 3. Mocks and Fakes

To achieve isolation in tests, we rely heavily on interfaces and mock implementations.

### 3.1. Mocking Strategy

- **Interfaces:** Core components define interfaces (e.g., `gitx.WorktreeManager`).
- **Mock Implementations:** For each interface, a mock implementation is provided in a `mock/` subdirectory (e.g., `internal/gitx/mock/manager.go`).
- **Usage:** In unit and integration tests, the real implementation is replaced with the mock, allowing us to control its behavior and verify that methods were called as expected.

### 3.2. Faking External Services

- **Claude CLI:** In integration and E2E tests, the real `claude` command is replaced by a mock script or a mock executor that returns predefined responses. This provides predictable behavior and avoids actual API calls.
- **Fixtures:** Predefined responses for mocked services are stored in fixture files like `test/integration/fixtures/claude_responses.json`. This keeps test data separate from test logic.

## 4. Running Tests

The `Makefile` provides convenient targets for running different test suites.

- `make test`: Runs all unit and integration tests.
- `make test-unit`: Runs only the fast unit tests.
- `make test-integration`: Runs only the integration tests.
- `make test-e2e`: Runs the slow end-to-end tests.
- `make test-all`: Runs all tests, including E2E.
- `make test-coverage`: Runs all tests and generates an HTML coverage report.

## 5. Adding New Tests

When adding new functionality, contributors should follow these guidelines:

1.  **New Logic:** Any new function or method containing business logic must be accompanied by unit tests covering its success paths, failure paths, and edge cases.
2.  **New Features:** A new feature that spans multiple components should have at least one integration test to verify the components work together correctly.
3.  **Critical User Workflows:** Major user-facing workflows (like creating a worktree) should be covered by an E2E test to prevent regressions.
4.  **Bug Fixes:** When fixing a bug, first write a failing test that reproduces the bug. The fix is complete when the test passes.
