package main

import (
	"testing"
)

func TestExitErrorError(t *testing.T) {
	e := &exitError{msg: "something failed", code: 2}
	if e.Error() != "something failed" {
		t.Fatalf("Error() = %q, want %q", e.Error(), "something failed")
	}
}

func TestSysErr(t *testing.T) {
	err := sysErr("boom")
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected *exitError, got %T", err)
	}
	if ee.code != 2 {
		t.Fatalf("code = %d, want 2", ee.code)
	}
}

func TestCommandErrorString(t *testing.T) {
	err := &commandError{msg: "failed"}
	if err.Error() != "failed" {
		t.Fatalf("Error() = %q, want %q", err.Error(), "failed")
	}
}
