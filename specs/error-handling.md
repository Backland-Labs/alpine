# Error Handling Specification

## Core Principles

- **Explicit handling**: Check and handle all errors
- **No panics**: Return errors instead of panicking in production
- **Clear messages**: Provide actionable error messages to users

## Basic Patterns

### Standard Error Handling
```go
data, err := ReadFile(path)
if err != nil {
    return fmt.Errorf("reading file: %w", err)
}
```

### Sentinel Errors
```go
var (
    ErrNotFound = errors.New("not found")
    ErrInvalid  = errors.New("invalid input")
)
```

### Error Wrapping
Always wrap errors with context:
```go
if err := operation(); err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

## CLI Error Display

- Show user-friendly messages in normal mode
- Include full error chain in debug/verbose mode
- Use appropriate exit codes (1 for general errors, 2 for usage errors)

## Key Rules

1. Never ignore errors
2. Add context when propagating errors
3. Log OR return, not both
4. Keep error messages concise and actionable