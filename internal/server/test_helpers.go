package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Backland-Labs/alpine/internal/events"
)

// TestServer provides test utilities for server testing
type TestServer struct {
	*Server
	BaseURL string
}

// NewTestServer creates a server for testing with automatic cleanup
func NewTestServer(t *testing.T) *TestServer {
	srv := NewServer(0) // Use random port

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Start server in background
	go func() {
		_ = srv.Start(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Get actual address
	addr := srv.Address()
	if addr == "" {
		t.Fatal("Server failed to start")
	}

	return &TestServer{
		Server:  srv,
		BaseURL: "http://" + addr,
	}
}

// NewTestServerWithConfig creates a configured server for testing
func NewTestServerWithConfig(t *testing.T, bufferSize, maxClients int) *TestServer {
	srv := NewServerWithConfig(0, bufferSize, maxClients)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Start server in background
	go func() {
		_ = srv.Start(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Get actual address
	addr := srv.Address()
	if addr == "" {
		t.Fatal("Server failed to start")
	}

	return &TestServer{
		Server:  srv,
		BaseURL: "http://" + addr,
	}
}

// Client returns an HTTP client for making requests to the test server
func (ts *TestServer) Client() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}

// MockBatchingEmitter for testing
type MockBatchingEmitter struct {
	EmittedEvents *[]events.BaseEvent
}

func (m *MockBatchingEmitter) EmitToolCallEvent(event events.BaseEvent) {
	*m.EmittedEvents = append(*m.EmittedEvents, event)
}

func (m *MockBatchingEmitter) RunStarted(runID string, task string)             {}
func (m *MockBatchingEmitter) RunFinished(runID string, task string)            {}
func (m *MockBatchingEmitter) RunError(runID string, task string, err error)    {}
func (m *MockBatchingEmitter) StateSnapshot(runID string, snapshot interface{}) {}
