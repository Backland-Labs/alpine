package cli

import (
	"context"
	"os"

	"github.com/maxmcd/river/internal/claude"
	"github.com/maxmcd/river/internal/config"
	"github.com/maxmcd/river/internal/workflow"
)

// ConfigLoader interface for dependency injection in tests
type ConfigLoader interface {
	Load() (*config.Config, error)
}

// WorkflowEngine interface for dependency injection in tests  
type WorkflowEngine interface {
	Run(ctx context.Context, taskDescription string, generatePlan bool) error
}

// FileReader interface for dependency injection in tests
type FileReader interface {
	ReadFile(filename string) ([]byte, error)
}

// Real implementations for production use

// RealConfigLoader implements ConfigLoader using the real config package
type RealConfigLoader struct{}

func (r *RealConfigLoader) Load() (*config.Config, error) {
	return config.New()
}

// RealWorkflowEngine implements WorkflowEngine using the real workflow package
type RealWorkflowEngine struct {
	engine *workflow.Engine
}

func NewRealWorkflowEngine() *RealWorkflowEngine {
	executor := claude.NewExecutor()
	engine := workflow.NewEngine(executor)
	return &RealWorkflowEngine{engine: engine}
}

func (r *RealWorkflowEngine) Run(ctx context.Context, taskDescription string, generatePlan bool) error {
	return r.engine.Run(ctx, taskDescription, generatePlan)
}

// RealFileReader implements FileReader using the os package
type RealFileReader struct{}

func (r *RealFileReader) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// NewRealDependencies creates production dependencies
func NewRealDependencies() *Dependencies {
	return &Dependencies{
		ConfigLoader:   &RealConfigLoader{},
		WorkflowEngine: NewRealWorkflowEngine(),
		FileReader:     &RealFileReader{},
	}
}

// Dependencies struct for injection (moved from test file for reuse)
type Dependencies struct {
	ConfigLoader   ConfigLoader
	WorkflowEngine WorkflowEngine
	FileReader     FileReader
}