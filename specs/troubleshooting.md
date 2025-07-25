# Troubleshooting Guide

This guide covers common issues and their solutions when using Alpine.

## Common Issues

### Claude Command Not Found

**Symptom**: Error message "claude: command not found" or similar

**Solution**:
1. Ensure Claude Code CLI is installed: https://docs.anthropic.com/en/docs/claude-code
2. Verify installation: `claude --version`
3. Check PATH includes Claude installation directory.

### State File Conflicts

**Symptom**: "State file is locked" or unexpected workflow behavior.

**Solution**:
1. Check if another Alpine instance is running: `ps aux | grep alpine`
2. If no other instance is running, you can safely remove the state directory: `rm -rf agent_state/`
3. The state file is located at `agent_state/agent_state.json`. For more details, see the [System Design](system-design.md#4-state-management) specification.

### Task Not Progressing

**Symptom**: Alpine seems stuck, state file not updating.

**Possible Causes & Solutions**:
1. **Claude Code is waiting for input**: Check if Claude is prompting for confirmation. Run with `ALPINE_SHOW_OUTPUT=true` to see Claude's output. See [Configuration](system-design.md#3-configuration) for more output options.
2. **State file permissions**: Check file permissions: `ls -la agent_state/agent_state.json`.
3. **Slash commands not working**: Verify your Claude Code version is up to date and supports the required slash commands.

### Color Output Issues

**Symptom**: Seeing escape codes like `\033[32m` instead of colors.

**Solution**:
1. Check if your terminal supports colors (`echo $TERM`).
2. For Windows, use Windows Terminal or a terminal that supports ANSI escape codes.
3. You can disable colors with `export NO_COLOR=1` or force them with `export FORCE_COLOR=1`.

### Performance Issues

**Symptom**: Alpine runs slowly or uses excessive resources.

**Solutions**:
1. **Check disk space**: Ensure sufficient space for worktree and state file operations.
2. **Reduce verbosity**: Use `ALPINE_VERBOSITY=normal`. Debug mode can be slow.
3. **Use --no-plan**: Skip the planning phase for simple tasks. See [CLI Commands](cli-commands.md).

### File Input Problems

**Symptom**: Error reading task file with `--file` flag.

**Common Issues**:
1. **File not found**: Use absolute paths or ensure the file is in the current working directory.
2. **Permissions**: Check that the file is readable (`chmod 644 my-task.md`).
3. For more details, see the [CLI Commands](cli-commands.md) specification.

### Environment Variable Issues

**Symptom**: Configuration is not being applied as expected.

**Debugging Steps**:
1. Verify environment variables are set and exported correctly: `env | grep ALPINE`
2. Check for typos in variable names (e.g., `ALPINE_WORKDIR`).
3. For a full list of variables, see the [Configuration](system-design.md#3-configuration) section in the system design spec.

Example:
```bash
# Correct: exports the variable to the current shell session
export ALPINE_VERBOSITY=debug

# Incorrect: only sets the variable for the next command
ALPINE_VERBOSITY=debug alpine "My task"
```

### Signal Handling Issues

**Symptom**: Can't interrupt Alpine with Ctrl+C.

**Solution**:
1. Alpine is designed to handle `Ctrl+C` gracefully by saving state and exiting. Try it a couple of times.
2. As a last resort, you may need to kill the process: `kill -9 $(pgrep alpine)`.
3. You may need to clean up the `agent_state/` directory after a force kill.

### Debug Mode

For detailed troubleshooting, enable debug mode. This will create a log file for analysis.
```bash
export ALPINE_VERBOSITY=debug
export ALPINE_SHOW_OUTPUT=true
alpine "Your task" 2>&1 | tee alpine-debug.log
```

## Platform-Specific Issues

### macOS

**Issue**: "cannot execute binary file"
- Check architecture: `file alpine`
- Download correct version (arm64 for M1/M2, amd64 for Intel)

**Issue**: "Operation not permitted"
- Remove quarantine: `xattr -d com.apple.quarantine alpine`
- Or allow in System Preferences > Security & Privacy

### Linux

**Issue**: "No such file or directory" for valid binary
- Check architecture: `uname -m`
- Install 32-bit libraries if needed: `sudo apt-get install libc6-i386`

**Issue**: Permission denied
- Make executable: `chmod +x alpine`
- Check mount options if on external drive

### Windows

**Issue**: "This app can't run on your PC"
- Verify 64-bit Windows
- Download Windows-specific binary (.exe)

**Issue**: Path issues
- Use forward slashes or escaped backslashes in paths
- Example: `C:/Users/Name/task.md` or `C:\Users\Name\task.md`

## Getting Help

### Diagnostic Information

When reporting issues, include:
1. Alpine version: `alpine --version`
2. OS and architecture: `uname -a` (Unix) or `systeminfo` (Windows)
3. Claude Code version: `claude --version`
4. Relevant environment variables: `env | grep ALPINE`
5. Debug log (see Debug Mode section)

### Support Channels

1. **GitHub Issues**: https://github.com/[username]/alpine/issues
2. **Debug Logs**: Run with `ALPINE_VERBOSITY=debug` and attach output.
3. **State File**: Include `agent_state/agent_state.json` content if relevant.

### Quick Fixes Checklist

- [ ] Claude Code is installed and in PATH
- [ ] No other Alpine instances running
- [ ] State file has write permissions
- [ ] Using correct binary for your platform
- [ ] Environment variables are exported
- [ ] Terminal supports UTF-8 and colors
- [ ] Sufficient disk space available
- [ ] Task description is not empty

## Error Messages Reference

| Error | Cause | Solution |
|-------|-------|----------|
| "task description required" | No task provided | Provide task as argument or via --file |
| "failed to read task file" | File not found or not readable | Check file path and permissions |
| "failed to create state file" | Permission or disk space issue | Check directory permissions and disk space |
| "failed to execute claude" | Claude not found or not executable | Install Claude Code CLI |
| "context canceled" | Operation interrupted | Normal when using Ctrl+C |
| "invalid state file format" | Corrupted JSON | Remove state file and restart |
