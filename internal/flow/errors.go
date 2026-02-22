package flow

import (
	"errors"

	"alpine/internal/apperr"
)

func ExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, apperr.ErrUsage):
		return 2
	case errors.Is(err, apperr.ErrPreflight):
		return 3
	case errors.Is(err, apperr.ErrAuth):
		return 4
	case errors.Is(err, apperr.ErrCleanup):
		return 5
	default:
		return 1
	}
}
