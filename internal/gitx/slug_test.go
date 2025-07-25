package gitx

import (
	"fmt"
	"testing"
	"time"
)

// TestSanitizeTaskName tests the sanitization of task names for git branches
func TestSanitizeTaskName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple task",
			input:    "implement feature",
			expected: "implement-feature",
		},
		{
			name:     "with special characters",
			input:    "fix bug #123!",
			expected: "fix-bug-123",
		},
		{
			name:     "with multiple spaces",
			input:    "implement   new    feature",
			expected: "implement-new-feature",
		},
		{
			name:     "uppercase letters",
			input:    "Fix Critical BUG",
			expected: "fix-critical-bug",
		},
		{
			name:     "leading and trailing spaces",
			input:    "  cleanup code  ",
			expected: "cleanup-code",
		},
		{
			name:     "only special characters",
			input:    "!!!@@@###",
			expected: "task",
		},
		{
			name:     "very long task name",
			input:    "this is a very long task name that exceeds the maximum allowed length for a git branch component",
			expected: "this-is-a-very-long-task-name-that-exceeds-the-max",
		},
		{
			name:     "with dots and slashes",
			input:    "feature/user.authentication",
			expected: "feature-user-authentication",
		},
		{
			name:     "already valid",
			input:    "valid-branch-name",
			expected: "valid-branch-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeTaskName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeTaskName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestGenerateUniqueBranchName tests generation of unique branch names
func TestGenerateUniqueBranchName(t *testing.T) {
	tests := []struct {
		name             string
		baseBranch       string
		existingBranches []string
		expected         string
	}{
		{
			name:             "no conflict",
			baseBranch:       "alpine/new-feature",
			existingBranches: []string{"main", "develop"},
			expected:         "alpine/new-feature",
		},
		{
			name:             "one conflict",
			baseBranch:       "alpine/fix-bug",
			existingBranches: []string{"main", "alpine/fix-bug", "develop"},
			expected:         "alpine/fix-bug-2",
		},
		{
			name:             "multiple conflicts",
			baseBranch:       "alpine/feature",
			existingBranches: []string{"alpine/feature", "alpine/feature-2", "alpine/feature-3"},
			expected:         "alpine/feature-4",
		},
		{
			name:             "empty existing branches",
			baseBranch:       "alpine/task",
			existingBranches: []string{},
			expected:         "alpine/task",
		},
		{
			name:             "non-sequential conflicts",
			baseBranch:       "alpine/test",
			existingBranches: []string{"alpine/test", "alpine/test-5", "alpine/test-10"},
			expected:         "alpine/test-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateUniqueBranchName(tt.baseBranch, tt.existingBranches)
			if result != tt.expected {
				t.Errorf("generateUniqueBranchName(%q, %v) = %q, want %q",
					tt.baseBranch, tt.existingBranches, result, tt.expected)
			}
		})
	}
}

// TestGenerateUniqueBranchName_manyConflicts tests the fallback to timestamp
func TestGenerateUniqueBranchName_manyConflicts(t *testing.T) {
	// Mock time for predictable test
	oldTimeNow := timeNow
	mockTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return mockTime }
	defer func() { timeNow = oldTimeNow }()

	baseBranch := "alpine/task"
	existingBranches := make([]string, 0, 101)
	existingBranches = append(existingBranches, baseBranch)

	// Add 100 numbered branches
	for i := 2; i <= 100; i++ {
		existingBranches = append(existingBranches, fmt.Sprintf("%s-%d", baseBranch, i))
	}

	result := generateUniqueBranchName(baseBranch, existingBranches)
	expected := fmt.Sprintf("%s-%d", baseBranch, int(mockTime.Unix()))

	if result != expected {
		t.Errorf("generateUniqueBranchName with 100 conflicts = %q, want %q", result, expected)
	}
}
