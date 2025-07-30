package main

import (
	"fmt"
	"os"

	"github.com/Backland-Labs/alpine/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		// Print error to stderr before exiting
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
