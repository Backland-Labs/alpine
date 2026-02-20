package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

type listOutput struct {
	Name         string         `json:"name"`
	Repo         string         `json:"repo"`
	ImageProfile string         `json:"image_profile"`
	State        lifecycleState `json:"state"`
	LastActivity string         `json:"last_activity"`
}

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "Show managed sandboxes",
	Aliases: []string{"ls"},
	RunE:    runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return userErr(fmt.Sprintf("failed to load config: %v", err))
	}

	orch := newOrchestrator(cfg)
	records, err := orch.list()
	if err != nil {
		return sysErr(fmt.Sprintf("failed to list sandboxes: %v", err))
	}

	items := make([]listOutput, 0, len(records))
	for _, rec := range records {
		items = append(items, listOutput{
			Name:         rec.Identity.Name,
			Repo:         rec.Identity.Repo,
			ImageProfile: rec.Identity.ImageProfile,
			State:        rec.State,
			LastActivity: rec.LastActivityAt,
		})
	}

	if jsonOutput {
		return outputJSON(items)
	}

	if len(items) == 0 {
		fmt.Println("No managed sandboxes. Run 'alpine launch <name> --repo <url>' to get started.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATE\tPROFILE\tLAST ACTIVITY\tREPO") //nolint:errcheck
	for _, item := range items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", item.Name, item.State, item.ImageProfile, item.LastActivity, item.Repo) //nolint:errcheck
	}
	return w.Flush()
}
