package runner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// RunAutoClaudeScript copies and executes the auto_claude.sh script in the specified worktree
func RunAutoClaudeScript(worktreePath, issueID, scriptPath string) error {
	// Copy auto_claude.sh to worktree
	destPath := filepath.Join(worktreePath, "auto_claude.sh")
	if err := copyFile(scriptPath, destPath); err != nil {
		return fmt.Errorf("failed to copy script: %w", err)
	}

	// Create command to run the script
	cmd := exec.Command("/bin/bash", "auto_claude.sh")
	cmd.Dir = worktreePath

	// Set up stdin to pass issueID
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Stream stdout and stderr to console
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start script: %w", err)
	}

	// Write issueID to stdin
	if _, err := fmt.Fprintln(stdin, issueID); err != nil {
		return fmt.Errorf("failed to write issue ID to stdin: %w", err)
	}
	stdin.Close()

	// Wait for the command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("script execution failed: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst, preserving permissions
func copyFile(src, dst string) error {
	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Get source file info for permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Create destination file
	destFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy file contents
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Ensure all data is written
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	return nil
}