package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Backland-Labs/alpine/internal/claude"
	"github.com/Backland-Labs/alpine/internal/events"
	"github.com/Backland-Labs/alpine/internal/gitx"
	"github.com/Backland-Labs/alpine/internal/logger"
	"github.com/Backland-Labs/alpine/internal/server"
)

// runWorkflowWithDependencies is the testable version of runWorkflow with dependency injection
func runWorkflowWithDependencies(ctx context.Context, args []string, noPlan bool, noWorktree bool, deps *Dependencies) error {
	// Check if we're in server-only mode
	serve, _ := ctx.Value(serveKey).(bool)
	if serve && len(args) == 0 {
		// Load configuration for server
		cfg, err := deps.ConfigLoader.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Initialize logger based on configuration (for production use)
		logger.InitializeFromConfig(cfg)
		logger.Infof("Starting Alpine in server-only mode")

		// Start HTTP server
		httpServer, err := startServerIfRequested(ctx)
		if err != nil {
			return fmt.Errorf("failed to start server: %w", err)
		}

		// Create Alpine workflow engine for REST API
		if httpServer != nil {
			// Create Claude executor
			claudeExecutor := claude.NewExecutor()

			// Create worktree manager
			cwd, _ := os.Getwd()
			var wtMgr gitx.WorktreeManager
			if cwd != "" && cfg.Git.WorktreeEnabled {
				wtMgr = gitx.NewCLIWorktreeManager(cwd, cfg.Git.BaseBranch)
			}

			// Create AlpineWorkflowEngine for REST API operations
			alpineEngine := server.NewAlpineWorkflowEngine(claudeExecutor, wtMgr, cfg)
			alpineEngine.SetServer(httpServer)
			httpServer.SetWorkflowEngine(alpineEngine)

			logger.Infof("Configured Alpine workflow engine for REST API")
		}

		// Keep the server running until context is cancelled
		<-ctx.Done()
		logger.Infof("Server shut down")
		return nil
	}

	var taskDescription string

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
	httpServer, err := startServerIfRequested(ctx)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Create workflow engine with finalized config if not already created
	if deps.WorkflowEngine == nil {
		// Check if we're in server mode and need to pass streamer
		var streamer events.Streamer
		if httpServer != nil {
			streamer = server.NewServerStreamer(httpServer)
		}

		engine, wtMgr, claudeExecutor := CreateWorkflowEngine(cfg, streamer)

		// Connect ServerEventEmitter when --serve mode is active
		if httpServer != nil {
			// Create a broadcast function that converts to server's WorkflowEvent format
			broadcastFunc := func(eventType, runID string, data map[string]interface{}) {
				// Import the server package's WorkflowEvent in the function scope to avoid cycles
				// We'll need to construct the proper WorkflowEvent structure here
				httpServer.BroadcastEvent(server.WorkflowEvent{
					Type:      eventType,
					RunID:     runID,
					Timestamp: time.Now(),
					Data:      data,
				})
			}

			// Create and set the event emitter - using empty runID for now, will be set per workflow
			eventEmitter := events.NewServerEventEmitter("", broadcastFunc)

			// Set the event emitter on the underlying workflow engine
			if realEngine, ok := engine.(*RealWorkflowEngine); ok {
				realEngine.engine.SetEventEmitter(eventEmitter)
			}
		}

		deps.WorkflowEngine = engine
		deps.WorktreeManager = wtMgr

		// Create server workflow engine if server is running and task is provided
		if httpServer != nil && len(args) > 0 {
			logger.Debugf("Creating Alpine workflow engine for server with shared resources")

			// Create AlpineWorkflowEngine with shared ClaudeExecutor and WorktreeManager
			alpineEngine := server.NewAlpineWorkflowEngine(claudeExecutor, wtMgr, cfg)

			// Set server reference on the AlpineWorkflowEngine
			alpineEngine.SetServer(httpServer)

			// Configure the server with the AlpineWorkflowEngine
			httpServer.SetWorkflowEngine(alpineEngine)
		}
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
// Returns the server instance if started, nil otherwise.
func startServerIfRequested(ctx context.Context) (*server.Server, error) {
	serve, ok := ctx.Value(serveKey).(bool)
	if !ok || !serve {
		return nil, nil // Server not requested
	}

	// Get port from context or use default
	const defaultServerPort = 3001
	port := defaultServerPort
	if p, ok := ctx.Value(portKey).(int); ok && p > 0 {
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
	const serverStartupDelay = 100 * time.Millisecond
	time.Sleep(serverStartupDelay)

	// Verify the server started successfully
	addr := httpServer.Address()
	if addr == "" {
		return nil, fmt.Errorf("server failed to start on port %d", port)
	}

	logger.Infof("HTTP server listening on %s", addr)
	return httpServer, nil
}
