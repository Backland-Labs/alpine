package main

import "github.com/spf13/cobra"

var teardownCmd = &cobra.Command{
	Use:   "teardown",
	Short: "Legacy command replaced by alpine down",
	RunE: func(cmd *cobra.Command, args []string) error {
		return newCommandError(
			1,
			ErrCommandReplaced,
			"the teardown command was replaced",
			"use `alpine down <session-id>`",
			false,
			"",
		)
	},
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(teardownCmd)
}
