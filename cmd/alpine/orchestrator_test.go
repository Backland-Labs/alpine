package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestOrchestrator(t *testing.T) *orchestrator {
	t.Helper()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	if err := os.Setenv("ALPINE_STATE_PATH", statePath); err != nil {
		t.Fatalf("set env: %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("ALPINE_STATE_PATH") })

	cfg := &Config{
		Sandbox: SandboxConfig{
			WebBaseURL:    "https://sandbox.example.com",
			AutoTeardown:  true,
			ImageProfile:  "default",
			ImageProfiles: map[string]string{"default": "class-a"},
		},
		GitHub:     GitHubConfig{BranchPrefix: "alpine", RequireAuth: false},
		Durability: DurabilityConfig{Bucket: "b", CheckpointPrefix: "p"},
		BaseImage:  "ubuntu:24.04",
	}
	orch := newOrchestrator(cfg)
	tick := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)
	orch.now = func() time.Time {
		tick = tick.Add(1 * time.Second)
		return tick
	}
	return orch
}

func TestOrchestratorLaunchAndIdentityRules(t *testing.T) {
	orch := newTestOrchestrator(t)

	result, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"})
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	if result.State != stateRunning || result.WebURL == "" {
		t.Fatalf("unexpected launch result: %+v", result)
	}

	second, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"})
	if err != nil {
		t.Fatalf("relaunch same identity: %v", err)
	}
	if !second.Reused {
		t.Fatal("expected reused=true")
	}

	_, err = orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/other.git", ImageProfile: "default"})
	if err == nil {
		t.Fatal("expected identity mismatch error")
	}

	forced, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/other.git", ImageProfile: "default", ForceRecreate: true})
	if err != nil {
		t.Fatalf("force recreate: %v", err)
	}
	if forced.Repo != "https://github.com/acme/other.git" {
		t.Fatalf("repo = %q", forced.Repo)
	}
}

func TestOrchestratorTeardownAndResume(t *testing.T) {
	orch := newTestOrchestrator(t)

	if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
		t.Fatalf("launch: %v", err)
	}

	if _, err := orch.teardown(teardownOptions{Name: "alpha", Force: true}); err != nil {
		t.Fatalf("teardown: %v", err)
	}

	rec, ok, _, err := orch.status("alpha")
	if err != nil || !ok {
		t.Fatalf("status: %v", err)
	}
	if rec.State != stateDestroyed || rec.Checkpoint == nil || !rec.Checkpoint.Verified {
		t.Fatalf("unexpected post-teardown state: %+v", rec)
	}

	resume, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"})
	if err != nil {
		t.Fatalf("resume launch: %v", err)
	}
	if !resume.Resumed {
		t.Fatal("expected resumed=true")
	}
}

func TestOrchestratorCheckpointMismatchTransitionsToError(t *testing.T) {
	orch := newTestOrchestrator(t)

	if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
		t.Fatalf("launch: %v", err)
	}
	if _, err := orch.teardown(teardownOptions{Name: "alpha", Force: true}); err != nil {
		t.Fatalf("teardown: %v", err)
	}

	store, err := orch.loadStore()
	if err != nil {
		t.Fatalf("loadStore: %v", err)
	}
	store.Sandboxes["alpha"].Checkpoint.Manifest.ContentHash = "bad-hash"
	if err := orch.saveStore(store); err != nil {
		t.Fatalf("saveStore: %v", err)
	}

	_, err = orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"})
	if err == nil {
		t.Fatal("expected checkpoint checksum mismatch")
	}

	rec, ok, _, err := orch.status("alpha")
	if err != nil || !ok {
		t.Fatalf("status: %v", err)
	}
	if rec.State != stateError {
		t.Fatalf("state = %s", rec.State)
	}
}

func TestOrchestratorExportContracts(t *testing.T) {
	orch := newTestOrchestrator(t)
	ctx := context.Background()

	if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
		t.Fatalf("launch: %v", err)
	}

	_, err := orch.export(ctx, exportOptions{Name: "alpha"})
	if err == nil {
		t.Fatal("expected checkpoint_missing before teardown")
	}

	if _, err := orch.teardown(teardownOptions{Name: "alpha", Force: true}); err != nil {
		t.Fatalf("teardown: %v", err)
	}

	res, err := orch.export(ctx, exportOptions{Name: "alpha"})
	if err != nil {
		t.Fatalf("export destroyed with checkpoint: %v", err)
	}
	if res.Branch == "" || res.Source != "checkpoint" {
		t.Fatalf("unexpected export result: %+v", res)
	}

	store, err := orch.loadStore()
	if err != nil {
		t.Fatalf("loadStore: %v", err)
	}
	store.Sandboxes["alpha"].State = stateSaving
	if err := orch.saveStore(store); err != nil {
		t.Fatalf("saveStore: %v", err)
	}

	_, err = orch.export(ctx, exportOptions{Name: "alpha"})
	if err == nil {
		t.Fatal("expected retryable error from saving")
	}
	ee, ok := err.(*exitError)
	if !ok || ee.code != 2 || ee.reasonCode != "export_retryable_state" || !ee.retryable {
		t.Fatalf("unexpected error contract: %#v", err)
	}
}

func TestOrchestratorOpenAndStaleStatus(t *testing.T) {
	orch := newTestOrchestrator(t)

	if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
		t.Fatalf("launch: %v", err)
	}

	url, err := orch.open("alpha")
	if err != nil || url == "" {
		t.Fatalf("open running: %v, %s", err, url)
	}

	if _, err := orch.teardown(teardownOptions{Name: "alpha", Force: true}); err != nil {
		t.Fatalf("teardown: %v", err)
	}
	if _, err := orch.open("alpha"); err == nil {
		t.Fatal("expected non-running open failure")
	}

	store, err := orch.loadStore()
	if err != nil {
		t.Fatalf("loadStore: %v", err)
	}
	store.Sandboxes["alpha"].RuntimeProbe.CheckedAt = time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC).Format(time.RFC3339)
	if err := orch.saveStore(store); err != nil {
		t.Fatalf("saveStore: %v", err)
	}

	_, ok, stale, err := orch.status("alpha")
	if err != nil || !ok {
		t.Fatalf("status: %v", err)
	}
	if !stale {
		t.Fatal("expected stale status probe")
	}
}

func TestCheckpointHelpers(t *testing.T) {
	m := newCheckpointManifest("repo", "default", "2026-02-20T00:00:00Z")
	if !validCheckpoint(m) {
		t.Fatal("new checkpoint should be valid")
	}
	m.ContentHash = "wrong"
	if validCheckpoint(m) {
		t.Fatal("tampered checkpoint should be invalid")
	}
	if validCheckpoint(checkpointManifest{}) {
		t.Fatal("empty checkpoint should be invalid")
	}

	in := []string{"a", "b"}
	out := appendUnique(in, "b")
	if len(out) != 2 {
		t.Fatal("appendUnique duplicated value")
	}
	out = appendUnique(out, "c")
	if len(out) != 3 {
		t.Fatal("appendUnique should append missing value")
	}
}

func TestOrchestratorStoreAndPathBranches(t *testing.T) {
	if err := os.Setenv("ALPINE_STATE_PATH", "/tmp/custom-state.json"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	if got := orchestratorStatePath(); got != "/tmp/custom-state.json" {
		t.Fatalf("state path = %q", got)
	}
	_ = os.Unsetenv("ALPINE_STATE_PATH")
	if got := orchestratorStatePath(); got == "" {
		t.Fatal("expected default state path")
	}
	origHome := os.Getenv("HOME")
	if err := os.Unsetenv("HOME"); err != nil {
		t.Fatalf("unset HOME: %v", err)
	}
	if got := orchestratorStatePath(); got != ".alpine-orchestrator-state.json" {
		t.Fatalf("expected fallback path on HOME error, got %q", got)
	}
	if origHome != "" {
		_ = os.Setenv("HOME", origHome)
	}

	orch := newTestOrchestrator(t)
	badStore := &orchestratorStore{}
	if err := orch.saveStore(badStore); err != nil {
		t.Fatalf("saveStore empty map should initialize: %v", err)
	}

	store, err := orch.loadStore()
	if err != nil {
		t.Fatalf("loadStore: %v", err)
	}
	if store.Sandboxes == nil {
		t.Fatal("sandboxes map should be initialized")
	}

	if err := os.WriteFile(orch.statePath, []byte("not-json"), 0644); err != nil {
		t.Fatalf("write bad state: %v", err)
	}
	if _, err := orch.loadStore(); err == nil {
		t.Fatal("expected parse error")
	}

	dirPath := filepath.Join(t.TempDir(), "state-dir")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	orch.statePath = dirPath
	if _, err := orch.loadStore(); err == nil {
		t.Fatal("expected read error when state path is directory")
	}

	writeErrDir := filepath.Join(t.TempDir(), "readonly")
	if err := os.MkdirAll(writeErrDir, 0755); err != nil {
		t.Fatalf("mkdir readonly dir: %v", err)
	}
	if err := os.Chmod(writeErrDir, 0555); err != nil {
		t.Fatalf("chmod readonly dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(writeErrDir, 0755) })
	orch.statePath = filepath.Join(writeErrDir, "state.json")
	if err := orch.saveStore(&orchestratorStore{Sandboxes: map[string]*sandboxRecord{}}); err == nil {
		t.Fatal("expected write temp error for readonly directory")
	}
}

func TestOrchestratorAdditionalBranches(t *testing.T) {
	orch := newTestOrchestrator(t)

	if _, ok, _, err := orch.status("missing"); err != nil || ok {
		t.Fatalf("status missing should be false,nil err got ok=%t err=%v", ok, err)
	}
	if _, err := orch.open("missing"); err == nil {
		t.Fatal("open missing should fail")
	}
	orch.statePath = filepath.Join(t.TempDir(), "broken-state.json")
	if err := os.WriteFile(orch.statePath, []byte("broken"), 0644); err != nil {
		t.Fatalf("write broken state: %v", err)
	}
	if _, err := orch.open("anything"); err == nil {
		t.Fatal("open should fail when state cannot be parsed")
	}
	orch = newTestOrchestrator(t)
	if _, err := orch.teardown(teardownOptions{Name: "missing", Force: true}); err == nil {
		t.Fatal("teardown missing should fail")
	}

	if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default", Task: "hello"}); err != nil {
		t.Fatalf("launch with task: %v", err)
	}
	if _, err := orch.export(context.Background(), exportOptions{Name: "alpha", FromLive: true, Branch: "custom/alpha"}); err != nil {
		t.Fatalf("export from live: %v", err)
	}

	store, err := orch.loadStore()
	if err != nil {
		t.Fatalf("loadStore: %v", err)
	}
	store.Sandboxes["alpha"].State = stateCompleted
	store.Sandboxes["alpha"].Checkpoint = nil
	if err := orch.saveStore(store); err != nil {
		t.Fatalf("saveStore: %v", err)
	}
	if _, err := orch.export(context.Background(), exportOptions{Name: "alpha", FromLive: true}); err != nil {
		t.Fatalf("from-live completed export should succeed: %v", err)
	}

	store, err = orch.loadStore()
	if err != nil {
		t.Fatalf("loadStore: %v", err)
	}
	store.Sandboxes["alpha"].State = stateNew
	if err := orch.saveStore(store); err != nil {
		t.Fatalf("saveStore: %v", err)
	}
	if _, err := orch.export(context.Background(), exportOptions{Name: "alpha"}); err == nil {
		t.Fatal("expected export_state_blocked")
	}

	store, err = orch.loadStore()
	if err != nil {
		t.Fatalf("loadStore: %v", err)
	}
	store.Sandboxes["alpha"].State = stateCompleted
	store.Sandboxes["alpha"].Checkpoint = nil
	if err := orch.saveStore(store); err != nil {
		t.Fatalf("saveStore: %v", err)
	}
	if _, err := orch.export(context.Background(), exportOptions{Name: "alpha"}); err == nil {
		t.Fatal("expected checkpoint_missing")
	}

	store, err = orch.loadStore()
	if err != nil {
		t.Fatalf("loadStore: %v", err)
	}
	store.Sandboxes["alpha"].Checkpoint = &checkpointPointer{Verified: false, Manifest: newCheckpointManifest("https://github.com/acme/repo.git", "default", time.Now().UTC().Format(time.RFC3339))}
	if err := orch.saveStore(store); err != nil {
		t.Fatalf("saveStore: %v", err)
	}
	if _, err := orch.export(context.Background(), exportOptions{Name: "alpha"}); err == nil {
		t.Fatal("expected checkpoint_missing with unverified checkpoint")
	}

	if _, err := orch.teardown(teardownOptions{Name: "alpha", Force: true}); err != nil {
		t.Fatalf("teardown complete: %v", err)
	}
	if _, err := orch.teardown(teardownOptions{Name: "alpha", Force: true}); err != nil {
		t.Fatalf("idempotent teardown should succeed: %v", err)
	}

	if _, err := orch.launch(launchOptions{Name: "beta", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
		t.Fatalf("launch beta: %v", err)
	}
	if _, err := orch.teardown(teardownOptions{Name: "beta", Force: false}); err == nil {
		t.Fatal("expected force requirement for running sandbox")
	}
	store, err = orch.loadStore()
	if err != nil {
		t.Fatalf("loadStore: %v", err)
	}
	store.Sandboxes["beta"].Checkpoint = &checkpointPointer{Verified: true, Manifest: checkpointManifest{SchemaVersion: "v1", CheckpointID: "id", ContentHash: "bad"}}
	if err := orch.saveStore(store); err != nil {
		t.Fatalf("saveStore: %v", err)
	}
	if _, err := orch.teardown(teardownOptions{Name: "beta", Force: true}); err == nil {
		t.Fatal("expected checkpoint mismatch on teardown")
	}

	store, err = orch.loadStore()
	if err != nil {
		t.Fatalf("loadStore: %v", err)
	}
	store.Sandboxes["beta"].State = stateSaving
	if err := orch.saveStore(store); err != nil {
		t.Fatalf("saveStore: %v", err)
	}
	if _, err := orch.launch(launchOptions{Name: "beta", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err == nil {
		t.Fatal("expected invalid transition launching from saving state")
	}

	// saveStore error branch: parent path points to a file.
	badDir := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(badDir, []byte("x"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	orch.statePath = filepath.Join(badDir, "state.json")
	if err := orch.saveStore(&orchestratorStore{Sandboxes: map[string]*sandboxRecord{}}); err == nil {
		t.Fatal("expected saveStore error")
	}

	orch.statePath = filepath.Join(t.TempDir(), "rename-target")
	if err := os.MkdirAll(orch.statePath, 0755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := orch.saveStore(&orchestratorStore{Sandboxes: map[string]*sandboxRecord{}}); err == nil {
		t.Fatal("expected rename error when target is directory")
	}

	orch.statePath = filepath.Join(t.TempDir(), "missing", "state.json")
	if _, _, err := orch.sandboxIdentity("alpha"); err != nil {
		t.Fatalf("sandboxIdentity on missing state should not fail: %v", err)
	}
	_ = os.MkdirAll(filepath.Dir(orch.statePath), 0755)
	_ = os.WriteFile(orch.statePath, []byte("bad"), 0644)
	if _, _, err := orch.sandboxIdentity("alpha"); err == nil {
		t.Fatal("expected sandboxIdentity parse failure")
	}
	if _, err := orch.list(); err == nil {
		t.Fatal("expected list parse failure")
	}

	launchErrDir := filepath.Join(t.TempDir(), "launch-readonly")
	if err := os.MkdirAll(launchErrDir, 0755); err != nil {
		t.Fatalf("mkdir launch readonly dir: %v", err)
	}
	if err := os.Chmod(launchErrDir, 0555); err != nil {
		t.Fatalf("chmod launch readonly dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(launchErrDir, 0755) })
	orch.statePath = filepath.Join(launchErrDir, "state.json")
	if _, err := orch.launch(launchOptions{Name: "gamma", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err == nil {
		t.Fatal("expected launch saveStore error on readonly directory")
	}
}
