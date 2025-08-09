package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "0.2.0"

// NewVersionCommand creates the version command
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of Alpine",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("alpine version %s\n", Version)
		},
	}
}
