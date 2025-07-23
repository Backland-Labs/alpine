package hooks

import _ "embed"

//go:embed todo-monitor.rs
var todoMonitorScript string

// GetTodoMonitorScript returns the TodoWrite PostToolUse hook script
func GetTodoMonitorScript() (string, error) {
	return todoMonitorScript, nil
}
