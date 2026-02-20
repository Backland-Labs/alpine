//go:build !test

package main

import (
	"context"
	"os"
	"os/signal"
)

var (
	notifyContext = signal.NotifyContext
	exitProcess   = os.Exit
)

func runMain() int {
	ctx, cancel := notifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Handle double Ctrl+C: second signal force-exits.
	go func() {
		<-ctx.Done()
		ctx2, cancel2 := notifyContext(context.Background(), os.Interrupt)
		defer cancel2()
		<-ctx2.Done()
		exitProcess(1)
	}()

	return execute(ctx)
}

func main() {
	exitProcess(runMain())
}
