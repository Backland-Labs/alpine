package cli

import (
	"fmt"
	"io"
)

// PrefixWriter wraps an io.Writer and prefixes each line with a project name
type PrefixWriter struct {
	writer      io.Writer
	prefix      string
	useColor    bool
	colorCode   string
	buffer      []byte
	atLineStart bool
}

// ANSI color codes for different project indexes
var prefixColors = []string{
	"\033[32m", // Green
	"\033[34m", // Blue
	"\033[36m", // Cyan
	"\033[35m", // Magenta
	"\033[33m", // Yellow
	"\033[31m", // Red
}

// NewPrefixWriter creates a new prefix writer
func NewPrefixWriter(w io.Writer, projectName string, useColor bool, colorIndex int) *PrefixWriter {
	colorCode := ""
	if useColor {
		colorCode = prefixColors[colorIndex%len(prefixColors)]
	}
	
	return &PrefixWriter{
		writer:      w,
		prefix:      fmt.Sprintf("[%s]", projectName),
		useColor:    useColor,
		colorCode:   colorCode,
		buffer:      make([]byte, 0, 1024),
		atLineStart: true,
	}
}

// Write implements io.Writer
func (pw *PrefixWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	
	for _, b := range p {
		if pw.atLineStart {
			// Write prefix at start of line
			var prefix string
			if pw.useColor {
				prefix = fmt.Sprintf("%s%s\033[0m ", pw.colorCode, pw.prefix)
			} else {
				prefix = fmt.Sprintf("%s ", pw.prefix)
			}
			
			if _, err := pw.writer.Write([]byte(prefix)); err != nil {
				return 0, err
			}
			pw.atLineStart = false
		}
		
		// Write the byte
		if _, err := pw.writer.Write([]byte{b}); err != nil {
			return 0, err
		}
		
		// Check if we're at a newline
		if b == '\n' {
			pw.atLineStart = true
		}
	}
	
	return n, nil
}