# Migration Guide: Python to Go Version

This guide helps users migrate from the Python version of River to the new Go implementation.

## Key Changes

### 1. Binary Distribution

**Python Version**: Required Python 3.x and pip installation
```bash
pip install -r requirements.txt
python main.py ABC-123
```

**Go Version**: Single standalone binary
```bash
./river "Implement user authentication"
```

### 2. Task Input Method

**Python Version**: Required Linear issue IDs
```bash
python main.py ABC-123              # Fetch from Linear API
python main.py ABC-123 --no-plan    # Required Linear API key
```

**Go Version**: Direct task descriptions
```bash
river "Implement user authentication"       # Direct description
river "Fix payment bug" --no-plan          # No API needed
river --file task.md                       # From file
```

### 3. Configuration

**Python Version**: Mix of environment variables and command-line flags
```bash
export LINEAR_API_KEY=lin_api_xxx
export WORK_DIR=/path/to/work
python main.py ABC-123
```

**Go Version**: Environment variables only (no API keys needed)
```bash
export RIVER_WORK_DIR=/path/to/work
export RIVER_VERBOSITY=debug
river "Build REST API"
```

### 4. Removed Dependencies

The Go version removes these dependencies:
- Linear API integration (no more API keys)
- Python runtime and packages
- Network calls for issue fetching

## Migration Steps

### Step 1: Install Go Version

1. Download the binary for your platform from the releases page
2. Make it executable: `chmod +x river` (Unix/macOS)
3. Optionally move to PATH: `sudo mv river /usr/local/bin/`

### Step 2: Update Environment Variables

Remove Linear-specific variables:
```bash
# Remove these
unset LINEAR_API_KEY
unset LINEAR_TEAM_ID

# Update these with RIVER_ prefix
export RIVER_WORK_DIR="$WORK_DIR"
export RIVER_VERBOSITY="$VERBOSITY"
export RIVER_SHOW_OUTPUT="$SHOW_OUTPUT"
```

### Step 3: Convert Task Workflow

**Old workflow** (with Linear):
1. Create Linear issue
2. Copy issue ID (e.g., ABC-123)
3. Run: `python main.py ABC-123`

**New workflow** (direct):
1. Run: `river "Your task description here"`

### Step 4: Update Scripts/Aliases

If you have scripts or aliases using the Python version:

**Before**:
```bash
alias plan='python ~/river/main.py'
plan ABC-123
```

**After**:
```bash
alias plan='river'
plan "Implement feature X"
```

### Step 5: Handle Existing State Files

The state file format remains the same (`claude_state.json`). However, existing state files from Linear-based runs will reference Linear issues. You can:

1. Let them complete naturally
2. Or clean them up: `rm claude_state.json`

## Feature Comparison

| Feature | Python | Go | Notes |
|---------|--------|-----|-------|
| Task input | Linear ID | Direct text | More flexible |
| File input | ❌ | ✅ | `--file` flag |
| Linear API | ✅ Required | ❌ Removed | No API dependencies |
| Color output | Basic | Enhanced | Auto-detection |
| Progress indicators | ❌ | ✅ | With timing |
| Debug logging | Basic | Enhanced | Timestamped |
| Binary size | N/A | ~6MB | Standalone |
| Startup time | ~500ms | ~100ms | 5x faster |

## Common Migration Issues

### Issue: "Linear issue ID required"
**Solution**: The Go version doesn't use Linear IDs. Provide the task description directly:
```bash
# Instead of: river ABC-123
river "The actual task you want to accomplish"
```

### Issue: Missing LINEAR_API_KEY error
**Solution**: This error shouldn't occur in the Go version. If you see it, ensure you're using the Go binary, not the Python script.

### Issue: Different command syntax
**Solution**: The basic syntax is the same, but instead of issue IDs, use task descriptions:
```bash
# Python: python main.py ABC-123 --no-plan
# Go:     river "Task description" --no-plan
```

## Benefits After Migration

1. **No External Dependencies**: No need for Linear API access
2. **Faster Execution**: 5x faster startup, 50% less memory
3. **Simpler Setup**: Just download and run - no Python environment needed
4. **Better UX**: Colored output, progress indicators, better error messages
5. **More Flexible**: Work with any task, not just Linear issues

## Rollback Plan

If you need to temporarily switch back to Python:
1. The Python version remains compatible with existing workflows
2. State files are compatible between versions
3. Keep both versions during transition period

## Need Help?

- Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common issues
- Report bugs via [GitHub Issues](https://github.com/[username]/river/issues)
- The Python version will remain available in the `python-legacy` branch