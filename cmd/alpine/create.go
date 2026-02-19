package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Legacy command replaced by alpine up",
	RunE: func(cmd *cobra.Command, args []string) error {
		return newCommandError(
			1,
			ErrCommandReplaced,
			"the create command was replaced",
			"use `alpine up <git-repo> --branch <branch>`",
			false,
			"",
		)
	},
	DisableFlagsInUseLine: true,
	Example:               fmt.Sprintf("  %s", "alpine up git@github.com:org/repo.git --branch main"),
}

func init() {
	rootCmd.AddCommand(createCmd)
}
