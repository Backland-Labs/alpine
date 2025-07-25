#!/usr/bin/env -S uv run --script

"""
DEPRECATED: This Python implementation is deprecated as of v0.2.0.
Please use the Go version instead. See DEPRECATED.md for migration instructions.

To use the new Go version:
  river "Your task description here"
  river --file task.md
  river "Your task" --no-plan
"""

import subprocess
import json
import os
import sys
import warnings

# Show deprecation warning
warnings.warn(
    "This Python implementation is deprecated. Please use the Go version instead. "
    "See DEPRECATED.md for migration instructions.",
    DeprecationWarning,
    stacklevel=2
)
print("\n⚠️  WARNING: This Python version is deprecated. Use the Go version instead.\n", file=sys.stderr)


def run_claude_code(prompt):
    print(f"Running Claude Code with prompt: {prompt}")
    cmd = [
        'claude', '-p', prompt, #linear_issue
        '--output-format', 'text',
        '--append-system-prompt', 'You are an expert software engineer with deep knowledge of TDD, Python, Typescript. Execute the following tasks with surgical precision while taking care not to overengineer solutions.',
        '--allowedTools', 'mcp__linear-server__*', 'mcp__context7__*', 'Bash', 'Read', 'Write', 'Edit', 'Remove'
    ]
    result = subprocess.run(cmd, capture_output=True, text=True)
    return result.stdout

def check_state():
    try:
        with open('agent_state/agent_state.json', 'r') as f:
            state = json.load(f)
        return state
    except (FileNotFoundError, json.JSONDecodeError):
        return None

# Run initial command
# 1. First read agent_state/agent_state.json
# 2. parse agent_state/agent_state.json to get the next step prompt if not none
# 3. then run prompt with run_claude_code
linear_issue = input("Enter the initial prompt for Claude Code: ")
NEED_PLAN = input("Do you need a plan? (True/False): ").strip().lower() == 'true'

if NEED_PLAN:
    print("Generating plan...")
    print(run_claude_code(f"/make_plan {linear_issue}"))

continue_flag = True
iteration = 0
while continue_flag:
    iteration += 1
    state = check_state()

    print(f"Starting command... (Iteration {iteration})")
    output = run_claude_code(state["next_step_prompt"])
    print(output)
    
    # Check status file for continuation
    if state["status"] == "completed":
        continue_flag = False
        print("Status file indicates completion. Stopping.")
