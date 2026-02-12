# Critical Patterns -- Required Reading

These patterns must be followed in every code change. Subagents should review this before generating code.

---

## 1. One Concern Per File, Named For What It Contains (ALWAYS REQUIRED)

### WRONG (Agents waste context reading hundreds of irrelevant lines)
```
cmd/alpine/
├── main.go       128 lines (config + errors + output + root cmd)
├── docker.go     735 lines (exec + docker + compose + dockerfile + git + container + dotenv)
└── status.go     146 lines (status command + container inspection + process checking)
```

### CORRECT
```
cmd/alpine/
├── main.go           63 lines  -- root command, flags, execute()
├── config.go         62 lines  -- Config struct, load, validate
├── errors.go         21 lines  -- exitError, userErr, sysErr
├── output.go         30 lines  -- outputJSON, outputError
├── exec.go           97 lines  -- run(), runInteractive(), ExecError
├── docker.go        159 lines  -- health check, compose up/down, image exists
├── compose.go       178 lines  -- Compose YAML templates + generation
├── dockerfile.go     47 lines  -- Dockerfile generation + hash
├── git.go            83 lines  -- clone, branch, configure, find root
├── container.go      57 lines  -- inspect, copy files, check processes
├── dotenv.go         48 lines  -- loadDotEnv()
└── status.go        105 lines  -- status command only
```

**Why:** AI coding agents navigate by listing directories and reading filenames. A file named `git.go` tells the agent "git operations live here" before it reads a single line. A 735-line `docker.go` tells it nothing. Every file should have one concern, named for that concern. Keep files under ~200 lines. Each test file mirrors its source file (`git.go` -> `git_test.go`).

**Placement/Context:** Every new file added to the codebase. When adding new functionality, create a new file rather than appending to an existing one that handles a different concern. If a file starts doing two unrelated things, split it immediately.

**Documented in:** `docs/solutions/developer-experience/god-file-refactor-to-single-concern-files-alpine-cli-20260212.md`
