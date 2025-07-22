// Package output provides colored terminal output with progress indicators
package output

import (
	"fmt"
	"sync"
	"time"
)

// Progress represents an active progress indicator
type Progress struct {
	printer      *Printer
	message      string
	iteration    int
	startTime    time.Time
	done         chan bool
	wg           sync.WaitGroup
	mu           sync.Mutex
	spinnerIndex int
}

// Spinner characters for animation
var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// StartProgress creates and starts a new progress indicator
func (p *Printer) StartProgress(message string) *Progress {
	progress := &Progress{
		printer:   p,
		message:   message,
		startTime: time.Now(),
		done:      make(chan bool),
	}
	
	progress.wg.Add(1)
	go progress.animate()
	
	// Small delay to ensure first render
	time.Sleep(10 * time.Millisecond)
	
	return progress
}

// StartProgressWithIteration creates a progress indicator with iteration counter
func (p *Printer) StartProgressWithIteration(message string, iteration int) *Progress {
	progress := &Progress{
		printer:   p,
		message:   message,
		iteration: iteration,
		startTime: time.Now(),
		done:      make(chan bool),
	}
	
	progress.wg.Add(1)
	go progress.animate()
	
	// Small delay to ensure first render
	time.Sleep(10 * time.Millisecond)
	
	return progress
}

// UpdateMessage updates the progress message
func (p *Progress) UpdateMessage(message string) {
	p.mu.Lock()
	p.message = message
	spinnerIndex := p.spinnerIndex
	p.mu.Unlock()
	
	// Immediately re-render with new message
	p.render(spinnerIndex)
}

// Stop stops the progress indicator and clears the line
func (p *Progress) Stop() {
	// Prevent multiple calls to Stop
	select {
	case <-p.done:
		// Already stopped
		return
	default:
		close(p.done)
	}
	
	p.wg.Wait()
	
	// Clear the line
	fmt.Fprintf(p.printer.out, "\r\033[K")
}

// animate runs the spinner animation in a goroutine
func (p *Progress) animate() {
	defer p.wg.Done()
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	spinnerIndex := 0
	
	// Render immediately
	p.render(spinnerIndex)
	
	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			spinnerIndex++
			p.mu.Lock()
			p.spinnerIndex = spinnerIndex
			p.mu.Unlock()
			p.render(spinnerIndex)
		}
	}
}

// render displays the current progress state
func (p *Progress) render(spinnerIndex int) {
	p.mu.Lock()
	message := p.message
	iteration := p.iteration
	p.mu.Unlock()
	
	elapsed := time.Since(p.startTime)
	
	// Build the progress line
	var line string
	if p.printer.useColor {
		spinner := spinnerChars[spinnerIndex%len(spinnerChars)]
		
		if iteration > 0 {
			line = fmt.Sprintf("\r%s%s%s %s (Iteration %d) %s[%s]%s",
				colorBold, colorCyan, spinner, message, iteration,
				colorGray, formatDuration(elapsed), colorReset)
		} else {
			line = fmt.Sprintf("\r%s%s%s %s %s[%s]%s",
				colorBold, colorCyan, spinner, message,
				colorGray, formatDuration(elapsed), colorReset)
		}
	} else {
		spinner := spinnerChars[spinnerIndex%len(spinnerChars)]
		
		if iteration > 0 {
			line = fmt.Sprintf("\r%s %s (Iteration %d) [%s]",
				spinner, message, iteration, formatDuration(elapsed))
		} else {
			line = fmt.Sprintf("\r%s %s [%s]",
				spinner, message, formatDuration(elapsed))
		}
	}
	
	// Clear to end of line and print
	fmt.Fprintf(p.printer.out, "%s\033[K", line)
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.0fs", d.Seconds())
}