//go:build !test

package main

import (
	"context"
	"os"
	"os/signal"
)

type notifyContextFunc func(context.Context, ...os.Signal) (context.Context, context.CancelFunc)

func runMain(executeFn func(context.Context) int, notifyFn notifyContextFunc, exitFn func(int)) {
	ctx, cancel := notifyFn(context.Background(), os.Interrupt)
	defer cancel()

	// Handle double Ctrl+C: second signal force-exits.
	go func() {
		<-ctx.Done()
		ctx2, cancel2 := notifyFn(context.Background(), os.Interrupt)
		defer cancel2()
		<-ctx2.Done()
		exitFn(1)
	}()

	exitFn(executeFn(ctx))
}

func main() {
	runMain(execute, signal.NotifyContext, os.Exit)
}
