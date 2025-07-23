package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrinter(t *testing.T) {
	tests := []struct {
		name         string
		useColor     bool
		method       func(p *Printer)
		wantContains string
		wantNoColor  string
		wantErr      bool
	}{
		{
			name:     "success with color",
			useColor: true,
			method: func(p *Printer) {
				p.Success("Operation completed")
			},
			wantContains: "✓ Operation completed",
			wantErr:      false,
		},
		{
			name:     "success without color",
			useColor: false,
			method: func(p *Printer) {
				p.Success("Operation completed")
			},
			wantNoColor: "✓ Operation completed\n",
			wantErr:     false,
		},
		{
			name:     "error with color",
			useColor: true,
			method: func(p *Printer) {
				p.Error("Something went wrong")
			},
			wantContains: "✗ Something went wrong",
			wantErr:      true,
		},
		{
			name:     "warning with format",
			useColor: false,
			method: func(p *Printer) {
				p.Warning("File %s not found", "test.txt")
			},
			wantNoColor: "⚠ File test.txt not found\n",
			wantErr:     true,
		},
		{
			name:     "info message",
			useColor: false,
			method: func(p *Printer) {
				p.Info("Processing %d items", 42)
			},
			wantNoColor: "→ Processing 42 items\n",
			wantErr:     false,
		},
		{
			name:     "step message",
			useColor: false,
			method: func(p *Printer) {
				p.Step("Running step %d/%d", 1, 3)
			},
			wantNoColor: "▶ Running step 1/3\n",
			wantErr:     false,
		},
		{
			name:     "detail message",
			useColor: false,
			method: func(p *Printer) {
				p.Detail("Additional information")
			},
			wantNoColor: "  Additional information\n",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outBuf, errBuf bytes.Buffer
			p := NewPrinterWithWriters(&outBuf, &errBuf, tt.useColor)

			tt.method(p)

			var got string
			if tt.wantErr {
				got = errBuf.String()
			} else {
				got = outBuf.String()
			}

			if tt.useColor {
				// For colored output, just check that it contains the message
				if !strings.Contains(got, tt.wantContains) {
					t.Errorf("output does not contain %q, got %q", tt.wantContains, got)
				}
				// Check that it contains color codes
				if !strings.Contains(got, "\033[") {
					t.Errorf("expected color codes in output, got %q", got)
				}
			} else {
				// For non-colored output, check exact match
				if got != tt.wantNoColor {
					t.Errorf("got %q, want %q", got, tt.wantNoColor)
				}
			}
		})
	}
}

func TestPrinterPlain(t *testing.T) {
	var outBuf bytes.Buffer
	p := NewPrinterWithWriters(&outBuf, nil, false)

	p.Print("Hello %s", "world")
	if got := outBuf.String(); got != "Hello world" {
		t.Errorf("Print() = %q, want %q", got, "Hello world")
	}

	outBuf.Reset()
	p.Println("Hello", "world")
	if got := outBuf.String(); got != "Hello world\n" {
		t.Errorf("Println() = %q, want %q", got, "Hello world\n")
	}
}
