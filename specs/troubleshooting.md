# Troubleshooting Guide

This guide covers common issues and their solutions when using River.

## Common Issues

### Claude Command Not Found

**Symptom**: Error message "claude: command not found" or similar

**Solution**:
1. Ensure Claude Code CLI is installed: https://docs.anthropic.com/en/docs/claude-code
2. Verify installation: `claude --version`
3. Check PATH includes Claude installation directory
4. On macOS/Linux: `export PATH="$PATH:/path/to/claude"`

### State File Conflicts

**Symptom**: "State file is locked" or unexpected workflow behavior

**Solution**:
1. Check if another River instance is running: `ps aux | grep river`
2. Remove stale state file if needed: `rm -rf .claude/river/claude_state.json`
3. State file location is now fixed at `.claude/river/claude_state.json` to avoid conflicts

### Task Not Progressing

**Symptom**: River seems stuck, state file not updating

**Possible Causes & Solutions**:

1. **Claude Code is waiting for input**
   - Check if Claude is prompting for confirmation
   - Run with `RIVER_SHOW_OUTPUT=true` to see Claude's output

2. **State file permissions**
   - Check file permissions: `ls -la .claude/river/claude_state.json`
   - Ensure write permissions: `chmod 644 .claude/river/claude_state.json`

3. **Slash commands not working**
   - Verify Claude Code supports required slash commands
   - Check Claude Code version is up to date

### Color Output Issues

**Symptom**: Seeing escape codes like `\033[32m` instead of colors

**Solution**:
1. Check terminal supports colors: `echo $TERM`
2. For Windows: Use Windows Terminal or enable ANSI colors
3. Disable colors if needed: `export NO_COLOR=1`
4. Force color output: `export FORCE_COLOR=1`

### Performance Issues

**Symptom**: River runs slowly or uses excessive resources

**Solutions**:
1. **Check disk space**: Ensure sufficient space for state file operations
2. **Reduce verbosity**: Use `RIVER_VERBOSITY=normal` (not debug)
3. **Monitor Claude execution**: Claude Code operations may be slow
4. **Use --no-plan**: Skip planning phase for simple tasks

### File Input Problems

**Symptom**: Error reading task file with `--file` flag

**Common Issues**:
1. **File not found**: Use absolute paths or check working directory
2. **Empty file**: Ensure file contains task description
3. **Encoding issues**: Save file as UTF-8
4. **Permissions**: Check file is readable

Example fix:
```bash
# Use absolute path
river --file /home/user/tasks/my-task.md

# Check file content
cat my-task.md

# Fix permissions
chmod 644 my-task.md
```

### Environment Variable Issues

**Symptom**: Configuration not being applied

**Debugging Steps**:
1. Verify environment variables are set:
   ```bash
   env | grep RIVER
   ```

2. Check for typos in variable names:
   - ✅ `RIVER_WORK_DIR`
   - ❌ `RIVER_WORKDIR`

3. Export variables properly:
   ```bash
   # Correct
   export RIVER_VERBOSITY=debug
   
   # Incorrect (not exported)
   RIVER_VERBOSITY=debug
   ```

### Signal Handling Issues

**Symptom**: Can't interrupt River with Ctrl+C

**Solution**:
1. Try Ctrl+C multiple times (handled gracefully)
2. As last resort: `kill -9 $(pgrep river)`
3. Clean up state file after force kill

### Debug Mode

For detailed troubleshooting, enable debug mode:
```bash
export RIVER_VERBOSITY=debug
export RIVER_SHOW_OUTPUT=true
river "Your task" 2>&1 | tee river-debug.log
```

This creates a log file for analysis.

## Platform-Specific Issues

### macOS

**Issue**: "cannot execute binary file"
- Check architecture: `file river`
- Download correct version (arm64 for M1/M2, amd64 for Intel)

**Issue**: "Operation not permitted"
- Remove quarantine: `xattr -d com.apple.quarantine river`
- Or allow in System Preferences > Security & Privacy

### Linux

**Issue**: "No such file or directory" for valid binary
- Check architecture: `uname -m`
- Install 32-bit libraries if needed: `sudo apt-get install libc6-i386`

**Issue**: Permission denied
- Make executable: `chmod +x river`
- Check mount options if on external drive

### Windows

**Issue**: "This app can't run on your PC"
- Verify 64-bit Windows
- Download Windows-specific binary (.exe)

**Issue**: Path issues
- Use forward slashes or escaped backslashes in paths
- Example: `C:/Users/Name/task.md` or `C:\\Users\\Name\\task.md`

## Getting Help

### Diagnostic Information

When reporting issues, include:
1. River version: `river --version`
2. OS and architecture: `uname -a` (Unix) or `systeminfo` (Windows)
3. Claude Code version: `claude --version`
4. Environment variables: `env | grep RIVER`
5. Debug log (see Debug Mode section)

### Support Channels

1. **GitHub Issues**: https://github.com/[username]/river/issues
2. **Debug Logs**: Run with `RIVER_VERBOSITY=debug` and attach output
3. **State File**: Include `.claude/river/claude_state.json` content if relevant

### Quick Fixes Checklist

- [ ] Claude Code is installed and in PATH
- [ ] No other River instances running
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