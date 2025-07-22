package cli

import (
	"os"
	"testing"

	"github.com/maxmcd/river/internal/config"
	"github.com/maxmcd/river/internal/gitx"
	"github.com/stretchr/testify/assert"
)

// TestNewRealDependencies tests that real dependencies can be created without error
func TestNewRealDependencies(t *testing.T) {
	deps := NewRealDependencies()
	
	assert.NotNil(t, deps)
	assert.NotNil(t, deps.ConfigLoader)
	assert.NotNil(t, deps.WorkflowEngine)
	assert.NotNil(t, deps.FileReader)
}

// TestRealConfigLoader tests the real config loader
func TestRealConfigLoader(t *testing.T) {
	loader := &RealConfigLoader{}
	
	// This should work in most environments, though config values may vary
	cfg, err := loader.Load()
	
	// We expect this to succeed in test environment
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
}

// TestRealFileReader tests the real file reader
func TestRealFileReader(t *testing.T) {
	reader := &RealFileReader{}
	
	// Test reading a file that doesn't exist
	_, err := reader.ReadFile("nonexistent-file-12345.txt")
	assert.Error(t, err)
	
	// Note: We could create a temp file to test successful reading,
	// but that would require more setup and the os.ReadFile function
	// is well-tested by the Go standard library
}

// TestNewRealWorkflowEngine tests creating a real workflow engine
func TestNewRealWorkflowEngine(t *testing.T) {
	// Create a test config
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeEnabled: true,
			BaseBranch:      "main",
			AutoCleanupWT:   true,
		},
	}
	
	// Create worktree manager
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	wtMgr := gitx.NewCLIWorktreeManager(cwd, cfg.Git.BaseBranch)
	
	engine := NewRealWorkflowEngine(cfg, wtMgr)
	
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.engine)
}