# Git Worktree Integration - Implementation Plan

## Overview
Add git worktree support to River CLI enabling isolated task execution in separate directories. Focus on creating worktrees at workflow start for clean isolation.

## Architecture Changes

### New Package: `internal/gitx/`
- **interfaces.go**: Core abstractions
  ```go
  type WorktreeManager interface {
      Create(ctx context.Context, taskName string) (*Worktree, error)
      Cleanup(ctx context.Context, wt *Worktree) error
  }

  type Worktree struct {
      Path       string // ../repo-river-<task>
      Branch     string // river/<sanitised-task>
      ParentRepo string // absolute path to main repo
  }
  ```

- **manager.go**: CLI-based implementation using system `git`
- **slug.go**: Task name sanitization helpers
- **mock/manager.go**: Mock implementation for testing

### Configuration Extensions (`internal/config/`)
Add Git subsection:
```go
type GitConfig struct {
    WorktreeEnabled bool   // default=true
    BaseBranch      string // default="main"
    AutoCleanupWT   bool   // default=true
}
```

### Workflow Engine Changes (`internal/workflow/`)
- Inject `WorktreeManager` into `Engine` constructor
- Add worktree lifecycle:
  - **Pre-workflow**: Create worktree, change working directory
  - **Post-workflow**: Optional cleanup
- Update state file path to live within worktree

## Implementation Phases

### Phase 1: Configuration Groundwork ✅
**Goal**: Add git configuration support
- [x] Extend `Config` struct with `GitConfig`
- [x] Add environment variables: `RIVER_GIT_ENABLED`, `RIVER_GIT_BASE_BRANCH`, `RIVER_GIT_AUTO_CLEANUP`
- [x] Write configuration unit tests
- [x] Update config loading logic

**Tests**:
- `TestConfigGitDefaults` ✅
- `TestConfigGitEnvironmentVariables` ✅

### Phase 2: WorktreeManager Implementation ✅
**Goal**: Core git worktree creation
- [x] Create `internal/gitx/` package
- [x] Implement `WorktreeManager` interface with CLI backend
- [x] Add task name sanitization logic
- [x] Create mock implementation for testing

**Tests**:
- `TestCreateWorktree_basic` - Creates directory, branch, registers worktree ✅
- `TestCleanup_removesWorktreeDirAndPrunes` - Proper cleanup ✅
- `TestBranchNameCollisionProducesUniqueNames` - Handles conflicts ✅
- `TestErrorPropagation_gitFailure` - Error handling ✅

### Phase 3: Engine Integration
**Goal**: Wire worktree creation into workflow engine
- [ ] Add `WorktreeManager` field to `Engine`
- [ ] Implement `createWorktree()` helper method
- [ ] Update `Run()` method to create worktree before workflow
- [ ] Handle working directory changes
- [ ] Update state file path to worktree location

**Tests**:
- `TestEngineCreatesWorktree` - Worktree creation with mock manager
- `TestEngineWorktreeDisabled` - Graceful fallback when disabled
- `TestEngineStateFileInWorktree` - State file location

### Phase 4: CLI Integration
**Goal**: Wire everything together at CLI level
- [ ] Update `NewRealDependencies()` to create `WorktreeManager`
- [ ] Add CLI flag: `--no-worktree`
- [ ] Handle dependency injection

**Tests**:
- `TestCLIWorktreeFlags` - Flag parsing
- `TestCLIWorktreeDisabled` - `--no-worktree` flag behavior

### Phase 5: Integration Testing
**Goal**: End-to-end validation
- [ ] Create test repository setup helpers
- [ ] Write integration tests with real git operations
- [ ] Test worktree isolation

**Tests** (build tag `e2e`):
- `TestRiverCreatesWorktree` - Complete workflow in isolated worktree
- `TestRiverWorktreeCleanup` - Proper cleanup after completion/failure

## Test Strategy

### Unit Tests
- **gitx package**: Test git operations with temporary repositories
- **workflow package**: Test integration with mock `WorktreeManager`
- **config package**: Test configuration loading and validation

### Integration Tests  
- Use temporary git repositories
- Test real git worktree operations
- Verify isolation between worktrees

### Mocking Strategy
- Mock `WorktreeManager` for workflow tests
- Mock `ClaudeExecutor` for git integration tests
- Use dependency injection throughout

## Integration Points

### Workflow Engine
```go
// Engine constructor changes
func NewEngine(exec ClaudeExecutor, wtMgr gitx.WorktreeManager, cfg *config.Config) *Engine

// Run() method hooks
func (e *Engine) Run(ctx context.Context, taskDescription string, generatePlan bool) error {
    // Before workflow: Create worktree if enabled
    if e.cfg.Git.WorktreeEnabled {
        wt, err := e.wtMgr.Create(ctx, taskDescription)
        if err != nil { return err }
        e.wt = wt
        
        // Change working directory to worktree
        if err := os.Chdir(wt.Path); err != nil {
            return fmt.Errorf("failed to change to worktree directory: %w", err)
        }
        
        // Update state file path
        e.stateFile = filepath.Join(wt.Path, "claude_state.json")
    }
    
    // ... existing workflow logic runs in worktree ...
    
    // After completion: Optional cleanup
    if e.wt != nil && e.cfg.Git.AutoCleanupWT {
        defer e.wtMgr.Cleanup(ctx, e.wt)
    }
}
```

### Core Worktree Operations
```go
// Create worktree with new branch
func (m *CLIWorktreeManager) Create(ctx context.Context, taskName string) (*Worktree, error) {
    branch := fmt.Sprintf("river/%s", sanitizeTaskName(taskName))
    repoName := filepath.Base(m.parentRepo)
    wtPath := filepath.Join("..", fmt.Sprintf("%s-river-%s", repoName, slugify(taskName)))
    
    // git worktree add -B <branch> <path> <base-branch>
    cmd := exec.CommandContext(ctx, "git", "worktree", "add", "-B", branch, wtPath, m.baseBranch)
    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("failed to create worktree: %w", err)
    }
    
    return &Worktree{
        Path:       wtPath,
        Branch:     branch,
        ParentRepo: m.parentRepo,
    }, nil
}
```
