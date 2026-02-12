---
module: System
date: 2026-02-12
problem_type: developer_experience
component: tooling
symptoms:
  - "docker.go at 735 lines contained 7 unrelated concerns (shell exec, Docker ops, Compose gen, Dockerfile gen, git ops, container ops, dotenv)"
  - "main.go at 128 lines mixed config, errors, output, and root command"
  - "Agent tools must read entire 735-line file to find a single function"
root_cause: incomplete_setup
resolution_type: workflow_improvement
severity: medium
tags: [file-structure, agent-navigation, refactor, single-responsibility, codebase-organization]
---

# Troubleshooting: God Files Hurt Agent Navigation -- Split by Concern

## Problem

AI coding agents navigate codebases by listing directories and reading filenames. A 735-line `docker.go` containing shell execution, Docker daemon ops, Compose YAML generation, Dockerfile generation, git operations, container interaction, and .env loading gave agents zero signal about what was inside until they read the whole file.

## Environment
- Module: System (all of cmd/alpine/)
- Go Version: 1.22
- Affected Component: CLI codebase file structure
- Date: 2026-02-12

## Symptoms
- `docker.go` at 735 lines was the largest file, doing 7 unrelated things
- `main.go` at 128 lines mixed config loading, error types, output formatting, and root command setup
- `status.go` owned `inspectContainer` and `checkClaudeProcess` which are container concerns, not status-command concerns
- `create.go` owned `userErr` and `sysErr` which are error-system concerns, not create-command concerns
- An agent needing to understand "how does git work in this tool" had to read 735 lines of docker.go

## What Didn't Work

**Direct solution:** The problem was identified and fixed on the first attempt.

## Solution

Split 6 source files into 15, each named for its single concern. Every filename became a table-of-contents entry.

**Before (6 files):**
```
cmd/alpine/
├── main.go          128 lines (config, errors, output, root cmd)
├── main_entry.go     26 lines
├── create.go        495 lines
├── docker.go        735 lines (everything)
├── list.go          221 lines
└── status.go        146 lines
```

**After (15 files):**
```
cmd/alpine/
├── main.go           63 lines  -- root command, flags, execute()
├── main_entry.go     26 lines  -- OS entrypoint
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
├── create.go        485 lines  -- create workflow
├── list.go          221 lines  -- list command
└── status.go        105 lines  -- status command
```

**Key moves:**
- `docker.go` (735 lines) -> 7 files: `exec.go`, `docker.go` (159), `compose.go`, `dockerfile.go`, `git.go`, `container.go`, `dotenv.go`
- `main.go` (128 lines) -> 4 files: `main.go` (63), `config.go`, `errors.go`, `output.go`
- `status.go` gave `inspectContainer`/`checkClaudeProcess` to `container.go`
- `create.go` gave `userErr`/`sysErr` to `errors.go`
- Test files reorganized 1:1 to mirror source files

**What was NOT changed:**
- Zero behavior changes -- same functions, same signatures, same package
- No new abstractions, interfaces, or indirection layers
- No renaming of functions or types
- 97.3% test coverage preserved exactly

## Why This Works

The main mechanism agentic tools use to navigate a codebase is the filesystem. They list directories, read filenames, search for strings, and pull files into context. A file called `git.go` communicates "git operations live here" before an agent reads a single line. A file called `docker.go` at 735 lines communicates nothing.

By splitting along concern boundaries:
1. **Filenames become documentation** -- an agent searching for "how does Compose YAML get built" finds `compose.go` immediately
2. **Context windows stay small** -- reading `git.go` (83 lines) costs far less than reading `docker.go` (735 lines) to find the same 6 functions
3. **Grep results are meaningful** -- searching for `generateComposeYAML` returns `compose.go`, not the generic `docker.go`

The convention added to CLAUDE.md ("one concern per file, named for what it contains") ensures future code follows the same pattern.

## Prevention

- Name files for the single concern they contain (e.g., `git.go` not `utils.go`)
- If a file does two unrelated things, split it before it grows
- Each test file mirrors its source file (`git.go` -> `git_test.go`)
- Keep files under ~200 lines; if larger, check if it has multiple concerns
- When adding new functionality, create a new file rather than appending to an existing one that handles a different concern

## Related Issues

- Promoted to Required Reading: [Critical Pattern #1](../patterns/critical-patterns.md)
