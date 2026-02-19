package main

import "fmt"

// LifecycleState is the canonical session lifecycle state stored in the ledger.
type LifecycleState string

const (
	StateRequested          LifecycleState = "requested"
	StateProvisioning       LifecycleState = "provisioning"
	StateRepoSyncing        LifecycleState = "repo_syncing"
	StateInstalling         LifecycleState = "installing"
	StateReady              LifecycleState = "ready"
	StateFailing            LifecycleState = "failing"
	StateStopping           LifecycleState = "stopping"
	StatePersisting         LifecycleState = "persisting"
	StateStopped            LifecycleState = "stopped"
	StateFailed             LifecycleState = "failed"
	StatePersistFailed      LifecycleState = "persist_failed"
	StateStoppedUnpersisted LifecycleState = "stopped_unpersisted"
	StateCleanupFailed      LifecycleState = "cleanup_failed"
	StateCloseFailed        LifecycleState = "close_failed"
)

const (
	persistSchemaVersion = "v1"
	maxPersistBytes      = 512 * 1024
)

var terminalStates = map[LifecycleState]bool{
	StateStopped:            true,
	StateFailed:             true,
	StatePersistFailed:      true,
	StateStoppedUnpersisted: true,
	StateCleanupFailed:      true,
	StateCloseFailed:        true,
}

// allowedTransitions defines legal lifecycle edges.
var allowedTransitions = map[LifecycleState]map[LifecycleState]bool{
	StateRequested: {
		StateProvisioning: true,
	},
	StateProvisioning: {
		StateRepoSyncing: true,
		StateFailing:     true,
	},
	StateRepoSyncing: {
		StateInstalling: true,
		StateFailing:    true,
	},
	StateInstalling: {
		StateReady:   true,
		StateFailing: true,
	},
	StateReady: {
		StateStopping: true,
	},
	StateFailing: {
		StateFailed:        true,
		StateCleanupFailed: true,
	},
	StateStopping: {
		StatePersisting:         true,
		StateStoppedUnpersisted: true,
		StateCloseFailed:        true,
	},
	StatePersisting: {
		StateStopped:            true,
		StatePersistFailed:      true,
		StateStoppedUnpersisted: true,
	},
	StateStoppedUnpersisted: {
		StatePersisting: true,
	},
	StatePersistFailed: {
		StatePersisting: true,
	},
}

func isTerminalState(state LifecycleState) bool {
	return terminalStates[state]
}

func canTransition(from, to LifecycleState) bool {
	next, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	return next[to]
}

func validateTransition(from, to LifecycleState) error {
	if canTransition(from, to) {
		return nil
	}
	return fmt.Errorf("illegal lifecycle transition: %s -> %s", from, to)
}
