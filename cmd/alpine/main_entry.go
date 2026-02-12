//go:build !test

package main

import (
	"context"
	"os"
	"os/signal"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Handle double Ctrl+C: second signal force-exits
	go func() {
		<-ctx.Done()
		ctx2, cancel2 := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel2()
		<-ctx2.Done()
		os.Exit(1)
	}()

	os.Exit(execute(ctx))
}
