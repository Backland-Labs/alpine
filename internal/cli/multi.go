package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
)

// multiCmd represents the multi command
type multiCmd struct {
	cmd *cobra.Command
}

// NewMultiCommand creates a new multi command (exported for tests)
func NewMultiCommand() *cobra.Command {
	return newMultiCmd().Command()
}

// newMultiCmd creates a new multi command
func newMultiCmd() *multiCmd {
	mc := &multiCmd{}

	mc.cmd = &cobra.Command{
		Use:   "multi [path task]...",
		Short: "Run multiple River agents in parallel",
		Long: `Run multiple River agents in different codebases simultaneously.
Each agent runs in its own process with isolated state.

Example:
  river multi ~/code/frontend "upgrade to React 18" ~/code/backend "add authentication"`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("requires at least one <path> <task> pair")
			}
			if len(args)%2 != 0 {
				return fmt.Errorf("requires pairs of <path> <task> arguments")
			}
			return nil
		},
		RunE: mc.execute,
	}

	return mc
}

// Command returns the cobra command
func (mc *multiCmd) Command() *cobra.Command {
	return mc.cmd
}

// execute runs the multi command
func (mc *multiCmd) execute(cmd *cobra.Command, args []string) error {
	// Validate arguments
	if err := ValidateArguments(args); err != nil {
		return err
	}

	// Parse path/task pairs
	pairs := ParsePathTaskPairs(args)

	// Spawn River processes
	return SpawnRiverProcesses(pairs)
}

// PathTaskPair represents a project path and task description
type PathTaskPair struct {
	Path string
	Task string
}

// ValidateArguments ensures arguments come in pairs
func ValidateArguments(args []string) error {
	if len(args)%2 != 0 {
		return fmt.Errorf("arguments must be in pairs of [path task]")
	}
	return nil
}

// ParsePathTaskPairs converts arguments into path/task pairs
func ParsePathTaskPairs(args []string) []PathTaskPair {
	var pairs []PathTaskPair
	for i := 0; i < len(args); i += 2 {
		pairs = append(pairs, PathTaskPair{
			Path: args[i],
			Task: args[i+1],
		})
	}
	return pairs
}

// ExtractProjectName gets the project name from a path
func ExtractProjectName(path string) string {
	// Handle special cases first
	if path == "/" {
		return "root"
	}

	if path == "." {
		// For current directory, just return "."
		return "."
	}

	// Remove trailing slashes
	path = filepath.Clean(path)

	// Get the base name
	base := filepath.Base(path)

	// If we get an empty string or slash, use "project" as fallback
	if base == "" || base == "/" {
		return "project"
	}

	return base
}

// SpawnRiverProcesses starts River processes for each path/task pair
func SpawnRiverProcesses(pairs []PathTaskPair) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(pairs))

	// Determine if we should use color based on terminal support
	useColor := isTerminalColor()

	for i, pair := range pairs {
		wg.Add(1)

		go func(index int, p PathTaskPair) {
			defer wg.Done()

			// Extract project name from path
			projectName := ExtractProjectName(p.Path)

			// Create prefix writers for stdout and stderr
			stdoutWriter := NewPrefixWriter(os.Stdout, projectName, useColor, index)
			stderrWriter := NewPrefixWriter(os.Stderr, projectName, useColor, index)

			// Create the River command
			cmd := exec.Command("river", p.Task)
			cmd.Dir = p.Path
			cmd.Stdout = stdoutWriter
			cmd.Stderr = stderrWriter
			cmd.Stdin = os.Stdin

			// Run the command
			if err := cmd.Run(); err != nil {
				errChan <- fmt.Errorf("failed to run River in %s: %w", p.Path, err)
			}
		}(i, pair)
	}

	// Wait for all processes to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var firstErr error
	for err := range errChan {
		if firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// isTerminalColor checks if stdout is a terminal with color support
func isTerminalColor() bool {
	// Re-use the logic from the output package
	// Check if NO_COLOR env var is set
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check if stdout is a terminal
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
