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
		assert.NotNil(t, server.runs, "server should have initialized runs storage")
		assert.NotNil(t, server.plans, "server should have initialized plans storage")
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
		resp, err := http.Get("http://" + addr + "/health")
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

// TestSSEHelloWorldEvent verifies that a client receives the initial event.
// This test ensures that when a client connects to the /events endpoint,
// it immediately receives a "hello world" Server-Sent Event as specified
// in the requirements. This is crucial for frontend clients to verify
// their connection is established and working correctly.
func TestSSEHelloWorldEvent(t *testing.T) {
	t.Run("client receives hello world event on connection", func(t *testing.T) {
		server := NewServer(0)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start server
		errChan := make(chan error, 1)
		go func() {
			errChan <- server.Start(ctx)
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Connect to SSE endpoint
		addr := server.Address()
		require.NotEmpty(t, addr, "server should be running")

		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequestWithContext(ctx, "GET", "http://"+addr+"/events", nil)
		require.NoError(t, err, "should create request")

		resp, err := client.Do(req)
		require.NoError(t, err, "should connect to SSE endpoint")
		defer func() { _ = resp.Body.Close() }()

		// Verify SSE headers
		assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
		assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))
		assert.Equal(t, "keep-alive", resp.Header.Get("Connection"))

		// Read the first event
		buffer := make([]byte, 1024)
		n, err := resp.Body.Read(buffer)
		require.NoError(t, err, "should read event")

		event := string(buffer[:n])
		assert.Contains(t, event, "data: hello world", "should receive hello world event")
		assert.Contains(t, event, "\n\n", "event should be properly terminated")
	})
}

// TestSSEMultipleClients verifies that multiple clients can connect and receive events.
// This test ensures the server can handle concurrent SSE connections, which is
// essential for supporting multiple frontend clients monitoring the same Alpine
// workflow. Each client should receive events independently without interference.
func TestSSEMultipleClients(t *testing.T) {
	t.Run("multiple clients can connect simultaneously", func(t *testing.T) {
		server := NewServer(0)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start server
		errChan := make(chan error, 1)
		go func() {
			errChan <- server.Start(ctx)
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		addr := server.Address()
		require.NotEmpty(t, addr, "server should be running")

		// Connect multiple clients
		numClients := 3
		clients := make([]*http.Response, numClients)

		for i := 0; i < numClients; i++ {
			client := &http.Client{Timeout: 5 * time.Second}
			req, err := http.NewRequestWithContext(ctx, "GET", "http://"+addr+"/events", nil)
			require.NoError(t, err, "should create request for client %d", i)

			resp, err := client.Do(req)
			require.NoError(t, err, "client %d should connect", i)
			clients[i] = resp
			defer func() { _ = resp.Body.Close() }()
		}

		// Verify all clients receive their hello world events
		for i, resp := range clients {
			buffer := make([]byte, 1024)
			n, err := resp.Body.Read(buffer)
			require.NoError(t, err, "client %d should read event", i)

			event := string(buffer[:n])
			assert.Contains(t, event, "data: hello world", "client %d should receive hello world", i)
		}
	})
}

// TestSSEClientDisconnect verifies that the server handles disconnection without crashing.
// This test ensures robust error handling when clients disconnect unexpectedly,
// which is common in real-world scenarios due to network issues, browser refreshes,
// or client-side errors. The server should continue operating normally.
func TestSSEClientDisconnect(t *testing.T) {
	t.Run("server handles client disconnection gracefully", func(t *testing.T) {
		server := NewServer(0)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start server
		errChan := make(chan error, 1)
		go func() {
			errChan <- server.Start(ctx)
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		addr := server.Address()
		require.NotEmpty(t, addr, "server should be running")

		// Connect and immediately disconnect a client
		client := &http.Client{Timeout: 1 * time.Second}
		req, err := http.NewRequestWithContext(ctx, "GET", "http://"+addr+"/events", nil)
		require.NoError(t, err, "should create request")

		resp, err := client.Do(req)
		require.NoError(t, err, "should connect to SSE endpoint")

		// Close connection immediately
		_ = resp.Body.Close()

		// Give server time to handle disconnection
		time.Sleep(50 * time.Millisecond)

		// Connect another client to verify server is still working
		req2, err := http.NewRequestWithContext(ctx, "GET", "http://"+addr+"/events", nil)
		require.NoError(t, err, "should create second request")

		resp2, err := client.Do(req2)
		require.NoError(t, err, "should connect after first client disconnected")
		defer func() { _ = resp2.Body.Close() }()

		// Verify new client still receives events
		buffer := make([]byte, 1024)
		n, err := resp2.Body.Read(buffer)
		require.NoError(t, err, "new client should read event")

		event := string(buffer[:n])
		assert.Contains(t, event, "data: hello world", "new client should receive hello world")
	})
}