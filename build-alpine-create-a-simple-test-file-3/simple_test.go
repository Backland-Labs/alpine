package main

import "testing"

// TestAdd tests the basic addition function using table-driven tests
func TestAdd(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive numbers", 2, 3, 5},
		{"addition with zero", 5, 0, 5},
		{"negative numbers", -2, -3, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Add(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Add(%d, %d) = %d; expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Add performs basic addition of two integers
func Add(a, b int) int {
	return a + b
}
