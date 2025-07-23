# Claude Code Hooks Integration Specification

## Overview

This specification defines how River integrates with Claude Code hooks to provide enhanced workflow control, validation, and monitoring capabilities during AI-assisted development sessions.

## Background

Claude Code hooks are configurable scripts that intercept and modify Claude Code's behavior at specific events. River can leverage these hooks to:
- Validate tool usage within River workflows
- Add contextual information about River's state
- Monitor and log River-orchestrated Claude sessions
- Implement custom permissions and safety checks

## Hook Event Types

### 1. PreToolUse
- **Trigger**: Before Claude executes any tool
- **River Use Case**: Validate tools are appropriate for current workflow step
- **Input**: Tool name, arguments, current state context

### 2. PostToolUse
- **Trigger**: After tool execution completes
- **River Use Case**: Update River state based on tool results, log progress
- **Input**: Tool name, arguments, execution results, exit status

### 3. UserPromptSubmit
- **Trigger**: When prompts are submitted to Claude
- **River Use Case**: Inject River state context, modify prompts for workflow alignment
- **Input**: Original prompt, current River state

### 4. Notification
- **Trigger**: During system notifications
- **River Use Case**: React to Claude status changes, workflow transitions
- **Input**: Notification type, message content

### 5. Stop/SubagentStop
- **Trigger**: When Claude responses complete
- **River Use Case**: Determine if workflow should continue or pause
- **Input**: Response completion status, generated content

## Claude Code Hooks Core Functionality

### Blocking Tool Execution
Hooks can prevent tools from executing by returning exit code 2:
```bash
#!/bin/bash
# Block dangerous operations during certain workflow steps
if [[ "$RIVER_CURRENT_STEP" == "planning" && "$TOOL_NAME" == "Bash" ]]; then
    echo "Bash execution blocked during planning phase"
    exit 2
fi
exit 0
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
  "hooks": [{"type": "command", "command": "validate-file-operations.sh"}]
}
```

### Context Injection
UserPromptSubmit hooks can augment prompts with additional context:
```bash
#!/bin/bash
original_prompt=$(cat)
echo "$original_prompt

Current River State: $RIVER_CURRENT_STEP
Status: $RIVER_STATUS
"
```

## Configuration Structure

River should support hooks configuration through:

### Environment Variables
```bash
export RIVER_HOOKS_CONFIG_PATH="/path/to/hooks.json"
export RIVER_HOOKS_ENABLED="true"
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
            "command": "river-validate-tool.sh"
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
            "command": "river-inject-context.sh"
          }
        ]
      }
    ]
  }
}
```

## River-Specific Hook Scripts

### Tool Validation Hook
**Purpose**: Ensure tools align with current workflow step
**Script**: `river-validate-tool.sh`
**Behavior**:
- Read current River state from `claude_state.json`
- Validate tool usage against workflow requirements
- Block inappropriate tools (exit code 2)
- Allow appropriate tools (exit code 0)

### Context Injection Hook
**Purpose**: Add River state to Claude prompts
**Script**: `river-inject-context.sh`
**Behavior**:
- Read current step description and status
- Append relevant context to user prompts
- Ensure Claude understands workflow position

### Progress Monitoring Hook
**Purpose**: Track and log workflow progress
**Script**: `river-monitor-progress.sh`
**Behavior**:
- Log tool usage patterns
- Update external monitoring systems
- Generate workflow analytics

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
```bash
#!/bin/bash
# Filter sensitive information from tool outputs
tool_output=$(cat)
echo "$tool_output" | sed 's/password=[^[:space:]]*/password=****/g'
```

### Workflow Step Transitions
Hooks can trigger workflow transitions by:
- Modifying `claude_state.json` directly
- Setting environment variables for next iteration
- Creating trigger files for River to detect

## Integration Points

### 1. Hook Configuration Management
River should:
- Generate appropriate hooks configuration based on workflow type
- Merge user-defined hooks with River defaults
- Validate hook script availability and permissions

### 2. State Context Sharing
River should:
- Expose current state via environment variables for hook scripts
- Provide state file path to hooks
- Ensure hooks have read access to workflow state

### 3. Hook Script Distribution
River should:
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
   - Export River state for hook consumption
   - Provide environment variables for hook scripts
   - Ensure secure state access patterns

3. **Hook Script Registry**
   - Manage built-in hook scripts
   - Support custom hook script registration
   - Validate hook script executability

### Configuration Integration

River should modify Claude Code settings to include hooks:

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
    os.Setenv("RIVER_CURRENT_STEP", e.state.CurrentStepDescription)
    os.Setenv("RIVER_STATUS", e.state.Status)
    os.Setenv("RIVER_STATE_FILE", e.stateFilePath)
}
```

## Security Considerations

1. **Script Validation**: Verify hook script permissions and ownership
2. **State Access**: Limit hook access to necessary state information
3. **Command Injection**: Sanitize hook script paths and arguments
4. **Resource Limits**: Implement timeouts for hook script execution