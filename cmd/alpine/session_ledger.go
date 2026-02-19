package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	defaultLockLease = 30 * time.Second
	recentWindow     = 24 * time.Hour
)

type sessionError struct {
	ErrorCode   string `json:"error_code"`
	Cause       string `json:"cause"`
	NextStep    string `json:"next_step"`
	Retryable   bool   `json:"retryable"`
	OperationID string `json:"operation_id,omitempty"`
}

type stateTransition struct {
	State       LifecycleState `json:"state"`
	Timestamp   string         `json:"timestamp"`
	Actor       string         `json:"actor"`
	Reason      string         `json:"reason"`
	OperationID string         `json:"operation_id"`
}

type operationRecord struct {
	OperationID    string `json:"operation_id"`
	IdempotencyKey string `json:"idempotency_key"`
	Action         string `json:"action"`
	StartedAt      string `json:"started_at"`
	CompletedAt    string `json:"completed_at,omitempty"`
	TerminalState  string `json:"terminal_state,omitempty"`
}

type sessionOperationLock struct {
	OperationID    string `json:"operation_id"`
	LeaseExpiresAt string `json:"lease_expires_at"`
}

type sessionRecord struct {
	SchemaVersion    string                `json:"schema_version"`
	SessionID        string                `json:"session_id"`
	PrincipalID      string                `json:"principal_id"`
	OwnerPrincipalID string                `json:"owner_principal_id"`
	RepoURL          string                `json:"repo_url"`
	Branch           string                `json:"branch"`
	TargetCommitSHA  string                `json:"target_commit_sha"`
	OpencodeURL      string                `json:"opencode_url"`
	BrowserOpened    bool                  `json:"browser_opened"`
	State            LifecycleState        `json:"state"`
	StateTransitions []stateTransition     `json:"state_transitions"`
	OperationHistory []operationRecord     `json:"operation_history"`
	LastError        *sessionError         `json:"last_error,omitempty"`
	StartedAt        string                `json:"started_at"`
	ReadyAt          string                `json:"ready_at,omitempty"`
	StoppedAt        string                `json:"stopped_at,omitempty"`
	PersistedAt      string                `json:"persisted_at,omitempty"`
	UpdatedAt        string                `json:"updated_at"`
	DurableObjectID  string                `json:"durable_object_id,omitempty"`
	PersistAttempt   int                   `json:"persist_attempt"`
	HistoryTruncated bool                  `json:"history_truncated,omitempty"`
	Lock             *sessionOperationLock `json:"lock,omitempty"`
}

type durableObjectRecord struct {
	DurableObjectID string      `json:"durable_object_id"`
	PersistedAt     string      `json:"persisted_at"`
	Payload         interface{} `json:"payload"`
}

type idempotencyRecord struct {
	OperationID        string `json:"operation_id"`
	SessionID          string `json:"session_id"`
	Action             string `json:"action"`
	RequestFingerprint string `json:"request_fingerprint,omitempty"`
}

type sessionLedger struct {
	SchemaVersion string                         `json:"schema_version"`
	Sessions      map[string]*sessionRecord      `json:"sessions"`
	Idempotency   map[string]idempotencyRecord   `json:"idempotency"`
	DurableObject map[string]durableObjectRecord `json:"durable_objects"`
}

type sessionStore struct {
	path string
	now  func() time.Time
}

func newSessionStore() (*sessionStore, error) {
	path := os.Getenv("ALPINE_LEDGER_PATH")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home dir: %w", err)
		}
		path = filepath.Join(home, ".alpine", "sessions.json")
	}

	return &sessionStore{
		path: path,
		now:  func() time.Time { return time.Now().UTC() },
	}, nil
}

func (s *sessionStore) withLedger(fn func(*sessionLedger) error) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("create ledger dir: %w", err)
	}

	lockPath := s.path + ".lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open ledger lock: %w", err)
	}
	defer lockFile.Close() //nolint:errcheck

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock session ledger: %w", err)
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN) //nolint:errcheck

	ledger, err := s.loadLedger()
	if err != nil {
		return err
	}

	if err := fn(ledger); err != nil {
		return err
	}

	return s.saveLedger(ledger)
}

func (s *sessionStore) loadLedger() (*sessionLedger, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &sessionLedger{
				SchemaVersion: persistSchemaVersion,
				Sessions:      map[string]*sessionRecord{},
				Idempotency:   map[string]idempotencyRecord{},
				DurableObject: map[string]durableObjectRecord{},
			}, nil
		}
		return nil, fmt.Errorf("read session ledger: %w", err)
	}

	var ledger sessionLedger
	if err := json.Unmarshal(data, &ledger); err != nil {
		return nil, fmt.Errorf("parse session ledger: %w", err)
	}

	if ledger.SchemaVersion == "" {
		ledger.SchemaVersion = persistSchemaVersion
	}
	if ledger.Sessions == nil {
		ledger.Sessions = map[string]*sessionRecord{}
	}
	if ledger.Idempotency == nil {
		ledger.Idempotency = map[string]idempotencyRecord{}
	}
	if ledger.DurableObject == nil {
		ledger.DurableObject = map[string]durableObjectRecord{}
	}

	return &ledger, nil
}

func (s *sessionStore) saveLedger(ledger *sessionLedger) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("create ledger dir: %w", err)
	}

	data, err := json.MarshalIndent(ledger, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session ledger: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write session ledger: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("persist session ledger: %w", err)
	}
	return nil
}

func newID(prefix string) (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(buf), nil
}

func currentPrincipalID() string {
	if v := strings.TrimSpace(os.Getenv("ALPINE_PRINCIPAL_ID")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("CF_ACCESS_SUB")); v != "" {
		return v
	}
	return ""
}

func hasAdminOverride() bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("ALPINE_ADMIN")), "1") {
		return true
	}
	roles := strings.Split(os.Getenv("ALPINE_ROLES"), ",")
	for _, role := range roles {
		if strings.TrimSpace(role) == "alpine.admin" {
			return true
		}
	}
	return false
}

func appendOperation(session *sessionRecord, operationID, idempotencyKey, action string, now time.Time) {
	session.OperationHistory = append(session.OperationHistory, operationRecord{
		OperationID:    operationID,
		IdempotencyKey: idempotencyKey,
		Action:         action,
		StartedAt:      now.Format(time.RFC3339),
	})
}

func completeOperation(session *sessionRecord, operationID string, terminalState LifecycleState, now time.Time) {
	for i := range session.OperationHistory {
		if session.OperationHistory[i].OperationID == operationID && session.OperationHistory[i].CompletedAt == "" {
			session.OperationHistory[i].CompletedAt = now.Format(time.RFC3339)
			session.OperationHistory[i].TerminalState = string(terminalState)
			return
		}
	}
}

func appendTransition(session *sessionRecord, to LifecycleState, actor, reason, operationID string, now time.Time) error {
	if session.State != "" {
		if err := validateTransition(session.State, to); err != nil {
			return err
		}
	}
	session.State = to
	stamp := now.Format(time.RFC3339)
	session.UpdatedAt = stamp
	session.StateTransitions = append(session.StateTransitions, stateTransition{
		State:       to,
		Timestamp:   stamp,
		Actor:       actor,
		Reason:      reason,
		OperationID: operationID,
	})
	return nil
}

func acquireSessionLock(session *sessionRecord, operationID string, now time.Time) error {
	if session.Lock != nil {
		expiry, err := time.Parse(time.RFC3339, session.Lock.LeaseExpiresAt)
		if err == nil && expiry.After(now) && session.Lock.OperationID != operationID {
			return newCommandError(
				2,
				ErrOperationConflict,
				"another operation is already active for this session",
				"retry after the active operation lease expires",
				true,
				session.Lock.OperationID,
			)
		}
	}

	session.Lock = &sessionOperationLock{
		OperationID:    operationID,
		LeaseExpiresAt: now.Add(defaultLockLease).Format(time.RFC3339),
	}
	return nil
}

func releaseSessionLock(session *sessionRecord, operationID string) {
	if session.Lock != nil && session.Lock.OperationID == operationID {
		session.Lock = nil
	}
}

func normalizeRepoURL(raw string) string {
	if strings.HasPrefix(raw, "git@") {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err == nil {
		parsed.User = nil
		return parsed.String()
	}
	return raw
}

func sortSessionsByUpdatedDesc(sessions []*sessionRecord) {
	sort.SliceStable(sessions, func(i, j int) bool {
		left, _ := time.Parse(time.RFC3339, sessions[i].UpdatedAt)
		right, _ := time.Parse(time.RFC3339, sessions[j].UpdatedAt)
		return left.After(right)
	})
}
