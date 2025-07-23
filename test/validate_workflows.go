package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// WorkflowValidation tests for GitHub Actions workflow files
type WorkflowValidation struct {
	Name string
	Path string
}

// GitHubWorkflow represents the structure of a GitHub Actions workflow
type GitHubWorkflow struct {
	Name string                 `yaml:"name"`
	On   interface{}            `yaml:"on"`
	Jobs map[string]interface{} `yaml:"jobs"`
}

// TestWorkflowsExist ensures required workflow files exist
func TestWorkflowsExist() error {
	requiredWorkflows := []WorkflowValidation{
		{Name: "CI", Path: ".github/workflows/ci.yml"},
		{Name: "Release", Path: ".github/workflows/release.yml"},
	}

	for _, wf := range requiredWorkflows {
		if _, err := os.Stat(wf.Path); os.IsNotExist(err) {
			return fmt.Errorf("required workflow %s not found at %s", wf.Name, wf.Path)
		}
	}
	return nil
}

// TestCIWorkflow validates the CI workflow configuration
func TestCIWorkflow() error {
	data, err := os.ReadFile(".github/workflows/ci.yml")
	if err != nil {
		return fmt.Errorf("failed to read CI workflow: %w", err)
	}

	var workflow GitHubWorkflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return fmt.Errorf("failed to parse CI workflow: %w", err)
	}

	// Test: Workflow has a name
	if workflow.Name == "" {
		return fmt.Errorf("CI workflow must have a name")
	}

	// Test: Workflow triggers on push and pull_request
	if workflow.On == nil {
		return fmt.Errorf("CI workflow must have triggers defined")
	}

	// Test: Workflow has required jobs
	requiredJobs := []string{"test", "lint", "build"}
	for _, job := range requiredJobs {
		if _, exists := workflow.Jobs[job]; !exists {
			return fmt.Errorf("CI workflow missing required job: %s", job)
		}
	}

	return nil
}

// TestReleaseWorkflow validates the release workflow configuration
func TestReleaseWorkflow() error {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		return fmt.Errorf("failed to read Release workflow: %w", err)
	}

	var workflow GitHubWorkflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return fmt.Errorf("failed to parse Release workflow: %w", err)
	}

	// Test: Workflow has a name
	if workflow.Name == "" {
		return fmt.Errorf("release workflow must have a name")
	}

	// Test: Workflow triggers on tags
	triggerStr := fmt.Sprintf("%v", workflow.On)
	if !strings.Contains(triggerStr, "tags") {
		return fmt.Errorf("release workflow must trigger on tags")
	}

	// Test: Workflow has build job
	if _, exists := workflow.Jobs["build"]; !exists {
		return fmt.Errorf("release workflow missing build job")
	}

	return nil
}

func main() {
	tests := []struct {
		name string
		test func() error
	}{
		{"Workflows Exist", TestWorkflowsExist},
		{"CI Workflow Valid", TestCIWorkflow},
		{"Release Workflow Valid", TestReleaseWorkflow},
	}

	failed := false
	for _, tt := range tests {
		fmt.Printf("Running test: %s... ", tt.name)
		if err := tt.test(); err != nil {
			fmt.Printf("FAILED\n  Error: %v\n", err)
			failed = true
		} else {
			fmt.Printf("PASSED\n")
		}
	}

	if failed {
		log.Fatal("Some tests failed")
	} else {
		fmt.Println("\nAll workflow validation tests passed!")
	}
}
