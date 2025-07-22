# Code Quality Specification

## Overview

This document defines the code quality standards and linting requirements for the River project. All code must pass these checks before being considered production-ready.

## Linting Tools

The project uses `golangci-lint` as the primary code quality tool, which includes multiple linters:

- **errcheck**: Ensures all error return values are checked
- **staticcheck**: Performs static analysis for common Go mistakes
- **gofmt**: Ensures code is properly formatted
- **govet**: Reports suspicious constructs

## Error Handling Standards

### 1. All Errors Must Be Handled

Every function that returns an error must have that error checked or explicitly ignored.

**Bad:**
```go
os.Setenv("KEY", "value")  // Error not checked
defer file.Close()          // Error not checked
```

**Good:**
```go
// Option 1: Check the error
if err := os.Setenv("KEY", "value"); err != nil {
    return fmt.Errorf("failed to set env: %w", err)
}

// Option 2: Explicitly ignore with blank identifier
_ = os.Setenv("KEY", "value")

// Option 3: For deferred cleanup, use anonymous function
defer func() {
    if err := file.Close(); err != nil {
        // Log or handle cleanup error
    }
}()
```

### 2. Error Messages

Error messages should follow Go conventions:
- Start with lowercase letter
- Be concise and descriptive
- Use `fmt.Errorf` with `%w` for wrapping errors

**Bad:**
```go
return fmt.Errorf("Failed to open file")  // Capitalized
```

**Good:**
```go
return fmt.Errorf("failed to open file: %w", err)
```

## Output and Logging

### 1. Output Functions

For functions that write to stdout/stderr (logging, progress, user messages):

**When error handling is not critical:**
```go
// Use blank identifier for fmt.Fprintf/Fprintln in output functions
_, _ = fmt.Fprintf(w, "message: %s\n", msg)
```

**When error handling is important:**
```go
// Return errors from functions that can fail
func WriteReport(w io.Writer, data string) error {
    if _, err := fmt.Fprintf(w, "Report: %s\n", data); err != nil {
        return fmt.Errorf("failed to write report: %w", err)
    }
    return nil
}
```

### 2. Test Cleanup

In tests, cleanup errors can be ignored since they don't affect test results:

```go
defer func() {
    if err := os.RemoveAll(tmpDir); err != nil {
        // Ignore cleanup errors in tests
    }
}()

// Or use blank identifier
defer func() { _ = os.RemoveAll(tmpDir) }()
```

## Import Standards

### 1. Use Modern Go Packages

Replace deprecated packages with their modern equivalents:

**Bad:**
```go
import "io/ioutil"

data, err := ioutil.ReadFile("file.txt")
```

**Good:**
```go
import "os"

data, err := os.ReadFile("file.txt")
```

Common replacements:
- `ioutil.ReadFile` → `os.ReadFile`
- `ioutil.WriteFile` → `os.WriteFile`
- `ioutil.ReadDir` → `os.ReadDir`
- `ioutil.TempFile` → `os.CreateTemp`
- `ioutil.TempDir` → `os.MkdirTemp`

## Static Analysis Compliance

### 1. Empty Error Blocks

Avoid empty blocks with only comments. Staticcheck (SA9003) flags these as potentially incorrect error handling.

**Bad:**
```go
if err := doSomething(); err != nil {
    // Ignore error - not critical
}
```

**Good:**
```go
// Option 1: Use blank identifier
_ = doSomething()

// Option 2: Actually handle or log the error
if err := doSomething(); err != nil {
    log.Printf("non-critical error: %v", err)
}
```

## Testing Standards

### 1. Linting Compliance Tests

The project includes automated tests to ensure linting compliance:

```go
// internal/quality/lint_test.go
func TestLintingCompliance(t *testing.T) {
    // Runs golangci-lint and fails if any issues found
}
```

### 2. CI/CD Integration

All pull requests must pass linting checks before merge:
- `make lint` runs golangci-lint locally
- CI pipeline runs the same checks automatically
- No linting warnings are acceptable in production code

## Development Workflow

1. **Before Committing:**
   ```bash
   go fmt ./...            # Format code
   golangci-lint run       # Check for linting issues
   go test ./...           # Run tests
   ```

2. **Fixing Linting Issues:**
   - Address errors systematically by category
   - Use automated fixes where available: `golangci-lint run --fix`
   - Run tests after fixes to ensure nothing broke

3. **Continuous Monitoring:**
   - The `internal/quality` package contains tests that enforce these standards
   - These tests run as part of the regular test suite

## Rationale

These standards ensure:
1. **Reliability**: Proper error handling prevents silent failures
2. **Maintainability**: Consistent style makes code easier to read and modify
3. **Security**: Checked errors prevent security vulnerabilities
4. **Performance**: Modern packages often have performance improvements
5. **Best Practices**: Following Go community standards