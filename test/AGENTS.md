# Test Directory

This directory contains all test suites for Alpine, including unit tests, integration tests, and end-to-end tests.

## Test Organization

### Directory Structure
- `e2e/` - End-to-end workflow tests
- `integration/` - Integration tests for major components
- `fixtures/` - Test data and mock responses
- `helpers/` - Shared test utilities

### Test Files in Package Directories
Each package contains its own unit tests:
- `*_test.go` - Standard unit tests
- `*_integration_test.go` - Integration tests requiring external resources
- `*_coverage_test.go` - Coverage-specific test cases

## Testing Philosophy

### Test Levels
1. **Unit Tests**: Fast, isolated, no external dependencies
2. **Integration Tests**: Test component interactions, may use external tools
3. **E2E Tests**: Full workflow validation, uses actual Claude Code

### Coverage Goals
- Minimum 80% code coverage for critical paths
- 100% coverage for error handling paths
- Focus on behavior, not implementation details

## Test Patterns

### Table-Driven Tests
```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "result", false},
        {"empty input", "", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Feature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Feature() error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("Feature() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Mock Patterns
```go
type MockExecutor struct {
    ExecuteFunc func(ctx context.Context, config Config) error
}

func (m *MockExecutor) Execute(ctx context.Context, config Config) error {
    if m.ExecuteFunc != nil {
        return m.ExecuteFunc(ctx, config)
    }
    return nil
}
```

### Test Helpers
```go
func setupTestEnvironment(t *testing.T) (cleanup func()) {
    t.Helper()
    
    tmpDir := t.TempDir()
    oldWd, _ := os.Getwd()
    os.Chdir(tmpDir)
    
    return func() {
        os.Chdir(oldWd)
    }
}
```

## Integration Test Guidelines

### Claude Integration Tests
Located in `integration/claude_integration_test.go`:
- Test actual Claude command execution
- Use mock responses when possible
- Verify stream processing
- Test error conditions

### Server Integration Tests
Located in `integration/rest_api_integration_test.go`:
- Full HTTP request/response cycles
- SSE streaming validation
- Concurrent request handling
- Error response verification

### Workflow Integration Tests
Located in `integration/workflow_integration_test.go`:
- Complete workflow execution
- State file management
- Plan generation and approval
- GitHub issue integration

## E2E Test Requirements

### Setup
```bash
# Ensure Claude Code is installed
claude --version

# Set required environment variables
export GEMINI_API_KEY="test-key"
export GITHUB_TOKEN="test-token"

# Run E2E tests
go test ./test/e2e/... -tags=e2e
```

### Test Scenarios
1. **Full Workflow**: Issue → Plan → Implementation → Completion
2. **Error Recovery**: Interrupted workflow resume
3. **Worktree Management**: Creation, isolation, cleanup
4. **Multi-step Tasks**: Complex state transitions
5. **Plan Feedback**: Iteration and refinement

## Test Fixtures

### Mock Responses (`fixtures/claude_responses.json`)
```json
{
  "plan_generation": {
    "output": "# Plan\n## Steps\n1. ...",
    "exitCode": 0
  },
  "implementation": {
    "output": "Implementing feature...",
    "exitCode": 0
  }
}
```

### Test States
```json
{
  "running": {
    "current_step_description": "Implementing feature",
    "next_step_prompt": "Continue implementation",
    "status": "running"
  },
  "completed": {
    "current_step_description": "All tasks complete",
    "next_step_prompt": "",
    "status": "completed"
  }
}
```

## Performance Testing

### Benchmarks
```go
func BenchmarkStreamProcessing(b *testing.B) {
    data := generateLargeOutput()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        processStream(data)
    }
}
```

### Memory Profiling
```bash
go test -bench=. -benchmem -memprofile=mem.prof
go tool pprof mem.prof
```

### Load Testing
```go
func TestConcurrentRequests(t *testing.T) {
    server := startTestServer()
    defer server.Close()
    
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            makeRequest(server.URL)
        }()
    }
    wg.Wait()
}
```

## Test Utilities

### Common Helpers (`helpers/test_helpers.go`)
- `CreateTestRepo()` - Initialize test Git repository
- `MockClaudeCommand()` - Replace Claude with mock
- `CaptureOutput()` - Capture stdout/stderr
- `WaitForState()` - Poll for state changes
- `AssertJSONEqual()` - Compare JSON structures

## CI/CD Integration

### GitHub Actions
```yaml
- name: Run Tests
  run: |
    go test -v -race -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
```

### Pre-commit Hooks
```bash
#!/bin/bash
go test ./... || exit 1
golangci-lint run || exit 1
```

## Debugging Tests

### Verbose Output
```bash
go test -v ./...
```

### Single Test Execution
```bash
go test -run TestSpecificFunction ./internal/cli
```

### Debug Logging
```go
func TestWithDebug(t *testing.T) {
    if testing.Verbose() {
        log.SetLevel(log.DebugLevel)
    }
    // test code
}
```

## Test Maintenance

### Adding New Tests
1. Follow existing patterns in the package
2. Include both positive and negative cases
3. Document complex test scenarios
4. Update fixtures as needed
5. Ensure tests are deterministic

### Updating Tests
1. Run affected tests before changes
2. Update test cases for new behavior
3. Maintain backward compatibility tests
4. Document breaking changes
5. Update integration tests if interfaces change

## Common Issues

### Flaky Tests
- Use proper synchronization (no sleep)
- Mock time-dependent operations
- Isolate file system operations
- Handle race conditions properly

### Slow Tests
- Use t.Parallel() for independent tests
- Mock expensive operations
- Use test caching appropriately
- Profile test execution time

### Test Isolation
- Clean up resources in defer blocks
- Use unique temp directories
- Reset global state between tests
- Avoid test interdependencies