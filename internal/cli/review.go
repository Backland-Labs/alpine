package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// reviewCmd represents the review command structure
type reviewCmd struct {
	cmd *cobra.Command
}

// newReviewCmd creates a new review command
func newReviewCmd() *reviewCmd {
	rc := &reviewCmd{}

	rc.cmd = &cobra.Command{
		Use:   "review <plan-file>",
		Short: "Review an implementation plan using Gemini CLI",
		Long: `Review a detailed implementation plan for a given task using Gemini CLI.
This command reads a plan.md file and analyzes it against the current codebase
to provide feedback and validation.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			planFile := args[0]

			// Check if the plan file exists
			if _, err := os.Stat(planFile); os.IsNotExist(err) {
				return fmt.Errorf("plan file not found: %s", planFile)
			}

			return generateReview(planFile)
		},
	}

	return rc
}

// Command returns the cobra command
func (rc *reviewCmd) Command() *cobra.Command {
	return rc.cmd
}

// generateReview generates a review of an implementation plan using Gemini CLI
func generateReview(planFile string) error {
	// This function will be implemented in Task 2.
	return nil
}
