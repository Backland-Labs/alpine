package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// WorkflowEngine interface for dependency injection without Linear dependency
type WorkflowEngine interface {
	Run(ctx context.Context, taskDescription string, generatePlan bool) error
}

// NewRunCommand creates the run command that accepts task descriptions
func NewRunCommand(engine WorkflowEngine) *cobra.Command {
	var noPlan bool
	var fromFile string

	cmd := &cobra.Command{
		Use:   "run <task-description>",
		Short: "Run River workflow for a task",
		Long: `Run River workflow for a task described in natural language.
		
Examples:
  river "Implement user authentication"
  river "Fix bug in payment processing" --no-plan
  river --file task.md`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var taskDescription string

			// Get task description from file or command line
			if fromFile != "" {
				content, err := os.ReadFile(fromFile)
				if err != nil {
					return fmt.Errorf("failed to read task file: %w", err)
				}
				taskDescription = strings.TrimSpace(string(content))
			} else {
				if len(args) == 0 {
					return fmt.Errorf("task description is required (use quotes for multi-word descriptions)")
				}
				taskDescription = args[0]
			}

			// Validate task description
			if strings.TrimSpace(taskDescription) == "" {
				return fmt.Errorf("task description cannot be empty")
			}

			// Run the workflow
			ctx := cmd.Context()
			generatePlan := !noPlan

			return engine.Run(ctx, taskDescription, generatePlan)
		},
	}

	cmd.Flags().BoolVar(&noPlan, "no-plan", false, "Skip plan generation and execute directly")
	cmd.Flags().StringVar(&fromFile, "file", "", "Read task description from a file")

	return cmd
}