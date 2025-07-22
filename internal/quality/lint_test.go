package quality

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLintingCompliance ensures that the codebase passes all golangci-lint checks.
// This test enforces code quality standards and ensures no linting warnings exist.
// The test will fail if any linting issues are found, encouraging developers to
// fix code quality issues before committing.
func TestLintingCompliance(t *testing.T) {
	// Skip if golangci-lint is not available
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		t.Skip("golangci-lint not found, skipping linting test")
	}

	// Change to project root directory
	projectRoot := "../.."
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}

	// Run golangci-lint
	cmd := exec.Command("golangci-lint", "run")
	output, err := cmd.CombinedOutput()

	// Parse the output to count issues
	outputStr := string(output)
	
	if err != nil {
		// If there are linting issues, the command will return exit code 1
		// Count the number of issues reported
		lines := strings.Split(strings.TrimSpace(outputStr), "\n")
		var issueCount int
		
		for _, line := range lines {
			// Lines that contain file paths and error descriptions are issues
			if strings.Contains(line, ":") && (strings.Contains(line, "Error") || 
				strings.Contains(line, "errcheck") || strings.Contains(line, "staticcheck")) {
				issueCount++
			}
		}
		
		t.Errorf("Found %d linting issues. All linting issues must be fixed.\nOutput:\n%s", 
			issueCount, outputStr)
		return
	}

	// If we reach here, linting passed
	assert.NoError(t, err, "golangci-lint should pass without errors")
	assert.NotContains(t, outputStr, "issues:", "Should not contain any linting issues")
}

