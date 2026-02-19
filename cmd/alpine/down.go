package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var downRetryPersist bool

type downOutput struct {
	SessionID           string         `json:"session_id"`
	OperationID         string         `json:"operation_id"`
	State               LifecycleState `json:"state"`
	DurableObjectID     string         `json:"durable_object_id,omitempty"`
	PersistedAt         string         `json:"persisted_at,omitempty"`
	PersistenceVerified bool           `json:"persistence_verified"`
	NextStep            string         `json:"next_step,omitempty"`
}

var downCmd = &cobra.Command{
	Use:   "down <session-id>",
	Short: "Stop a session and persist it to the durable ledger",
	Args:  cobra.ExactArgs(1),
	RunE:  runDown,
}

func init() {
	downCmd.Flags().BoolVar(&downRetryPersist, "retry-persist", false, "retry persistence for stopped_unpersisted sessions")
	rootCmd.AddCommand(downCmd)
}

func runDown(cmd *cobra.Command, args []string) error {
	sessionID := strings.TrimSpace(args[0])
	principalID := currentPrincipalID()
	if principalID == "" {
		return newCommandError(1, ErrCallerIdentityRequired, "caller identity is required", "set ALPINE_PRINCIPAL_ID or CF_ACCESS_SUB", false, "")
	}

	store, err := newSessionStore()
	if err != nil {
		return sysErr(fmt.Sprintf("failed to initialize session store: %v", err))
	}

	var result downOutput
	var opErr error
	err = store.withLedger(func(ledger *sessionLedger) error {
		session, ok := ledger.Sessions[sessionID]
		if !ok {
			opErr = newCommandError(1, ErrSessionNotFound, "session not found", "run alpine list to discover active/recent sessions", false, "")
			return nil
		}

		if session.OwnerPrincipalID != principalID && !hasAdminOverride() {
			opErr = newCommandError(1, ErrSessionForbidden, "session is owned by another principal", "use the owning principal or alpine.admin role", false, "")
			return nil
		}

		attempt := session.PersistAttempt
		if session.State == StatePersistFailed || session.State == StateStoppedUnpersisted {
			if !downRetryPersist {
				next := fmt.Sprintf("run alpine down %s --retry-persist", session.SessionID)
				opErr = newCommandError(1, ErrPersistenceVerification, "session is stopped but not persisted", next, true, "")
				return nil
			}
			attempt++
		}

		idempotencyKey := fmt.Sprintf("down:%s:%d", sessionID, attempt)
		if existing, ok := ledger.Idempotency[idempotencyKey]; ok && existing.Action == "down" {
			result = downOutput{
				SessionID:           session.SessionID,
				OperationID:         existing.OperationID,
				State:               session.State,
				DurableObjectID:     session.DurableObjectID,
				PersistedAt:         session.PersistedAt,
				PersistenceVerified: session.PersistedAt != "" && session.DurableObjectID != "",
			}
			if session.State == StateStoppedUnpersisted || session.State == StatePersistFailed {
				result.NextStep = fmt.Sprintf("alpine down %s --retry-persist", session.SessionID)
			}
			return nil
		}

		now := store.now()
		operationID, idErr := newID("op")
		if idErr != nil {
			opErr = sysErr(fmt.Sprintf("failed to generate operation id: %v", idErr))
			return nil
		}

		if err := acquireSessionLock(session, operationID, now); err != nil {
			opErr = err
			return nil
		}

		appendOperation(session, operationID, idempotencyKey, "down", now)

		if session.State == StateReady {
			if err := appendTransition(session, StateStopping, principalID, "session stop started", operationID, store.now()); err != nil {
				opErr = sysErr(err.Error())
				releaseSessionLock(session, operationID)
				return nil
			}
		} else if session.State != StatePersistFailed && session.State != StateStoppedUnpersisted && session.State != StateStopped {
			releaseSessionLock(session, operationID)
			opErr = newCommandError(1, ErrOperationConflict, fmt.Sprintf("cannot run down from state %q", session.State), "wait for the active operation to finish and retry", true, operationID)
			return nil
		}

		if session.State != StateStopped {
			if session.State != StatePersisting {
				if err := appendTransition(session, StatePersisting, principalID, "persistence started", operationID, store.now()); err != nil {
					opErr = sysErr(err.Error())
					releaseSessionLock(session, operationID)
					return nil
				}
			}

			durableObjectID, persistedAt, persistErr := persistSessionToDurableObject(ledger, session, operationID, store.now())
			if persistErr != nil {
				session.LastError = &sessionError{
					ErrorCode:   ErrPersistPayloadTooLarge,
					Cause:       persistErr.Error(),
					NextStep:    fmt.Sprintf("run alpine down %s --retry-persist", session.SessionID),
					Retryable:   true,
					OperationID: operationID,
				}
				session.StoppedAt = store.now().Format(time.RFC3339)
				session.PersistAttempt = attempt
				_ = appendTransition(session, StateStoppedUnpersisted, principalID, "persistence failed", operationID, store.now())
				completeOperation(session, operationID, session.State, store.now())
				releaseSessionLock(session, operationID)
				ledger.Idempotency[idempotencyKey] = idempotencyRecord{
					OperationID: operationID,
					SessionID:   session.SessionID,
					Action:      "down",
				}
				result = downOutput{
					SessionID:           session.SessionID,
					OperationID:         operationID,
					State:               session.State,
					PersistenceVerified: false,
					NextStep:            fmt.Sprintf("alpine down %s --retry-persist", session.SessionID),
				}
				opErr = newCommandError(2, ErrPersistPayloadTooLarge, persistErr.Error(), result.NextStep, true, operationID)
				return nil
			}

			session.DurableObjectID = durableObjectID
			session.PersistedAt = persistedAt
			session.StoppedAt = store.now().Format(time.RFC3339)
			session.PersistAttempt = attempt
			if err := appendTransition(session, StateStopped, principalID, "session stopped", operationID, store.now()); err != nil {
				opErr = sysErr(err.Error())
				releaseSessionLock(session, operationID)
				return nil
			}
		}

		completeOperation(session, operationID, session.State, store.now())
		releaseSessionLock(session, operationID)
		ledger.Idempotency[idempotencyKey] = idempotencyRecord{
			OperationID: operationID,
			SessionID:   session.SessionID,
			Action:      "down",
		}

		result = downOutput{
			SessionID:           session.SessionID,
			OperationID:         operationID,
			State:               session.State,
			DurableObjectID:     session.DurableObjectID,
			PersistedAt:         session.PersistedAt,
			PersistenceVerified: session.PersistedAt != "" && session.DurableObjectID != "",
		}
		return nil
	})
	if err != nil {
		return sysErr(fmt.Sprintf("failed to persist session state: %v", err))
	}
	if opErr != nil {
		return opErr
	}

	if jsonOutput {
		return outputJSON(result)
	}

	fmt.Printf("Session:              %s\n", result.SessionID)
	fmt.Printf("Operation:            %s\n", result.OperationID)
	fmt.Printf("State:                %s\n", result.State)
	if result.DurableObjectID != "" {
		fmt.Printf("Durable object:       %s\n", result.DurableObjectID)
	}
	if result.PersistedAt != "" {
		fmt.Printf("Persisted at:         %s\n", result.PersistedAt)
	}
	fmt.Printf("Persistence verified: %t\n", result.PersistenceVerified)
	if result.NextStep != "" {
		fmt.Printf("Next step:            %s\n", result.NextStep)
	}
	return nil
}

func persistSessionToDurableObject(ledger *sessionLedger, session *sessionRecord, operationID string, now time.Time) (string, string, error) {
	payload, truncated, err := persistedPayloadWithCompaction(session)
	if err != nil {
		return "", "", err
	}
	session.HistoryTruncated = truncated

	persistedAt := now.Format(time.RFC3339)
	durableObjectID := "do_" + session.SessionID
	ledger.DurableObject[session.SessionID] = durableObjectRecord{
		DurableObjectID: durableObjectID,
		PersistedAt:     persistedAt,
		Payload:         payload,
	}

	record, ok := ledger.DurableObject[session.SessionID]
	if !ok {
		return "", "", fmt.Errorf("durable object verification failed: missing written record")
	}

	data, err := json.Marshal(record.Payload)
	if err != nil {
		return "", "", fmt.Errorf("durable object verification failed: marshal payload: %w", err)
	}

	var verified struct {
		SchemaVersion string `json:"schema_version"`
		SessionID     string `json:"session_id"`
		PrincipalID   string `json:"principal_id"`
		Branch        string `json:"branch"`
		State         string `json:"state"`
		PersistedAt   string `json:"persisted_at"`
	}
	if err := json.Unmarshal(data, &verified); err != nil {
		return "", "", fmt.Errorf("durable object verification failed: decode payload: %w", err)
	}

	if verified.SchemaVersion != persistSchemaVersion || verified.SessionID != session.SessionID || verified.PrincipalID != session.PrincipalID || verified.Branch != session.Branch {
		return "", "", fmt.Errorf("durable object verification failed: canonical fields mismatch")
	}
	if verified.State == "" {
		return "", "", fmt.Errorf("durable object verification failed: missing state")
	}

	_ = operationID // retained for future provider correlation

	return durableObjectID, persistedAt, nil
}

func persistedPayloadWithCompaction(session *sessionRecord) (map[string]interface{}, bool, error) {
	transitions := append([]stateTransition(nil), session.StateTransitions...)
	operations := append([]operationRecord(nil), session.OperationHistory...)
	truncated := false

	build := func() map[string]interface{} {
		return map[string]interface{}{
			"schema_version":    persistSchemaVersion,
			"session_id":        session.SessionID,
			"principal_id":      session.PrincipalID,
			"repo_url":          normalizeRepoURL(session.RepoURL),
			"branch":            session.Branch,
			"state":             session.State,
			"state_transitions": transitions,
			"started_at":        session.StartedAt,
			"stopped_at":        session.StoppedAt,
			"persisted_at":      time.Now().UTC().Format(time.RFC3339),
			"operation_history": operations,
			"last_error":        session.LastError,
			"history_truncated": truncated,
		}
	}

	payload := build()
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, false, fmt.Errorf("marshal persistence payload: %w", err)
	}
	if len(bytes) <= maxPersistBytes {
		return payload, truncated, nil
	}

	for len(operations) > 1 {
		truncated = true
		operations = operations[len(operations)/2:]
		payload = build()
		bytes, err = json.Marshal(payload)
		if err != nil {
			return nil, false, fmt.Errorf("marshal persistence payload: %w", err)
		}
		if len(bytes) <= maxPersistBytes {
			return payload, truncated, nil
		}
	}

	for len(transitions) > 1 {
		truncated = true
		transitions = transitions[len(transitions)/2:]
		payload = build()
		bytes, err = json.Marshal(payload)
		if err != nil {
			return nil, false, fmt.Errorf("marshal persistence payload: %w", err)
		}
		if len(bytes) <= maxPersistBytes {
			return payload, truncated, nil
		}
	}

	return nil, truncated, fmt.Errorf("persistence payload exceeds %d bytes after compaction", maxPersistBytes)
}
