package docs_test

import (
	"os"
	"strings"
	"testing"
)

// TestCLAUDEDocumentationComplete verifies that AGENTS.md contains all required REST API documentation
func TestCLAUDEDocumentationComplete(t *testing.T) {
	content, err := os.ReadFile("../../AGENTS.md")
	if err != nil {
		t.Fatal("Failed to read AGENTS.md:", err)
	}

	doc := string(content)

	// Test 1: REST API Server Usage section exists
	t.Run("REST API Server Usage Section", func(t *testing.T) {
		if !strings.Contains(doc, "## REST API Server Usage") && !strings.Contains(doc, "### REST API Server Usage") {
			t.Error("AGENTS.md missing REST API Server Usage section")
		}
	})

	// Test 2: Server startup examples
	t.Run("Server Startup Examples", func(t *testing.T) {
		requiredExamples := []string{
			"./alpine --serve",
			"--port",
			"curl",
		}
		for _, example := range requiredExamples {
			if !strings.Contains(doc, example) {
				t.Errorf("AGENTS.md missing server startup example with: %s", example)
			}
		}
	})

	// Test 3: REST API endpoints documentation
	t.Run("REST API Endpoints Documentation", func(t *testing.T) {
		requiredEndpoints := []struct {
			endpoint     string
			alternatives []string
		}{
			{"/health", nil},
			{"/agents/list", nil},
			{"/agents/run", nil},
			{"/runs", nil},
			{"/runs/{id}", []string{"/runs/{run-id}"}},
			{"/runs/{id}/cancel", []string{"/runs/{run-id}/cancel"}},
			{"/plans/{runId}", []string{"/plans/{run-id}"}},
			{"/plans/{runId}/approve", []string{"/plans/{run-id}/approve"}},
			{"/plans/{runId}/feedback", []string{"/plans/{run-id}/feedback"}},
		}
		for _, ep := range requiredEndpoints {
			found := strings.Contains(doc, ep.endpoint)
			if !found {
				// Check alternatives
				for _, alt := range ep.alternatives {
					if strings.Contains(doc, alt) {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("AGENTS.md missing documentation for endpoint: %s", ep.endpoint)
			}
		}
	})

	// Test 4: Curl examples for common workflows
	t.Run("Curl Examples", func(t *testing.T) {
		requiredExamples := []string{
			"curl -X POST",
			"application/json",
			"Content-Type",
		}
		for _, example := range requiredExamples {
			if !strings.Contains(doc, example) {
				t.Errorf("AGENTS.md missing curl example element: %s", example)
			}
		}
	})

	// Test 5: Integration patterns
	t.Run("Integration Patterns", func(t *testing.T) {
		if !strings.Contains(doc, "Python") || !strings.Contains(doc, "JavaScript") {
			t.Error("AGENTS.md missing integration examples in multiple languages")
		}
	})
}

// TestCLICommandsDocumentationComplete verifies cli-commands.md has REST API server documentation
func TestCLICommandsDocumentationComplete(t *testing.T) {
	content, err := os.ReadFile("../../specs/cli-commands.md")
	if err != nil {
		t.Fatal("Failed to read cli-commands.md:", err)
	}

	doc := string(content)

	// Test 1: REST API endpoints section
	t.Run("REST API Endpoints Section", func(t *testing.T) {
		if !strings.Contains(doc, "## REST API Endpoints") && !strings.Contains(doc, "### REST API Endpoints") {
			t.Error("cli-commands.md missing REST API Endpoints section")
		}
	})

	// Test 2: Server mode with REST API documentation
	t.Run("Server Mode REST API Documentation", func(t *testing.T) {
		if !strings.Contains(doc, "REST API") || !strings.Contains(doc, "/health") {
			t.Error("cli-commands.md missing REST API documentation in server mode section")
		}
	})

	// Test 3: Examples of REST API usage
	t.Run("REST API Usage Examples", func(t *testing.T) {
		requiredContent := []string{
			"POST /agents/run",
			"GET /runs",
			"workflow",
		}
		for _, content := range requiredContent {
			if !strings.Contains(doc, content) {
				t.Errorf("cli-commands.md missing REST API example content: %s", content)
			}
		}
	})
}

// TestDocumentationConsistency verifies that documentation is consistent across files
func TestDocumentationConsistency(t *testing.T) {
	claudeContent, err1 := os.ReadFile("../../AGENTS.md")
	cliContent, err2 := os.ReadFile("../../specs/cli-commands.md")

	if err1 != nil || err2 != nil {
		t.Fatal("Failed to read documentation files")
	}

	claudeDoc := string(claudeContent)
	cliDoc := string(cliContent)

	// Test: Port numbers are consistent
	t.Run("Port Number Consistency", func(t *testing.T) {
		if strings.Contains(claudeDoc, "3001") && strings.Contains(cliDoc, "3001") {
			// Good - both use same default port
		} else {
			t.Error("Port numbers are inconsistent between AGENTS.md and cli-commands.md")
		}
	})

	// Test: Server flags are documented consistently
	t.Run("Server Flags Consistency", func(t *testing.T) {
		if strings.Contains(claudeDoc, "--serve") != strings.Contains(cliDoc, "--serve") {
			t.Error("--serve flag documentation is inconsistent")
		}
		if strings.Contains(claudeDoc, "--port") != strings.Contains(cliDoc, "--port") {
			t.Error("--port flag documentation is inconsistent")
		}
	})
}

// TestRESTAPIWorkflowExamples verifies complete workflow examples exist
func TestRESTAPIWorkflowExamples(t *testing.T) {
	content, err := os.ReadFile("../../AGENTS.md")
	if err != nil {
		t.Fatal("Failed to read AGENTS.md:", err)
	}

	doc := string(content)

	// Test: Complete workflow example from start to finish
	t.Run("Complete Workflow Example", func(t *testing.T) {
		workflowSteps := []string{
			"Start the server",
			"Create a new run",
			"Monitor progress",
			"Approve plan",
		}

		missingSteps := []string{}
		for _, step := range workflowSteps {
			if !strings.Contains(strings.ToLower(doc), strings.ToLower(step)) {
				missingSteps = append(missingSteps, step)
			}
		}

		if len(missingSteps) > 0 {
			t.Errorf("AGENTS.md missing workflow steps: %v", missingSteps)
		}
	})
}
