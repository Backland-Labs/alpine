package validation

import (
	"regexp"
	"strings"
)

// outputValidator implements OutputValidator interface
type outputValidator struct{}

// NewOutputValidator creates a new output validator
func NewOutputValidator() OutputValidator {
	return &outputValidator{}
}

// CompareOutputs compares Python and Go outputs
func (v *outputValidator) CompareOutputs(pythonOutput, goOutput string) ComparisonResult {
	// Normalize both outputs before comparison
	normalizedPython := v.NormalizeOutput(pythonOutput)
	normalizedGo := v.NormalizeOutput(goOutput)

	result := ComparisonResult{
		Match:       true,
		Differences: []Difference{},
	}

	if normalizedPython != normalizedGo {
		result.Match = false
		result.Differences = append(result.Differences, Difference{
			Type:        "output_content",
			PythonValue: truncateForDisplay(normalizedPython, 100),
			GoValue:     truncateForDisplay(normalizedGo, 100),
			Description: "Output content mismatch",
		})
	}

	return result
}

// ExtractKeyMetrics extracts important metrics from output
func (v *outputValidator) ExtractKeyMetrics(output string) OutputMetrics {
	metrics := OutputMetrics{
		ErrorCount:   0,
		WarningCount: 0,
		HasErrors:    false,
	}

	lines := strings.Split(output, "\n")

	// Patterns for detecting errors and warnings
	errorPattern := regexp.MustCompile(`(?i)ERROR:`)
	warningPattern := regexp.MustCompile(`(?i)WARNING:`)
	completionPattern := regexp.MustCompile(`(?i)(task completed successfully|workflow completed|done successfully)`)

	for _, line := range lines {
		if errorPattern.MatchString(line) {
			metrics.ErrorCount++
			metrics.HasErrors = true
		}
		if warningPattern.MatchString(line) {
			metrics.WarningCount++
		}
		if matches := completionPattern.FindStringSubmatch(line); len(matches) > 0 {
			metrics.CompletionMessage = strings.TrimSpace(matches[0])
		}
	}

	return metrics
}

// NormalizeOutput normalizes output for comparison
func (v *outputValidator) NormalizeOutput(output string) string {
	// Normalize line endings (CRLF -> LF)
	output = strings.ReplaceAll(output, "\r\n", "\n")

	// Split into lines for processing
	lines := strings.Split(output, "\n")

	// Trim trailing whitespace from each line
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	// Rejoin lines
	output = strings.Join(lines, "\n")

	// Collapse multiple blank lines into max 2
	multipleBlankLines := regexp.MustCompile(`\n{3,}`)
	output = multipleBlankLines.ReplaceAllString(output, "\n\n")

	// Trim leading and trailing whitespace
	output = strings.TrimSpace(output)

	return output
}

// truncateForDisplay truncates a string for display in comparison results
func truncateForDisplay(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
