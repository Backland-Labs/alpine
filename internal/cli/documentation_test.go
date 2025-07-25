package cli

import (
	"os"
	"strings"
	"testing"
)

// TestDocumentation_GhIssueCommand tests that gh-issue command is properly documented
// This test ensures that both README.md and specs/cli-commands.md are updated with the new command
func TestDocumentation_GhIssueCommand(t *testing.T) {
	// Test README.md documentation
	t.Run("README.md contains gh-issue documentation", func(t *testing.T) {
		readmeContent, err := os.ReadFile("../../README.md")
		if err != nil {
			t.Fatalf("Failed to read README.md: %v", err)
		}

		readme := string(readmeContent)

		// Check for gh-issue command documentation
		if !strings.Contains(readme, "alpine plan gh-issue") {
			t.Error("README.md should document the 'alpine plan gh-issue' command")
		}

		// Check for example usage
		if !strings.Contains(readme, "github.com") && !strings.Contains(readme, "issues") {
			t.Error("README.md should include an example of using gh-issue with a GitHub URL")
		}

		// Check for gh CLI requirement mention
		if !strings.Contains(readme, "gh CLI") && !strings.Contains(readme, "GitHub CLI") {
			t.Error("README.md should mention the gh CLI requirement for gh-issue command")
		}
	})

	// Test specs/cli-commands.md documentation
	t.Run("cli-commands.md contains gh-issue specification", func(t *testing.T) {
		specsContent, err := os.ReadFile("../../specs/cli-commands.md")
		if err != nil {
			t.Fatalf("Failed to read specs/cli-commands.md: %v", err)
		}

		specs := string(specsContent)

		// Check for gh-issue command specification
		if !strings.Contains(specs, "alpine plan gh-issue") {
			t.Error("specs/cli-commands.md should document the 'alpine plan gh-issue' command")
		}

		// Check for command syntax
		if !strings.Contains(specs, "gh-issue <url>") && !strings.Contains(specs, "gh-issue <github-issue-url>") {
			t.Error("specs/cli-commands.md should include the command syntax for gh-issue")
		}

		// Check for behavior description
		if !strings.Contains(specs, "fetch") && !strings.Contains(specs, "GitHub issue") {
			t.Error("specs/cli-commands.md should describe the behavior of fetching GitHub issues")
		}
	})
}
