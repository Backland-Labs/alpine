package cli

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrefixWriterCore tests the core PrefixWriter implementation
func TestPrefixWriterCore(t *testing.T) {
	t.Run("BasicFunctionality", func(t *testing.T) {
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "myproject", false, 0)

		// Test single line
		_, err := pw.Write([]byte("Hello, world!\n"))
		require.NoError(t, err)
		assert.Equal(t, "[myproject] Hello, world!\n", buf.String())

		// Test multiple lines in one write
		buf.Reset()
		_, err = pw.Write([]byte("Line 1\nLine 2\nLine 3\n"))
		require.NoError(t, err)
		assert.Equal(t, "[myproject] Line 1\n[myproject] Line 2\n[myproject] Line 3\n", buf.String())
	})

	t.Run("NoTrailingNewline", func(t *testing.T) {
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "test", false, 0)

		_, err := pw.Write([]byte("No newline"))
		require.NoError(t, err)
		assert.Equal(t, "[test] No newline", buf.String())

		// Next write should not add prefix
		_, err = pw.Write([]byte(" continues"))
		require.NoError(t, err)
		assert.Equal(t, "[test] No newline continues", buf.String())

		// Newline should trigger prefix on next write
		_, err = pw.Write([]byte("\nNew line"))
		require.NoError(t, err)
		assert.Equal(t, "[test] No newline continues\n[test] New line", buf.String())
	})

	t.Run("EmptyWrite", func(t *testing.T) {
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "empty", false, 0)

		// Empty write should do nothing
		n, err := pw.Write([]byte{})
		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, "", buf.String())

		// Write after empty should still work
		_, err = pw.Write([]byte("After empty\n"))
		require.NoError(t, err)
		assert.Equal(t, "[empty] After empty\n", buf.String())
	})

	t.Run("ByteCount", func(t *testing.T) {
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "count", false, 0)

		data := []byte("Test data\n")
		n, err := pw.Write(data)
		require.NoError(t, err)
		// Should return the number of bytes in the original data, not including prefix
		assert.Equal(t, len(data), n)
	})

	t.Run("ColorSupport", func(t *testing.T) {
		testCases := []struct {
			colorIndex int
			expected   string
		}{
			{0, "\033[32m"}, // Green
			{1, "\033[34m"}, // Blue
			{2, "\033[36m"}, // Cyan
			{3, "\033[35m"}, // Magenta
			{4, "\033[33m"}, // Yellow
			{5, "\033[31m"}, // Red
			{6, "\033[32m"}, // Wraps to green
		}

		for _, tc := range testCases {
			var buf bytes.Buffer
			pw := NewPrefixWriter(&buf, "color", true, tc.colorIndex)

			_, err := pw.Write([]byte("Test\n"))
			require.NoError(t, err)

			output := buf.String()
			assert.Contains(t, output, tc.expected+"[color]\033[0m Test\n",
				"Color index %d should produce correct color", tc.colorIndex)
		}
	})

	t.Run("MultilineWithEmptyLines", func(t *testing.T) {
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "multi", false, 0)

		input := "First\n\nThird\n\n\nSixth\n"
		_, err := pw.Write([]byte(input))
		require.NoError(t, err)

		expected := "[multi] First\n[multi] \n[multi] Third\n[multi] \n[multi] \n[multi] Sixth\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("PartialLineWrites", func(t *testing.T) {
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "partial", false, 0)

		// Simulate output coming in chunks
		chunks := []string{"Par", "tial ", "line", " output", "\n", "Next", " line"}

		for _, chunk := range chunks {
			_, err := pw.Write([]byte(chunk))
			require.NoError(t, err)
		}

		expected := "[partial] Partial line output\n[partial] Next line"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("CarriageReturn", func(t *testing.T) {
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "cr", false, 0)

		// Carriage return should not trigger new prefix
		_, err := pw.Write([]byte("Progress: 0%\rProgress: 50%\rProgress: 100%\n"))
		require.NoError(t, err)

		expected := "[cr] Progress: 0%\rProgress: 50%\rProgress: 100%\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("ANSISequences", func(t *testing.T) {
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "ansi", false, 0)

		// ANSI sequences should pass through unchanged
		_, err := pw.Write([]byte("\033[1mBold\033[0m text\n\033[31mRed\033[0m text\n"))
		require.NoError(t, err)

		expected := "[ansi] \033[1mBold\033[0m text\n[ansi] \033[31mRed\033[0m text\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("WriteError", func(t *testing.T) {
		// Create a writer that always fails
		failWriter := &failingWriter{failAfter: 0}
		pw := NewPrefixWriter(failWriter, "fail", false, 0)

		_, err := pw.Write([]byte("This will fail\n"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "write failed")
	})

	t.Run("PartialWriteError", func(t *testing.T) {
		// Create a writer that fails after writing the prefix
		failWriter := &failingWriter{failAfter: 7} // "[fail] " is 7 bytes
		pw := NewPrefixWriter(failWriter, "fail", false, 0)

		_, err := pw.Write([]byte("Test\n"))
		assert.Error(t, err)
	})
}

// TestPrefixWriterConcurrency tests concurrent access to PrefixWriter
func TestPrefixWriterConcurrency(t *testing.T) {
	t.Run("ConcurrentWritesNotThreadSafe", func(t *testing.T) {
		// This test documents that PrefixWriter is NOT thread-safe
		// When multiple goroutines write concurrently, output can be interleaved

		var buf safeBuffer
		pw := NewPrefixWriter(&buf, "concurrent", false, 0)

		var wg sync.WaitGroup
		numGoroutines := 10
		linesPerGoroutine := 5

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < linesPerGoroutine; j++ {
					line := fmt.Sprintf("Goroutine %d, line %d\n", id, j)
					_, _ = pw.Write([]byte(line))
				}
			}(i)
		}

		wg.Wait()

		// When writes are concurrent, output may be interleaved
		// We can only verify that we got some output
		output := buf.String()
		assert.NotEmpty(t, output)

		// Count occurrences of the prefix - there should be at least some
		prefixCount := strings.Count(output, "[concurrent] ")
		assert.Greater(t, prefixCount, 0)

		// Note: In a real implementation, each River process would have its own
		// PrefixWriter, so concurrent writes from different processes wouldn't interfere
	})

	t.Run("SerializedWrites", func(t *testing.T) {
		// This test shows that serialized writes work correctly
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "serial", false, 0)

		var mu sync.Mutex
		var wg sync.WaitGroup

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Serialize writes with a mutex
				mu.Lock()
				defer mu.Unlock()

				line := fmt.Sprintf("Goroutine %d\n", id)
				_, _ = pw.Write([]byte(line))
			}(i)
		}

		wg.Wait()

		// With serialized writes, output should be properly prefixed
		lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
		assert.Len(t, lines, 5)

		for _, line := range lines {
			assert.True(t, strings.HasPrefix(line, "[serial] "))
			assert.Contains(t, line, "Goroutine")
		}
	})
}

// TestPrefixWriterEdgeCases tests edge cases and error conditions
func TestPrefixWriterEdgeCases(t *testing.T) {
	t.Run("VeryLongLine", func(t *testing.T) {
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "long", false, 0)

		// Create a very long line (10KB)
		longLine := strings.Repeat("x", 10*1024) + "\n"
		n, err := pw.Write([]byte(longLine))
		require.NoError(t, err)
		assert.Equal(t, len(longLine), n)

		output := buf.String()
		assert.True(t, strings.HasPrefix(output, "[long] "))
		assert.True(t, strings.HasSuffix(output, "\n"))
		assert.Len(t, output, len("[long] ")+10*1024+1) // prefix + content + newline
	})

	t.Run("BinaryData", func(t *testing.T) {
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "binary", false, 0)

		// Write some binary data with null bytes and other control chars
		binaryData := []byte{0x00, 0x01, 0x02, 'H', 'i', 0x7F, '\n', 0xFF}
		_, err := pw.Write(binaryData)
		require.NoError(t, err)

		output := buf.Bytes()
		// The newline in the binary data will trigger a new prefix
		// So we expect: [binary] <data up to \n>\n[binary] <remaining data>
		expected := []byte{
			'[', 'b', 'i', 'n', 'a', 'r', 'y', ']', ' ',
			0x00, 0x01, 0x02, 'H', 'i', 0x7F, '\n',
			'[', 'b', 'i', 'n', 'a', 'r', 'y', ']', ' ',
			0xFF,
		}
		assert.Equal(t, expected, output)
	})

	t.Run("ProjectNameWithSpecialChars", func(t *testing.T) {
		specialNames := []string{
			"my-project",
			"my_project",
			"my.project",
			"my project",
			"@scope/package",
			"προジェクト", // Japanese
			"проект",  // Russian
			"🚀rocket", // Emoji
		}

		for _, name := range specialNames {
			var buf bytes.Buffer
			pw := NewPrefixWriter(&buf, name, false, 0)

			_, err := pw.Write([]byte("Test\n"))
			require.NoError(t, err)

			expected := fmt.Sprintf("[%s] Test\n", name)
			assert.Equal(t, expected, buf.String(),
				"Project name %q should work correctly", name)
		}
	})
}

// TestPrefixWriterWithStdoutStderr tests simulating stdout/stderr behavior
func TestPrefixWriterWithStdoutStderr(t *testing.T) {
	t.Run("SimulatedProcessOutput", func(t *testing.T) {
		var output bytes.Buffer

		// Create separate prefix writers for stdout and stderr
		stdoutPW := NewPrefixWriter(&output, "myapp", true, 0) // Green
		stderrPW := NewPrefixWriter(&output, "myapp", true, 5) // Red for errors

		// Simulate mixed stdout/stderr output
		_, _ = stdoutPW.Write([]byte("Starting application...\n"))
		_, _ = stderrPW.Write([]byte("WARNING: Config file not found, using defaults\n"))
		_, _ = stdoutPW.Write([]byte("Server listening on port 8080\n"))
		_, _ = stderrPW.Write([]byte("ERROR: Failed to connect to database\n"))
		_, _ = stdoutPW.Write([]byte("Shutting down...\n"))

		outputStr := output.String()

		// Check that output contains both regular and error prefixes with different colors
		assert.Contains(t, outputStr, "\033[32m[myapp]\033[0m Starting application")
		assert.Contains(t, outputStr, "\033[31m[myapp]\033[0m WARNING:")
		assert.Contains(t, outputStr, "\033[31m[myapp]\033[0m ERROR:")
		assert.Contains(t, outputStr, "\033[32m[myapp]\033[0m Server listening")
	})
}

// Helper types for testing

type failingWriter struct {
	written   int
	failAfter int
}

func (f *failingWriter) Write(p []byte) (n int, err error) {
	if f.written >= f.failAfter {
		return 0, fmt.Errorf("write failed")
	}
	toWrite := len(p)
	if f.written+toWrite > f.failAfter {
		toWrite = f.failAfter - f.written
	}
	f.written += toWrite
	return toWrite, nil
}

type safeBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (s *safeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// Benchmarks

func BenchmarkPrefixWriterCore(b *testing.B) {
	b.Run("ShortLines", func(b *testing.B) {
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "bench", false, 0)
		line := []byte("This is a typical log line\n")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			pw.atLineStart = true // Reset state
			_, _ = pw.Write(line)
		}
	})

	b.Run("LongLines", func(b *testing.B) {
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "bench", false, 0)
		line := []byte(strings.Repeat("x", 1000) + "\n")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			pw.atLineStart = true // Reset state
			_, _ = pw.Write(line)
		}
	})

	b.Run("ManySmallWrites", func(b *testing.B) {
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "bench", false, 0)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			pw.atLineStart = true // Reset state

			// Simulate character-by-character output
			for _, ch := range []byte("Hello, world!\n") {
				_, _ = pw.Write([]byte{ch})
			}
		}
	})

	b.Run("WithColor", func(b *testing.B) {
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "bench", true, 0)
		line := []byte("Colored output line\n")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			pw.atLineStart = true // Reset state
			_, _ = pw.Write(line)
		}
	})
}

// TestIsTerminalDetection tests terminal detection for color support
func TestIsTerminalDetection(t *testing.T) {
	t.Run("NO_COLOR_env", func(t *testing.T) {
		// Save and restore NO_COLOR
		oldNoColor := os.Getenv("NO_COLOR")
		defer os.Setenv("NO_COLOR", oldNoColor)

		// Set NO_COLOR
		os.Setenv("NO_COLOR", "1")

		// Even if we request color, it should be disabled
		var buf bytes.Buffer
		pw := NewPrefixWriter(&buf, "test", true, 0)

		// In real implementation, this would check NO_COLOR env var
		// For now, we just verify the logic works
		_, _ = pw.Write([]byte("Test\n"))

		// This test documents the expected behavior
		// In real implementation, NO_COLOR=1 would disable colors
	})

	t.Run("PipeVsTTY", func(t *testing.T) {
		// This test documents expected behavior for TTY detection
		// In real implementation:
		// - If stdout is a TTY, colors are enabled by default
		// - If stdout is a pipe/file, colors are disabled by default

		// For unit tests, we just verify the PrefixWriter handles both cases
		var buf bytes.Buffer

		// Test with colors disabled (pipe behavior)
		pwNoColor := NewPrefixWriter(&buf, "pipe", false, 0)
		_, _ = pwNoColor.Write([]byte("No color\n"))
		assert.NotContains(t, buf.String(), "\033[")

		// Test with colors enabled (TTY behavior)
		buf.Reset()
		pwColor := NewPrefixWriter(&buf, "tty", true, 0)
		_, _ = pwColor.Write([]byte("With color\n"))
		assert.Contains(t, buf.String(), "\033[")
	})
}
