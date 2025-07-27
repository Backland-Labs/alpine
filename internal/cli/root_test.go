package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRootCommandFlags tests that the root command has all expected flags.
// This ensures that CLI users can access the --serve and --port flags
// to enable the HTTP server functionality.
func TestRootCommandFlags(t *testing.T) {
	t.Run("root command has --serve flag", func(t *testing.T) {
		cmd := NewRootCommand()

		// The --serve flag should exist as a boolean flag
		serveFlag := cmd.Flags().Lookup("serve")
		require.NotNil(t, serveFlag, "root command should have --serve flag")
		assert.Equal(t, "bool", serveFlag.Value.Type(), "--serve should be a boolean flag")
		assert.Equal(t, "false", serveFlag.DefValue, "--serve should default to false")
		assert.Contains(t, serveFlag.Usage, "HTTP server", "--serve flag should mention HTTP server in usage")
	})

	t.Run("root command has --port flag", func(t *testing.T) {
		cmd := NewRootCommand()

		// The --port flag should exist as an integer flag with default 3001
		portFlag := cmd.Flags().Lookup("port")
		require.NotNil(t, portFlag, "root command should have --port flag")
		assert.Equal(t, "int", portFlag.Value.Type(), "--port should be an integer flag")
		assert.Equal(t, "3001", portFlag.DefValue, "--port should default to 3001")
		assert.Contains(t, portFlag.Usage, "HTTP server port", "--port flag should mention HTTP server port in usage")
	})
}

// TestRootCommandFlagParsing tests that the CLI correctly parses the server flags.
// This validates that users can specify these flags on the command line
// and that they are accessible to the workflow logic.
func TestRootCommandFlagParsing(t *testing.T) {
	t.Run("alpine --serve is a valid command without task", func(t *testing.T) {
		cmd := NewRootCommand()

		// Override RunE to prevent actual workflow execution
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			// Just verify we can get the flag value
			serve, err := cmd.Flags().GetBool("serve")
			require.NoError(t, err)
			assert.True(t, serve)
			return nil
		}

		cmd.SetArgs([]string{"--serve"})
		err := cmd.Execute()
		require.NoError(t, err)
	})

	t.Run("alpine --serve with task description should error", func(t *testing.T) {
		cmd := NewRootCommand()

		cmd.SetArgs([]string{"--serve", "test task"})
		err := cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot use --serve with a task description")
	})

	t.Run("alpine --port 8080 is a valid command", func(t *testing.T) {
		cmd := NewRootCommand()

		// Override RunE to prevent actual workflow execution
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			// Just verify we can get the flag value
			port, err := cmd.Flags().GetInt("port")
			require.NoError(t, err)
			assert.Equal(t, 8080, port)
			return nil
		}

		cmd.SetArgs([]string{"--port", "8080", "test task"})
		err := cmd.Execute()
		require.NoError(t, err)
	})

	t.Run("default port is 3001 when --port is not specified", func(t *testing.T) {
		cmd := NewRootCommand()

		// Override RunE to prevent actual workflow execution
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			// Just verify we can get the flag value
			port, err := cmd.Flags().GetInt("port")
			require.NoError(t, err)
			assert.Equal(t, 3001, port)
			return nil
		}

		cmd.SetArgs([]string{"test task"})
		err := cmd.Execute()
		require.NoError(t, err)
	})
}

// TestRootCommandHelp tests that the help text includes server-related flags.
// This ensures users can discover the HTTP server functionality through --help.
func TestRootCommandHelp(t *testing.T) {
	cmd := NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	helpText := buf.String()
	assert.Contains(t, helpText, "--serve", "help text should mention --serve flag")
	assert.Contains(t, helpText, "--port", "help text should mention --port flag")
	assert.Contains(t, strings.ToLower(helpText), "http server", "help text should mention HTTP server functionality")
}
