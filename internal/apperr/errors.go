package apperr

import "errors"

var (
	ErrUsage     = errors.New("usage error")
	ErrPreflight = errors.New("preflight error")
	ErrAuth      = errors.New("auth error")
	ErrCleanup   = errors.New("cleanup error")
	ErrSilent    = errors.New("silent error")
)
