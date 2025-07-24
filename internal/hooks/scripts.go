package hooks

import _ "embed"

//go:embed todo-monitor.rs
var todoMonitorScript string

// GetTodoMonitorScript returns the todo-monitor hook script that handles:
// - PostToolUse events (for TodoWrite and other tool monitoring)
// - subagent:stop events (for Task tool completion tracking)
func GetTodoMonitorScript() (string, error) {
	return todoMonitorScript, nil
}
