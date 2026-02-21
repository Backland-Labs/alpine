package main

import (
	"context"
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

func runTeardown(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return userErr(fmt.Sprintf("failed to load config: %v", err))
	}

	if cfg.usesControlPlane() {
		return runTeardownControlPlane(cmd.Context(), cfg, args[0])
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

func runTeardownControlPlane(ctx context.Context, cfg *Config, name string) error {
	client := newControlPlaneClient(cfg.Sandbox.ControlPlaneURL)

	resp, err := client.TeardownSandbox(ctx, name, teardownForce)
	if err != nil {
		return err
	}

	result := teardownResult{
		Name:         resp.Name,
		State:        resp.State,
		CheckpointID: resp.CheckpointID,
		Destroyed:    resp.Destroyed,
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
