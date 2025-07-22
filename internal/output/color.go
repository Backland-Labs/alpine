package output

import (
	"fmt"
	"io"
	"os"
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
)

// Printer handles colored output
type Printer struct {
	out      io.Writer
	err      io.Writer
	useColor bool
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