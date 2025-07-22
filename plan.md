# Git Worktree Integration - Implementation Plan

## Overview
Add git worktree support to River CLI enabling isolated task execution in separate directories with automatic branch management, commits, and PR creation.

## Architecture Changes

### New Package: `internal/gitx/`
- **interfaces.go**: Core abstractions
  ```go
  type WorktreeManager interface {
      Create(ctx context.Context, taskName string) (*Worktree, error)
      CommitAll(ctx context.Context, wt *Worktree, msg string) error
      Push(ctx context.Context, wt *Worktree) error
      CreatePR(ctx context.Context, wt *Worktree) (url string, err error)
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
    Remote          string // default="origin" 
    BaseBranch      string // default="main"
    GHEnabled       bool   // create PR with gh CLI
    AutoCleanupWT   bool   // default=true
}
```

### Workflow Engine Changes (`internal/workflow/`)
- Inject `WorktreeManager` into `Engine` constructor
- Add worktree lifecycle hooks:
  - **Pre-workflow**: Create worktree, change working directory
  - **Post-workflow**: Commit, push, create PR, cleanup
- Update state file path to live within worktree

## Implementation Phases

### Phase 1: Configuration Groundwork
**Goal**: Add git configuration support
- [ ] Extend `Config` struct with `GitConfig`
- [ ] Add environment variables: `RIVER_GIT_ENABLED`, `RIVER_GIT_REMOTE`, etc.
- [ ] Write configuration unit tests
- [ ] Update config loading logic

**Tests**:
- `TestConfigGitDefaults`
- `TestConfigGitEnvironmentVariables`
- `TestConfigGitValidation`

### Phase 2: WorktreeManager Implementation  
**Goal**: Core git worktree operations
- [ ] Create `internal/gitx/` package
- [ ] Implement `WorktreeManager` interface with CLI backend
- [ ] Add task name sanitization logic
- [ ] Create mock implementation for testing

**Tests**:
- `TestCreateWorktree_basic` - Creates directory, branch, registers worktree
- `TestCommitAndPush` - Commits changes and pushes to remote
- `TestCleanup_removesWorktreeDirAndPrunes` - Proper cleanup
- `TestBranchNameCollisionProducesUniqueNames` - Handles conflicts
- `TestErrorPropagation_pushFailure` - Error handling

### Phase 3: Engine Integration
**Goal**: Wire worktree lifecycle into workflow engine
- [ ] Add `WorktreeManager` field to `Engine`
- [ ] Implement `createWorktree()` helper method
- [ ] Implement `finalizeWorktree()` helper method  
- [ ] Update `Run()` method with worktree hooks
- [ ] Handle working directory changes

**Tests**:
- `TestEngineCreatesAndCleansWorktree` - Full lifecycle with mock manager
- `TestEngineCleanupOnInterrupt` - Cleanup on context cancellation
- `TestEngineWorktreeDisabled` - Graceful fallback when disabled

### Phase 4: CLI Integration
**Goal**: Wire everything together at CLI level
- [ ] Update `NewRealDependencies()` to create `WorktreeManager`
- [ ] Add CLI flags: `--no-worktree`, `--git-base-branch`
- [ ] Update command help and documentation
- [ ] Handle dependency injection

**Tests**:
- `TestCLIWorktreeFlags` - Flag parsing
- `TestCLIWorktreeDisabled` - `--no-worktree` flag behavior

### Phase 5: Integration Testing
**Goal**: End-to-end validation
- [ ] Create test repository setup helpers
- [ ] Write integration tests with real git operations
- [ ] Add smoke tests for PR creation
- [ ] Performance testing with large repositories

**Tests** (build tag `e2e`):
- `TestRiverCreatesPR` - Complete workflow including PR creation
- `TestRiverParallelWorktrees` - Multiple tasks don't conflict
- `TestRiverWorktreeCleanup` - Proper cleanup after completion/failure

## Test Strategy

### Unit Tests
- **gitx package**: Test git operations with temporary repositories
- **workflow package**: Test integration with mock `WorktreeManager`
- **config package**: Test configuration loading and validation

### Integration Tests  
- Use temporary git repositories with bare remotes
- Test real git operations but avoid external dependencies
- Mock GitHub API calls for PR creation tests

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
        os.Chdir(wt.Path) // Change working directory
    }
    
    // ... existing workflow logic ...
    
    // After completion: Finalize worktree
    if e.wt != nil {
        defer e.finalizeWorktree(ctx)
    }
}
```

### State File Management
- State file path becomes `{worktree_path}/claude_state.json`
- Each task maintains isolated state
- No changes to `core.State` structure needed

## Error Handling

### Git Operation Failures
- Graceful degradation when git commands fail
- Clear error messages for common issues (no git, no remote, etc.)
- Automatic cleanup on failures

### Worktree Conflicts
- Generate unique branch names with timestamps
- Handle existing worktree directories
- Provide clear conflict resolution guidance

## Future Enhancements

### Not in Scope (Phase 1)
- [ ] Parallel task orchestration (`river queue` command)
- [ ] Live commits after each Claude iteration  
- [ ] Advanced PR template customization
- [ ] Integration with other git hosting providers

### Configuration Questions
1. Should we support custom worktree directory patterns?
2. Do you want automatic PR creation enabled by default?
3. Should we support custom commit message templates?
4. Any specific requirements for branch naming conventions?

## Dependencies
- System `git` binary (required)
- GitHub CLI `gh` (optional, for PR creation)
- Existing River architecture (no breaking changes)
