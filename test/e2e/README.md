# End-to-End Tests

This directory contains end-to-end tests for River's git worktree functionality.

## Running E2E Tests

These tests are tagged with the `e2e` build tag and are not run by default. They require:
- A working git installation
- Ability to create temporary directories and git repositories
- The River binary to be buildable

To run the e2e tests:

```bash
# Run all e2e tests
go test -tags=e2e ./test/e2e/...

# Run with verbose output
go test -v -tags=e2e ./test/e2e/...

# Run a specific test
go test -tags=e2e -run TestRiverCreatesWorktree ./test/e2e/
```

## Test Coverage

The e2e tests cover:
- **Worktree Creation**: Verifies worktrees are created with proper naming and structure
- **Worktree Cleanup**: Tests auto-cleanup behavior on success/failure
- **Worktree Disabled**: Tests the `--no-worktree` flag functionality
- **Worktree Isolation**: Ensures changes in worktrees don't affect the main repository
- **Branch Naming**: Tests branch name sanitization and collision handling
- **Environment Variables**: Tests git configuration via environment variables

## Test Implementation

The tests use:
- Temporary git repositories created for each test
- Mock Claude scripts that simulate Claude's behavior
- The actual River binary built from source
- Real git operations (not mocked)

## Debugging

If tests fail:
1. Check git is installed and accessible: `git --version`
2. Run tests with `-v` flag for verbose output
3. Examine test output for git command failures
4. Check temporary directories aren't being cleaned up prematurely