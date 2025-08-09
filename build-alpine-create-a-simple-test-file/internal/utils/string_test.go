package utils

import "testing"

func TestNormalizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal input",
			input:    "feature-branch",
			expected: "feature-branch",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "special characters",
			input:    "feature/branch@name#with$special%chars",
			expected: "feature-branch-name-with-special-chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeBranchName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeBranchName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
