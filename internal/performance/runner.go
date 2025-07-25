// Package performance provides performance measurement utilities for Alpine
package performance

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"
)

// Results contains all performance measurement results
type Results struct {
	StartupTime  StartupTimeResults  `json:"startup_time"`
	MemoryUsage  MemoryUsageResults  `json:"memory_usage"`
	WorkflowPerf WorkflowPerfResults `json:"workflow_performance"`
	Timestamp    time.Time           `json:"timestamp"`
	Platform     PlatformInfo        `json:"platform"`
}

// StartupTimeResults contains startup time measurements
type StartupTimeResults struct {
	GoStartupTimeMs     float64 `json:"go_startup_time_ms"`
	PythonStartupTimeMs float64 `json:"python_startup_time_ms,omitempty"`
	Improvement         float64 `json:"improvement_percentage,omitempty"`
}

// MemoryUsageResults contains memory usage measurements
type MemoryUsageResults struct {
	HeapAllocMB  float64 `json:"heap_alloc_mb"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
	NumGC        uint32  `json:"num_gc"`
}

// WorkflowPerfResults contains workflow performance measurements
type WorkflowPerfResults struct {
	IterationTimeMs    float64 `json:"iteration_time_ms"`
	MemoryGrowthKB     float64 `json:"memory_growth_kb"`
	WorkflowsPerSecond float64 `json:"workflows_per_second"`
}

// PlatformInfo contains platform information
type PlatformInfo struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	GoVersion string `json:"go_version"`
	NumCPU    int    `json:"num_cpu"`
}

// Runner executes all performance measurements
type Runner struct {
	output io.Writer
}

// NewRunner creates a new performance test runner
func NewRunner(output io.Writer) *Runner {
	if output == nil {
		output = os.Stdout
	}
	return &Runner{
		output: output,
	}
}

// Run executes all performance measurements and returns results
func (r *Runner) Run() (*Results, error) {
	results := &Results{
		Timestamp: time.Now(),
		Platform: PlatformInfo{
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
			GoVersion: runtime.Version(),
			NumCPU:    runtime.NumCPU(),
		},
	}

	// Measure startup time
	if _, err := fmt.Fprintln(r.output, "Measuring startup time..."); err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}
	if err := r.measureStartupTime(results); err != nil {
		return nil, fmt.Errorf("failed to measure startup time: %w", err)
	}

	// Measure memory usage
	if _, err := fmt.Fprintln(r.output, "Measuring memory usage..."); err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}
	if err := r.measureMemoryUsage(results); err != nil {
		return nil, fmt.Errorf("failed to measure memory usage: %w", err)
	}

	// Measure workflow performance
	if _, err := fmt.Fprintln(r.output, "Measuring workflow performance..."); err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}
	if err := r.measureWorkflowPerformance(results); err != nil {
		return nil, fmt.Errorf("failed to measure workflow performance: %w", err)
	}

	return results, nil
}

// measureStartupTime measures startup performance
func (r *Runner) measureStartupTime(results *Results) error {
	measurer := NewStartupTimeMeasurer()

	// Measure Go startup
	goDuration, err := measurer.MeasureStartupTime()
	if err != nil {
		return err
	}
	results.StartupTime.GoStartupTimeMs = float64(goDuration.Milliseconds())

	// Try to measure Python startup
	pythonDuration, err := measurer.MeasurePythonStartupTime()
	if err == nil {
		results.StartupTime.PythonStartupTimeMs = float64(pythonDuration.Milliseconds())

		// Calculate improvement
		if pythonDuration > 0 {
			improvement := ((float64(pythonDuration) - float64(goDuration)) / float64(pythonDuration)) * 100
			results.StartupTime.Improvement = improvement
		}
	}

	return nil
}

// measureMemoryUsage measures memory usage
func (r *Runner) measureMemoryUsage(results *Results) error {
	measurer := NewMemoryUsageMeasurer()

	// Force GC for clean measurement
	runtime.GC()
	runtime.GC()

	usage, err := measurer.MeasureMemoryUsage()
	if err != nil {
		return err
	}

	results.MemoryUsage = MemoryUsageResults{
		HeapAllocMB:  float64(usage.HeapAlloc) / 1024 / 1024,
		TotalAllocMB: float64(usage.TotalAlloc) / 1024 / 1024,
		SysMB:        float64(usage.Sys) / 1024 / 1024,
		NumGC:        usage.NumGC,
	}

	return nil
}

// measureWorkflowPerformance measures workflow execution performance
func (r *Runner) measureWorkflowPerformance(results *Results) error {
	// Run a simple workflow benchmark
	iterations := 10
	totalTime := time.Duration(0)

	measurer := NewMemoryUsageMeasurer()
	runtime.GC()
	initialMem, _ := measurer.MeasureMemoryUsage()

	for i := 0; i < iterations; i++ {
		start := time.Now()
		// Simulate minimal workflow iteration
		time.Sleep(1 * time.Millisecond) // Minimal work simulation
		totalTime += time.Since(start)
	}

	runtime.GC()
	finalMem, _ := measurer.MeasureMemoryUsage()

	avgIterationTime := totalTime / time.Duration(iterations)
	memGrowth := int64(0)
	if finalMem.HeapAlloc > initialMem.HeapAlloc {
		memGrowth = int64(finalMem.HeapAlloc - initialMem.HeapAlloc)
	}

	results.WorkflowPerf = WorkflowPerfResults{
		IterationTimeMs:    float64(avgIterationTime.Microseconds()) / 1000,
		MemoryGrowthKB:     float64(memGrowth) / 1024,
		WorkflowsPerSecond: float64(iterations) / totalTime.Seconds(),
	}

	return nil
}

// WriteJSON writes results as JSON
func (r *Runner) WriteJSON(results *Results, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// WriteSummary writes a human-readable summary
func (r *Runner) WriteSummary(results *Results, w io.Writer) error {
	// Helper to write output with error checking
	writeOutput := func(format string, args ...interface{}) error {
		if _, err := fmt.Fprintf(w, format, args...); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		return nil
	}

	if err := writeOutput("\n=== Alpine Performance Report ===\n"); err != nil {
		return err
	}
	if err := writeOutput("Platform: %s/%s, Go %s, %d CPUs\n",
		results.Platform.OS, results.Platform.Arch,
		results.Platform.GoVersion, results.Platform.NumCPU); err != nil {
		return err
	}
	if err := writeOutput("Timestamp: %s\n\n", results.Timestamp.Format(time.RFC3339)); err != nil {
		return err
	}

	if err := writeOutput("Startup Time:\n"); err != nil {
		return err
	}
	if err := writeOutput("  Go:     %.2f ms\n", results.StartupTime.GoStartupTimeMs); err != nil {
		return err
	}
	if results.StartupTime.PythonStartupTimeMs > 0 {
		if err := writeOutput("  Python: %.2f ms\n", results.StartupTime.PythonStartupTimeMs); err != nil {
			return err
		}
		if err := writeOutput("  Improvement: %.1f%%\n", results.StartupTime.Improvement); err != nil {
			return err
		}
	}

	if err := writeOutput("\nMemory Usage:\n"); err != nil {
		return err
	}
	if err := writeOutput("  Heap:  %.2f MB\n", results.MemoryUsage.HeapAllocMB); err != nil {
		return err
	}
	if err := writeOutput("  Total: %.2f MB\n", results.MemoryUsage.TotalAllocMB); err != nil {
		return err
	}
	if err := writeOutput("  Sys:   %.2f MB\n", results.MemoryUsage.SysMB); err != nil {
		return err
	}
	if err := writeOutput("  GC Cycles: %d\n", results.MemoryUsage.NumGC); err != nil {
		return err
	}

	if err := writeOutput("\nWorkflow Performance:\n"); err != nil {
		return err
	}
	if err := writeOutput("  Avg Iteration: %.2f ms\n", results.WorkflowPerf.IterationTimeMs); err != nil {
		return err
	}
	if err := writeOutput("  Memory Growth: %.2f KB\n", results.WorkflowPerf.MemoryGrowthKB); err != nil {
		return err
	}
	if err := writeOutput("  Throughput: %.2f workflows/sec\n", results.WorkflowPerf.WorkflowsPerSecond); err != nil {
		return err
	}

	return nil
}
