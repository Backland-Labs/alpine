package hooks

import (
	"os"
	"path/filepath"
)

// GetTodoMonitorScript returns the path to the todo-monitor hook binary that handles:
// - PostToolUse events (for TodoWrite and other tool monitoring)
// - subagent:stop events (for Task tool completion tracking)
func GetTodoMonitorScript() (string, error) {
	// Get the current executable path to find the hooks directory
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Get the directory containing the executable
	execDir := filepath.Dir(execPath)

	// Try different paths where the hook binary might be located
	possiblePaths := []string{
		// Development environment - hooks at project root (from build/ directory)
		filepath.Join(execDir, "..", "hooks", "alpine-todo-monitor"),
		// Development environment - hooks at project root (from deeper nested directories)
		filepath.Join(execDir, "..", "..", "hooks", "alpine-todo-monitor"),
		// Installed environment - hooks relative to binary
		filepath.Join(execDir, "hooks", "alpine-todo-monitor"),
		// Current working directory (for testing)
		filepath.Join("hooks", "alpine-todo-monitor"),
	}

	for _, path := range possiblePaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			return absPath, nil
		}
	}

	// If no binary found, return empty string (backward compatibility)
	return "", nil
}
