package main

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

func TestRunMainReturnsExecuteCode(t *testing.T) {
	resetFlags(t)
	origNotify := notifyContext
	origExit := exitProcess
	notifyContext = func(_ context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		return context.WithCancel(context.Background())
	}
	exitProcess = func(int) {}
	t.Cleanup(func() {
		notifyContext = origNotify
		exitProcess = origExit
	})

	origArgs := os.Args
	os.Args = []string{"alpine", "--version"}
	t.Cleanup(func() { os.Args = origArgs })

	if code := runMain(); code != 0 {
		t.Fatalf("runMain code=%d want 0", code)
	}
}

func TestRunMainDoubleInterruptPath(t *testing.T) {
	resetFlags(t)
	origNotify := notifyContext
	origExit := exitProcess

	var mu sync.Mutex
	call := 0
	notifyContext = func(_ context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		mu.Lock()
		defer mu.Unlock()
		call++
		ctx, cancel := context.WithCancel(context.Background())
		if call <= 2 {
			cancel()
		}
		return ctx, cancel
	}

	exited := make(chan struct{}, 1)
	exitProcess = func(code int) {
		if code == 1 {
			exited <- struct{}{}
		}
	}
	t.Cleanup(func() {
		notifyContext = origNotify
		exitProcess = origExit
	})

	origArgs := os.Args
	os.Args = []string{"alpine", "--version"}
	t.Cleanup(func() { os.Args = origArgs })

	_ = runMain()
	select {
	case <-exited:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected forced exit path on double interrupt")
	}
}

func TestMainUsesExitProcess(t *testing.T) {
	resetFlags(t)
	origNotify := notifyContext
	origExit := exitProcess
	notifyContext = func(_ context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		return context.WithCancel(context.Background())
	}

	called := false
	exitProcess = func(int) {
		called = true
	}
	t.Cleanup(func() {
		notifyContext = origNotify
		exitProcess = origExit
	})

	origArgs := os.Args
	os.Args = []string{"alpine", "--version"}
	t.Cleanup(func() { os.Args = origArgs })

	main()
	if !called {
		t.Fatal("expected main to call exitProcess")
	}
}
