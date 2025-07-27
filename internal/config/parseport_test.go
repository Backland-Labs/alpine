package config

import (
	"strings"
	"testing"
)

// TestParsePortHelper tests the parsePort helper function directly
// This ensures the port parsing logic is thoroughly tested in isolation
func TestParsePortHelper(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid port 8080",
			input:   "8080",
			want:    8080,
			wantErr: false,
		},
		{
			name:    "minimum valid port",
			input:   "1",
			want:    1,
			wantErr: false,
		},
		{
			name:    "maximum valid port",
			input:   "65535",
			want:    65535,
			wantErr: false,
		},
		{
			name:    "port too low",
			input:   "0",
			want:    0,
			wantErr: true,
			errMsg:  "must be between 1 and 65535",
		},
		{
			name:    "port too high",
			input:   "65536",
			want:    0,
			wantErr: true,
			errMsg:  "must be between 1 and 65535",
		},
		{
			name:    "negative port",
			input:   "-100",
			want:    0,
			wantErr: true,
			errMsg:  "must be between 1 and 65535",
		},
		{
			name:    "not a number",
			input:   "abc",
			want:    0,
			wantErr: true,
			errMsg:  "invalid port number",
		},
		{
			name:    "floating point",
			input:   "8080.5",
			want:    0,
			wantErr: true,
			errMsg:  "invalid port number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePort(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("error message %q does not contain %q", err.Error(), tt.errMsg)
			}
			if got != tt.want {
				t.Errorf("parsePort() = %d, want %d", got, tt.want)
			}
		})
	}
}