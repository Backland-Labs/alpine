// Package events provides state monitoring functionality for Alpine workflows.
// The StateMonitor watches agent_state.json files for changes and emits events
// when the workflow state is updated.
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Backland-Labs/alpine/internal/core"
	"github.com/Backland-Labs/alpine/internal/logger"
)

const (
	// defaultPollInterval is the default interval for checking state file changes
	defaultPollInterval = 100 * time.Millisecond
)

// StateMonitor watches the agent_state.json file for changes and emits StateSnapshot events.
// It polls the file at regular intervals and emits events when changes are detected.
// The monitor is safe for concurrent use.
type StateMonitor struct {
	stateFile    string
	emitter      EventEmitter
	runID        string
	pollInterval time.Duration
	lastState    *core.State
	mu           sync.Mutex
	stopChan     chan struct{}
	stoppedChan  chan struct{}
}

// NewStateMonitor creates a new state monitor that watches the specified state file.
// It uses the default polling interval of 100ms.
func NewStateMonitor(stateFile string, emitter EventEmitter, runID string) *StateMonitor {
	return &StateMonitor{
		stateFile:    stateFile,
		emitter:      emitter,
		runID:        runID,
		pollInterval: defaultPollInterval,
		stopChan:     make(chan struct{}),
		stoppedChan:  make(chan struct{}),
	}
}

// Start begins monitoring the state file for changes.
// It returns an error if the monitor is already running or if validation fails.
func (m *StateMonitor) Start(ctx context.Context) error {
	// Validate inputs
	if m.stateFile == "" {
		return fmt.Errorf("state file path cannot be empty")
	}
	if m.emitter == nil {
		return fmt.Errorf("event emitter cannot be nil")
	}
	if m.runID == "" {
		return fmt.Errorf("run ID cannot be empty")
	}

	logger.Debugf("Starting state file monitoring for: %s", m.stateFile)

	// Start the monitoring goroutine
	go m.monitor(ctx)

	return nil
}

// monitor is the main loop that checks for state file changes
func (m *StateMonitor) monitor(ctx context.Context) {
	defer close(m.stoppedChan)

	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	// Initial check for existing file
	m.checkAndEmit()

	for {
		select {
		case <-ctx.Done():
			logger.Debug("State monitor stopping due to context cancellation")
			return
		case <-m.stopChan:
			logger.Debug("State monitor stopping due to stop signal")
			return
		case <-ticker.C:
			m.checkAndEmit()
		}
	}
}

// checkAndEmit checks if the state file has changed and emits an event if it has
func (m *StateMonitor) checkAndEmit() {
	// Check if file exists
	stateDir := filepath.Dir(m.stateFile)
	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		// Directory doesn't exist yet, skip
		return
	}

	// Try to read the state file
	data, err := os.ReadFile(m.stateFile)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Debugf("Error reading state file: %v", err)
		}
		return
	}

	// Parse the state
	var state core.State
	if err := json.Unmarshal(data, &state); err != nil {
		logger.Debugf("Error parsing state file: %v", err)
		return
	}

	// Check if state has changed
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.hasStateChanged(&state) {
		// Update last state
		m.lastState = &state

		// Emit StateSnapshot event
		logger.Debug("State change detected, emitting StateSnapshot event")
		m.emitter.StateSnapshot(m.runID, state)
	}
}

// hasStateChanged checks if the state has changed since the last check
func (m *StateMonitor) hasStateChanged(newState *core.State) bool {
	if m.lastState == nil {
		// First time seeing state
		return true
	}

	// Compare states
	return m.lastState.CurrentStepDescription != newState.CurrentStepDescription ||
		m.lastState.NextStepPrompt != newState.NextStepPrompt ||
		m.lastState.Status != newState.Status
}

// Stop stops the monitoring
func (m *StateMonitor) Stop() {
	close(m.stopChan)
	<-m.stoppedChan
}