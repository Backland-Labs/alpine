package performance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// StartupTimeMeasurer measures the startup time of River
type StartupTimeMeasurer struct {
	binaryPath       string
	pythonScriptPath string
}

// NewStartupTimeMeasurer creates a new startup time measurer
func NewStartupTimeMeasurer() *StartupTimeMeasurer {
	return &StartupTimeMeasurer{
		binaryPath:       "river",
		pythonScriptPath: "main.py",
	}
}

// MeasureStartupTime measures the startup time of the Go River binary
func (m *StartupTimeMeasurer) MeasureStartupTime() (time.Duration, error) {
	// First, build the binary if needed
	binaryPath, err := m.ensureBinaryExists()
	if err != nil {
		return 0, err
	}

	// Measure time to run with --help (minimal execution)
	start := time.Now()
	cmd := exec.Command(binaryPath, "--help")
	err = cmd.Run()
	duration := time.Since(start)

	if err != nil {
		return 0, err
	}

	return duration, nil
}

// MeasurePythonStartupTime measures the startup time of the Python prototype
func (m *StartupTimeMeasurer) MeasurePythonStartupTime() (time.Duration, error) {
	// Find project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		return 0, err
	}

	// Create a minimal test that just checks state and exits
	testScript := `#!/usr/bin/env python3
import time
start = time.time()
# Minimal startup - just import what's needed
import subprocess
import json
import os
# Exit immediately
print("startup time:", time.time() - start)
`

	// Write test script to temp file
	tmpFile := filepath.Join(projectRoot, ".startup_test.py")
	if err := os.WriteFile(tmpFile, []byte(testScript), 0755); err != nil {
		return 0, err
	}
	defer func() {
		_ = os.Remove(tmpFile)
	}()

	// Measure time to run the minimal script
	start := time.Now()
	cmd := exec.Command("python3", tmpFile)
	cmd.Dir = projectRoot
	if _, err := cmd.CombinedOutput(); err != nil {
		// Try with just 'python' if 'python3' fails
		start = time.Now()
		cmd = exec.Command("python", tmpFile)
		cmd.Dir = projectRoot
		if _, err := cmd.CombinedOutput(); err != nil {
			return 0, fmt.Errorf("failed to run Python: %w", err)
		}
	}
	duration := time.Since(start)

	return duration, nil
}

// ensureBinaryExists builds the River binary if it doesn't exist
func (m *StartupTimeMeasurer) ensureBinaryExists() (string, error) {
	// Check if binary exists in current directory
	if _, err := exec.LookPath(m.binaryPath); err == nil {
		return m.binaryPath, nil
	}

	// Get the absolute path to the binary
	absPath, err := filepath.Abs(m.binaryPath)
	if err != nil {
		return "", err
	}

	// Find project root (where go.mod is)
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}

	// Build the binary from the project root
	buildCmd := exec.Command("go", "build", "-o", absPath, "./cmd/river")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to build binary: %w\nOutput: %s", err, output)
	}

	return absPath, nil
}

// findProjectRoot finds the project root by looking for go.mod
func findProjectRoot() (string, error) {
	dir, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (go.mod)")
		}
		dir = parent
	}
}
