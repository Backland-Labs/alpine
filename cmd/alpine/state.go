package main

import "fmt"

type lifecycleState string

const (
	stateNew          lifecycleState = "new"
	stateProvisioning lifecycleState = "provisioning"
	stateRunning      lifecycleState = "running"
	stateSaving       lifecycleState = "saving"
	stateCompleted    lifecycleState = "completed"
	stateTearingDown  lifecycleState = "tearing_down"
	stateDestroyed    lifecycleState = "destroyed"
	stateError        lifecycleState = "error"
)

var validLifecycleTransitions = map[lifecycleState]map[lifecycleState]bool{
	stateNew: {
		stateProvisioning: true,
		stateError:        true,
	},
	stateProvisioning: {
		stateRunning: true,
		stateError:   true,
	},
	stateRunning: {
		stateSaving:      true,
		stateCompleted:   true,
		stateTearingDown: true,
		stateError:       true,
	},
	stateSaving: {
		stateRunning:     true,
		stateCompleted:   true,
		stateTearingDown: true,
		stateError:       true,
	},
	stateCompleted: {
		stateRunning:     true,
		stateTearingDown: true,
		stateError:       true,
	},
	stateTearingDown: {
		stateDestroyed: true,
		stateError:     true,
	},
	stateDestroyed: {
		stateRunning: true,
		stateError:   true,
	},
	stateError: {
		stateRunning: true,
	},
}

func transitionLifecycle(from, to lifecycleState) error {
	if from == to {
		return nil
	}
	next, ok := validLifecycleTransitions[from]
	if !ok || !next[to] {
		return fmt.Errorf("invalid lifecycle transition %q -> %q", from, to)
	}
	return nil
}

func canExportFrom(state lifecycleState) bool {
	return state == stateRunning || state == stateCompleted || state == stateDestroyed
}
