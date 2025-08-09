package main

import "testing"

// Add function for testing
func Add(a, b int) int {
	return a + b
}

// TestAdd demonstrates a simple test with table-driven pattern
func TestAdd(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive numbers", 2, 3, 5},
		{"negative numbers", -1, -2, -3},
		{"mixed numbers", -5, 10, 5},
		{"zero values", 0, 0, 0},
		{"one zero", 5, 0, 5},
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

// TestAddBoundaryConditions tests edge cases
func TestAddBoundaryConditions(t *testing.T) {
	// Test with large numbers
	result := Add(1000000, 2000000)
	expected := 3000000
	if result != expected {
		t.Errorf("Add(1000000, 2000000) = %d; expected %d", result, expected)
	}

	// Test with negative boundary
	result = Add(-1000, 1000)
	expected = 0
	if result != expected {
		t.Errorf("Add(-1000, 1000) = %d; expected %d", result, expected)
	}
}
