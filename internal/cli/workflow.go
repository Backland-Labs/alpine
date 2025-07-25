package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/maxmcd/alpine/internal/logger"
)

// runWorkflowWithDependencies is the testable version of runWorkflow with dependency injection
func runWorkflowWithDependencies(ctx context.Context, args []string, noPlan bool, noWorktree bool, fromFile string, continueFlag bool, deps *Dependencies) error {
	var taskDescription string

	// Check for --continue flag first
	if continueFlag {
		// Check if state file exists
		if _, err := deps.FileReader.ReadFile("claude_state.json"); err != nil {
			return fmt.Errorf("no existing state file found to continue from")
		}
		// Continue mode: empty task description
		taskDescription = ""
	} else if fromFile != "" {
		// Get task description from file
		content, err := deps.FileReader.ReadFile(fromFile)
		if err != nil {
			return fmt.Errorf("failed to read task file: %w", err)
		}
		taskDescription = string(content)
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

	// Create workflow engine with finalized config if not already created
	if deps.WorkflowEngine == nil {
		engine, wtMgr := CreateWorkflowEngine(cfg)
		deps.WorkflowEngine = engine
		deps.WorktreeManager = wtMgr
	}

	// Run the workflow (generatePlan is opposite of noPlan)
	generatePlan := !noPlan
	return deps.WorkflowEngine.Run(ctx, taskDescription, generatePlan)
}
