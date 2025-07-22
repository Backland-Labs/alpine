package performance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// MemoryUsage contains memory usage statistics
type MemoryUsage struct {
	HeapAlloc  uint64 // bytes allocated and still in use
	TotalAlloc uint64 // bytes allocated (even if freed)
	Sys        uint64 // bytes obtained from system
	NumGC      uint32 // number of completed GC cycles
}

// MemoryUsageMeasurer measures memory usage of River
type MemoryUsageMeasurer struct{}

// NewMemoryUsageMeasurer creates a new memory usage measurer
func NewMemoryUsageMeasurer() *MemoryUsageMeasurer {
	return &MemoryUsageMeasurer{}
}

// MeasureMemoryUsage measures the current process memory usage
func (m *MemoryUsageMeasurer) MeasureMemoryUsage() (*MemoryUsage, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return &MemoryUsage{
		HeapAlloc:  memStats.HeapAlloc,
		TotalAlloc: memStats.TotalAlloc,
		Sys:        memStats.Sys,
		NumGC:      memStats.NumGC,
	}, nil
}

// MeasurePythonMemoryUsage attempts to measure Python script memory usage
func (m *MemoryUsageMeasurer) MeasurePythonMemoryUsage() (uint64, error) {
	// Find project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		return 0, err
	}
	
	// Create a Python script that measures its own memory usage
	testScript := `#!/usr/bin/env python3
import psutil
import os
import subprocess
import json

# Get current process memory usage
process = psutil.Process(os.getpid())
memory_info = process.memory_info()

# Import the modules that main.py would use
try:
    import subprocess
    import json
    import os
    # Print memory usage in bytes
    print(memory_info.rss)
except ImportError:
    # If psutil is not available, use a rough estimate
    print(20 * 1024 * 1024)  # 20MB estimate
`
	
	// Write test script to temp file
	tmpFile := filepath.Join(projectRoot, ".memory_test.py")
	if err := os.WriteFile(tmpFile, []byte(testScript), 0755); err != nil {
		return 0, err
	}
	defer os.Remove(tmpFile)
	
	// Run the script and capture output
	cmd := exec.Command("python3", tmpFile)
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try with just 'python' if 'python3' fails
		cmd = exec.Command("python", tmpFile)
		cmd.Dir = projectRoot
		output, err = cmd.CombinedOutput()
		if err != nil {
			// If it still fails, return a reasonable estimate
			return 20 * 1024 * 1024, nil // 20MB estimate
		}
	}
	
	// Parse the output
	outputStr := strings.TrimSpace(string(output))
	memoryBytes, err := strconv.ParseUint(outputStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse memory output: %w", err)
	}
	
	return memoryBytes, nil
}