package main

import (
	"os"

	"github.com/Backland-Labs/alpine/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
