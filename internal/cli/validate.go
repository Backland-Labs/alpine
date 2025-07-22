package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/maxmcd/river/internal/config"
	"github.com/maxmcd/river/internal/logger"
	"github.com/maxmcd/river/internal/validation"
	"github.com/spf13/cobra"
)

// NewValidateCommand creates the validate command
func NewValidateCommand() *cobra.Command {
	var pythonPath string
	var goPath string
	var workDir string
	var cleanup bool

	cmd := &cobra.Command{
		Use:   "validate <issue-id>",
		Short: "Validate feature parity between Python and Go implementations",
		Long: `Validate feature parity between Python and Go implementations
		
This command runs both the Python prototype and Go implementation with the same
Linear issue ID and compares their behavior, including:
- Generated Claude commands
- State file contents  
- Output and error messages`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]

			// Load configuration for logger initialization
			cfg, err := config.New()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Initialize logger
			logger.InitializeFromConfig(cfg)
			logger.Debugf("Starting validation for issue %s", issueID)

			// Validate Linear issue ID format
			if !isValidLinearID(issueID) {
				return fmt.Errorf("invalid Linear issue ID format: %s", issueID)
			}

			// Set default paths if not provided
			if pythonPath == "" {
				// Look for main.py in current directory or parent
				if _, err := os.Stat("main.py"); err == nil {
					pythonPath = "main.py"
				} else if _, err := os.Stat("../main.py"); err == nil {
					pythonPath = "../main.py"
				} else {
					return fmt.Errorf("Python river script not found. Use --python-path to specify location")
				}
			}

			if goPath == "" {
				// Default to current binary
				goPath = os.Args[0]
			}

			if workDir == "" {
				// Create temp directory
				tmpDir, err := os.MkdirTemp("", "river-parity-*")
				if err != nil {
					return fmt.Errorf("failed to create temp directory: %w", err)
				}
				workDir = tmpDir
			}

			// Create parity config
			parityConfig := &validation.ParityConfig{
				PythonPath:    pythonPath,
				GoPath:        goPath,
				WorkDir:       workDir,
				CleanupOnExit: cleanup,
			}

			// Create and run parity runner
			runner := validation.NewParityRunner(parityConfig)
			ctx := context.Background()

			fmt.Fprintf(cmd.OutOrStdout(), "Running parity validation for issue %s...\n\n", issueID)
			fmt.Fprintf(cmd.OutOrStdout(), "Python: %s\n", pythonPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Go:     %s\n", goPath)
			fmt.Fprintf(cmd.OutOrStdout(), "WorkDir: %s\n\n", workDir)

			results, err := runner.Run(ctx, issueID)
			if err != nil {
				return fmt.Errorf("parity validation failed: %w", err)
			}

			// Generate and print report
			report := runner.GenerateReport(results)
			fmt.Fprint(cmd.OutOrStdout(), report)

			// Exit with non-zero code if parity check failed
			if !results.Success {
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&pythonPath, "python-path", "", "Path to Python river script (default: ./main.py)")
	cmd.Flags().StringVar(&goPath, "go-path", "", "Path to Go river binary (default: current binary)")
	cmd.Flags().StringVar(&workDir, "work-dir", "", "Working directory for test runs (default: temp dir)")
	cmd.Flags().BoolVar(&cleanup, "cleanup", true, "Clean up temporary files after validation")

	return cmd
}


// findPythonScript attempts to locate the Python river script
func findPythonScript() string {
	// Check common locations
	locations := []string{
		"main.py",
		"./main.py",
		"../main.py",
		filepath.Join("..", "python", "main.py"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			abs, _ := filepath.Abs(loc)
			return abs
		}
	}

	return ""
}