package main

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

func TestRunMain(t *testing.T) {
	var mu sync.Mutex
	exitCodes := make([]int, 0, 2)

	var firstCancel context.CancelFunc
	notifyCalls := 0
	notifyFn := func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		if notifyCalls == 0 {
			firstCancel = cancel
		} else {
			// Immediately cancel second context so watcher path exits deterministically.
			cancel()
		}
		notifyCalls++
		return ctx, cancel
	}

	executeFn := func(ctx context.Context) int {
		if firstCancel != nil {
			firstCancel()
		}
		<-ctx.Done()
		return 0
	}

	exitFn := func(code int) {
		mu.Lock()
		exitCodes = append(exitCodes, code)
		mu.Unlock()
	}

	runMain(executeFn, notifyFn, exitFn)

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		mu.Lock()
		count := len(exitCodes)
		mu.Unlock()
		if count >= 2 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(exitCodes) == 0 {
		t.Fatal("expected at least one exit code")
	}
	foundZero := false
	foundOne := false
	for _, code := range exitCodes {
		if code == 0 {
			foundZero = true
		}
		if code == 1 {
			foundOne = true
		}
	}
	if !foundZero {
		t.Fatalf("expected normal exit code 0, got %v", exitCodes)
	}
	if !foundOne {
		t.Fatalf("expected forced exit code 1 path, got %v", exitCodes)
	}
}
