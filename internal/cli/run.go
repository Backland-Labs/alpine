package cli

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// WorkflowEngine interface for dependency injection
type WorkflowEngine interface {
	Run(ctx context.Context, issueID string, generatePlan bool) error
}

// NewRunCommand creates the run command
func NewRunCommand(engine WorkflowEngine) *cobra.Command {
	var noPlan bool

	cmd := &cobra.Command{
		Use:   "run <issue-id>",
		Short: "Run River workflow for a Linear issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]

			// Validate Linear issue ID format
			if !isValidLinearID(issueID) {
				return fmt.Errorf("invalid Linear issue ID format: %s", issueID)
			}

			// Run the workflow
			ctx := cmd.Context()
			generatePlan := !noPlan

			return engine.Run(ctx, issueID, generatePlan)
		},
	}

	cmd.Flags().BoolVar(&noPlan, "no-plan", false, "Skip plan generation and execute directly")

	return cmd
}

// isValidLinearID checks if the given ID is a valid Linear issue ID
// Linear IDs follow the format: UPPERCASE-NUMBER (e.g., ABC-123)
func isValidLinearID(id string) bool {
	if id == "" {
		return false
	}

	// Check format: UPPERCASE-NUMBER
	pattern := `^[A-Z]+-[0-9]+$`
	matched, err := regexp.MatchString(pattern, id)
	if err != nil || !matched {
		return false
	}

	// Split and validate parts
	parts := strings.Split(id, "-")
	if len(parts) != 2 {
		return false
	}

	// Check that the number part is >= 1
	num, err := strconv.Atoi(parts[1])
	if err != nil || num < 1 {
		return false
	}

	return true
}
