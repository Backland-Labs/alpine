package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/maxkrieger/river/internal/claude"
)

func main() {
	// Create a simple plan command
	cmd := claude.Command{
		Type:         claude.CommandTypePlan,
		Content:      "Process Linear issue TEST-123 following TDD methodology",
		OutputFormat: "json",
		AllowedTools: []string{"linear-server", "code-editing"},
		SystemPrompt: "You are an AI assistant helping to implement features from Linear issues using Test-Driven Development.",
	}

	// Build the command
	builder := claude.New()
	args, err := builder.BuildCommand(context.Background(), cmd)
	if err != nil {
		fmt.Printf("Error building command: %v\n", err)
		os.Exit(1)
	}

	// Print the command that would be executed
	fmt.Printf("Command: %s\n", strings.Join(args, " "))
	fmt.Println("\nArguments:")
	for i, arg := range args {
		fmt.Printf("  [%d]: %q\n", i, arg)
	}

	// Try different approaches
	fmt.Println("\n\nTesting different command approaches:")
	
	// Test 1: Without slash command
	fmt.Println("\n1. Testing without slash command:")
	testCmd1 := exec.Command("claude", "-p", "--output-format", "json", "Process Linear issue TEST-123 following TDD methodology")
	testCmd1.Stdout = os.Stdout
	testCmd1.Stderr = os.Stderr
	if err := testCmd1.Run(); err != nil {
		fmt.Printf("   Failed: %v\n", err)
	}
	
	// Test 2: Using --permission-mode plan
	fmt.Println("\n2. Testing with --permission-mode plan:")
	testCmd2 := exec.Command("claude", "-p", "--output-format", "json", "--permission-mode", "plan", "Process Linear issue TEST-123 following TDD methodology")
	testCmd2.Stdout = os.Stdout
	testCmd2.Stderr = os.Stderr
	if err := testCmd2.Run(); err != nil {
		fmt.Printf("   Failed: %v\n", err)
	}
}