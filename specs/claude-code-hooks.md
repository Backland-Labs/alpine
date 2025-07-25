# Claude Code Hooks Integration Specification

## Overview

This specification defines how Alpine integrates with Claude Code hooks to provide enhanced workflow control, validation, and monitoring capabilities during AI-assisted development sessions.

## Background

Claude Code hooks are configurable scripts that intercept and modify Claude Code's behavior at specific events. Alpine can leverage these hooks to:
- Validate tool usage within Alpine workflows
- Add contextual information about Alpine's state
- Monitor and log Alpine-orchestrated Claude sessions
- Implement custom permissions and safety checks

## Hook Event Types

### 1. PreToolUse
- **Trigger**: Before Claude executes any tool
- **Alpine Use Case**: Validate tools are appropriate for current workflow step
- **Input**: Tool name, arguments, current state context

### 2. PostToolUse
- **Trigger**: After tool execution completes
- **Alpine Use Case**: Update Alpine state based on tool results, log progress
- **Input**: Tool name, arguments, execution results, exit status

### 3. UserPromptSubmit
- **Trigger**: When prompts are submitted to Claude
- **Alpine Use Case**: Inject Alpine state context, modify prompts for workflow alignment
- **Input**: Original prompt, current Alpine state

### 4. Notification
- **Trigger**: During system notifications
- **Alpine Use Case**: React to Claude status changes, workflow transitions
- **Input**: Notification type, message content

### 5. Stop/SubagentStop
- **Trigger**: When Claude responses complete
- **Alpine Use Case**: Determine if workflow should continue or pause
- **Input**: Response completion status, generated content

## Claude Code Hooks Core Functionality

### Hook Implementation Language
Alpine hooks are implemented in **Rust** using `rust-script` for executable scripts. This provides:
- Fast JSON parsing with `serde_json`
- Type-safe error handling
- Native performance
- Self-contained executable scripts

### Blocking Tool Execution
Hooks can prevent tools from executing by returning exit code 2:
```rust
#!/usr/bin/env rust-script
//! ```cargo
//! [dependencies]
//! serde_json = "1.0"
//! ```

use serde_json::Value;
use std::env;
use std::io::{self, Read};
use std::process;

fn main() -> io::Result<()> {
    let mut input = String::new();
    io::stdin().read_to_string(&mut input)?;
    
    let data: Value = serde_json::from_str(&input).unwrap_or_default();
    let tool = data["tool"].as_str().unwrap_or("");
    
    let current_step = env::var("ALPINE_CURRENT_STEP").unwrap_or_default();
    
    if current_step == "planning" && tool == "Bash" {
        eprintln!("Bash execution blocked during planning phase");
        process::exit(2);
    }
    
    process::exit(0);
}
```

### Input/Output Modification
Hooks receive JSON input via stdin and can modify behavior through JSON output:
```json
{
  "tool": "Edit",
  "args": {"file_path": "/path/to/file", "old_string": "...", "new_string": "..."},
  "blocking": false
}
```

### Regex Pattern Matching
Hooks support sophisticated pattern matching for selective activation:
```json
{
  "matcher": "(Bash|Edit|Write)",
  "hooks": [{"type": "command", "command": "validate-file-operations.rs"}]
}
```

### Context Injection
UserPromptSubmit hooks can augment prompts with additional context:
```rust
#!/usr/bin/env rust-script

use std::env;
use std::io::{self, Read};

fn main() -> io::Result<()> {
    let mut original_prompt = String::new();
    io::stdin().read_to_string(&mut original_prompt)?;
    
    let current_step = env::var("ALPINE_CURRENT_STEP").unwrap_or_default();
    let status = env::var("ALPINE_STATUS").unwrap_or_default();
    
    println!("{}\n\nCurrent Alpine State: {}\nStatus: {}", 
             original_prompt.trim(), current_step, status);
    
    Ok(())
}
```

## Hook Storage and Configuration

### Claude Code Settings Files

Claude Code hooks are configured through Claude's settings files, which follow a hierarchical precedence order:

1. **Global User Settings**: `~/.claude/settings.json`
   - Applied to all Claude Code sessions for the user
   - Suitable for general Alpine integration hooks

2. **Project Settings**: `.claude/settings.json`
   - Project-specific hooks configuration
   - Committed to version control for team sharing
   - Ideal for Alpine workflow-specific hooks

3. **Local Project Settings**: `.claude/settings.local.json`
   - Local overrides not committed to version control
   - Used for developer-specific customizations
   - Takes precedence over project settings

### Hook Script Storage

Hook scripts referenced in configuration should be stored in:

- **Built-in Alpine Hooks**: Embedded in Alpine binary or distributed alongside
- **Project Hooks**: `./claude/hooks/` directory (relative to project root)
- **User Hooks**: `~/.claude/hooks/` directory for user-global scripts
- **Custom Paths**: Absolute paths specified in hook configuration

### Settings File Structure

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "~/.claude/hooks/alpine-validate-tool.rs"
          }
        ]
      }
    ]
  }
}
```

## Configuration Structure

Alpine should support hooks configuration through:

### Environment Variables
```bash
export ALPINE_HOOKS_CONFIG_PATH="/path/to/hooks.json"
export ALPINE_HOOKS_ENABLED="true"
```

### Configuration File Format
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "alpine-validate-tool.rs"
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command", 
            "command": "alpine-inject-context.rs"
          }
        ]
      }
    ]
  }
}
```

## Alpine-Specific Hook Scripts

### Tool Validation Hook
**Purpose**: Ensure tools align with current workflow step
**Script**: `alpine-validate-tool.rs`
**Behavior**:
- Read current Alpine state from `claude_state.json`
- Validate tool usage against workflow requirements
- Block inappropriate tools (exit code 2)
- Allow appropriate tools (exit code 0)

### Context Injection Hook
**Purpose**: Add Alpine state to Claude prompts
**Script**: `alpine-inject-context.rs`
**Behavior**:
- Read current step description and status
- Append relevant context to user prompts
- Ensure Claude understands workflow position

### Progress Monitoring Hook
**Purpose**: Track and log workflow progress
**Script**: `alpine-monitor-progress.rs`
**Behavior**:
- Log tool usage patterns
- Update external monitoring systems
- Generate workflow analytics

### Todo Monitor Hook
**Purpose**: Monitor all Claude Code tool usage and track TodoWrite updates
**Script**: `todo-monitor.rs`
**Behavior**:
- Display all tool calls with timestamps to stderr for real-time visibility
- Show tool-specific information (file paths, commands, search patterns)
- Track TodoWrite updates with task counts (Completed/In Progress/Pending)
- Display current in-progress task
- Write current task to file specified by `ALPINE_TODO_FILE` environment variable
- Support both Claude Code PostToolUse format (`tool_name`/`tool_input`) and legacy format (`tool`/`args`)

**Example Output**:
```
[14:32:15] [TODO] Updated - Completed: 2, In Progress: 1, Pending: 3
[14:32:15] [TODO] Current task: Implementing user authentication
[14:32:16] [READ] Reading file: /src/auth/login.js
[14:32:18] [EDIT] Editing file: /src/auth/login.js
[14:32:20] [BASH] Executing: npm test
[14:32:22] [GREP] Searching for 'authenticate' in src/
[14:32:23] [GLOB] Finding files matching '*.test.js' in tests/
[14:32:24] [WEB] Fetching: https://api.example.com/docs
```

**Implementation Details**:
```rust
#!/usr/bin/env rust-script
//! ```cargo
//! [dependencies]
//! serde_json = "1.0"
//! chrono = "0.4"
//! ```

use serde_json::Value;
use std::env;
use std::io::{self, Read, Write};
use std::fs::File;
use chrono::Local;

fn main() -> io::Result<()> {
    // Read JSON input from Claude Code
    let mut input = String::new();
    io::stdin().read_to_string(&mut input)?;
    
    let data: Value = match serde_json::from_str(&input) {
        Ok(v) => v,
        Err(_) => return Ok(()), // Exit gracefully on invalid JSON
    };
    
    // Get timestamp
    let timestamp = Local::now().format("%H:%M:%S");
    
    // Check both possible field names for tool name (for compatibility)
    let tool_name = data["tool_name"].as_str()
        .or_else(|| data["tool"].as_str())
        .unwrap_or("");
    
    // Get tool input - check both possible field names
    let tool_input = data["tool_input"].as_object()
        .or_else(|| data["args"].as_object());
    
    // Process and display all tool calls
    match tool_name {
        "TodoWrite" => handle_todo_write(&data, &timestamp, tool_input),
        "Read" => /* display file path */,
        "Write" => /* display file path */,
        "Edit" | "MultiEdit" => /* display file path */,
        "Bash" => /* display command */,
        "Grep" => /* display pattern and path */,
        "Glob" => /* display pattern and path */,
        "LS" => /* display directory path */,
        "WebFetch" => /* display URL */,
        "WebSearch" => /* display query */,
        "Task" => /* display agent description */,
        _ => /* display generic tool message */
    }
    
    Ok(())
}
```

## Advanced Hook Features

### Conditional Execution
Hooks can implement complex conditional logic based on:
- Current workflow state
- Tool arguments and context
- Previous tool execution results
- Environment variables and configuration

### State Persistence
Hooks can maintain state across executions through:
- Temporary files in workflow directory
- Environment variable updates
- External storage systems
- State file annotations

### Tool Result Modification
PostToolUse hooks can modify tool results before Claude processes them:
```rust
#!/usr/bin/env rust-script
//! ```cargo
//! [dependencies]
//! regex = "1.0"
//! ```

use regex::Regex;
use std::io::{self, Read};

fn main() -> io::Result<()> {
    let mut tool_output = String::new();
    io::stdin().read_to_string(&mut tool_output)?;
    
    // Filter sensitive information from tool outputs
    let re = Regex::new(r"password=[^\s]*").unwrap();
    let filtered = re.replace_all(&tool_output, "password=****");
    
    print!("{}", filtered);
    
    Ok(())
}
```

### Workflow Step Transitions
Hooks can trigger workflow transitions by:
- Modifying `claude_state.json` directly
- Setting environment variables for next iteration
- Creating trigger files for Alpine to detect

## Integration Points

### 1. Hook Configuration Management
Alpine should:
- Generate appropriate hooks configuration based on workflow type
- Merge user-defined hooks with Alpine defaults
- Validate hook script availability and permissions

### 2. State Context Sharing
Alpine should:
- Expose current state via environment variables for hook scripts
- Provide state file path to hooks
- Ensure hooks have read access to workflow state

### 3. Hook Script Distribution
Alpine should:
- Include default hook scripts in distribution
- Support custom hook script directories
- Provide hook script templates and examples

## Implementation Requirements

### Core Components

1. **Hook Configuration Manager**
   - Load and validate hooks configuration
   - Merge default and custom configurations
   - Generate Claude Code settings with hooks enabled

2. **State Context Provider**
   - Export Alpine state for hook consumption
   - Provide environment variables for hook scripts
   - Ensure secure state access patterns

3. **Hook Script Registry**
   - Manage built-in hook scripts
   - Support custom hook script registration
   - Validate hook script executability

### Configuration Integration

Alpine should modify Claude Code settings to include hooks:

```go
func (e *Executor) configureHooks() error {
    if !e.config.HooksEnabled {
        return nil
    }
    
    hooksConfig := e.loadHooksConfiguration()
    claudeSettings := e.generateClaudeSettings(hooksConfig)
    
    return e.writeClaudeSettings(claudeSettings)
}
```

### State Export for Hooks

```go
func (e *Executor) exportStateForHooks() {
    os.Setenv("ALPINE_CURRENT_STEP", e.state.CurrentStepDescription)
    os.Setenv("ALPINE_STATUS", e.state.Status)
    // State file location is fixed at .claude/alpine/claude_state.json
}
```

## Security Considerations

1. **Script Validation**: Verify hook script permissions and ownership
2. **State Access**: Limit hook access to necessary state information
3. **Command Injection**: Sanitize hook script paths and arguments
4. **Resource Limits**: Implement timeouts for hook script execution