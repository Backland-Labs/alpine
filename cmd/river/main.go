package main

import (
	"os"

	"github.com/maxmcd/river/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
