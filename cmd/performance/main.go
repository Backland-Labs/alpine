// Command performance runs performance measurements for River
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/maxmcd/river/internal/performance"
)

func main() {
	var (
		jsonOutput = flag.Bool("json", false, "Output results as JSON")
		outputFile = flag.String("output", "", "Write results to file (default: stdout)")
	)
	flag.Parse()

	// Create runner
	runner := performance.NewRunner(os.Stderr)
	
	// Run performance measurements
	results, err := runner.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running performance tests: %v\n", err)
		os.Exit(1)
	}
	
	// Determine output destination
	output := os.Stdout
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			if closeErr := f.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "Error closing output file: %v\n", closeErr)
			}
		}()
		output = f
	}
	
	// Write results
	if *jsonOutput {
		if err := runner.WriteJSON(results, output); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := runner.WriteSummary(results, output); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing summary: %v\n", err)
			os.Exit(1)
		}
		
		// If writing to file, also print summary to stdout
		if *outputFile != "" {
			if err := runner.WriteSummary(results, os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing summary to stdout: %v\n", err)
			}
		}
	}
	
	// Check if performance meets expectations
	if results.StartupTime.GoStartupTimeMs > 1000 {
		fmt.Fprintf(os.Stderr, "\nWarning: Startup time exceeds 1 second\n")
		os.Exit(1)
	}
	
	if results.MemoryUsage.HeapAllocMB > 100 {
		fmt.Fprintf(os.Stderr, "\nWarning: Memory usage exceeds 100MB\n")
		os.Exit(1)
	}
}