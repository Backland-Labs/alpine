package performance

import (
	"bytes"
	"strings"
	"testing"
)

// TestGenerateComparison tests the comparison report generation
func TestGenerateComparison(t *testing.T) {
	// This test verifies that we can generate a performance comparison
	// between Go and Python versions

	report, err := GenerateComparison()
	if err != nil {
		t.Fatalf("Failed to generate comparison: %v", err)
	}

	// Verify Go results
	if report.GoResults.StartupTimeMs <= 0 {
		t.Errorf("Expected positive Go startup time, got %f", report.GoResults.StartupTimeMs)
	}

	if report.GoResults.MemoryUsageMB <= 0 {
		t.Errorf("Expected positive Go memory usage, got %f", report.GoResults.MemoryUsageMB)
	}

	// Python results might fail, which is OK
	if report.PythonResults.StartupTimeMs > 0 {
		t.Logf("Python startup time: %.2f ms", report.PythonResults.StartupTimeMs)
		t.Logf("Python memory usage: %.2f MB", report.PythonResults.MemoryUsageMB)
	}

	// Verify summary exists
	if report.Summary.OverallAssessment == "" {
		t.Error("Expected overall assessment in summary")
	}

	t.Logf("Comparison summary: %s", report.Summary.OverallAssessment)
}

// TestWriteComparisonReport tests the report formatting
func TestWriteComparisonReport(t *testing.T) {
	// This test verifies that the comparison report is formatted correctly

	report := &ComparisonReport{
		GoResults: VersionResults{
			StartupTimeMs: 5.5,
			MemoryUsageMB: 12.3,
		},
		PythonResults: VersionResults{
			StartupTimeMs: 25.0,
			MemoryUsageMB: 35.0,
		},
		Summary: Summary{
			StartupImprovement: "4.5x faster",
			MemoryImprovement:  "2.8x less memory",
			OverallAssessment:  "Go version shows significant performance improvements",
		},
	}

	var buf bytes.Buffer
	err := WriteComparisonReport(report, &buf)
	if err != nil {
		t.Fatalf("Failed to write report: %v", err)
	}

	output := buf.String()

	// Verify key elements are present
	if !strings.Contains(output, "Performance Comparison Report") {
		t.Error("Report missing title")
	}

	if !strings.Contains(output, "5.50 ms") {
		t.Error("Report missing Go startup time")
	}

	if !strings.Contains(output, "25.00 ms") {
		t.Error("Report missing Python startup time")
	}

	if !strings.Contains(output, "4.5x faster") {
		t.Error("Report missing startup improvement")
	}

	if !strings.Contains(output, "significant performance improvements") {
		t.Error("Report missing overall assessment")
	}
}

// TestComparisonWithMissingPython tests comparison when Python is unavailable
func TestComparisonWithMissingPython(t *testing.T) {
	// This test verifies that comparison works even without Python

	report := &ComparisonReport{
		GoResults: VersionResults{
			StartupTimeMs: 5.5,
			MemoryUsageMB: 12.3,
		},
		PythonResults: VersionResults{
			StartupTimeMs: -1,
			MemoryUsageMB: -1,
		},
	}

	report.Summary = generateSummary(report.GoResults, report.PythonResults)

	if report.Summary.StartupImprovement != "Python comparison unavailable" {
		t.Errorf("Expected unavailable message, got %s", report.Summary.StartupImprovement)
	}

	if report.Summary.OverallAssessment != "Go version performance measured successfully" {
		t.Errorf("Expected success message, got %s", report.Summary.OverallAssessment)
	}
}
