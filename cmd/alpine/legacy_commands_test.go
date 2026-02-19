package main

import "testing"

func TestLegacyCreateCommand(t *testing.T) {
	resetFlags(t)
	err := createCmd.RunE(createCmd, []string{})
	if err == nil {
		t.Fatal("expected error")
	}
	ce, ok := err.(*commandError)
	if !ok {
		t.Fatalf("expected *commandError, got %T", err)
	}
	if ce.errorCode != ErrCommandReplaced {
		t.Fatalf("errorCode = %q, want %q", ce.errorCode, ErrCommandReplaced)
	}
}

func TestLegacyTeardownCommand(t *testing.T) {
	resetFlags(t)
	err := teardownCmd.RunE(teardownCmd, []string{})
	if err == nil {
		t.Fatal("expected error")
	}
	ce, ok := err.(*commandError)
	if !ok {
		t.Fatalf("expected *commandError, got %T", err)
	}
	if ce.errorCode != ErrCommandReplaced {
		t.Fatalf("errorCode = %q, want %q", ce.errorCode, ErrCommandReplaced)
	}
}
