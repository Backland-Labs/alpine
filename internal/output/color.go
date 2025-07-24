package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"

	// maxToolLogs defines the maximum number of tool logs to keep in the circular buffer
	maxToolLogs = 4
)

// Printer handles colored output
type Printer struct {
	out      io.Writer
	err      io.Writer
	useColor bool

	// Tool log state management
	toolLogs    []string   // Circular buffer for tool logs
	currentTask string     // Current task being displayed
	mu          sync.Mutex // Mutex for thread-safe access to toolLogs and currentTask
}

// NewPrinter creates a new printer with color support
func NewPrinter() *Printer {
	return &Printer{
		out:      os.Stdout,
		err:      os.Stderr,
		useColor: isTerminal(),
	}
}

// NewPrinterWithWriters creates a printer with custom writers (for testing)
func NewPrinterWithWriters(out, err io.Writer, useColor bool) *Printer {
	return &Printer{
		out:      out,
		err:      err,
		useColor: useColor,
	}
}

// Success prints a success message in green
func (p *Printer) Success(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if p.useColor {
		_, _ = fmt.Fprintf(p.out, "%s%s✓ %s%s\n", colorBold, colorGreen, message, colorReset)
	} else {
		_, _ = fmt.Fprintf(p.out, "✓ %s\n", message)
	}
}

// Error prints an error message in red
func (p *Printer) Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if p.useColor {
		_, _ = fmt.Fprintf(p.err, "%s%s✗ %s%s\n", colorBold, colorRed, message, colorReset)
	} else {
		_, _ = fmt.Fprintf(p.err, "✗ %s\n", message)
	}
}

// Warning prints a warning message in yellow
func (p *Printer) Warning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if p.useColor {
		_, _ = fmt.Fprintf(p.err, "%s%s⚠ %s%s\n", colorBold, colorYellow, message, colorReset)
	} else {
		_, _ = fmt.Fprintf(p.err, "⚠ %s\n", message)
	}
}

// Info prints an info message in cyan
func (p *Printer) Info(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if p.useColor {
		_, _ = fmt.Fprintf(p.out, "%s%s→ %s%s\n", colorBold, colorCyan, message, colorReset)
	} else {
		_, _ = fmt.Fprintf(p.out, "→ %s\n", message)
	}
}

// Step prints a step message in blue
func (p *Printer) Step(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if p.useColor {
		_, _ = fmt.Fprintf(p.out, "%s%s▶ %s%s\n", colorBold, colorBlue, message, colorReset)
	} else {
		_, _ = fmt.Fprintf(p.out, "▶ %s\n", message)
	}
}

// Detail prints a detail message in gray
func (p *Printer) Detail(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if p.useColor {
		_, _ = fmt.Fprintf(p.out, "%s  %s%s\n", colorGray, message, colorReset)
	} else {
		_, _ = fmt.Fprintf(p.out, "  %s\n", message)
	}
}

// Print prints a plain message without color
func (p *Printer) Print(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(p.out, format, args...)
}

// Println prints a plain message with newline
func (p *Printer) Println(args ...interface{}) {
	_, _ = fmt.Fprintln(p.out, args...)
}

// StartTodoMonitoring prints an initial message for TODO monitoring
func (p *Printer) StartTodoMonitoring() {
	if p.useColor {
		_, _ = fmt.Fprintf(p.out, "%s%s⚡ Monitoring Claude's progress...%s\n", colorBold, colorCyan, colorReset)
	} else {
		_, _ = fmt.Fprintf(p.out, "⚡ Monitoring Claude's progress...\n")
	}
}

// UpdateCurrentTask updates the current task being displayed
func (p *Printer) UpdateCurrentTask(task string) {
	if task == "" {
		return
	}

	p.mu.Lock()
	p.currentTask = task
	p.mu.Unlock()

	// Clear current line and show new task
	_, _ = fmt.Fprintf(p.out, "\r\033[K")
	if p.useColor {
		_, _ = fmt.Fprintf(p.out, "%s%s🔄 Working on: %s%s\n", colorBold, colorBlue, task, colorReset)
	} else {
		_, _ = fmt.Fprintf(p.out, "🔄 Working on: %s\n", task)
	}

	// Render tool logs to maintain sticky header
	p.RenderToolLogs()
}

// StopTodoMonitoring prints a completion message and clears the line
func (p *Printer) StopTodoMonitoring() {
	_, _ = fmt.Fprintf(p.out, "\r\033[K") // Clear line
	if p.useColor {
		_, _ = fmt.Fprintf(p.out, "%s%s✓ Task completed%s\n", colorBold, colorGreen, colorReset)
	} else {
		_, _ = fmt.Fprintf(p.out, "✓ Task completed\n")
	}
}

// AddToolLog adds a new tool log message to the circular buffer
func (p *Printer) AddToolLog(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Append the new message
	p.toolLogs = append(p.toolLogs, message)

	// If we exceed maxToolLogs, remove the oldest entry
	if len(p.toolLogs) > maxToolLogs {
		p.toolLogs = p.toolLogs[len(p.toolLogs)-maxToolLogs:]
	}
}

// RenderToolLogs renders the tool logs with ANSI escape codes for a sticky header effect
func (p *Printer) RenderToolLogs() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If no tool logs, return empty string
	if len(p.toolLogs) == 0 {
		return ""
	}

	var output strings.Builder

	// Calculate total lines to move up (number of logs + 1 for header)
	linesToMoveUp := len(p.toolLogs) + 1

	// Move cursor up
	output.WriteString(fmt.Sprintf("\033[%dA", linesToMoveUp))

	// Clear and re-print the header line (primary task)
	output.WriteString("\033[K") // Clear line
	if p.currentTask != "" {
		if p.useColor {
			output.WriteString(fmt.Sprintf("%s%s🔄 Working on: %s%s\n", colorBold, colorBlue, p.currentTask, colorReset))
		} else {
			output.WriteString(fmt.Sprintf("🔄 Working on: %s\n", p.currentTask))
		}
	} else {
		if p.useColor {
			output.WriteString(fmt.Sprintf("%s%s🔄 Working on: Real-time tool monitoring%s\n", colorBold, colorBlue, colorReset))
		} else {
			output.WriteString("🔄 Working on: Real-time tool monitoring\n")
		}
	}

	// Print each tool log with indentation and gray color
	for _, log := range p.toolLogs {
		output.WriteString("\033[K") // Clear line
		if p.useColor {
			output.WriteString(fmt.Sprintf("  %s%s%s\n", colorGray, log, colorReset))
		} else {
			output.WriteString(fmt.Sprintf("  %s\n", log))
		}
	}

	// Write the output to the terminal
	_, _ = fmt.Fprint(p.out, output.String())

	return output.String()
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	// Check if NO_COLOR env var is set
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check if stdout is a terminal
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
