package performance

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestStartupTimeMeasurement tests that we can accurately measure startup time
func TestStartupTimeMeasurement(t *testing.T) {
	// This test verifies that our startup time measurement infrastructure works correctly
	// It should measure the time it takes for the Alpine binary to start and exit
	measurer := NewStartupTimeMeasurer()

	duration, err := measurer.MeasureStartupTime()
	if err != nil {
		t.Fatalf("Failed to measure startup time: %v", err)
	}

	// Verify we got a reasonable duration
	if duration <= 0 {
		t.Errorf("Expected positive duration, got %v", duration)
	}

	// Startup should be fast (less than 1 second)
	if duration > time.Second {
		t.Errorf("Startup time too slow: %v", duration)
	}
}

// BenchmarkStartupTime benchmarks the startup time of the Alpine binary
func BenchmarkStartupTime(b *testing.B) {
	// This benchmark measures how long it takes to start the Alpine binary
	// and immediately exit (with --help flag to avoid actual execution)

	// Build the binary once before benchmarking
	binaryPath := buildTestBinary(b)
	defer func() {
		_ = os.Remove(binaryPath)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		cmd := exec.Command(binaryPath, "--help")
		err := cmd.Run()
		if err != nil {
			b.Fatalf("Failed to run binary: %v", err)
		}
		b.StopTimer()
		duration := time.Since(start)
		b.ReportMetric(float64(duration.Milliseconds()), "ms/startup")
		b.StartTimer()
	}
}

// buildTestBinary builds the Alpine binary for testing
func buildTestBinary(tb testing.TB) string {
	tb.Helper()

	// Find project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		tb.Fatalf("Failed to find project root: %v", err)
	}

	tmpDir := tb.TempDir()
	binaryPath := filepath.Join(tmpDir, "alpine")

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/alpine")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		tb.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	return binaryPath
}
