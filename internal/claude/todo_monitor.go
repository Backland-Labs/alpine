package claude

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/Backland-Labs/alpine/internal/logger"
)

// TodoMonitor monitors a file for TODO updates from Claude
type TodoMonitor struct {
	filePath string
	lastTask string
	updates  chan string
}

// NewTodoMonitor creates a new TodoMonitor for the specified file
func NewTodoMonitor(filePath string) *TodoMonitor {
	return &TodoMonitor{
		filePath: filePath,
		updates:  make(chan string, 10), // Buffered channel to prevent blocking
	}
}

// Start begins monitoring the todo file for changes
func (tm *TodoMonitor) Start(ctx context.Context) {
	logger.WithFields(map[string]interface{}{
		"file_path": tm.filePath,
		"poll_interval": "500ms",
	}).Info("Starting TODO monitor")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.WithField("reason", "context_done").Info("TODO monitor stopped")
			close(tm.updates)
			return
		case <-ticker.C:
			if task := tm.readCurrentTask(); task != tm.lastTask {
				if task != "" || tm.lastTask != "" { // Only send updates when there's a change
					tm.lastTask = task
					select {
					case tm.updates <- task:
						logger.WithFields(map[string]interface{}{
							"task": task,
							"task_length": len(task),
						}).Debug("TODO update sent")
					default:
						// Channel full, skip this update
						logger.WithField("channel_size", 10).Warn("TODO update channel full, skipping")
					}
				}
			}
		}
	}
}

// Updates returns the channel that receives TODO updates
func (tm *TodoMonitor) Updates() <-chan string {
	return tm.updates
}

// readCurrentTask reads the current task from the file
func (tm *TodoMonitor) readCurrentTask() string {
	data, err := os.ReadFile(tm.filePath)
	if err != nil {
		// File might not exist yet or be temporarily unavailable
		return ""
	}

	task := strings.TrimSpace(string(data))
	return task
}
