package flow

import (
	"errors"
	"testing"

	"alpine/internal/apperr"
)

func TestExitCodeMapping(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{nil, 0},
		{apperr.ErrUsage, 2},
		{apperr.ErrPreflight, 3},
		{apperr.ErrAuth, 4},
		{apperr.ErrCleanup, 5},
		{errors.New("other"), 1},
	}
	for _, tc := range tests {
		if got := ExitCode(tc.err); got != tc.want {
			t.Fatalf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
		}
	}
}
