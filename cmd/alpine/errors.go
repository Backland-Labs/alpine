package main

// exitError carries a specific exit code alongside the error message.
// Commands return this to signal user errors (1) vs system errors (2).
type exitError struct {
	msg  string
	code int
}

func (e *exitError) Error() string { return e.msg }

// userErr returns an exitError with code 1 (user error).
func userErr(msg string) error {
	return &exitError{msg: msg, code: 1}
}

// sysErr returns an exitError with code 2 (system error).
func sysErr(msg string) error {
	return &exitError{msg: msg, code: 2}
}
