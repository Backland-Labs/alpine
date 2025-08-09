package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	noPlan     bool
	noWorktree bool
)

// NewRootCommand creates the root command for Alpine CLI
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "alpine [task description]",
		Short: "Alpine CLI orchestrator for Claude Code workflows",
		Long:  "Alpine is a CLI orchestrator that automates iterative AI-assisted development workflows.",
		Args:  cobra.ArbitraryArgs,
		RunE:  runRoot,
	}

	// Add flags
	rootCmd.PersistentFlags().BoolVar(&noPlan, "no-plan", false, "Skip plan generation")
	rootCmd.PersistentFlags().BoolVar(&noWorktree, "no-worktree", false, "Skip worktree creation")

	// Add commands
	rootCmd.AddCommand(NewVersionCommand())

	return rootCmd
}

func runRoot(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	taskDescription := args[0]
	fmt.Printf("Running Alpine with task: %s\n", taskDescription)
	fmt.Printf("Options: no-plan=%v, no-worktree=%v\n", noPlan, noWorktree)

	return nil
}
