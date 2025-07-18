# Type Usage Requirements

This document establishes mandatory type conventions for River development. You MUST follow these patterns when implementing any feature.

## Required Type Usage

You MUST use these standard Go types for the following purposes:

| Type Category | Go Types | Usage in River |
|---------------|----------|----------------|
| String | `string` | Issue IDs, file paths, branch names, command output |
| Boolean | `bool` | Function return status (implicit via error) |
| Integer | `int` | Array lengths, indices |
| Error | `error` | All functions return error as last value |

## String Type Requirements

You MUST use string types for these specific purposes:

| Usage | Type | Example | Description |
|-------|------|---------|-------------|
| Issue IDs | `string` | `"PROJ-123"` | Linear issue identifiers |
| File Paths | `string` | `"../river-proj-123"` | Worktree and script paths |
| Branch Names | `string` | `"proj-123"` | Git branch identifiers |
| Command Args | `[]string` | `[]string{"add", "-b", branch}` | Shell command arguments |

## File I/O Type Standards

You MUST use these standard library types for file operations:

| Type | Usage | Example |
|------|-------|---------|
| `*os.File` | File handles | `os.Open(src)` |
| `os.FileMode` | File permissions | `sourceInfo.Mode()` |
| `os.FileInfo` | File metadata | `sourceFile.Stat()` |
| `io.Reader` | Reading data | Source for `io.Copy()` |
| `io.Writer` | Writing data | Destination for `io.Copy()` |
| `*exec.Cmd` | Process execution | Git and script commands |

## Required Function Signature Patterns

You MUST follow these function signature patterns:

```go
// Action functions MUST return error as last value
func CreateWorktree(baseDir, issueID string) (string, error)
func RemoveWorktree(worktreePath string) error

// Pure transformation functions MUST NOT return errors
func SanitizeIssueID(issueID string) string

// I/O operations MUST return only error
func copyFile(src, dst string) error
```

### Function Signature Rules

1. **ALWAYS return `error` as the last return value** for any function that can fail
2. **NEVER return `error` from pure functions** that only transform data
3. **USE descriptive parameter names** that indicate purpose
4. **RETURN concrete types**, not interfaces (unless explicitly required)

## Custom Type Implementation Requirements

You SHOULD implement these custom types to improve type safety:

| Custom Type | Definition | Purpose |
|-------------|------------|---------|
| `IssueID` | `type IssueID string` | Distinguish issue IDs from regular strings |
| `WorktreePath` | `type WorktreePath string` | Type-safe file paths for worktrees |
| `BranchName` | `type BranchName string` | Git branch identifiers |

### Required Implementation Pattern:
```go
// DEFINE custom types for domain concepts
type IssueID string
type WorktreePath string
type BranchName string

// USE custom types in function signatures
func CreateWorktree(baseDir string, id IssueID) (WorktreePath, error)

// IMPLEMENT type conversion methods
func (id IssueID) ToBranchName() BranchName {
    // SANITIZE and convert
    return BranchName(SanitizeIssueID(string(id)))
}
```

## Mandatory Type Conventions

### Zero Value Requirements
- **USE `""`** (empty string) to indicate uninitialized strings
- **RETURN `nil`** to indicate no error condition
- **AVOID pointers** unless absolutely necessary

### Type Validation Requirements
- **VALIDATE all inputs** at function boundaries
- **USE `SanitizeIssueID`** before creating branch names
- **CHECK file existence** with `os.Stat` before file operations
- **FAIL FAST** on invalid input types

### Type Conversion Rules
- **ALWAYS use explicit conversion** between custom types
- **NEVER rely on implicit type coercion**
- **IMPLEMENT sanitization functions** for all user input
- **VALIDATE after conversion** to ensure data integrity

## Package Type Requirements

| Package | MUST Use These Types | Type Responsibility |
|---------|---------------------|---------------------|
| `main` | `string` for CLI args | VALIDATE and convert to domain types |
| `git` | Custom types for safety | USE `IssueID`, `WorktreePath`, `BranchName` |
| `runner` | `string` for flexibility | VALIDATE paths before operations |

### Type Usage by Package

1. **`main` package**: ACCEPT raw strings, CONVERT to typed values
2. **`git` package**: REQUIRE typed parameters for safety
3. **`runner` package**: USE strings for shell compatibility

## Type Safety Checklist

When implementing new features, FOLLOW this checklist:

1. **IDENTIFY domain concepts** that deserve custom types
2. **CREATE type definitions** before implementation
3. **USE custom types** in all internal APIs
4. **VALIDATE at boundaries** when converting from strings
5. **DOCUMENT type invariants** in comments
6. **TEST type conversions** and validations