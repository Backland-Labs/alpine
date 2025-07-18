# Error Handling Development Guidelines

This document establishes mandatory error handling standards for all River development. Follow these patterns consistently across the codebase.

## Implementation Requirements

When implementing any feature in River, you MUST follow these error handling patterns:
- Go code: ALWAYS return `error` as the last return value
- Bash scripts: VALIDATE all outputs and check exit codes
- Process execution: CAPTURE and log all command outputs

## Required Error Handling Patterns

### Go Error Implementation Standards

FOLLOW these patterns for all Go functions:

```go
// CORRECT: How to implement error handling
func YourFunction(baseDir, issueID string) (string, error) {
    // VALIDATE inputs first
    if issueID == "" {
        return "", fmt.Errorf("issueID cannot be empty")
    }
    
    // CHECK filesystem operations
    if _, err := os.Stat(worktreePath); err == nil {
        return "", fmt.Errorf("worktree already exists at %s", worktreePath)
    }
    
    // CAPTURE command output for debugging
    output, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("failed to create worktree: %w\nOutput: %s", err, string(output))
    }
    
    return worktreePath, nil
}
```

Mandatory patterns:
- USE `fmt.Errorf` with `%w` verb to wrap errors with context
- INCLUDE debugging information (paths, IDs, command output)
- RETURN early on error conditions
- PROVIDE zero value for success type when returning error

### Error Propagation Rules

WRAP errors with context at each layer:

```go
// IMPLEMENT error propagation like this
func runWorkflow(issueID, worktreePath string) error {
    // WRAP errors with operation context
    worktreeFullPath, err := git.CreateWorktree(parentDir, issueID)
    if err != nil {
        return fmt.Errorf("failed to create worktree: %w", err)
    }
    
    // ADD context for each operation
    if err := runner.RunAutoClaudeScript(worktreeFullPath, issueID, scriptPath); err != nil {
        return fmt.Errorf("failed to run auto_claude.sh: %w", err)
    }
    
    return nil
}
```

### CLI Error Display Requirements

HANDLE final errors in main function:

```go
// DISPLAY errors to stderr with proper exit codes
if err := runWorkflow(issueID, worktreePath); err != nil {
    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    os.Exit(1)
}
```

### Bash Script Error Handling Standards

IMPLEMENT JSON validation in all bash scripts:

```bash
# ALWAYS validate JSON outputs
next_issue_json=$(get_next_subissue "$parent_issue_id")
has_next=$(echo "$next_issue_json" | jq -r '.has_next // false')

# CHECK expected values explicitly
if [[ "$has_next" != "true" ]]; then
    echo "No more sub-issues to process!"
    break
fi

# PROVIDE default values for missing fields
plan_status=$(echo "$plan_json" | jq -r '.status // "incomplete"')
if [[ "$plan_status" != "complete" ]]; then
    echo "Failed to create plan, skipping issue"
    continue
fi
```

## Error Categories and Required Handling

| Error Category | Required Handling | Implementation |
|----------------|-------------------|----------------|
| File Operations | WRAP with file path context | `fmt.Errorf("failed to open %s: %w", path, err)` |
| Git Operations | INCLUDE git command output | `fmt.Errorf("git failed: %w\nOutput: %s", err, output)` |
| Process Execution | CAPTURE stdout/stderr | Use `CombinedOutput()` for debugging |
| Validation | SHOW usage and exit | Call `usage()` then `os.Exit(1)` |
| Environment | FAIL FAST with context | `fmt.Errorf("env error: %w", err)` |

## Mandatory Error Handling Practices

### 1. NEVER Ignore Errors
```go
// CORRECT: Always check and handle
if err := cmd.Start(); err != nil {
    return fmt.Errorf("failed to start script: %w", err)
}

// WRONG: Never ignore errors
cmd.Start()  // FORBIDDEN
```

### 2. ALWAYS Provide Context
```go
// CORRECT: Context explains the operation
return fmt.Errorf("failed to copy file contents: %w", err)

// WRONG: No context
return err  // INSUFFICIENT
```

### 3. INCLUDE Debugging Information
```go
// ALWAYS capture command output
return fmt.Errorf("failed to create worktree: %w\nOutput: %s", err, string(output))
```

### 4. CLASSIFY Error Severity
- Non-critical: LOG but continue (e.g., cleanup failures)
- Critical: STOP execution immediately (e.g., missing dependencies)

### 5. VALIDATE Inputs First
```go
// VALIDATE before processing
if len(args) != 1 {
    usage()  // SHOW help and exit
}

if issueID == "" {
    fmt.Fprintf(os.Stderr, "Error: LINEAR-ISSUE-ID cannot be empty\n")
    usage()
}
```

## Required Exit Codes

| Code | Usage | When to Use |
|------|-------|-------------|
| 0 | Success | ONLY when all operations complete successfully |
| 1 | General Error | ANY error or validation failure |

## Error Handling Implementation Checklist

When implementing new functionality, FOLLOW this checklist:

1. **RETURN errors** from all functions that can fail
2. **WRAP errors** with context using `fmt.Errorf("context: %w", err)`
3. **INCLUDE debugging info** (file paths, command output, IDs)
4. **VALIDATE inputs** before any processing
5. **USE early returns** for all error conditions
6. **LET errors bubble up** to appropriate handlers

Required implementation pattern:
```go
func YourNewFeature(input string) error {
    // 1. VALIDATE input first
    if input == "" {
        return fmt.Errorf("input cannot be empty")
    }
    
    // 2. PERFORM operation with error handling
    result, err := performOperation(input)
    if err != nil {
        // 3. WRAP with context and input info
        return fmt.Errorf("failed to perform operation on %q: %w", input, err)
    }
    
    // 4. ONLY return nil on complete success
    return nil
}
```