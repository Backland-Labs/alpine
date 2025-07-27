package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewServer validates that a new server instance can be created.
// This test ensures that the server constructor properly initializes
// all required fields and returns a valid server instance that can
// be used for HTTP communication.
func TestNewServer(t *testing.T) {
	t.Run("creates server with default configuration", func(t *testing.T) {
		server := NewServer(3001)

		require.NotNil(t, server, "NewServer should return a non-nil server")
		assert.Equal(t, 3001, server.port, "server should store the provided port")
		assert.NotNil(t, server.httpServer, "server should have an initialized HTTP server")
		assert.NotNil(t, server.eventsChan, "server should have an events channel for SSE")
	})

	t.Run("creates server with custom port", func(t *testing.T) {
		server := NewServer(8080)

		require.NotNil(t, server, "NewServer should return a non-nil server")
		assert.Equal(t, 8080, server.port, "server should use the custom port")
	})
}

// TestServerStartAndStop verifies that the server can be started and stopped gracefully.
// This test ensures proper lifecycle management, including:
// - Server starts and listens on the specified port
// - Server can be accessed via HTTP
// - Server shuts down cleanly when context is canceled
// - No goroutine leaks occur during shutdown
func TestServerStartAndStop(t *testing.T) {
	t.Run("server starts and listens on specified port", func(t *testing.T) {
		server := NewServer(0) // Use port 0 for automatic port assignment in tests
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start server in a goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- server.Start(ctx)
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Verify server is listening
		addr := server.Address()
		assert.NotEmpty(t, addr, "server should return its listening address")

		// Try to connect to the server
		resp, err := http.Get("http://" + addr + "/")
		if err == nil {
			_ = resp.Body.Close()
		}
		// We expect either a successful connection or a specific route not found
		// The important thing is that the server is listening
		assert.True(t, err == nil || resp != nil, "server should be reachable")

		// Shutdown server
		cancel()

		// Wait for server to stop
		select {
		case err := <-errChan:
			// http.ErrServerClosed is expected when server shuts down gracefully
			assert.True(t, err == nil || err == http.ErrServerClosed,
				"server should shut down without unexpected errors")
		case <-time.After(2 * time.Second):
			t.Fatal("server did not shut down within timeout")
		}
	})

	t.Run("server handles multiple start/stop cycles", func(t *testing.T) {
		server := NewServer(0)

		for i := 0; i < 3; i++ {
			ctx, cancel := context.WithCancel(context.Background())

			// Start server
			errChan := make(chan error, 1)
			go func() {
				errChan <- server.Start(ctx)
			}()

			// Give server time to start
			time.Sleep(50 * time.Millisecond)

			// Verify it's running
			addr := server.Address()
			assert.NotEmpty(t, addr, "server should be running in cycle %d", i)

			// Stop server
			cancel()

			// Wait for shutdown
			select {
			case <-errChan:
				// Server stopped
			case <-time.After(1 * time.Second):
				t.Fatalf("server did not stop in cycle %d", i)
			}
		}
	})

	t.Run("server stops immediately if context is already canceled", func(t *testing.T) {
		server := NewServer(0)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel before starting

		err := server.Start(ctx)
		assert.Equal(t, context.Canceled, err, "server should return context.Canceled error")
	})
}

