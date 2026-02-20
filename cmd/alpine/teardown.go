package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

type teardownOptions struct {
	Name  string
	Force bool
}

var teardownForce bool

var teardownCmd = &cobra.Command{
	Use:   "teardown <name>",
	Short: "Tear down a managed sandbox",
	Args:  cobra.ExactArgs(1),
	RunE:  runTeardown,
}

func init() {
	teardownCmd.Flags().BoolVar(&teardownForce, "force", false, "force teardown of active sandbox")
	rootCmd.AddCommand(teardownCmd)
}

func runTeardown(_ *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return userErr(fmt.Sprintf("failed to load config: %v", err))
	}

	orch := newOrchestrator(cfg)
	result, err := orch.teardown(teardownOptions{Name: args[0], Force: teardownForce})
	if err != nil {
		return err
	}

	if jsonOutput {
		return outputJSON(result)
	}

	fmt.Printf("Sandbox %s is %s\n", result.Name, result.State)
	if result.CheckpointID != "" {
		fmt.Printf("Verified checkpoint: %s\n", result.CheckpointID)
	}
	return nil
}
