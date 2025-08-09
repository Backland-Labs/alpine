package integration

import (
	"os/exec"
	"strings"
	"testing"
)

// TestHelpTextFormat tests that help command produces properly formatted output
func TestHelpTextFormat(t *testing.T) {
	cmd := exec.Command("go", "run", "../../cmd/alpine/main.go", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	outputStr := string(output)

	// Critical: Help must contain basic usage information
	if !strings.Contains(outputStr, "Usage:") {
		t.Error("Help output missing 'Usage:' section")
	}

	// Critical: Must show available commands
	if !strings.Contains(outputStr, "Available Commands:") || !strings.Contains(outputStr, "Flags:") {
		t.Error("Help output missing command or flag sections")
	}
}

// TestVersionCommandOutput tests version command produces expected format
func TestVersionCommandOutput(t *testing.T) {
	cmd := exec.Command("go", "run", "../../cmd/alpine/main.go", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Version command failed: %v", err)
	}

	outputStr := strings.TrimSpace(string(output))

	// Critical: Version output must contain version information
	if !strings.Contains(outputStr, "alpine") {
		t.Error("Version output should contain 'alpine'")
	}
}

// TestInvalidFlagErrorFormat tests error message format for invalid flags
func TestInvalidFlagErrorFormat(t *testing.T) {
	cmd := exec.Command("go", "run", "../../cmd/alpine/main.go", "--invalid-flag")
	output, err := cmd.CombinedOutput()

	// Command should exit with error
	if err == nil {
		t.Error("Invalid flag should return non-zero exit code")
	}

	outputStr := string(output)

	// Critical: Error message should be helpful
	if !strings.Contains(outputStr, "unknown flag") && !strings.Contains(outputStr, "Error:") {
		t.Error("Error message should indicate unknown flag")
	}
}
