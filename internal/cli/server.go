package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/Backland-Labs/alpine/internal/claude"
	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/events"
	"github.com/Backland-Labs/alpine/internal/gitx"
	"github.com/Backland-Labs/alpine/internal/logger"
	"github.com/Backland-Labs/alpine/internal/output"
	"github.com/Backland-Labs/alpine/internal/server"
	"github.com/Backland-Labs/alpine/internal/workflow"
	"github.com/spf13/cobra"
)

type serverFlags struct {
	port          int
	eventEndpoint string
}

// newServerCommand creates the server subcommand
func newServerCommand() *cobra.Command {
	flags := &serverFlags{}
	
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start Alpine HTTP server for UI integration",
		Long:  `Start an HTTP server that exposes Alpine's workflow execution via REST API and emits ag-ui protocol events for UI integration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(cmd.Context(), flags)
		},
	}
	
	cmd.Flags().IntVarP(&flags.port, "port", "p", 8080, "Port to run the HTTP server on")
	cmd.Flags().StringVar(&flags.eventEndpoint, "event-endpoint", "", "Default endpoint to send ag-ui events to")
	
	return cmd
}

// runServer starts the HTTP server and handles workflow execution
func runServer(ctx context.Context, flags *serverFlags) error {
	// Initialize logger
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	logger.InitializeFromConfig(cfg)
	
	// Create context that cancels on interrupt
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	
	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Create server instance
	srv := server.NewServer(flags.port)
	
	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		logger.Infof("Starting Alpine HTTP server on port %d", flags.port)
		serverErr <- srv.Start()
	}()
	
	// Create workflow executor
	executor := &workflowExecutor{
		defaultEventEndpoint: flags.eventEndpoint,
		activeRuns:          make(map[string]*runContext),
		mu:                  &sync.Mutex{},
	}
	
	// Set the executor on the server
	srv.SetRunHandler(executor.handleRun)
	
	// Wait for shutdown signal or server error
	select {
	case <-sigChan:
		logger.Info("Received shutdown signal")
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case <-ctx.Done():
		logger.Info("Context cancelled")
	}
	
	// Shutdown server gracefully
	logger.Info("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	
	if err := srv.Stop(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}
	
	return nil
}

// workflowExecutor handles workflow execution for HTTP requests
type workflowExecutor struct {
	defaultEventEndpoint string
	activeRuns          map[string]*runContext
	mu                  *sync.Mutex
}

type runContext struct {
	runID         string
	task          string
	eventEndpoint string
	status        string
	workDir       string
	cancel        context.CancelFunc
	error         error
	completedAt   time.Time
}

// handleRun executes a workflow based on the HTTP request
func (e *workflowExecutor) handleRun(ctx context.Context, req server.RunRequest) error {
	// Generate run ID
	runID := req.ID
	
	// Determine event endpoint
	eventEndpoint := req.EventEndpoint
	if eventEndpoint == "" {
		eventEndpoint = e.defaultEventEndpoint
	}
	
	// Create run directory
	runDir := filepath.Join("runs", runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return fmt.Errorf("failed to create run directory: %w", err)
	}
	
	// Create run context
	runCtx, cancel := context.WithCancel(ctx)
	run := &runContext{
		runID:         runID,
		task:          req.Task,
		eventEndpoint: eventEndpoint,
		status:        "running",
		workDir:       runDir,
		cancel:        cancel,
	}
	
	// Store run context
	e.mu.Lock()
	e.activeRuns[runID] = run
	e.mu.Unlock()
	
	// Execute workflow in background
	go e.executeWorkflow(runCtx, run)
	
	return nil
}

// executeWorkflow runs the Alpine workflow for a specific run
func (e *workflowExecutor) executeWorkflow(ctx context.Context, run *runContext) {
	// Update status when done
	defer func() {
		e.mu.Lock()
		if run.status == "running" {
			run.status = "completed"
			run.completedAt = time.Now()
		}
		e.mu.Unlock()
	}()
	
	// Get absolute path for run directory
	runPath, err := filepath.Abs(run.workDir)
	if err != nil {
		logger.Errorf("Failed to resolve run directory: %v", err)
		e.setRunError(run, err)
		return
	}
	
	// Initialize git repo in run directory using git command
	cmd := exec.Command("git", "init")
	cmd.Dir = runPath
	if err := cmd.Run(); err != nil {
		logger.Errorf("Failed to initialize git repo: %v", err)
		e.setRunError(run, err)
		return
	}
	
	// Configure git user (required for Alpine)
	cmd = exec.Command("git", "config", "user.email", "alpine@example.com")
	cmd.Dir = runPath
	if err := cmd.Run(); err != nil {
		logger.Debugf("Failed to configure git user.email: %v", err)
	}
	
	cmd = exec.Command("git", "config", "user.name", "Alpine Server")
	cmd.Dir = runPath
	if err := cmd.Run(); err != nil {
		logger.Debugf("Failed to configure git user.name: %v", err)
	}
	
	// Create event emitter if endpoint is configured
	var emitter events.EventEmitter
	if run.eventEndpoint != "" {
		// Create event client
		client := events.NewClient(run.eventEndpoint, run.runID)
		emitter = client
		
		// Emit RunStarted event
		emitter.RunStarted(run.runID, run.task)
		
		// Setup Claude hooks for tool events
		if err := e.setupClaudeHooks(run); err != nil {
			logger.Errorf("Failed to setup Claude hooks: %v", err)
			// Continue without hooks - not critical
		}
		
		// Start state monitoring
		// Ensure agent_state directory exists
		stateDir := filepath.Join("agent_state")
		if err := os.MkdirAll(stateDir, 0755); err != nil {
			logger.Errorf("Failed to create agent_state directory: %v", err)
			// Continue without state monitoring - not critical for workflow execution
		}
		
		stateFile := filepath.Join(stateDir, "agent_state.json")
		stateMonitor := events.NewStateMonitor(stateFile, emitter, run.runID)
		go stateMonitor.Start(ctx)
		defer stateMonitor.Stop()
	} else {
		// Use no-op emitter
		emitter = events.NewNoOpEmitter()
	}
	
	// Load configuration
	cfg, err := config.New()
	if err != nil {
		logger.Errorf("Failed to load config: %v", err)
		e.setRunError(run, err)
		return
	}
	
	// Create printer and Claude executor
	printer := output.NewPrinter()
	executor := claude.NewExecutorWithConfig(cfg, printer)
	
	// Create workflow engine with a no-op worktree manager
	// (we don't use worktrees in server mode, each run has its own directory)
	wtMgr := &noOpWorktreeManager{runDir: runPath}
	engine := workflow.NewEngine(executor, wtMgr, cfg)
	engine.SetEventEmitter(emitter)
	
	// Change working directory for workflow execution
	originalDir, err := os.Getwd()
	if err != nil {
		logger.Errorf("Failed to get current directory: %v", err)
		e.setRunError(run, err)
		return
	}
	if err := os.Chdir(runPath); err != nil {
		logger.Errorf("Failed to change to run directory: %v", err)
		e.setRunError(run, err)
		return
	}
	defer os.Chdir(originalDir)
	
	// Execute workflow
	if err := engine.Run(ctx, run.task, false); err != nil {
		logger.Errorf("Workflow execution failed: %v", err)
		e.setRunError(run, err)
		emitter.RunError(run.runID, run.task, err)
		return
	}
	
	// Emit RunFinished event
	emitter.RunFinished(run.runID, "Workflow completed successfully")
}

// setupClaudeHooks configures Claude with ag-ui hooks for tool event emission
func (e *workflowExecutor) setupClaudeHooks(run *runContext) error {
	// Set environment variables for the hook
	os.Setenv("ALPINE_EVENTS_ENDPOINT", run.eventEndpoint)
	os.Setenv("ALPINE_RUN_ID", run.runID)
	
	// Setup the hooks using the existing implementation
	executor := claude.NewExecutor()
	cleanup, err := executor.SetupAgUIHooks(run.eventEndpoint, run.runID)
	if err != nil {
		return err
	}
	// Don't need cleanup in server mode since each run has its own directory
	_ = cleanup
	return nil
}

// setRunError updates the run status to failed
func (e *workflowExecutor) setRunError(run *runContext, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	run.status = "failed"
	run.error = err
	run.completedAt = time.Now()
}

// noOpWorktreeManager is a no-op implementation for server mode
// where each run has its own directory instead of using git worktrees
type noOpWorktreeManager struct{
	runDir string
}

func (n *noOpWorktreeManager) Create(ctx context.Context, taskName string) (*gitx.Worktree, error) {
	// Return a dummy worktree pointing to run directory
	return &gitx.Worktree{
		Path:       n.runDir,
		Branch:     "server-run",
		ParentRepo: n.runDir,
	}, nil
}

func (n *noOpWorktreeManager) Cleanup(ctx context.Context, wt *gitx.Worktree) error {
	// No-op - we don't actually create worktrees in server mode
	return nil
}