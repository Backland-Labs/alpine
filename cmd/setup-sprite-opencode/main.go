package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"alpine/internal/apperr"
	"alpine/internal/cli"
	"alpine/internal/flow"
)

func main() {
	ctx := context.Background()
	cfg, err := cli.Parse(os.Args[1:], os.Environ(), os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(flow.ExitCode(err))
	}
	if cfg.ShowHelp {
		os.Exit(0)
	}

	if err := flow.Run(ctx, cfg, os.Stdout, os.Stderr); err != nil {
		if !errors.Is(err, apperr.ErrSilent) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(flow.ExitCode(err))
	}
}
