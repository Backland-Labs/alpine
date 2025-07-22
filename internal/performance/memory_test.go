package performance

import (
	"runtime"
	"testing"
)

// TestMemoryUsageMeasurement tests that we can accurately measure memory usage
func TestMemoryUsageMeasurement(t *testing.T) {
	// This test verifies that our memory usage measurement infrastructure works correctly
	// It should capture the memory footprint of the River process
	measurer := NewMemoryUsageMeasurer()
	
	usage, err := measurer.MeasureMemoryUsage()
	if err != nil {
		t.Fatalf("Failed to measure memory usage: %v", err)
	}
	
	// Verify we got a reasonable memory usage
	if usage.HeapAlloc <= 0 {
		t.Errorf("Expected positive heap allocation, got %d", usage.HeapAlloc)
	}
	
	if usage.TotalAlloc <= 0 {
		t.Errorf("Expected positive total allocation, got %d", usage.TotalAlloc)
	}
	
	if usage.Sys <= 0 {
		t.Errorf("Expected positive system memory, got %d", usage.Sys)
	}
	
	t.Logf("Memory usage - Heap: %d MB, Total: %d MB, Sys: %d MB", 
		usage.HeapAlloc/1024/1024, usage.TotalAlloc/1024/1024, usage.Sys/1024/1024)
}

// BenchmarkMemoryUsage benchmarks the memory usage of River
func BenchmarkMemoryUsage(b *testing.B) {
	// This benchmark measures the memory footprint during typical operations
	// We'll measure memory usage while running a simple workflow
	
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Force GC to get a clean baseline
		runtime.GC()
		runtime.GC()
		
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		
		// Report memory metrics
		b.ReportMetric(float64(m.HeapAlloc)/1024/1024, "MB/heap")
		b.ReportMetric(float64(m.Sys)/1024/1024, "MB/sys")
		b.ReportMetric(float64(m.NumGC), "gc_cycles")
	}
}

// TestMemoryUsageBaseline tests that memory usage meets our baseline requirements
func TestMemoryUsageBaseline(t *testing.T) {
	// This test ensures memory usage is reasonable compared to expectations
	// Go binaries should generally use less memory than Python scripts
	
	measurer := NewMemoryUsageMeasurer()
	
	// Measure Go version memory usage
	goUsage, err := measurer.MeasureMemoryUsage()
	if err != nil {
		t.Fatalf("Failed to measure Go memory usage: %v", err)
	}
	
	// Measure Python version memory usage (if possible)
	pythonUsage, err := measurer.MeasurePythonMemoryUsage()
	if err != nil {
		// Python measurement might fail, which is OK for this test
		t.Logf("Could not measure Python memory usage: %v", err)
	} else {
		// If we could measure both, Go should use equal or less memory
		if goUsage.HeapAlloc > pythonUsage {
			t.Logf("Warning: Go heap usage (%d MB) is higher than Python estimate (%d MB)", 
				goUsage.HeapAlloc/1024/1024, pythonUsage/1024/1024)
		}
	}
	
	// Ensure Go version uses reasonable memory (less than 100MB for basic operations)
	maxMemoryMB := uint64(100)
	if goUsage.HeapAlloc > maxMemoryMB*1024*1024 {
		t.Errorf("Memory usage too high: %d MB (max: %d MB)", 
			goUsage.HeapAlloc/1024/1024, maxMemoryMB)
	}
	
	t.Logf("Go memory usage - Heap: %d MB, Total: %d MB, Sys: %d MB", 
		goUsage.HeapAlloc/1024/1024, goUsage.TotalAlloc/1024/1024, goUsage.Sys/1024/1024)
}

// TestMemoryLeaks tests for memory leaks during repeated operations
func TestMemoryLeaks(t *testing.T) {
	// This test runs multiple iterations and checks if memory grows unbounded
	// which would indicate a memory leak
	
	measurer := NewMemoryUsageMeasurer()
	
	// Get initial baseline
	runtime.GC()
	initialUsage, err := measurer.MeasureMemoryUsage()
	if err != nil {
		t.Fatalf("Failed to measure initial memory usage: %v", err)
	}
	
	// Run multiple iterations
	iterations := 10
	for i := 0; i < iterations; i++ {
		// Simulate some work
		_ = NewStartupTimeMeasurer()
		_ = NewMemoryUsageMeasurer()
	}
	
	// Force GC and measure again
	runtime.GC()
	runtime.GC()
	finalUsage, err := measurer.MeasureMemoryUsage()
	if err != nil {
		t.Fatalf("Failed to measure final memory usage: %v", err)
	}
	
	// Memory should not grow significantly (allow 10MB growth)
	maxGrowthBytes := uint64(10 * 1024 * 1024)
	
	// Calculate growth, accounting for the possibility of memory decreasing
	var growth int64
	if finalUsage.HeapAlloc > initialUsage.HeapAlloc {
		growth = int64(finalUsage.HeapAlloc - initialUsage.HeapAlloc)
	} else {
		growth = -int64(initialUsage.HeapAlloc - finalUsage.HeapAlloc)
	}
	
	if growth > int64(maxGrowthBytes) {
		t.Errorf("Memory grew by %d MB, which may indicate a leak", growth/1024/1024)
	}
	
	t.Logf("Memory growth after %d iterations: %d KB", iterations, growth/1024)
}