package cli

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Backland-Labs/alpine/internal/logger"
	"github.com/Backland-Labs/alpine/internal/server"
)

// runWorkflowWithDependencies is the testable version of runWorkflow with dependency injection
func runWorkflowWithDependencies(ctx context.Context, args []string, noPlan bool, noWorktree bool, continueFlag bool, deps *Dependencies) error {
	var taskDescription string

	// Check for --continue flag first
	if continueFlag {
		// Check if state file exists
		if _, err := deps.FileReader.ReadFile("agent_state/agent_state.json"); err != nil {
			return fmt.Errorf("no existing state file found to continue from")
		}
		// Continue mode: empty task description
		taskDescription = ""
	} else {
		if len(args) == 0 {
			// Check if we're in bare mode (both flags set)
			if !noPlan || !noWorktree {
				return fmt.Errorf("task description is required")
			}
			// In bare mode, empty args is allowed
			taskDescription = ""
		} else {
			taskDescription = args[0]
		}
	}

	// Validate task description (trim whitespace)
	taskDescription = strings.TrimSpace(taskDescription)
	if taskDescription == "" {
		// Check if we're in bare mode (both flags set)
		if !noPlan || !noWorktree {
			return fmt.Errorf("task description cannot be empty")
		}
		// In bare mode, empty task description is allowed
	}

	// Load configuration
	cfg, err := deps.ConfigLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override worktree setting if --no-worktree flag is used
	if noWorktree {
		cfg.Git.WorktreeEnabled = false
	}

	// Initialize logger based on configuration (for production use)
	logger.InitializeFromConfig(cfg)
	logger.Debugf("Starting Alpine workflow for task: %s", taskDescription)

	// Start HTTP server if requested
	if err := startServerIfRequested(ctx); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Create workflow engine with finalized config if not already created
	if deps.WorkflowEngine == nil {
		engine, wtMgr := CreateWorkflowEngine(cfg)
		deps.WorkflowEngine = engine
		deps.WorktreeManager = wtMgr
	}

	// Run the workflow (generatePlan is opposite of noPlan)
	generatePlan := !noPlan
	workflowErr := deps.WorkflowEngine.Run(ctx, taskDescription, generatePlan)
	
	// Workflow has completed, no need to keep the server running
	// The server will shut down when the context is cancelled
	
	return workflowErr
}

// startServerIfRequested starts the HTTP server if the --serve flag is set in the context.
// The server runs in a separate goroutine and will be shut down when the context is cancelled.
func startServerIfRequested(ctx context.Context) error {
	serve, ok := ctx.Value(serveKey).(bool)
	if !ok || !serve {
		return nil // Server not requested
	}

	// Get port from context or use default
	port := 3001
	if p, ok := ctx.Value(portKey).(int); ok {
		port = p
	}

	// Create and start the server
	httpServer := server.NewServer(port)
	
	go func() {
		logger.Infof("Starting HTTP server on port %d", port)
		if err := httpServer.Start(ctx); err != nil {
			// Only log unexpected errors (not normal shutdown)
			if err != context.Canceled && err != http.ErrServerClosed {
				logger.Errorf("Server error: %v", err)
			}
		}
		logger.Debugf("HTTP server stopped")
	}()
	
	// Give the server a moment to start
	// TODO: Implement a proper readiness check
	time.Sleep(100 * time.Millisecond)
	
	// Verify the server started successfully
	addr := httpServer.Address()
	if addr == "" {
		return fmt.Errorf("server failed to start on port %d", port)
	}
	
	logger.Infof("HTTP server listening on %s", addr)
	return nil
}
