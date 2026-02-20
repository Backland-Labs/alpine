package main

// Tests mutate package-level variables and must NOT use t.Parallel().

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Pure function tests (table-driven)
// ---------------------------------------------------------------------------

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "single char", input: "a", wantErr: false},
		{name: "two chars", input: "ab", wantErr: false},
		{name: "simple name", input: "my-feature", wantErr: false},
		{name: "alphanumeric", input: "abc123", wantErr: false},
		{name: "hyphen in middle", input: "a-b", wantErr: false},
		{name: "double hyphens valid", input: "ab--cd", wantErr: false},
		{name: "50 char name", input: strings.Repeat("a", 50), wantErr: false},
		{name: "numbers only", input: "123", wantErr: false},

		{name: "empty", input: "", wantErr: true},
		{name: "uppercase", input: "ABC", wantErr: true},
		{name: "mixed case", input: "MyFeature", wantErr: true},
		{name: "leading hyphen", input: "-leading", wantErr: true},
		{name: "trailing hyphen", input: "trailing-", wantErr: true},
		{name: "dots", input: "a..b", wantErr: true},
		{name: "has spaces", input: "has spaces", wantErr: true},
		{name: "51 chars", input: strings.Repeat("a", 51), wantErr: true},
		{name: "special chars", input: "a@b", wantErr: true},
		{name: "underscore", input: "a_b", wantErr: true},
		{name: "slash", input: "a/b", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateName(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("validateName(%q) = nil, want error", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateName(%q) = %v, want nil", tt.input, err)
			}
		})
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "already valid", input: "my-feature", want: "my-feature"},
		{name: "slash separated", input: "refactor/make-agent-ready", want: "refactor-make-agent-ready"},
		{name: "uppercase", input: "MyFeature", want: "myfeature"},
		{name: "mixed case with slash", input: "Feature/Add-Login", want: "feature-add-login"},
		{name: "underscores", input: "my_feature_branch", want: "my-feature-branch"},
		{name: "dots", input: "v1.2.3", want: "v1-2-3"},
		{name: "multiple slashes", input: "a/b/c", want: "a-b-c"},
		{name: "leading slash", input: "/leading", want: "leading"},
		{name: "trailing slash", input: "trailing/", want: "trailing"},
		{name: "consecutive separators", input: "a//b", want: "a-b"},
		{name: "special chars stripped", input: "a@b!c", want: "abc"},
		{name: "empty after sanitize", input: "@!!", want: ""},
		{name: "truncate to 50", input: strings.Repeat("a", 60), want: strings.Repeat("a", 50)},
		{name: "single char", input: "a", want: "a"},
		{name: "numbers", input: "123", want: "123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUserErr(t *testing.T) {
	err := userErr("something went wrong")
	var ee *exitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *exitError, got %T", err)
	}
	if ee.code != 1 {
		t.Fatalf("code = %d, want 1", ee.code)
	}
	if ee.msg != "something went wrong" {
		t.Fatalf("msg = %q, want %q", ee.msg, "something went wrong")
	}
}

func TestSysErr(t *testing.T) {
	err := sysErr("system failure")
	var ee *exitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *exitError, got %T", err)
	}
	if ee.code != 2 {
		t.Fatalf("code = %d, want 2", ee.code)
	}
	if ee.msg != "system failure" {
		t.Fatalf("msg = %q, want %q", ee.msg, "system failure")
	}
}

// ---------------------------------------------------------------------------
// runCreate workflow tests -- helpers
// ---------------------------------------------------------------------------

// setupCreateTest is a common setup for runCreate tests. It resets flags,
// changes to a temp dir (for loadConfig), sets detach+json mode, and
// configures the environment so git auth and HOME pass silently.
// Returns the temp dir path (which is also the CWD).
func setupCreateTest(t *testing.T) string {
	t.Helper()
	resetFlags(t)
	jsonOutput = true
	detach = true

	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir tempDir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	t.Setenv("SSH_AUTH_SOCK", "/tmp/fake.sock")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("HOME", tempDir) // no .claude dir -> skip copy step

	return tempDir
}

// runCreateCmd executes the create command and returns the error.
// It creates a minimal Cobra command, sets args, and calls Execute.
func runCreateCmd(t *testing.T, ctx context.Context, name string) error {
	t.Helper()
	cmd := newTestCreateCmd(ctx)
	cmd.SetArgs([]string{name})
	return cmd.Execute()
}

// assertExitCode checks that the error is an *exitError with the expected code.
func assertExitCode(t *testing.T, err error, wantCode int) {
	t.Helper()
	if wantCode == 0 {
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("expected error with code %d, got nil", wantCode)
	}
	var ee *exitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *exitError, got %T: %v", err, err)
	}
	if ee.code != wantCode {
		t.Fatalf("exit code = %d, want %d (msg: %s)", ee.code, wantCode, ee.msg)
	}
}

// assertContains checks that the error message contains the expected substring.
func assertContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("error %q does not contain %q", err.Error(), substr)
	}
}

// ---------------------------------------------------------------------------
// Step 1: Invalid name
// ---------------------------------------------------------------------------

func TestRunCreate_Step1_InvalidName(t *testing.T) {
	setupCreateTest(t)
	// No run() calls needed -- sanitizeName produces "" which fails before any shell-out.
	mockRun(t, []cmdResult{})
	err := runCreateCmd(t, context.Background(), "@!!")
	assertExitCode(t, err, 1)
	assertContains(t, err, "invalid name")
}

// ---------------------------------------------------------------------------
// Step 2: Docker not running
// ---------------------------------------------------------------------------

func TestRunCreate_Step2_DockerNotRunning(t *testing.T) {
	setupCreateTest(t)
	responses := []cmdResult{
		errResult("Cannot connect to Docker daemon"), // docker info (fails)
	}
	if runtime.GOOS == "darwin" {
		// On darwin, dockerHealthCheck tries to start Docker Desktop.
		responses = append(responses, errResult("app not found")) // open -a Docker (fails)
	}
	mockRun(t, responses)
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 2)
}

// ---------------------------------------------------------------------------
// Step 3: Not in a git repo
// ---------------------------------------------------------------------------

func TestRunCreate_Step3_NotGitRepo(t *testing.T) {
	setupCreateTest(t)
	mockRun(t, []cmdResult{
		{stdout: ""},                      // 0: docker info
		errResult("not a git repository"), // 1: git rev-parse --show-toplevel
	})
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 1)
	assertContains(t, err, "not inside a git repository")
}

// ---------------------------------------------------------------------------
// Step 4: Duplicate environment
// ---------------------------------------------------------------------------

func TestRunCreate_Step4_DuplicateEnv(t *testing.T) {
	setupCreateTest(t)
	mockRun(t, []cmdResult{
		{stdout: ""},                         // 0: docker info
		{stdout: "/tmp/fakegitroot"},         // 1: git rev-parse --show-toplevel
		{stdout: `[{"Name":"alpine-test"}]`}, // 2: docker compose ls (non-empty)
	})
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 1)
	assertContains(t, err, "already exists")
}

// ---------------------------------------------------------------------------
// Step 5: --from branch (skips gitGetCurrentBranch)
// ---------------------------------------------------------------------------

func TestRunCreate_Step5_FromBranch(t *testing.T) {
	tempDir := setupCreateTest(t)
	fromBranch = "develop"

	// When fromBranch is set, gitGetCurrentBranch (index 3 in happy path)
	// is skipped. Build responses manually without that call.
	container := "alpine-test-dev-1"
	responses := []cmdResult{
		{stdout: ""},      // 0: docker info
		{stdout: tempDir}, // 1: git rev-parse (use tempDir)
		{stdout: "[]"},    // 2: docker compose ls
		// NO gitGetCurrentBranch call (skipped when fromBranch is set)
		{stdout: ""}, // 3: docker image inspect
		{stdout: ""}, // 4: docker compose up
		{stdout: fmt.Sprintf(`{"Name":"%s","Service":"dev"}`, container)}, // 5: docker compose ps
		{stdout: "git@github.com:user/repo.git"},                          // 6: git remote get-url origin
		{stdout: ""},                                                      // 7: docker exec git clone
		{stdout: ""},                                                      // 8: docker exec git checkout -b
		{stdout: "Test User"},                                             // 9: git config user.name
		{stdout: "test@example.com"},                                      // 10: git config user.email
		{stdout: ""},                                                      // 11: docker exec git config user.name
		{stdout: ""},                                                      // 12: docker exec git config user.email
		{stdout: ""},                                                      // 13: docker exec sh -c (claude setup)
		{stdout: "set"},                                                   // 14: docker exec sh -c (token check)
	}
	mockRun(t, responses)

	out := captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, out)
	}
	if result["branch"] != "feature/test" {
		t.Fatalf("branch = %v, want %q", result["branch"], "feature/test")
	}
}

// ---------------------------------------------------------------------------
// Step 5: --from with leading dash (flag injection prevention)
// ---------------------------------------------------------------------------

func TestRunCreate_Step5_FromLeadingDash(t *testing.T) {
	setupCreateTest(t)
	fromBranch = "-malicious"

	mockRun(t, []cmdResult{
		{stdout: ""},                 // 0: docker info
		{stdout: "/tmp/fakegitroot"}, // 1: git rev-parse --show-toplevel
		{stdout: "[]"},               // 2: docker compose ls
		// fromBranch is set, so gitGetCurrentBranch is skipped.
		// The leading dash check fires immediately.
	})
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 1)
	assertContains(t, err, "must not start with '-'")
}

// ---------------------------------------------------------------------------
// Step 5: Detached HEAD
// ---------------------------------------------------------------------------

func TestRunCreate_Step5_DetachedHead(t *testing.T) {
	setupCreateTest(t)
	mockRun(t, []cmdResult{
		{stdout: ""},                 // 0: docker info
		{stdout: "/tmp/fakegitroot"}, // 1: git rev-parse --show-toplevel
		{stdout: "[]"},               // 2: docker compose ls
		{stdout: "HEAD"},             // 3: git rev-parse --abbrev-ref HEAD
	})
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 1)
	assertContains(t, err, "HEAD is detached")
}

// ---------------------------------------------------------------------------
// Step 5: git rev-parse fails (error, not HEAD)
// ---------------------------------------------------------------------------

func TestRunCreate_Step5_GitRevParseFails(t *testing.T) {
	setupCreateTest(t)
	mockRun(t, []cmdResult{
		{stdout: ""},                 // 0: docker info
		{stdout: "/tmp/fakegitroot"}, // 1: git rev-parse --show-toplevel
		{stdout: "[]"},               // 2: docker compose ls
		errResult("not a git repo"),  // 3: git rev-parse --abbrev-ref HEAD fails
	})
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 1)
	assertContains(t, err, "HEAD is detached")
}

// ---------------------------------------------------------------------------
// Step 6: No git auth
// ---------------------------------------------------------------------------

func TestRunCreate_Step6_NoGitAuth(t *testing.T) {
	setupCreateTest(t)
	t.Setenv("SSH_AUTH_SOCK", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")

	mockRun(t, []cmdResult{
		{stdout: ""},                 // 0: docker info
		{stdout: "/tmp/fakegitroot"}, // 1: git rev-parse --show-toplevel
		{stdout: "[]"},               // 2: docker compose ls
		{stdout: "main"},             // 3: git rev-parse --abbrev-ref HEAD
	})
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 1)
	assertContains(t, err, "no git auth")
}

// ---------------------------------------------------------------------------
// Step 6: Auth via GITHUB_TOKEN
// ---------------------------------------------------------------------------

func TestRunCreate_Step6_GitHubTokenAuth(t *testing.T) {
	setupCreateTest(t)
	t.Setenv("SSH_AUTH_SOCK", "")
	t.Setenv("GITHUB_TOKEN", "ghp_test123")

	mockRun(t, happyCreateResponses("test"))
	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})
}

// ---------------------------------------------------------------------------
// Step 6: Auth via GH_TOKEN
// ---------------------------------------------------------------------------

func TestRunCreate_Step6_GHTokenAuth(t *testing.T) {
	setupCreateTest(t)
	t.Setenv("SSH_AUTH_SOCK", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "gho_test456")

	mockRun(t, happyCreateResponses("test"))
	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})
}

// ---------------------------------------------------------------------------
// Step 7: Image exists (skips build)
// ---------------------------------------------------------------------------

func TestRunCreate_Step7_ImageExistsSkipsBuild(t *testing.T) {
	setupCreateTest(t)
	responses := happyCreateResponses("test")
	calls := mockRunRecording(t, responses)

	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	for _, c := range *calls {
		if c.name == "docker" && len(c.args) > 0 && c.args[0] == "build" {
			t.Fatal("docker build should not be called when image exists")
		}
	}
}

// ---------------------------------------------------------------------------
// Step 7: Image build needed + build succeeds
// ---------------------------------------------------------------------------

func TestRunCreate_Step7_BuildSucceeds(t *testing.T) {
	setupCreateTest(t)
	responses := happyCreateResponses("test")
	responses[4] = errResult("no such image")
	buildOK := cmdResult{stdout: "Successfully built abc123"}
	newResp := make([]cmdResult, 0, len(responses)+1)
	newResp = append(newResp, responses[:5]...)
	newResp = append(newResp, buildOK)
	newResp = append(newResp, responses[5:]...)

	calls := mockRunRecording(t, newResp)

	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	foundBuild := false
	for _, c := range *calls {
		if c.name == "docker" && len(c.args) > 0 && c.args[0] == "build" {
			foundBuild = true
			break
		}
	}
	if !foundBuild {
		t.Fatal("expected docker build call when image doesn't exist")
	}
}

// ---------------------------------------------------------------------------
// Step 7: Image build needed + build fails
// ---------------------------------------------------------------------------

func TestRunCreate_Step7_BuildFails(t *testing.T) {
	setupCreateTest(t)
	// Build fails at step 5 (docker build), so only 6 responses consumed.
	mockRun(t, []cmdResult{
		{stdout: ""},                            // 0: docker info
		{stdout: "/tmp/fakegitroot"},            // 1: git rev-parse --show-toplevel
		{stdout: "[]"},                          // 2: docker compose ls
		{stdout: "main"},                        // 3: git rev-parse --abbrev-ref HEAD
		errResult("no such image"),              // 4: docker image inspect (not found)
		errResult("build error: no space left"), // 5: docker build (fails)
	})
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 2)
	assertContains(t, err, "Docker image build failed")
}

// ---------------------------------------------------------------------------
// Step 8: Compose up fails
// ---------------------------------------------------------------------------

func TestRunCreate_Step8_ComposeUpFails(t *testing.T) {
	setupCreateTest(t)
	responses := happyCreateResponses("test")
	responses[5] = errResult("network error")
	mockRun(t, responses[:6])
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 2)
	assertContains(t, err, "failed to start environment")
}

// ---------------------------------------------------------------------------
// Step 9: Discover container fails (with rollback)
// ---------------------------------------------------------------------------

func TestRunCreate_Step9_DiscoverContainerFails(t *testing.T) {
	setupCreateTest(t)
	responses := happyCreateResponses("test")
	responses[6] = errResult("container not found")
	rollbackResponses := make([]cmdResult, 7, 8)
	copy(rollbackResponses, responses[:7])
	rollbackResponses = append(rollbackResponses, cmdResult{stdout: ""})
	calls := mockRunRecording(t, rollbackResponses)

	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 2)
	assertContains(t, err, "failed to discover dev container")

	foundDown := false
	for _, c := range *calls {
		if c.name == "docker" && len(c.args) >= 2 {
			for _, a := range c.args {
				if a == "down" {
					foundDown = true
					break
				}
			}
		}
	}
	if !foundDown {
		t.Fatal("expected composeDown rollback call, but none found")
	}
}

// ---------------------------------------------------------------------------
// Step 10: Remote URL error
// ---------------------------------------------------------------------------

func TestRunCreate_Step10_RemoteURLError(t *testing.T) {
	setupCreateTest(t)
	responses := happyCreateResponses("test")
	responses[7] = errResult("no such remote")
	rollbackResponses := make([]cmdResult, 8, 9)
	copy(rollbackResponses, responses[:8])
	rollbackResponses = append(rollbackResponses, cmdResult{stdout: ""})
	mockRun(t, rollbackResponses)
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 1)
	assertContains(t, err, "no git remote")
}

// ---------------------------------------------------------------------------
// Step 11: Git clone error
// ---------------------------------------------------------------------------

func TestRunCreate_Step11_GitCloneError(t *testing.T) {
	setupCreateTest(t)
	responses := happyCreateResponses("test")
	responses[8] = errResult("clone failed: auth error")
	rollbackResponses := make([]cmdResult, 9, 10)
	copy(rollbackResponses, responses[:9])
	rollbackResponses = append(rollbackResponses, cmdResult{stdout: ""})
	mockRun(t, rollbackResponses)
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 2)
	assertContains(t, err, "failed to clone repository")
}

// ---------------------------------------------------------------------------
// Step 12: Git create branch error
// ---------------------------------------------------------------------------

func TestRunCreate_Step12_GitCreateBranchError(t *testing.T) {
	setupCreateTest(t)
	responses := happyCreateResponses("test")
	responses[9] = errResult("branch already exists")
	rollbackResponses := make([]cmdResult, 10, 11)
	copy(rollbackResponses, responses[:10])
	rollbackResponses = append(rollbackResponses, cmdResult{stdout: ""})
	mockRun(t, rollbackResponses)
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 2)
	assertContains(t, err, "failed to create branch")
}

// ---------------------------------------------------------------------------
// Step 13: Git configure user error
// ---------------------------------------------------------------------------

func TestRunCreate_Step13_GitConfigureUserError(t *testing.T) {
	setupCreateTest(t)
	responses := happyCreateResponses("test")
	responses[12] = errResult("permission denied")
	rollbackResponses := make([]cmdResult, 13, 14)
	copy(rollbackResponses, responses[:13])
	rollbackResponses = append(rollbackResponses, cmdResult{stdout: ""})
	mockRun(t, rollbackResponses)
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 2)
	assertContains(t, err, "failed to configure git user")
}

// ---------------------------------------------------------------------------
// Step 13: Host git config missing uses fallback values
// ---------------------------------------------------------------------------

func TestRunCreate_Step13_HostConfigFallback(t *testing.T) {
	setupCreateTest(t)
	responses := happyCreateResponses("test")
	responses[10] = errResult("not set")
	responses[11] = errResult("not set")

	calls := mockRunRecording(t, responses)

	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	if len(*calls) < 14 {
		t.Fatalf("expected at least 14 calls, got %d", len(*calls))
	}
	call12 := (*calls)[12]
	call13 := (*calls)[13]

	if call12.args[len(call12.args)-1] != "alpine" {
		t.Errorf("expected fallback user.name 'alpine', got args: %v", call12.args)
	}
	if call13.args[len(call13.args)-1] != "alpine@localhost" {
		t.Errorf("expected fallback user.email 'alpine@localhost', got args: %v", call13.args)
	}
}

// ---------------------------------------------------------------------------
// Step 14: Copy ~/.claude into container
// ---------------------------------------------------------------------------

func TestRunCreate_Step14_CopyClaudeDir(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tempDir := setupCreateTest(t)
		claudeDir := filepath.Join(tempDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("mkdir .claude: %v", err)
		}
		if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("write settings.json: %v", err)
		}

		responses := happyCreateResponses("test")
		withCopy := make([]cmdResult, 0, len(responses)+2)
		withCopy = append(withCopy, responses[:14]...)
		withCopy = append(withCopy, cmdResult{stdout: ""}) // docker cp
		withCopy = append(withCopy, cmdResult{stdout: ""}) // chown
		withCopy = append(withCopy, responses[14:]...)

		calls := mockRunRecording(t, withCopy)

		captureStdout(t, func() {
			err := runCreateCmd(t, context.Background(), "test")
			assertExitCode(t, err, 0)
		})

		foundCp := false
		for _, c := range *calls {
			if c.name == "docker" && len(c.args) >= 2 && c.args[0] == "cp" {
				for _, a := range c.args {
					if strings.Contains(a, ".claude") {
						foundCp = true
						break
					}
				}
			}
		}
		if !foundCp {
			t.Fatal("expected docker cp call for ~/.claude")
		}
	})

	t.Run("no .claude dir skips copy", func(t *testing.T) {
		setupCreateTest(t)
		mockRun(t, happyCreateResponses("test"))
		captureStdout(t, func() {
			err := runCreateCmd(t, context.Background(), "test")
			assertExitCode(t, err, 0)
		})
	})

	t.Run("copy fails non-fatal", func(t *testing.T) {
		tempDir := setupCreateTest(t)
		if err := os.MkdirAll(filepath.Join(tempDir, ".claude"), 0755); err != nil {
			t.Fatalf("mkdir .claude: %v", err)
		}

		responses := happyCreateResponses("test")
		withCopy := make([]cmdResult, 0, len(responses)+1)
		withCopy = append(withCopy, responses[:14]...)
		withCopy = append(withCopy, errResult("cp failed"))
		withCopy = append(withCopy, responses[14:]...)
		mockRun(t, withCopy)

		captureStdout(t, func() {
			err := runCreateCmd(t, context.Background(), "test")
			assertExitCode(t, err, 0)
		})
	})
}

// ---------------------------------------------------------------------------
// Step 14: Copy ~/.claude.json into container
// ---------------------------------------------------------------------------

func TestRunCreate_Step14_CopyClaudeJSON(t *testing.T) {
	tempDir := setupCreateTest(t)
	if err := os.WriteFile(filepath.Join(tempDir, ".claude.json"), []byte(`{"hasCompletedOnboarding":true}`), 0644); err != nil {
		t.Fatalf("write .claude.json: %v", err)
	}

	responses := happyCreateResponses("test")
	withJSON := make([]cmdResult, 0, len(responses)+2)
	withJSON = append(withJSON, responses[:14]...)
	withJSON = append(withJSON, cmdResult{stdout: ""}) // docker cp
	withJSON = append(withJSON, cmdResult{stdout: ""}) // chown
	withJSON = append(withJSON, responses[14:]...)
	mockRun(t, withJSON)

	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})
}

// ---------------------------------------------------------------------------
// Step 17: Install command -- success
// ---------------------------------------------------------------------------

func TestRunCreate_Step17_InstallSuccess(t *testing.T) {
	tempDir := setupCreateTest(t)
	if err := os.WriteFile(filepath.Join(tempDir, "alpine.yaml"), []byte("install: \"make build\"\n"), 0644); err != nil {
		t.Fatalf("write alpine.yaml: %v", err)
	}

	container := "alpine-test-dev-1"
	responses := []cmdResult{
		{stdout: ""},                 // 0: docker info
		{stdout: "/tmp/fakegitroot"}, // 1: git rev-parse
		{stdout: "[]"},               // 2: docker compose ls
		{stdout: "main"},             // 3: git rev-parse --abbrev-ref HEAD
		{stdout: ""},                 // 4: docker image inspect
		{stdout: ""},                 // 5: docker compose up
		{stdout: fmt.Sprintf(`{"Name":"%s","Service":"dev"}`, container)}, // 6: docker compose ps
		{stdout: "git@github.com:user/repo.git"},                          // 7: git remote get-url origin
		{stdout: ""},                                                      // 8: git clone
		{stdout: ""},                                                      // 9: git checkout -b
		{stdout: "Test User"},                                             // 10: git config user.name
		{stdout: "test@example.com"},                                      // 11: git config user.email
		{stdout: ""},                                                      // 12: container git config user.name
		{stdout: ""},                                                      // 13: container git config user.email
		{stdout: ""},                                                      // 14: install command
		{stdout: ""},                                                      // 15: claude setup
		{stdout: "set"},                                                   // 16: token check
	}
	mockRun(t, responses)

	out := captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, out)
	}
	if result["install_status"] != "success" {
		t.Fatalf("install_status = %v, want %q", result["install_status"], "success")
	}
}

// ---------------------------------------------------------------------------
// Step 17: Install command -- failure (exit code 3)
// ---------------------------------------------------------------------------

func TestRunCreate_Step17_InstallFails(t *testing.T) {
	tempDir := setupCreateTest(t)
	if err := os.WriteFile(filepath.Join(tempDir, "alpine.yaml"), []byte("install: \"make build\"\n"), 0644); err != nil {
		t.Fatalf("write alpine.yaml: %v", err)
	}

	container := "alpine-test-dev-1"
	responses := []cmdResult{
		{stdout: ""},                 // 0: docker info
		{stdout: "/tmp/fakegitroot"}, // 1: git rev-parse
		{stdout: "[]"},               // 2: docker compose ls
		{stdout: "main"},             // 3: git rev-parse --abbrev-ref HEAD
		{stdout: ""},                 // 4: docker image inspect
		{stdout: ""},                 // 5: docker compose up
		{stdout: fmt.Sprintf(`{"Name":"%s","Service":"dev"}`, container)}, // 6: docker compose ps
		{stdout: "git@github.com:user/repo.git"},                          // 7: git remote get-url origin
		{stdout: ""},                                                      // 8: git clone
		{stdout: ""},                                                      // 9: git checkout -b
		{stdout: "Test User"},                                             // 10: git config user.name
		{stdout: "test@example.com"},                                      // 11: git config user.email
		{stdout: ""},                                                      // 12: container git config user.name
		{stdout: ""},                                                      // 13: container git config user.email
		errResult("make: *** Error 2"),                                    // 14: install command (fails)
		{stdout: ""},                                                      // 15: claude setup
		{stdout: "set"},                                                   // 16: token check
	}
	mockRun(t, responses)

	out := captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 3)
		assertContains(t, err, "install command failed")
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, out)
	}
	if result["install_status"] != "failed" {
		t.Fatalf("install_status = %v, want %q", result["install_status"], "failed")
	}
	if result["warning"] == nil {
		t.Fatal("expected warning field in JSON output")
	}
}

// ---------------------------------------------------------------------------
// Step 18: Token not set (non-fatal warning)
// ---------------------------------------------------------------------------

func TestRunCreate_Step18_TokenNotSet(t *testing.T) {
	setupCreateTest(t)
	responses := happyCreateResponses("test")
	responses[15] = cmdResult{stdout: "unset"}
	mockRun(t, responses)

	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})
}

// ---------------------------------------------------------------------------
// Step 18: Claude onboarding setup fails (non-fatal)
// ---------------------------------------------------------------------------

func TestRunCreate_Step18_ClaudeSetupFails(t *testing.T) {
	setupCreateTest(t)
	responses := happyCreateResponses("test")
	responses[14] = errResult("permission denied")
	mockRun(t, responses)

	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})
}

// ---------------------------------------------------------------------------
// Step 19: Detach mode with JSON output
// ---------------------------------------------------------------------------

func TestRunCreate_Step19_DetachJSON(t *testing.T) {
	setupCreateTest(t)
	mockRun(t, happyCreateResponses("test"))

	out := captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nraw: %s", err, out)
	}

	expectedFields := map[string]string{
		"name":      "test",
		"branch":    "feature/test",
		"status":    "running",
		"container": "alpine-test-dev-1",
		"project":   "alpine-test",
	}
	for field, want := range expectedFields {
		got, ok := result[field]
		if !ok {
			t.Errorf("missing field %q in JSON output", field)
			continue
		}
		if got != want {
			t.Errorf("%s = %v, want %q", field, got, want)
		}
	}
	if result["install_status"] != nil {
		t.Errorf("install_status should be nil when no install, got %v", result["install_status"])
	}
}

// ---------------------------------------------------------------------------
// Step 19: Detach mode non-JSON (pretty print)
// ---------------------------------------------------------------------------

func TestRunCreate_Step19_DetachNonJSON(t *testing.T) {
	setupCreateTest(t)
	detach = true
	jsonOutput = false
	mockRun(t, happyCreateResponses("test"))

	out := captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	if !strings.Contains(out, `"name": "test"`) {
		t.Fatalf("expected pretty-printed JSON, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// Step 19: Install failed + detach returns exit code 3
// ---------------------------------------------------------------------------

func TestRunCreate_Step19_InstallFailedDetach(t *testing.T) {
	tempDir := setupCreateTest(t)
	if err := os.WriteFile(filepath.Join(tempDir, "alpine.yaml"), []byte("install: \"npm install\"\n"), 0644); err != nil {
		t.Fatalf("write alpine.yaml: %v", err)
	}

	container := "alpine-test-dev-1"
	responses := []cmdResult{
		{stdout: ""},                 // 0
		{stdout: "/tmp/fakegitroot"}, // 1
		{stdout: "[]"},               // 2
		{stdout: "main"},             // 3
		{stdout: ""},                 // 4
		{stdout: ""},                 // 5
		{stdout: fmt.Sprintf(`{"Name":"%s","Service":"dev"}`, container)}, // 6
		{stdout: "git@github.com:user/repo.git"},                          // 7
		{stdout: ""},                                                      // 8
		{stdout: ""},                                                      // 9
		{stdout: "Test User"},                                             // 10
		{stdout: "test@example.com"},                                      // 11
		{stdout: ""},                                                      // 12
		{stdout: ""},                                                      // 13
		errResult("npm ERR! code ENOENT"),                                 // 14
		{stdout: ""},                                                      // 15
		{stdout: "set"},                                                   // 16
	}
	mockRun(t, responses)

	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 3)
		assertContains(t, err, "install command failed")
	})
}

// ---------------------------------------------------------------------------
// Step 19: Interactive mode (no detach, no json)
// ---------------------------------------------------------------------------

func TestRunCreate_Step19_InteractiveMode(t *testing.T) {
	setupCreateTest(t)
	detach = false
	jsonOutput = false

	mockRun(t, happyCreateResponses("test"))
	mockRunInteractive(t, nil)

	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 0)
}

// ---------------------------------------------------------------------------
// Step 19: Interactive mode + install failed returns exit code 3
// ---------------------------------------------------------------------------

func TestRunCreate_Step19_InteractiveInstallFailed(t *testing.T) {
	tempDir := setupCreateTest(t)
	detach = false
	jsonOutput = false
	if err := os.WriteFile(filepath.Join(tempDir, "alpine.yaml"), []byte("install: \"npm install\"\n"), 0644); err != nil {
		t.Fatalf("write alpine.yaml: %v", err)
	}

	container := "alpine-test-dev-1"
	responses := []cmdResult{
		{stdout: ""},                 // 0
		{stdout: "/tmp/fakegitroot"}, // 1
		{stdout: "[]"},               // 2
		{stdout: "main"},             // 3
		{stdout: ""},                 // 4
		{stdout: ""},                 // 5
		{stdout: fmt.Sprintf(`{"Name":"%s","Service":"dev"}`, container)}, // 6
		{stdout: "git@github.com:user/repo.git"},                          // 7
		{stdout: ""},                                                      // 8
		{stdout: ""},                                                      // 9
		{stdout: "Test User"},                                             // 10
		{stdout: "test@example.com"},                                      // 11
		{stdout: ""},                                                      // 12
		{stdout: ""},                                                      // 13
		errResult("install failed"),                                       // 14
		{stdout: ""},                                                      // 15
		{stdout: "set"},                                                   // 16
	}
	mockRun(t, responses)
	mockRunInteractive(t, nil)

	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 3)
	assertContains(t, err, "install command failed")
}

// ---------------------------------------------------------------------------
// Rollback: compose up succeeds, later step fails, composeDown called
// ---------------------------------------------------------------------------

func TestRunCreate_Rollback(t *testing.T) {
	setupCreateTest(t)
	responses := happyCreateResponses("test")
	responses[7] = errResult("no remote")
	rollbackResponses := make([]cmdResult, 8, 9)
	copy(rollbackResponses, responses[:8])
	rollbackResponses = append(rollbackResponses, cmdResult{stdout: ""})
	calls := mockRunRecording(t, rollbackResponses)

	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 1)

	foundDown := false
	for _, c := range *calls {
		if c.name == "docker" && len(c.args) >= 2 {
			for _, a := range c.args {
				if a == "down" {
					foundDown = true
					break
				}
			}
		}
	}
	if !foundDown {
		t.Fatal("expected composeDown rollback after post-compose-up failure")
	}
}

// ---------------------------------------------------------------------------
// No rollback when compose was never started
// ---------------------------------------------------------------------------

func TestRunCreate_NoRollbackBeforeCompose(t *testing.T) {
	setupCreateTest(t)
	mockRun(t, []cmdResult{
		{stdout: ""},                         // 0: docker info
		{stdout: "/tmp/fakegitroot"},         // 1: git rev-parse
		{stdout: `[{"Name":"alpine-test"}]`}, // 2: compose ls (duplicate)
	})
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 1)
}

// ---------------------------------------------------------------------------
// Config load error
// ---------------------------------------------------------------------------

func TestRunCreate_ConfigLoadError(t *testing.T) {
	tempDir := setupCreateTest(t)
	if err := os.WriteFile(filepath.Join(tempDir, "alpine.yaml"), []byte(":\n  invalid"), 0644); err != nil {
		t.Fatalf("write alpine.yaml: %v", err)
	}

	mockRun(t, []cmdResult{
		{stdout: ""},      // 0: docker info
		{stdout: tempDir}, // 1: git rev-parse
		{stdout: "[]"},    // 2: compose ls
		{stdout: "main"},  // 3: git rev-parse --abbrev-ref HEAD
	})
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 1)
	assertContains(t, err, "failed to load config")
}

// ---------------------------------------------------------------------------
// Security: --from=-malicious rejected
// ---------------------------------------------------------------------------

func TestRunCreate_Security_FromFlagInjection(t *testing.T) {
	setupCreateTest(t)
	fromBranch = "-malicious"

	mockRun(t, []cmdResult{
		{stdout: ""},                 // 0: docker info
		{stdout: "/tmp/fakegitroot"}, // 1: git rev-parse
		{stdout: "[]"},               // 2: compose ls
	})
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 1)
	assertContains(t, err, "must not start with '-'")
}

// ---------------------------------------------------------------------------
// Security: Env file outside git root rejected (non-fatal)
// ---------------------------------------------------------------------------

func TestRunCreate_Security_EnvFileEscapesRoot(t *testing.T) {
	tempDir := setupCreateTest(t)
	if err := os.WriteFile(filepath.Join(tempDir, "alpine.yaml"),
		[]byte("env_files:\n  - \"../../etc/passwd\"\n"), 0644); err != nil {
		t.Fatalf("write alpine.yaml: %v", err)
	}

	mockRun(t, happyCreateResponses("test"))

	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})
}

// ---------------------------------------------------------------------------
// Security: Compose YAML uses passthrough env syntax (no literal secrets)
// ---------------------------------------------------------------------------

func TestRunCreate_Security_PassthroughEnvSyntax(t *testing.T) {
	cfg := &Config{BaseImage: "ubuntu:24.04"}
	yamlBytes, err := generateComposeYAML(cfg, "test", "main", "darwin", "alpine-dev:abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(yamlBytes)

	sensitiveVars := []string{"ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"}
	for _, v := range sensitiveVars {
		if !strings.Contains(s, "- "+v) {
			t.Errorf("missing passthrough env var: %s", v)
		}
		for _, line := range strings.Split(s, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "- "+v+"=") {
				t.Errorf("env var %s has literal value in compose YAML: %s", v, trimmed)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Happy path end-to-end with JSON output
// ---------------------------------------------------------------------------

func TestRunCreate_HappyPath(t *testing.T) {
	setupCreateTest(t)
	mockRun(t, happyCreateResponses("test"))

	out := captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, out)
	}
	if result["name"] != "test" {
		t.Errorf("name = %v, want %q", result["name"], "test")
	}
	if result["status"] != "running" {
		t.Errorf("status = %v, want %q", result["status"], "running")
	}
}

// ---------------------------------------------------------------------------
// Verify call count in happy path
// ---------------------------------------------------------------------------

func TestRunCreate_HappyPath_CallCount(t *testing.T) {
	setupCreateTest(t)
	calls := mockRunRecording(t, happyCreateResponses("test"))

	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	if len(*calls) != 16 {
		t.Fatalf("expected 16 run() calls, got %d", len(*calls))
	}
}

// ---------------------------------------------------------------------------
// Env file not found (non-fatal warning)
// ---------------------------------------------------------------------------

func TestRunCreate_EnvFileNotFound(t *testing.T) {
	tempDir := setupCreateTest(t)
	if err := os.WriteFile(filepath.Join(tempDir, "alpine.yaml"),
		[]byte("env_files:\n  - .env.missing\n"), 0644); err != nil {
		t.Fatalf("write alpine.yaml: %v", err)
	}

	mockRun(t, happyCreateResponses("test"))
	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})
}

// ---------------------------------------------------------------------------
// Auto-config file detection (step 15)
// ---------------------------------------------------------------------------

func TestRunCreate_AutoConfigFiles(t *testing.T) {
	setupCreateTest(t)
	gitRoot, _ := os.Getwd()
	if err := os.WriteFile(filepath.Join(gitRoot, ".tool-versions"), []byte("nodejs 20.0.0\n"), 0644); err != nil {
		t.Fatalf("write .tool-versions: %v", err)
	}

	responses := happyCreateResponses("test")
	// Override response[1] to return the real CWD so os.Stat finds files.
	responses[1] = cmdResult{stdout: gitRoot}
	// .tool-versions triggers copyPathToContainer (2 calls: cp + chown).
	withConfig := make([]cmdResult, 0, len(responses)+2)
	withConfig = append(withConfig, responses[:14]...)
	withConfig = append(withConfig, cmdResult{stdout: ""}) // docker cp
	withConfig = append(withConfig, cmdResult{stdout: ""}) // chown
	withConfig = append(withConfig, responses[14:]...)

	calls := mockRunRecording(t, withConfig)

	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	foundCp := false
	for _, c := range *calls {
		if c.name == "docker" && len(c.args) >= 2 && c.args[0] == "cp" {
			for _, a := range c.args {
				if strings.Contains(a, ".tool-versions") {
					foundCp = true
					break
				}
			}
		}
	}
	if !foundCp {
		t.Fatal("expected docker cp call for .tool-versions")
	}
}

// ---------------------------------------------------------------------------
// Cobra Args validator (not exercised through runCreate directly)
// ---------------------------------------------------------------------------

func TestCreateCmdArgsValidator(t *testing.T) {
	args := createCmd.Args

	t.Run("no args returns error", func(t *testing.T) {
		err := args(createCmd, []string{})
		if err == nil {
			t.Fatal("expected error for no args")
		}
		if !strings.Contains(err.Error(), "requires a name") {
			t.Fatalf("error = %q, want to contain 'requires a name'", err.Error())
		}
	})

	t.Run("too many args returns error", func(t *testing.T) {
		err := args(createCmd, []string{"a", "b"})
		if err == nil {
			t.Fatal("expected error for too many args")
		}
		if !strings.Contains(err.Error(), "accepts 1 argument") {
			t.Fatalf("error = %q, want to contain 'accepts 1 argument'", err.Error())
		}
	})

	t.Run("one arg succeeds", func(t *testing.T) {
		err := args(createCmd, []string{"valid"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// .env loading at gitRoot (Step 3 side effect)
// ---------------------------------------------------------------------------

func TestRunCreate_DotEnvLoading(t *testing.T) {
	tempDir := setupCreateTest(t)

	// Write .env at the gitRoot (tempDir).
	if err := os.WriteFile(filepath.Join(tempDir, ".env"), []byte("MY_TEST_VAR=loaded\n"), 0644); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	t.Setenv("MY_TEST_VAR", "")
	_ = os.Unsetenv("MY_TEST_VAR")

	// Make gitFindRoot return tempDir so .env loading fires.
	// Also, step 15 auto-detect finds .env at gitRoot and copies it (2 calls).
	responses := happyCreateResponses("test")
	responses[1] = cmdResult{stdout: tempDir}
	withCopy := make([]cmdResult, 0, len(responses)+2)
	withCopy = append(withCopy, responses[:14]...)
	withCopy = append(withCopy, cmdResult{stdout: ""}) // docker cp .env (auto-detect)
	withCopy = append(withCopy, cmdResult{stdout: ""}) // chown .env
	withCopy = append(withCopy, responses[14:]...)
	mockRun(t, withCopy)

	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	if os.Getenv("MY_TEST_VAR") != "loaded" {
		t.Errorf("MY_TEST_VAR = %q, want %q", os.Getenv("MY_TEST_VAR"), "loaded")
	}
}

// ---------------------------------------------------------------------------
// Non-JSON mode: various warning branches
// ---------------------------------------------------------------------------

func TestRunCreate_NonJSONTokenWarning(t *testing.T) {
	setupCreateTest(t)
	detach = true
	jsonOutput = false

	responses := happyCreateResponses("test")
	responses[15] = cmdResult{stdout: "unset"} // token not set
	mockRun(t, responses)

	out := captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	// Non-JSON detach mode outputs pretty-printed JSON.
	if !strings.Contains(out, `"name": "test"`) {
		t.Fatalf("expected pretty-printed JSON, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// Env file copy success (Step 16)
// ---------------------------------------------------------------------------

func TestRunCreate_EnvFileCopySuccess(t *testing.T) {
	tempDir := setupCreateTest(t)

	// Config with an env file that exists at gitRoot.
	if err := os.WriteFile(filepath.Join(tempDir, "alpine.yaml"),
		[]byte("env_files:\n  - .env.local\n"), 0644); err != nil {
		t.Fatalf("write alpine.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, ".env.local"), []byte("SECRET=val\n"), 0644); err != nil {
		t.Fatalf("write .env.local: %v", err)
	}

	// Make gitFindRoot return tempDir so the file is found.
	responses := happyCreateResponses("test")
	responses[1] = cmdResult{stdout: tempDir}
	// Env file triggers copyPathToContainer (2 calls: cp + chown) after step 13.
	withEnv := make([]cmdResult, 0, len(responses)+2)
	withEnv = append(withEnv, responses[:14]...)
	withEnv = append(withEnv, cmdResult{stdout: ""}) // docker cp .env.local
	withEnv = append(withEnv, cmdResult{stdout: ""}) // chown .env.local
	withEnv = append(withEnv, responses[14:]...)

	calls := mockRunRecording(t, withEnv)

	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	// Verify docker cp was called with .env.local.
	foundCp := false
	for _, c := range *calls {
		if c.name == "docker" && len(c.args) >= 2 && c.args[0] == "cp" {
			for _, a := range c.args {
				if strings.Contains(a, ".env.local") {
					foundCp = true
					break
				}
			}
		}
	}
	if !foundCp {
		t.Fatal("expected docker cp call for .env.local")
	}
}

// ---------------------------------------------------------------------------
// Env file already copied in auto-detect (Step 16 dedup)
// ---------------------------------------------------------------------------

func TestRunCreate_EnvFileAlreadyCopied(t *testing.T) {
	tempDir := setupCreateTest(t)

	// Config references .env which is also in autoConfigPaths.
	if err := os.WriteFile(filepath.Join(tempDir, "alpine.yaml"),
		[]byte("env_files:\n  - .env\n"), 0644); err != nil {
		t.Fatalf("write alpine.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, ".env"), []byte("KEY=val\n"), 0644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	// Make gitFindRoot return tempDir so auto-detect finds .env.
	responses := happyCreateResponses("test")
	responses[1] = cmdResult{stdout: tempDir}
	// .env auto-copied in step 15, then step 16 skips it.
	// Auto-copy: .claude dir (step 15 index 0) -- .claude dir might or might not exist at gitRoot.
	// Actually, autoConfigPaths checks gitRoot for: .claude, .env, .tool-versions, etc.
	// Since tempDir has .env, that's 1 copy (2 calls: cp + chown).
	withCopy := make([]cmdResult, 0, len(responses)+2)
	withCopy = append(withCopy, responses[:14]...)
	withCopy = append(withCopy, cmdResult{stdout: ""}) // docker cp .env
	withCopy = append(withCopy, cmdResult{stdout: ""}) // chown .env
	withCopy = append(withCopy, responses[14:]...)

	mockRun(t, withCopy)

	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})
	// If step 16 tried to copy .env again, we'd get "unexpected call to run()" --
	// the fact that the test passes proves dedup worked.
}

// ---------------------------------------------------------------------------
// generateComposeYAML error path (bad service triggers error in runCreate)
// ---------------------------------------------------------------------------

func TestRunCreate_ComposeYAMLError(t *testing.T) {
	tempDir := setupCreateTest(t)

	// Config with unsupported service -- generateComposeYAML returns error.
	if err := os.WriteFile(filepath.Join(tempDir, "alpine.yaml"),
		[]byte("base_image: ubuntu:24.04\nservices:\n  - mysql\n"), 0644); err != nil {
		t.Fatalf("write alpine.yaml: %v", err)
	}

	mockRun(t, []cmdResult{
		{stdout: ""},      // 0: docker info
		{stdout: tempDir}, // 1: git rev-parse --show-toplevel
		{stdout: "[]"},    // 2: docker compose ls
		{stdout: "main"},  // 3: git rev-parse --abbrev-ref HEAD
	})
	// loadConfig validates services, so it should fail before generateComposeYAML.
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 1)
	assertContains(t, err, "failed to load config")
}

// ---------------------------------------------------------------------------
// imageExists returns error (context cancelled) in runCreate
// ---------------------------------------------------------------------------

func TestRunCreate_ImageExistsError(t *testing.T) {
	setupCreateTest(t)

	// Use a cancelled context so imageExists returns an error.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockRun(t, []cmdResult{
		{stdout: ""},                  // 0: docker info
		{stdout: "/tmp/fakegitroot"},  // 1: git rev-parse
		{stdout: "[]"},                // 2: docker compose ls
		{stdout: "main"},              // 3: git rev-parse --abbrev-ref HEAD
		errResult("context canceled"), // 4: docker image inspect fails
	})
	err := runCreateCmd(t, ctx, "test")
	assertExitCode(t, err, 2)
	assertContains(t, err, "failed to check image")
}

// ---------------------------------------------------------------------------
// Non-JSON mode: copy failure warnings write to stderr
// ---------------------------------------------------------------------------

func TestRunCreate_NonJSON_CopyClaudeFailWarning(t *testing.T) {
	tempDir := setupCreateTest(t)
	jsonOutput = false
	detach = true

	// Create ~/.claude so the copy path runs.
	if err := os.MkdirAll(filepath.Join(tempDir, ".claude"), 0755); err != nil {
		t.Fatalf("mkdir .claude: %v", err)
	}

	responses := happyCreateResponses("test")
	// Insert a copy failure after step 13 (git config user.email).
	withCopy := make([]cmdResult, 0, len(responses)+1)
	withCopy = append(withCopy, responses[:14]...)
	withCopy = append(withCopy, errResult("cp failed")) // docker cp ~/.claude fails
	withCopy = append(withCopy, responses[14:]...)
	mockRun(t, withCopy)

	stdout, stderr := captureOutputs(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	// Non-JSON detach mode produces pretty-printed JSON on stdout.
	if !strings.Contains(stdout, `"name": "test"`) {
		t.Fatalf("expected pretty-printed JSON on stdout, got: %s", stdout)
	}
	// Warning should appear on stderr.
	if !strings.Contains(stderr, "warning: failed to copy ~/.claude") {
		t.Errorf("expected warning about ~/.claude copy failure on stderr, got: %s", stderr)
	}
}

// ---------------------------------------------------------------------------
// Non-JSON mode: env file not found warning
// ---------------------------------------------------------------------------

func TestRunCreate_NonJSON_EnvFileNotFound(t *testing.T) {
	tempDir := setupCreateTest(t)
	jsonOutput = false
	detach = true

	if err := os.WriteFile(filepath.Join(tempDir, "alpine.yaml"),
		[]byte("env_files:\n  - .env.missing\n"), 0644); err != nil {
		t.Fatalf("write alpine.yaml: %v", err)
	}

	mockRun(t, happyCreateResponses("test"))

	stdout, stderr := captureOutputs(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	if !strings.Contains(stdout, `"name": "test"`) {
		t.Fatalf("expected pretty-printed JSON on stdout, got: %s", stdout)
	}
	if !strings.Contains(stderr, "env file") || !strings.Contains(stderr, "not found") {
		t.Errorf("expected warning about missing env file on stderr, got: %s", stderr)
	}
}

// ---------------------------------------------------------------------------
// Non-JSON mode: env file escapes git root warning
// ---------------------------------------------------------------------------

func TestRunCreate_NonJSON_EnvFileEscapesRoot(t *testing.T) {
	tempDir := setupCreateTest(t)
	jsonOutput = false
	detach = true

	if err := os.WriteFile(filepath.Join(tempDir, "alpine.yaml"),
		[]byte("env_files:\n  - \"../../etc/passwd\"\n"), 0644); err != nil {
		t.Fatalf("write alpine.yaml: %v", err)
	}

	mockRun(t, happyCreateResponses("test"))

	stdout, stderr := captureOutputs(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	if !strings.Contains(stdout, `"name": "test"`) {
		t.Fatalf("expected pretty-printed JSON on stdout, got: %s", stdout)
	}
	if !strings.Contains(stderr, "escapes repository root") {
		t.Errorf("expected warning about env file escaping root on stderr, got: %s", stderr)
	}
}

// ---------------------------------------------------------------------------
// Non-JSON mode: env file copy failure warning
// ---------------------------------------------------------------------------

func TestRunCreate_NonJSON_EnvFileCopyFails(t *testing.T) {
	tempDir := setupCreateTest(t)
	jsonOutput = false
	detach = true

	if err := os.WriteFile(filepath.Join(tempDir, "alpine.yaml"),
		[]byte("env_files:\n  - .env.local\n"), 0644); err != nil {
		t.Fatalf("write alpine.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, ".env.local"), []byte("KEY=val\n"), 0644); err != nil {
		t.Fatalf("write .env.local: %v", err)
	}

	// gitFindRoot returns tempDir so env file is found.
	responses := happyCreateResponses("test")
	responses[1] = cmdResult{stdout: tempDir}
	// .env auto-detect triggers copy for .env.local? No, autoConfigPaths doesn't
	// include .env.local. But .env at gitRoot is checked... there's no .env file.
	// So only the explicit env_files copy runs.
	// Env file copy: cp fails.
	withEnv := make([]cmdResult, 0, len(responses)+1)
	withEnv = append(withEnv, responses[:14]...)
	withEnv = append(withEnv, errResult("cp failed")) // docker cp .env.local fails
	withEnv = append(withEnv, responses[14:]...)
	mockRun(t, withEnv)

	stdout, stderr := captureOutputs(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	if !strings.Contains(stdout, `"name": "test"`) {
		t.Fatalf("expected pretty-printed JSON on stdout, got: %s", stdout)
	}
	if !strings.Contains(stderr, "failed to copy env file") {
		t.Errorf("expected warning about env file copy failure on stderr, got: %s", stderr)
	}
}

// ---------------------------------------------------------------------------
// Non-JSON mode: auto-config copy failure warning
// ---------------------------------------------------------------------------

func TestRunCreate_NonJSON_AutoConfigCopyFails(t *testing.T) {
	tempDir := setupCreateTest(t)
	jsonOutput = false
	detach = true

	// Create .tool-versions at gitRoot so auto-detect finds it.
	if err := os.WriteFile(filepath.Join(tempDir, ".tool-versions"), []byte("nodejs 20\n"), 0644); err != nil {
		t.Fatalf("write .tool-versions: %v", err)
	}

	responses := happyCreateResponses("test")
	responses[1] = cmdResult{stdout: tempDir}
	// Auto-copy .tool-versions: cp fails.
	withCopy := make([]cmdResult, 0, len(responses)+1)
	withCopy = append(withCopy, responses[:14]...)
	withCopy = append(withCopy, errResult("cp failed")) // docker cp .tool-versions fails
	withCopy = append(withCopy, responses[14:]...)
	mockRun(t, withCopy)

	stdout, stderr := captureOutputs(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})

	if !strings.Contains(stdout, `"name": "test"`) {
		t.Fatalf("expected pretty-printed JSON on stdout, got: %s", stdout)
	}
	if !strings.Contains(stderr, "failed to copy") || !strings.Contains(stderr, ".tool-versions") {
		t.Errorf("expected warning about .tool-versions copy failure on stderr, got: %s", stderr)
	}
}

// ---------------------------------------------------------------------------
// Interactive mode: install failed (exercises the non-detach installFailed path)
// (This covers create.go line ~484-486 specifically)
// ---------------------------------------------------------------------------

func TestRunCreate_InteractiveInstallFailed_NonJSON(t *testing.T) {
	tempDir := setupCreateTest(t)
	detach = false
	jsonOutput = false

	if err := os.WriteFile(filepath.Join(tempDir, "alpine.yaml"), []byte("install: \"make build\"\n"), 0644); err != nil {
		t.Fatalf("write alpine.yaml: %v", err)
	}

	container := "alpine-test-dev-1"
	responses := []cmdResult{
		{stdout: ""},                 // 0: docker info
		{stdout: "/tmp/fakegitroot"}, // 1: git rev-parse
		{stdout: "[]"},               // 2: docker compose ls
		{stdout: "main"},             // 3: git rev-parse --abbrev-ref HEAD
		{stdout: ""},                 // 4: docker image inspect
		{stdout: ""},                 // 5: docker compose up
		{stdout: fmt.Sprintf(`{"Name":"%s","Service":"dev"}`, container)}, // 6: docker compose ps
		{stdout: "git@github.com:user/repo.git"},                          // 7: git remote get-url origin
		{stdout: ""},                                                      // 8: git clone
		{stdout: ""},                                                      // 9: git checkout -b
		{stdout: "Test User"},                                             // 10: git config user.name
		{stdout: "test@example.com"},                                      // 11: git config user.email
		{stdout: ""},                                                      // 12: container git config user.name
		{stdout: ""},                                                      // 13: container git config user.email
		errResult("make failed"),                                          // 14: install command fails
		{stdout: ""},                                                      // 15: claude setup
		{stdout: "set"},                                                   // 16: token check
	}
	mockRun(t, responses)
	mockRunInteractive(t, nil)

	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 3)
	assertContains(t, err, "install command failed")
}

// ---------------------------------------------------------------------------
// .env loading error path (create.go lines 102-104)
// ---------------------------------------------------------------------------

func TestRunCreate_DotEnvLoadError(t *testing.T) {
	tempDir := setupCreateTest(t)

	// Create a .env file that loadDotEnv can stat but not read.
	envPath := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envPath, []byte("KEY=val\n"), 0644); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	if err := os.Chmod(envPath, 0000); err != nil {
		t.Fatalf("chmod .env 0000: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(envPath, 0644)
	})

	// gitFindRoot returns tempDir so .env is found by os.Stat.
	responses := happyCreateResponses("test")
	responses[1] = cmdResult{stdout: tempDir}
	// Step 15 auto-detect also finds .env at gitRoot (stat succeeds even with
	// chmod 0000) and tries to copy it (2 extra calls: cp + chown).
	withCopy := make([]cmdResult, 0, len(responses)+2)
	withCopy = append(withCopy, responses[:14]...)
	withCopy = append(withCopy, cmdResult{stdout: ""}) // docker cp .env
	withCopy = append(withCopy, cmdResult{stdout: ""}) // chown .env
	withCopy = append(withCopy, responses[14:]...)
	mockRun(t, withCopy)

	// loadDotEnv error is non-fatal; create should still succeed.
	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})
}

// ---------------------------------------------------------------------------
// Step 14: Copy ~/.claude.json fails (create.go lines 309-311)
// ---------------------------------------------------------------------------

func TestRunCreate_Step14_CopyClaudeJSONFail(t *testing.T) {
	tempDir := setupCreateTest(t)
	if err := os.WriteFile(filepath.Join(tempDir, ".claude.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("write .claude.json: %v", err)
	}

	responses := happyCreateResponses("test")
	// .claude.json triggers copyPathToContainer (docker cp), which fails.
	withJSON := make([]cmdResult, 0, len(responses)+1)
	withJSON = append(withJSON, responses[:14]...)
	withJSON = append(withJSON, errResult("cp failed")) // docker cp ~/.claude.json fails
	withJSON = append(withJSON, responses[14:]...)
	mockRun(t, withJSON)

	// Copy failure is non-fatal.
	captureStdout(t, func() {
		err := runCreateCmd(t, context.Background(), "test")
		assertExitCode(t, err, 0)
	})
}

// ---------------------------------------------------------------------------
// Interactive shell returns error (create.go lines 484-486)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Step 7: os.MkdirTemp fails for build temp dir (create.go:176-179)
// ---------------------------------------------------------------------------

func TestRunCreate_Step7_MkdirTempBuildFails(t *testing.T) {
	setupCreateTest(t)
	// MkdirTemp fails after imageExists (step 4), so only 5 responses consumed.
	mockRun(t, []cmdResult{
		{stdout: ""},                 // 0: docker info
		{stdout: "/tmp/fakegitroot"}, // 1: git rev-parse --show-toplevel
		{stdout: "[]"},               // 2: docker compose ls
		{stdout: "main"},             // 3: git rev-parse --abbrev-ref HEAD
		errResult("no such image"),   // 4: docker image inspect (not found)
	})

	// Break TMPDIR so os.MkdirTemp fails inside runCreate.
	// setupCreateTest already created its temp dirs, so this only affects runCreate.
	t.Setenv("TMPDIR", "/nonexistent/path")

	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 2)
	assertContains(t, err, "failed to create temp dir")
}

// ---------------------------------------------------------------------------
// Step 8: os.MkdirTemp fails for compose temp dir (create.go:206-208)
// ---------------------------------------------------------------------------

func TestRunCreate_Step8_MkdirTempComposeFails(t *testing.T) {
	setupCreateTest(t)
	// Image exists (skip build), MkdirTemp fails after step 4. Only 5 responses consumed.
	mockRun(t, []cmdResult{
		{stdout: ""},                 // 0: docker info
		{stdout: "/tmp/fakegitroot"}, // 1: git rev-parse --show-toplevel
		{stdout: "[]"},               // 2: docker compose ls
		{stdout: "main"},             // 3: git rev-parse --abbrev-ref HEAD
		{stdout: ""},                 // 4: docker image inspect (exists)
	})

	// Break TMPDIR so os.MkdirTemp fails inside runCreate.
	t.Setenv("TMPDIR", "/nonexistent/path")

	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 2)
	assertContains(t, err, "failed to create temp dir")
}

// ---------------------------------------------------------------------------
// Interactive shell returns error (create.go lines 484-486)
// ---------------------------------------------------------------------------

func TestRunCreate_Step19_InteractiveShellError(t *testing.T) {
	setupCreateTest(t)
	detach = false
	jsonOutput = false

	mockRun(t, happyCreateResponses("test"))
	mockRunInteractive(t, fmt.Errorf("shell exited with code 130"))

	// shellErr is non-fatal when installFailed is false.
	err := runCreateCmd(t, context.Background(), "test")
	assertExitCode(t, err, 0)
}
