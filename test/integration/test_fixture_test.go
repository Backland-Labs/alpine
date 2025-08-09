package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFixtureFileExists verifies that the test.txt fixture file exists
// with the correct content and line endings
func TestFixtureFileExists(t *testing.T) {
	// Arrange
	fixturePath := filepath.Join("fixtures", "test.txt")
	expectedContent := "Testing hooks"

	// Act - read the file
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("Expected test.txt fixture to exist at %s, got error: %v", fixturePath, err)
	}

	// Assert - verify content
	actualContent := string(content)
	if actualContent != expectedContent {
		t.Errorf("Expected file content %q, got %q", expectedContent, actualContent)
	}

	// Assert - verify Unix line endings (no CR characters)
	if strings.Contains(actualContent, "\r") {
		t.Error("Expected Unix line endings (LF), but found Windows line endings (CRLF)")
	}
}
