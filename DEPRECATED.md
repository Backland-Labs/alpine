# DEPRECATED - Python Version

⚠️ **This Python implementation (main.py) is deprecated as of v0.2.0**

The River CLI has been completely rewritten in Go for better performance and reliability.

## Migration to Go Version

Please use the new Go implementation instead:

### Installation

```bash
# Download the latest release from GitHub
# https://github.com/[your-username]/river/releases/latest

# Or build from source
go build -o river cmd/river/main.go
```

### New Usage

The Go version has a simpler, more intuitive interface:

```bash
# Old Python usage (DEPRECATED):
python main.py
# Enter the initial prompt for Claude Code: ABC-123
# Do you need a plan? (True/False): true

# New Go usage:
river "Implement user authentication"
river "Implement user authentication" --no-plan
river --file task.md
```

### Key Improvements in Go Version

1. **No Linear API Dependency**: Works with direct task descriptions
2. **Better Performance**: ~5x faster startup, ~50% less memory usage
3. **Single Binary**: No Python dependencies required
4. **Enhanced UX**: Colored output, progress indicators, better error messages
5. **Cross-Platform**: Native binaries for Linux, macOS, and Windows

### Migration Guide

For detailed migration instructions, see [MIGRATION.md](MIGRATION.md).

## Support

The Python version (main.py) will no longer receive updates or bug fixes. All development efforts are focused on the Go implementation.

If you encounter any issues with the migration, please open an issue on GitHub.