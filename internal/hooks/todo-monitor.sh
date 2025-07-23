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