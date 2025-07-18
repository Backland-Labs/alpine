# Test-Driven Development Requirements

This document establishes mandatory TDD practices for all River development. You MUST follow these patterns when implementing any feature or fix.

## TDD Implementation Standards

ALWAYS follow the RED-GREEN-REFACTOR cycle without exceptions. NEVER write implementation code before tests. Each phase has specific requirements you MUST meet.

## RED-GREEN-REFACTOR Implementation Requirements

### RED Phase: Test Creation Standards

You MUST write failing tests before any implementation. FOLLOW these mandatory steps:

#### Required Implementation Steps

1. **ANALYZE the feature requirements** from the Linear issue
2. **DESIGN comprehensive test scenarios** that MUST include:
   - Happy path functionality tests
   - Edge case and boundary tests
   - Error condition tests
   - Integration point tests

3. **CREATE test files** following these requirements:
   - USE descriptive test function names that explain behavior
   - INCLUDE docstrings documenting test purpose
   - WRITE assertions that will fail initially
   - DO NOT write any implementation code

#### Required Test Implementation Pattern

IMPLEMENT tests following this pattern:
```go
// Test file created in RED phase
func TestCreateWorktree_Success(t *testing.T) {
    // Docstring: Tests that CreateWorktree successfully creates a new git worktree
    // with the correct branch name and returns the worktree path.
    // This ensures the core functionality works for the happy path.
    
    // Test will fail until CreateWorktree is implemented
    path, err := git.CreateWorktree("/tmp", "PROJ-123")
    assert.NoError(t, err)
    assert.Contains(t, path, "river-proj-123")
}
```

#### Mandatory Verification Steps

- RUN all tests and VERIFY they fail appropriately
- ENSURE failure messages clearly indicate missing functionality
- CONFIRM no test accidentally passes in RED phase
- FIX any test that passes before implementation

### GREEN Phase: Implementation Requirements

You MUST implement only enough code to pass tests. FOLLOW these strict requirements:

#### Required Implementation Approach

1. **WRITE minimal code** that satisfies test assertions only
2. **AVOID over-engineering** - implement NOTHING beyond test requirements
3. **RUN tests** and VERIFY all pass
4. **DO NOT refactor** - keep code simple, even if repetitive

#### Implementation Guidelines

```go
// Minimal implementation to pass tests
func CreateWorktree(baseDir, issueID string) (string, error) {
    // Just enough to make tests pass
    sanitizedID := strings.ToLower(issueID)
    worktreePath := filepath.Join(baseDir, "river-"+sanitizedID)
    
    // Simple implementation, no optimization
    cmd := exec.Command("git", "worktree", "add", worktreePath)
    err := cmd.Run()
    
    return worktreePath, err
}
```

#### Required Success Validation

- VERIFY all RED phase tests pass
- CONFIRM no functionality exists beyond test requirements
- ACCEPT crude but functional code at this stage

### REFACTOR Phase: Code Quality Requirements

You MUST improve code quality while maintaining all passing tests. FOLLOW these requirements:

#### Required Refactoring Process

1. **IDENTIFY required improvements**:
   - REMOVE code duplication
   - SIMPLIFY complex logic
   - ADD comprehensive error handling
   - OPTIMIZE performance bottlenecks

2. **REFACTOR incrementally** - one change at a time
3. **RUN tests after EVERY change** - ENSURE tests stay green
4. **EXTRACT common patterns** into reusable functions/types

#### Refactoring Examples

```go
// After refactoring: cleaner, more robust
func CreateWorktree(baseDir, issueID string) (string, error) {
    // Extracted sanitization logic
    sanitizedID := SanitizeIssueID(issueID)
    
    // Better naming and structure
    worktreeName := fmt.Sprintf("river-%s", sanitizedID)
    worktreePath := filepath.Join(baseDir, worktreeName)
    
    // Added validation
    if _, err := os.Stat(worktreePath); err == nil {
        return "", fmt.Errorf("worktree already exists at %s", worktreePath)
    }
    
    // Improved error handling
    branchName := sanitizedID
    cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("failed to create worktree: %w\nOutput: %s", err, string(output))
    }
    
    return worktreePath, nil
}
```
