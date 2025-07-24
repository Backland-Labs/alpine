package config

import (
	"os"
	"testing"
)

// TestShowTodoUpdatesConfiguration tests the ShowTodoUpdates configuration
func TestShowTodoUpdatesConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		want    bool
		wantErr bool
	}{
		{
			name:    "default value",
			envVar:  "",
			want:    true,
			wantErr: false,
		},
		{
			name:    "explicitly true",
			envVar:  "true",
			want:    true,
			wantErr: false,
		},
		{
			name:    "explicitly false",
			envVar:  "false",
			want:    false,
			wantErr: false,
		},
		{
			name:    "invalid value",
			envVar:  "yes",
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar != "" {
				_ = os.Setenv("RIVER_SHOW_TODO_UPDATES", tt.envVar)
				defer func() { _ = os.Unsetenv("RIVER_SHOW_TODO_UPDATES") }()
			} else {
				_ = os.Unsetenv("RIVER_SHOW_TODO_UPDATES")
			}

			cfg, err := New()
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && cfg.ShowTodoUpdates != tt.want {
				t.Errorf("ShowTodoUpdates = %v, want %v", cfg.ShowTodoUpdates, tt.want)
			}
		})
	}
}
