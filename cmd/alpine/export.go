package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	exportBranch   string
	exportFromLive bool
)

var exportCmd = &cobra.Command{
	Use:   "export <name>",
	Short: "Export sandbox work to a GitHub branch",
	Args:  cobra.ExactArgs(1),
	RunE:  runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportBranch, "branch", "", "explicit branch name")
	exportCmd.Flags().BoolVar(&exportFromLive, "from-live", false, "export from live runtime instead of checkpoint")
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return userErr(fmt.Sprintf("failed to load config: %v", err))
	}

	orch := newOrchestrator(cfg)
	result, err := orch.export(cmd.Context(), exportOptions{
		Name:     args[0],
		Branch:   exportBranch,
		FromLive: exportFromLive,
	})
	if err != nil {
		return err
	}

	if jsonOutput {
		return outputJSON(result)
	}

	fmt.Printf("Exported %s to branch %s from %s\n", result.Name, result.Branch, result.Source)
	if result.AutoToreDown {
		fmt.Printf("Sandbox auto-tore down after verified durable save\n")
	}
	return nil
}
