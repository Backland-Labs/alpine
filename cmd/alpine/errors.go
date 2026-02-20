package main

// exitError carries a specific exit code alongside the error message.
// Commands return this to signal user errors (1) vs system errors (2).
type exitError struct {
	msg        string
	code       int
	reasonCode string
	retryable  bool
}

func (e *exitError) Error() string { return e.msg }

func (e *exitError) ReasonCode() string { return e.reasonCode }

func (e *exitError) Retryable() bool { return e.retryable }

// userErr returns an exitError with code 1 (user error).
func userErr(msg string) error {
	return &exitError{msg: msg, code: 1}
}

func userErrReason(msg, reasonCode string) error {
	return &exitError{msg: msg, code: 1, reasonCode: reasonCode}
}

// sysErr returns an exitError with code 2 (system error).
func sysErr(msg string) error {
	return &exitError{msg: msg, code: 2}
}

func sysErrReason(msg, reasonCode string, retryable bool) error {
	return &exitError{msg: msg, code: 2, reasonCode: reasonCode, retryable: retryable}
}
