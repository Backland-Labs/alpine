package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// reviewCmd represents the review command
type reviewCmd struct {
	cmd *cobra.Command
}

// NewReviewCommand creates a new review command (exported for tests)
func NewReviewCommand() *cobra.Command {
	return newReviewCmd().Command()
}

// newReviewCmd creates a new review command
func newReviewCmd() *reviewCmd {
	rc := &reviewCmd{}

	rc.cmd = &cobra.Command{
		Use:   "review <plan-file>",
		Short: "Review a plan file",
		Long:  `Review a plan file to verify implementation status.`,
		Args:  cobra.ExactArgs(1),
		RunE:  rc.execute,
	}

	return rc
}

// Command returns the cobra command
func (rc *reviewCmd) Command() *cobra.Command {
	return rc.cmd
}

// execute runs the review command
func (rc *reviewCmd) execute(cmd *cobra.Command, args []string) error {
	planFile := args[0]

	// Check if the plan file exists
	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		return fmt.Errorf("plan file not found: %s", planFile)
	}

	// Review the plan file
	fmt.Fprintf(cmd.OutOrStdout(), "Reviewing plan file: %s\n", planFile)

	file, err := os.Open(planFile)
	if err != nil {
		return fmt.Errorf("failed to open plan file: %w", err)
	}
	defer file.Close()

	var totalTasks, implementedTasks, pendingTasks int
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		// Look for task headers (## Task... or similar patterns)
		if strings.HasPrefix(line, "## Task") || strings.HasPrefix(line, "### Task") || strings.HasPrefix(line, "#### Task") {
			totalTasks++
			// Check if the line contains implementation markers
			if strings.Contains(line, "✅") || strings.Contains(line, "IMPLEMENTED") || strings.Contains(line, "COMPLETE") {
				implementedTasks++
			} else {
				pendingTasks++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading plan file: %w", err)
	}

	// Print summary
	fmt.Fprintf(cmd.OutOrStdout(), "Tasks found: %d\n", totalTasks)
	fmt.Fprintf(cmd.OutOrStdout(), "Implemented: %d\n", implementedTasks)
	fmt.Fprintf(cmd.OutOrStdout(), "Pending: %d\n", pendingTasks)

	if pendingTasks == 0 && totalTasks > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "All tasks are implemented!\n")
	} else if pendingTasks > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "There are %d pending tasks\n", pendingTasks)
	}

	return nil
}
