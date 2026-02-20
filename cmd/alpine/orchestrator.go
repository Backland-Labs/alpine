package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const runtimeProbeStaleAfter = 2 * time.Minute

type operationLock struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	StartedAt string `json:"started_at"`
	ExpiresAt string `json:"expires_at"`
}

type checkpointManifest struct {
	SchemaVersion string `json:"schema_version"`
	CreatedAt     string `json:"created_at"`
	RepoRef       string `json:"repo_ref"`
	ImageProfile  string `json:"image_profile"`
	ContentHash   string `json:"content_hash"`
	CheckpointID  string `json:"checkpoint_id"`
}

type checkpointPointer struct {
	Verified bool               `json:"verified"`
	Manifest checkpointManifest `json:"manifest"`
}

type runtimeProbe struct {
	Status    string `json:"status"`
	CheckedAt string `json:"checked_at"`
}

type sandboxIdentity struct {
	Name         string `json:"name"`
	Repo         string `json:"repo"`
	ImageProfile string `json:"image_profile"`
}

type sandboxRecord struct {
	Identity         sandboxIdentity    `json:"identity"`
	State            lifecycleState     `json:"state"`
	InitialTask      string             `json:"initial_task,omitempty"`
	WebURL           string             `json:"web_url"`
	CreatedAt        string             `json:"created_at"`
	UpdatedAt        string             `json:"updated_at"`
	LastActivityAt   string             `json:"last_activity_at"`
	OperationLock    operationLock      `json:"operation_lock"`
	Checkpoint       *checkpointPointer `json:"checkpoint,omitempty"`
	ExportLock       bool               `json:"export_lock"`
	CompletionSignal bool               `json:"completion_signal"`
	TeardownBlockers []string           `json:"teardown_blockers,omitempty"`
	RuntimeProbe     runtimeProbe       `json:"runtime_probe"`
	LastExportBranch string             `json:"last_export_branch,omitempty"`
	LastExportAt     string             `json:"last_export_at,omitempty"`
	ErrorReason      string             `json:"error_reason,omitempty"`
}

type orchestratorStore struct {
	Sandboxes map[string]*sandboxRecord `json:"sandboxes"`
}

type launchOptions struct {
	Name          string
	Repo          string
	ImageProfile  string
	Task          string
	ForceRecreate bool
}

type launchResult struct {
	Name          string         `json:"name"`
	State         lifecycleState `json:"state"`
	Repo          string         `json:"repo"`
	ImageProfile  string         `json:"image_profile"`
	WebURL        string         `json:"web_url"`
	Reused        bool           `json:"reused"`
	CheckpointID  string         `json:"checkpoint_id,omitempty"`
	Resumed       bool           `json:"resumed"`
	LastActivity  string         `json:"last_activity_at"`
	OperationID   string         `json:"operation_id"`
	OperationType string         `json:"operation_type"`
}

type exportOptions struct {
	Name     string
	Branch   string
	FromLive bool
}

type exportResult struct {
	Name            string         `json:"name"`
	State           lifecycleState `json:"state"`
	Branch          string         `json:"branch"`
	Source          string         `json:"source"`
	AutoToreDown    bool           `json:"auto_tore_down"`
	CheckpointID    string         `json:"checkpoint_id,omitempty"`
	ExportedAt      string         `json:"exported_at"`
	RetryableReason string         `json:"retryable_reason,omitempty"`
}

type teardownResult struct {
	Name         string         `json:"name"`
	State        lifecycleState `json:"state"`
	CheckpointID string         `json:"checkpoint_id,omitempty"`
	Destroyed    bool           `json:"destroyed"`
}

type orchestrator struct {
	config    *Config
	statePath string
	now       func() time.Time
}

func newOrchestrator(cfg *Config) *orchestrator {
	return &orchestrator{
		config:    cfg,
		statePath: orchestratorStatePath(),
		now:       time.Now,
	}
}

func orchestratorStatePath() string {
	if path := strings.TrimSpace(os.Getenv("ALPINE_STATE_PATH")); path != "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".alpine-orchestrator-state.json"
	}
	return filepath.Join(home, ".alpine", "orchestrator-state.json")
}

func (o *orchestrator) sandboxIdentity(name string) (sandboxIdentity, bool, error) {
	store, err := o.loadStore()
	if err != nil {
		return sandboxIdentity{}, false, err
	}
	rec, ok := store.Sandboxes[name]
	if !ok {
		return sandboxIdentity{}, false, nil
	}
	return rec.Identity, true, nil
}

func (o *orchestrator) launch(opts launchOptions) (launchResult, error) {
	store, err := o.loadStore()
	if err != nil {
		return launchResult{}, err
	}

	now := o.now().UTC().Format(time.RFC3339)
	rec, exists := store.Sandboxes[opts.Name]
	resumed := false
	reused := false

	if exists {
		if rec.Identity.Repo != opts.Repo || rec.Identity.ImageProfile != opts.ImageProfile {
			if !opts.ForceRecreate {
				return launchResult{}, userErr(fmt.Sprintf("sandbox %q already exists with repo %q and image profile %q; rerun with --force-recreate to replace", opts.Name, rec.Identity.Repo, rec.Identity.ImageProfile))
			}
			rec = nil
			exists = false
		}
	}

	if !exists {
		rec = &sandboxRecord{
			Identity: sandboxIdentity{
				Name:         opts.Name,
				Repo:         opts.Repo,
				ImageProfile: opts.ImageProfile,
			},
			State:        stateNew,
			WebURL:       fmt.Sprintf("%s/sandboxes/%s", strings.TrimRight(o.config.Sandbox.WebBaseURL, "/"), opts.Name),
			CreatedAt:    now,
			UpdatedAt:    now,
			RuntimeProbe: runtimeProbe{Status: "unknown", CheckedAt: now},
		}
		store.Sandboxes[opts.Name] = rec
	} else {
		reused = true
	}

	if rec.State == stateDestroyed {
		if rec.Checkpoint == nil || !rec.Checkpoint.Verified || !validCheckpoint(rec.Checkpoint.Manifest) {
			rec.State = stateError
			rec.ErrorReason = "checkpoint_checksum_mismatch"
			rec.UpdatedAt = now
			rec.LastActivityAt = now
			rec.RuntimeProbe = runtimeProbe{Status: "unavailable", CheckedAt: now}
			if saveErr := o.saveStore(store); saveErr != nil {
				return launchResult{}, saveErr
			}
			return launchResult{}, sysErrReason("checkpoint checksum mismatch; repair or recreate sandbox", "checkpoint_checksum_mismatch", false)
		}
		if err := transitionLifecycle(rec.State, stateRunning); err != nil {
			return launchResult{}, userErr(err.Error())
		}
		rec.State = stateRunning
		rec.RuntimeProbe = runtimeProbe{Status: "running", CheckedAt: now}
		resumed = true
	}

	if rec.State == stateRunning {
		rec.OperationLock = o.newOperationLock("launch")
		rec.UpdatedAt = now
		rec.LastActivityAt = now
		if strings.TrimSpace(opts.Task) != "" {
			rec.InitialTask = strings.TrimSpace(opts.Task)
		}
		if err := o.saveStore(store); err != nil {
			return launchResult{}, err
		}

		checkpointID := ""
		if rec.Checkpoint != nil {
			checkpointID = rec.Checkpoint.Manifest.CheckpointID
		}

		return launchResult{
			Name:          opts.Name,
			State:         rec.State,
			Repo:          rec.Identity.Repo,
			ImageProfile:  rec.Identity.ImageProfile,
			WebURL:        rec.WebURL,
			Reused:        reused,
			CheckpointID:  checkpointID,
			Resumed:       resumed,
			LastActivity:  rec.LastActivityAt,
			OperationID:   rec.OperationLock.ID,
			OperationType: rec.OperationLock.Type,
		}, nil
	}

	if err := transitionLifecycle(rec.State, stateProvisioning); err != nil {
		return launchResult{}, userErr(err.Error())
	}
	rec.State = stateProvisioning
	rec.OperationLock = o.newOperationLock("launch")
	rec.UpdatedAt = now
	rec.LastActivityAt = now
	rec.RuntimeProbe = runtimeProbe{Status: "starting", CheckedAt: now}

	if err := transitionLifecycle(rec.State, stateRunning); err != nil {
		return launchResult{}, userErr(err.Error())
	}
	rec.State = stateRunning
	rec.RuntimeProbe = runtimeProbe{Status: "running", CheckedAt: now}
	if strings.TrimSpace(opts.Task) != "" {
		rec.InitialTask = strings.TrimSpace(opts.Task)
	}

	rec.UpdatedAt = now
	rec.LastActivityAt = now
	rec.TeardownBlockers = nil

	if err := o.saveStore(store); err != nil {
		return launchResult{}, err
	}

	checkpointID := ""
	if rec.Checkpoint != nil {
		checkpointID = rec.Checkpoint.Manifest.CheckpointID
	}

	return launchResult{
		Name:          opts.Name,
		State:         rec.State,
		Repo:          rec.Identity.Repo,
		ImageProfile:  rec.Identity.ImageProfile,
		WebURL:        rec.WebURL,
		Reused:        reused,
		CheckpointID:  checkpointID,
		Resumed:       resumed,
		LastActivity:  rec.LastActivityAt,
		OperationID:   rec.OperationLock.ID,
		OperationType: rec.OperationLock.Type,
	}, nil
}

func (o *orchestrator) list() ([]sandboxRecord, error) {
	store, err := o.loadStore()
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(store.Sandboxes))
	for name := range store.Sandboxes {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	result := make([]sandboxRecord, 0, len(keys))
	for _, name := range keys {
		result = append(result, *store.Sandboxes[name])
	}
	return result, nil
}

func (o *orchestrator) status(name string) (*sandboxRecord, bool, bool, error) {
	store, err := o.loadStore()
	if err != nil {
		return nil, false, false, err
	}
	rec, ok := store.Sandboxes[name]
	if !ok {
		return nil, false, false, nil
	}
	checkedAt, err := time.Parse(time.RFC3339, rec.RuntimeProbe.CheckedAt)
	if err != nil {
		return rec, true, true, nil
	}
	return rec, true, o.now().UTC().Sub(checkedAt) > runtimeProbeStaleAfter, nil
}

func (o *orchestrator) export(ctx context.Context, opts exportOptions) (exportResult, error) {
	_ = ctx
	store, err := o.loadStore()
	if err != nil {
		return exportResult{}, err
	}

	rec, ok := store.Sandboxes[opts.Name]
	if !ok {
		return exportResult{}, userErrReason(fmt.Sprintf("sandbox %q not found", opts.Name), "sandbox_not_found")
	}

	if rec.State == stateSaving || rec.State == stateTearingDown {
		return exportResult{}, sysErrReason("sandbox is busy; retry export shortly", "export_retryable_state", true)
	}

	if !canExportFrom(rec.State) {
		return exportResult{}, userErrReason(fmt.Sprintf("cannot export from state %q", rec.State), "export_state_blocked")
	}

	if o.config.GitHub.RequireAuth && os.Getenv("GH_TOKEN") == "" && os.Getenv("GITHUB_TOKEN") == "" {
		return exportResult{}, userErrReason("missing GitHub auth; set GH_TOKEN or GITHUB_TOKEN", "github_auth_missing")
	}

	source := "checkpoint"
	if opts.FromLive {
		source = "live"
	} else {
		if rec.Checkpoint == nil || !rec.Checkpoint.Verified || !validCheckpoint(rec.Checkpoint.Manifest) {
			return exportResult{}, sysErrReason("no verified checkpoint available for export", "checkpoint_missing", false)
		}
	}

	branch := strings.TrimSpace(opts.Branch)
	if branch == "" {
		branch = fmt.Sprintf("%s/%s", o.config.GitHub.BranchPrefix, opts.Name)
	}

	now := o.now().UTC().Format(time.RFC3339)
	rec.ExportLock = true
	rec.OperationLock = o.newOperationLock("export")
	rec.LastExportBranch = branch
	rec.LastExportAt = now
	rec.LastActivityAt = now
	rec.UpdatedAt = now
	rec.ExportLock = false

	autoToreDown := false
	if rec.State == stateCompleted && o.config.Sandbox.AutoTeardown {
		if rec.Checkpoint != nil && rec.Checkpoint.Verified && validCheckpoint(rec.Checkpoint.Manifest) {
			rec.State = stateTearingDown
			rec.State = stateDestroyed
			rec.RuntimeProbe = runtimeProbe{Status: "stopped", CheckedAt: now}
			autoToreDown = true
		} else {
			rec.TeardownBlockers = appendUnique(rec.TeardownBlockers, "checkpoint_unverified")
		}
	}

	if err := o.saveStore(store); err != nil {
		return exportResult{}, err
	}

	checkpointID := ""
	if rec.Checkpoint != nil {
		checkpointID = rec.Checkpoint.Manifest.CheckpointID
	}

	return exportResult{
		Name:         opts.Name,
		State:        rec.State,
		Branch:       branch,
		Source:       source,
		AutoToreDown: autoToreDown,
		CheckpointID: checkpointID,
		ExportedAt:   now,
	}, nil
}

func (o *orchestrator) teardown(opts teardownOptions) (teardownResult, error) {
	store, err := o.loadStore()
	if err != nil {
		return teardownResult{}, err
	}
	rec, ok := store.Sandboxes[opts.Name]
	if !ok {
		return teardownResult{}, userErrReason(fmt.Sprintf("sandbox %q not found", opts.Name), "sandbox_not_found")
	}

	if rec.State == stateDestroyed {
		checkpointID := ""
		if rec.Checkpoint != nil {
			checkpointID = rec.Checkpoint.Manifest.CheckpointID
		}
		return teardownResult{Name: opts.Name, State: rec.State, CheckpointID: checkpointID, Destroyed: true}, nil
	}

	if (rec.State == stateRunning || rec.State == stateProvisioning || rec.State == stateNew) && !opts.Force {
		return teardownResult{}, userErrReason("teardown of active sandbox requires --force", "teardown_requires_force")
	}

	now := o.now().UTC().Format(time.RFC3339)
	rec.OperationLock = o.newOperationLock("teardown")
	rec.UpdatedAt = now
	rec.LastActivityAt = now
	rec.State = stateSaving

	if rec.Checkpoint == nil {
		manifest := newCheckpointManifest(rec.Identity.Repo, rec.Identity.ImageProfile, now)
		rec.Checkpoint = &checkpointPointer{Verified: true, Manifest: manifest}
	}

	if !validCheckpoint(rec.Checkpoint.Manifest) {
		rec.State = stateError
		rec.ErrorReason = "checkpoint_checksum_mismatch"
		rec.RuntimeProbe = runtimeProbe{Status: "unavailable", CheckedAt: now}
		if saveErr := o.saveStore(store); saveErr != nil {
			return teardownResult{}, saveErr
		}
		return teardownResult{}, sysErrReason("checkpoint checksum mismatch; teardown halted", "checkpoint_checksum_mismatch", false)
	}

	rec.State = stateTearingDown
	rec.State = stateDestroyed
	rec.RuntimeProbe = runtimeProbe{Status: "stopped", CheckedAt: now}
	rec.TeardownBlockers = nil

	if err := o.saveStore(store); err != nil {
		return teardownResult{}, err
	}

	return teardownResult{
		Name:         opts.Name,
		State:        rec.State,
		CheckpointID: rec.Checkpoint.Manifest.CheckpointID,
		Destroyed:    true,
	}, nil
}

func (o *orchestrator) open(name string) (string, error) {
	store, err := o.loadStore()
	if err != nil {
		return "", err
	}
	rec, ok := store.Sandboxes[name]
	if !ok {
		return "", userErrReason(fmt.Sprintf("sandbox %q not found", name), "sandbox_not_found")
	}
	if rec.State != stateRunning {
		return "", userErrReason(
			fmt.Sprintf("sandbox %q is %q; launch or resume before opening", name, rec.State),
			"sandbox_not_running",
		)
	}
	return rec.WebURL, nil
}

func (o *orchestrator) loadStore() (*orchestratorStore, error) {
	data, err := os.ReadFile(o.statePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &orchestratorStore{Sandboxes: map[string]*sandboxRecord{}}, nil
		}
		return nil, fmt.Errorf("read orchestrator state: %w", err)
	}

	store := &orchestratorStore{}
	if err := json.Unmarshal(data, store); err != nil {
		return nil, fmt.Errorf("parse orchestrator state: %w", err)
	}
	if store.Sandboxes == nil {
		store.Sandboxes = map[string]*sandboxRecord{}
	}
	return store, nil
}

func (o *orchestrator) saveStore(store *orchestratorStore) error {
	if err := os.MkdirAll(filepath.Dir(o.statePath), 0755); err != nil {
		return fmt.Errorf("create orchestrator state directory: %w", err)
	}

	payload, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("encode orchestrator state: %w", err)
	}

	tmpPath := o.statePath + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0644); err != nil {
		return fmt.Errorf("write orchestrator state temp file: %w", err)
	}
	if err := os.Rename(tmpPath, o.statePath); err != nil {
		return fmt.Errorf("replace orchestrator state file: %w", err)
	}
	return nil
}

func (o *orchestrator) newOperationLock(operationType string) operationLock {
	now := o.now().UTC()
	idInput := fmt.Sprintf("%s|%d", operationType, now.UnixNano())
	sum := sha256.Sum256([]byte(idInput))
	return operationLock{
		ID:        hex.EncodeToString(sum[:8]),
		Type:      operationType,
		StartedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(2 * time.Minute).Format(time.RFC3339),
	}
}

func newCheckpointManifest(repo, profile, createdAt string) checkpointManifest {
	idInput := fmt.Sprintf("%s|%s|%s", repo, profile, createdAt)
	idSum := sha256.Sum256([]byte(idInput))
	checkpointID := hex.EncodeToString(idSum[:8])

	manifest := checkpointManifest{
		SchemaVersion: "v1",
		CreatedAt:     createdAt,
		RepoRef:       repo,
		ImageProfile:  profile,
		CheckpointID:  checkpointID,
	}
	manifest.ContentHash = checkpointHash(manifest)
	return manifest
}

func checkpointHash(m checkpointManifest) string {
	input := strings.Join([]string{m.SchemaVersion, m.CreatedAt, m.RepoRef, m.ImageProfile, m.CheckpointID}, "|")
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}

func validCheckpoint(m checkpointManifest) bool {
	if m.SchemaVersion == "" || m.CheckpointID == "" || m.ContentHash == "" {
		return false
	}
	return checkpointHash(m) == m.ContentHash
}

func appendUnique(in []string, item string) []string {
	for _, existing := range in {
		if existing == item {
			return in
		}
	}
	return append(in, item)
}
