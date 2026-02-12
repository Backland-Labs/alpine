---
date: 2026-02-12
topic: test-suite-100-coverage
---

# Test Suite: 100% Coverage with Speed

## What We're Building

A comprehensive test suite for the Alpine CLI (~1,700 lines of Go, 5 files, all `package main`). The hard constraint is 100% code coverage from unit tests alone, with test execution speed as the top priority.

## Why This Approach

Three strategies were considered:

- **Interface + mock (chosen):** Extract a `CommandRunner` interface from the central `run()` function. Unit tests inject a mock that returns canned responses. Zero external dependencies, sub-second execution, 100% coverage achievable. Requires moderate refactoring.
- **exec.Command test helper:** No source changes, but brittle test code that's hard to maintain. Rejected for maintainability concerns.
- **Real Docker integration:** Highest confidence, but slow and flaky. Rejected as the primary strategy.

**Hybrid decision:** Interface + mock for unit tests (speed + coverage), plus a small set of real Docker integration tests behind a build tag for confidence on critical paths.

## Key Decisions

- **Mock strategy:** Interface + dependency injection (idiomatic Go, improves code structure)
- **Coverage target:** 100% from unit tests alone (strict standard)
- **Integration tests:** Opt-in via `go test -tags=integration` (default `go test` stays fast)
- **Test location:** Same package (`package main`, required by code structure)
- **Coverage enforcement:** CI fails if coverage < 100%

## Test File Structure

| File | Covers | Dependencies |
|------|--------|-------------|
| `docker_test.go` | `run()`, Docker/Compose helpers, Dockerfile/YAML generation | Mock runner for commands, none for pure functions |
| `create_test.go` | 15-step create workflow, rollback, error paths | Mock runner |
| `list_test.go` | Parsing, formatting, label extraction | Mock runner + pure function tests |
| `status_test.go` | Container inspection, process checking | Mock runner |
| `main_test.go` | Config loading, validation, CLI setup | Temp files for config |
| `integration_test.go` | Real `alpine create` + `list` + `status` | Real Docker (build-tagged) |

## Open Questions

- Exact shape of the `CommandRunner` interface (function type vs interface with methods)
- Whether to use a struct to hold the runner or pass it through function parameters
- How to handle `os.Exit` calls in `userErr()`/`sysErr()` for testability

## Next Steps

- `/workflows:plan` for implementation details
