package utils

import (
	"regexp"
	"strings"
)

// NormalizeBranchName normalizes a branch name by replacing invalid characters with hyphens.
// It converts special characters to hyphens to create valid branch names.
func NormalizeBranchName(name string) string {
	if name == "" {
		return ""
	}

	// Replace special characters with hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9-]+`)
	normalized := reg.ReplaceAllString(name, "-")

	// Clean up multiple consecutive hyphens
	reg2 := regexp.MustCompile(`-+`)
	normalized = reg2.ReplaceAllString(normalized, "-")

	// Trim leading/trailing hyphens
	normalized = strings.Trim(normalized, "-")

	return normalized
}
