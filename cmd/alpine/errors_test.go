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

func TestErrorHelpers(t *testing.T) {
	u := userErrReason("bad input", "invalid_input").(*exitError)
	if u.code != 1 || u.reasonCode != "invalid_input" || u.retryable {
		t.Fatalf("unexpected userErrReason: %+v", u)
	}

	s := sysErrReason("temporary", "provider_busy", true).(*exitError)
	if s.code != 2 || s.ReasonCode() != "provider_busy" || !s.Retryable() {
		t.Fatalf("unexpected sysErrReason: %+v", s)
	}
}
