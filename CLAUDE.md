# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a shell script automation project that integrates Linear (project management) with Claude Code to automate software development workflows. The main script `auto_claude.sh` processes Linear sub-issues and implements features using Test-Driven Development.

## Commands

### Running the Main Script
```bash
# Execute the automation workflow
./auto_claude.sh

# The script requires these dependencies to be installed:
# - claude (Claude Code CLI)
# - jq (JSON processor)
```

### Git Operations
```bash
# Check status
git status

# Stage and commit changes
git add .
git commit -m "commit message"
```

## Architecture

### Core Components

1. **auto_claude.sh** - Main automation script that orchestrates the entire workflow:
   - Fetches sub-issues from Linear using the API
   - Creates implementation plans via Claude Code
   - Implements features following TDD methodology
   - Updates Linear issues upon completion

### Key Functions in auto_claude.sh

- `fetch_sub_issues()` - Retrieves sub-issues for a given Linear parent issue
- `process_sub_issue()` - Main workflow for each sub-issue
- `create_implementation_plan()` - Uses Claude to generate detailed plans
- `implement_with_tdd()` - Executes TDD cycle (red-green-refactor)
- `update_linear_issue()` - Marks issues as complete in Linear

### Workflow Architecture

The script follows this high-level flow:
1. Fetch sub-issues from Linear (requires PARENT_ISSUE_ID environment variable)
2. For each sub-issue:
   - Create an implementation plan using Claude
   - Implement using TDD methodology
   - Update Linear issue status
3. Continue until all sub-issues are processed

### Environment Requirements

- `LINEAR_API_KEY` - Required for Linear API access
- `PARENT_ISSUE_ID` - The Linear parent issue ID to process sub-issues from
- System must have `claude` and `jq` commands available in PATH

## Development Notes

- The script uses sub-agents pattern where Claude instances are spawned for specific tasks
- All Linear API calls are made using curl with proper authentication
- The TDD implementation follows strict red-green-refactor cycles
- Error handling includes automatic retries for transient failures