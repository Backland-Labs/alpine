# Implementation Plan: Claude TODO Visibility - PostToolUse TodoWrite Hook (IMPLEMENTED)

## Overview

Implement a single PostToolUse hook for the TodoWrite tool to show users what Claude is currently working on, replacing the generic "Executing Claude" spinner with real task updates.

## Simple Architecture

1. **River sets up one hook**: PostToolUse for TodoWrite tool
2. **Hook script writes updates**: To a temp file when TodoWrite is used  
3. **River monitors file**: Shows latest TODO status to user
4. **Fallback gracefully**: If hooks fail, show normal spinner

## Implementation

### 1. Hook Script
**File**: `internal/hooks/todo-monitor.sh`
```bash
#!/bin/bash
# PostToolUse hook for TodoWrite tool only

# Read JSON input from Claude Code
input=$(cat)
tool=$(echo "$input" | jq -r '.tool // empty')

# Only process TodoWrite tool
if [[ "$tool" != "TodoWrite" ]]; then
    exit 0
fi

# Extract current in_progress task
current_task=$(echo "$input" | jq -r '.args.todos[]? | select(.status == "in_progress") | .content' | head -1)

if [[ -n "$current_task" && -n "$RIVER_TODO_FILE" ]]; then
    echo "$current_task" > "$RIVER_TODO_FILE"
fi

exit 0
```

### 2. Hook Setup  
**File**: `internal/claude/hooks.go`
```go
func (c *ClaudeExecutor) setupTodoHook() error {
    // Create .claude directory
    // Write settings.json with PostToolUse hook for TodoWrite
    // Copy hook script and make executable
    // Set RIVER_TODO_FILE environment variable
}

func (c *ClaudeExecutor) generateClaudeSettings(hookScriptPath, todoFilePath string) string {
    settings := map[string]interface{}{
        "hooks": map[string]interface{}{
            "PostToolUse": []map[string]interface{}{
                {
                    "matcher": "TodoWrite",
                    "hooks": []map[string]interface{}{
                        {
                            "type": "command",
                            "command": hookScriptPath,
                        },
                    },
                },
            },
        },
    }
    // Return JSON string
}
```

### 3. File Monitor
**File**: `internal/claude/todo_monitor.go`  
```go
type TodoMonitor struct {
    filePath string
    lastTask string
    updates  chan string
}

func (tm *TodoMonitor) Start(ctx context.Context) {
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if task := tm.readCurrentTask(); task != tm.lastTask {
                tm.lastTask = task
                tm.updates <- task
            }
        case <-ctx.Done():
            return
        }
    }
}

func (tm *TodoMonitor) readCurrentTask() string {
    data, err := os.ReadFile(tm.filePath)
    if err != nil {
        return ""
    }
    return strings.TrimSpace(string(data))
}
```

### 4. Enhanced Executor
**File**: `internal/claude/executor.go`
```go
func (c *ClaudeExecutor) Execute(ctx context.Context, prompt string) error {
    if c.config.ShowTodoUpdates {
        return c.executeWithTodoMonitoring(ctx, prompt)
    }
    return c.executeWithSpinner(ctx, prompt)
}

func (c *ClaudeExecutor) executeWithTodoMonitoring(ctx context.Context, prompt string) error {
    // Setup hook (fallback to spinner on failure)
    todoFile, cleanup, err := c.setupTodoHook()
    if err != nil {
        return c.executeWithSpinner(ctx, prompt)
    }
    defer cleanup()
    
    // Start monitoring
    monitor := NewTodoMonitor(todoFile)
    go monitor.Start(ctx)
    
    // Start Claude execution  
    claudeErr := make(chan error, 1)
    go func() {
        claudeErr <- c.executeClaudeCommand(ctx, prompt)
    }()
    
    // Show updates
    c.printer.StartTodoMonitoring()
    for {
        select {
        case task := <-monitor.Updates():
            c.printer.UpdateCurrentTask(task)
        case err := <-claudeErr:
            c.printer.StopTodoMonitoring()
            return err
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

### 5. Display Updates
**File**: `internal/output/printer.go`
```go
func (p *Printer) UpdateCurrentTask(task string) {
    if task == "" {
        return
    }
    
    // Clear current line and show new task
    fmt.Print("\r\033[K")
    p.printWithIcon("🔄", fmt.Sprintf("Working on: %s", task))
}

func (p *Printer) StartTodoMonitoring() {
    p.printWithIcon("⚡", "Monitoring Claude's progress...")
}

func (p *Printer) StopTodoMonitoring() {
    fmt.Print("\r\033[K") // Clear line
    p.printWithIcon("✓", "Task completed")
}
```

## File Structure

```
internal/
├── claude/
│   ├── executor.go       # Enhanced with simple hook setup
│   ├── hooks.go          # Hook configuration (minimal)
│   └── todo_monitor.go   # File monitoring
├── hooks/
│   └── todo-monitor.sh   # Single hook script
└── output/
    └── printer.go        # Enhanced display
```

## User Experience

```
$ river "Implement user authentication"  
⚡ Monitoring Claude's progress...
🔄 Working on: Research existing authentication patterns
🔄 Working on: Create user model and database schema
🔄 Working on: Implement JWT token generation  
🔄 Working on: Add password hashing utilities
🔄 Working on: Create login/logout endpoints
✓ Task completed
```

## Configuration

Just one new config option:
```go
ShowTodoUpdates bool `env:"RIVER_SHOW_TODO_UPDATES" default:"true"`
```

## Success Criteria

- [x] Single hook for TodoWrite PostToolUse
- [x] Shows current task being worked on
- [x] Graceful fallback to spinner
- [x] Minimal code complexity
- [x] Immediate user value

This gives us 80% of the value with 20% of the complexity!

## Implementation Status

**COMPLETED**: All features have been successfully implemented and tested.

### What was implemented:
1. ✅ Hook script (`internal/hooks/todo-monitor.sh`) - Embedded as Go string
2. ✅ Hook setup (`internal/claude/hooks.go`) - Creates .claude directory and settings.json
3. ✅ File monitor (`internal/claude/todo_monitor.go`) - Monitors todo file for updates
4. ✅ Enhanced executor (`internal/claude/executor.go`) - Supports todo monitoring mode
5. ✅ Display updates (`internal/output/color.go`) - Shows real-time task updates
6. ✅ Configuration (`internal/config/config.go`) - ShowTodoUpdates option with env var

### Tests added:
- `internal/claude/todo_monitor_test.go` - Tests for TodoMonitor
- `internal/claude/hooks_test.go` - Tests for hook setup and configuration
- `internal/claude/executor_todo_test.go` - Tests for executor todo monitoring
- `internal/output/todo_display_test.go` - Tests for display functions
- `internal/config/config_showtodo_test.go` - Tests for ShowTodoUpdates config

The feature is fully functional and provides real-time visibility into Claude's current tasks when using the TodoWrite tool.