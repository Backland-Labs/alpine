#!/usr/bin/env python3

import time
import sys

def print_hello_world():
    """Simple function that prints 'Hello, World!' with some progress messages."""
    
    # Print to stderr to trigger the sticky header tool logs
    print("Tool: Starting Hello World function...", file=sys.stderr)
    time.sleep(0.5)
    
    print("Tool: Preparing greeting message...", file=sys.stderr)
    time.sleep(0.5)
    
    # Print the main output to stdout
    print("Hello, World!")
    
    print("Tool: Greeting delivered successfully!", file=sys.stderr)
    time.sleep(0.5)
    
    print("Tool: Function execution complete.", file=sys.stderr)
    
    return "Hello, World!"

if __name__ == "__main__":
    print("=== Testing Sticky Header Feature ===")
    print("This will demonstrate how River captures tool logs from stderr")
    print("Watch the terminal for the sticky header updates!\n")
    
    result = print_hello_world()
    
    print(f"\nFunction returned: {result}")
    print("=== Test Complete ===")