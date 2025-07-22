package performance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/maxmcd/river/internal/claude"
	"github.com/maxmcd/river/internal/config"
	"github.com/maxmcd/river/internal/core"
	gitxmock "github.com/maxmcd/river/internal/gitx/mock"
	"github.com/maxmcd/river/internal/workflow"
)

// TestLongRunningWorkflowPerformance tests performance during long-running workflows
func TestLongRunningWorkflowPerformance(t *testing.T) {
	// This test verifies that River can handle long-running workflows efficiently
	// without degrading performance over time
	
	// Create a mock workflow that simulates multiple iterations
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "claude_state.json")
	
	// Create initial state
	initialState := &core.State{
		CurrentStepDescription: "Starting long-running test",
		NextStepPrompt:        "Continue test",
		Status:                core.StatusRunning,
	}
	
	if err := initialState.Save(stateFile); err != nil {
		t.Fatalf("Failed to save initial state: %v", err)
	}
	
	// Mock executor
	mockExecutor := &mockLongRunningExecutor{
		iterations: 5,
		stateFile:  stateFile,
	}
	
	// Create workflow engine with mock dependencies
	mockWtMgr := &gitxmock.WorktreeManager{}
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeEnabled: false, // Disable worktree for performance tests
		},
	}
	engine := workflow.NewEngine(mockExecutor, mockWtMgr, cfg)
	engine.SetStateFile(stateFile)
	
	// Measure performance over the workflow execution
	ctx := context.Background()
	start := time.Now()
	
	// Track memory usage at intervals
	measurer := NewMemoryUsageMeasurer()
	initialMemory, _ := measurer.MeasureMemoryUsage()
	
	// Run workflow
	err := engine.Run(ctx, "Long-running performance test workflow", true)
	if err != nil {
		t.Fatalf("Workflow failed: %v", err)
	}
	
	duration := time.Since(start)
	finalMemory, _ := measurer.MeasureMemoryUsage()
	
	// Verify performance metrics
	if duration > 10*time.Second {
		t.Errorf("Workflow took too long: %v", duration)
	}
	
	// Check memory growth
	memoryGrowth := int64(0)
	if finalMemory.HeapAlloc > initialMemory.HeapAlloc {
		memoryGrowth = int64(finalMemory.HeapAlloc - initialMemory.HeapAlloc)
	}
	
	t.Logf("Long-running workflow completed in %v", duration)
	t.Logf("Memory growth: %d KB", memoryGrowth/1024)
	t.Logf("Iterations completed: %d", mockExecutor.completedIterations)
}

// BenchmarkLongRunningWorkflow benchmarks a long-running workflow
func BenchmarkLongRunningWorkflow(b *testing.B) {
	// This benchmark measures performance characteristics of long workflows
	
	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		stateFile := filepath.Join(tmpDir, "claude_state.json")
		
		// Create initial state
		initialState := &core.State{
			CurrentStepDescription: "Benchmark iteration",
			NextStepPrompt:        "Continue",
			Status:                core.StatusRunning,
		}
		
		if err := initialState.Save(stateFile); err != nil {
			b.Fatalf("Failed to save state: %v", err)
		}
		
		mockExecutor := &mockLongRunningExecutor{
			iterations: 3,
			stateFile:  stateFile,
		}
		
		// Create engine with mock dependencies
		mockWtMgr := &gitxmock.WorktreeManager{}
		cfg := &config.Config{
			Git: config.GitConfig{
				WorktreeEnabled: false,
			},
		}
		engine := workflow.NewEngine(mockExecutor, mockWtMgr, cfg)
		engine.SetStateFile(stateFile)
		
		b.StartTimer()
		ctx := context.Background()
		if err := engine.Run(ctx, "Benchmark workflow test", true); err != nil {
			b.Fatalf("Workflow failed: %v", err)
		}
		b.StopTimer()
		
		// Clean up
		_ = os.RemoveAll(tmpDir)
	}
}

// mockLongRunningExecutor simulates a long-running Claude execution
type mockLongRunningExecutor struct {
	iterations          int
	completedIterations int
	stateFile          string
}

func (m *mockLongRunningExecutor) Execute(ctx context.Context, config claude.ExecuteConfig) (string, error) {
	// Simulate some work
	time.Sleep(10 * time.Millisecond)
	
	m.completedIterations++
	
	// Load current state
	state, err := core.LoadState(m.stateFile)
	if err != nil {
		return "", err
	}
	
	// Update state based on iteration
	if m.completedIterations >= m.iterations {
		state.CurrentStepDescription = "Completed all iterations"
		state.NextStepPrompt = ""
		state.Status = core.StatusCompleted
	} else {
		state.CurrentStepDescription = fmt.Sprintf("Completed iteration %d", m.completedIterations)
		state.NextStepPrompt = "Continue to next iteration"
		state.Status = core.StatusRunning
	}
	
	return "Mock execution output", state.Save(m.stateFile)
}


// TestWorkflowMemoryStability tests memory stability during repeated workflows
func TestWorkflowMemoryStability(t *testing.T) {
	// This test runs multiple workflow iterations and checks for memory leaks
	
	measurer := NewMemoryUsageMeasurer()
	
	// Force GC and get baseline
	runtime.GC()
	runtime.GC()
	baselineMemory, _ := measurer.MeasureMemoryUsage()
	
	// Run multiple workflow iterations
	iterations := 10
	for i := 0; i < iterations; i++ {
		tmpDir := t.TempDir()
		stateFile := filepath.Join(tmpDir, "claude_state.json")
		
		// Create and run a mini workflow
		initialState := &core.State{
			CurrentStepDescription: "Test iteration",
			NextStepPrompt:        "Run test",
			Status:                core.StatusRunning,
		}
		
		if err := initialState.Save(stateFile); err != nil {
			t.Fatalf("Failed to save state: %v", err)
		}
		
		mockExecutor := &mockLongRunningExecutor{
			iterations: 1,
			stateFile:  stateFile,
		}
		
		// Create engine with mock dependencies
		mockWtMgr := &gitxmock.WorktreeManager{}
		cfg := &config.Config{
			Git: config.GitConfig{
				WorktreeEnabled: false,
			},
		}
		engine := workflow.NewEngine(mockExecutor, mockWtMgr, cfg)
		engine.SetStateFile(stateFile)
		ctx := context.Background()
		
		if err := engine.Run(ctx, "Memory stability test workflow", true); err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}
		
		// Clean up
		_ = os.RemoveAll(tmpDir)
	}
	
	// Force GC and measure final memory
	runtime.GC()
	runtime.GC()
	finalMemory, _ := measurer.MeasureMemoryUsage()
	
	// Calculate memory growth
	var growth int64
	if finalMemory.HeapAlloc > baselineMemory.HeapAlloc {
		growth = int64(finalMemory.HeapAlloc - baselineMemory.HeapAlloc)
	} else {
		growth = -int64(baselineMemory.HeapAlloc - finalMemory.HeapAlloc)
	}
	
	// Allow up to 5MB growth for workflow overhead
	maxGrowthMB := int64(5)
	if growth > maxGrowthMB*1024*1024 {
		t.Errorf("Memory grew by %d MB after %d workflows, possible leak", 
			growth/1024/1024, iterations)
	}
	
	t.Logf("Memory stability test - Growth: %d KB after %d workflows", 
		growth/1024, iterations)
}