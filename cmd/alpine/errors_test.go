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
