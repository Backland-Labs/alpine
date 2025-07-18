#!/bin/bash
# Setup script for Claude Code Logfire hook

echo "Claude Code Logfire Hook Setup"
echo "=============================="

# Check if logfire is installed
if ! python3 -c "import logfire" 2>/dev/null; then
    echo "Installing pydantic-logfire..."
    pip install pydantic-logfire
fi

# Check for LOGFIRE_TOKEN
if [ -z "$LOGFIRE_TOKEN" ]; then
    echo "WARNING: LOGFIRE_TOKEN environment variable not set!"
    echo "Please set it with: export LOGFIRE_TOKEN='your-token-here'"
    echo ""
fi

# Create Claude settings directory if it doesn't exist
mkdir -p ~/.claude

# Check if user wants to install globally or just for this project
echo "Install hook configuration:"
echo "1) Globally (in ~/.claude/settings.json)"
echo "2) Project-specific (in ./.claude/settings.json)"
read -p "Choose option (1 or 2): " choice

case $choice in
    1)
        SETTINGS_FILE="$HOME/.claude/settings.json"
        ;;
    2)
        mkdir -p .claude
        SETTINGS_FILE="./.claude/settings.json"
        ;;
    *)
        echo "Invalid choice. Exiting."
        exit 1
        ;;
esac

# Backup existing settings if they exist
if [ -f "$SETTINGS_FILE" ]; then
    cp "$SETTINGS_FILE" "${SETTINGS_FILE}.backup"
    echo "Backed up existing settings to ${SETTINGS_FILE}.backup"
fi

# Copy the hooks configuration
cp claude-hooks-config.json "$SETTINGS_FILE"
echo "Installed hooks configuration to $SETTINGS_FILE"

# Make hook script executable
chmod +x claude_logfire_hook.py

echo ""
echo "Setup complete! The hook will now send Claude Code data to Logfire."
echo ""
echo "To test the integration, run Claude Code and check your Logfire dashboard."
echo ""
echo "Remember to set LOGFIRE_TOKEN if you haven't already:"
echo "  export LOGFIRE_TOKEN='your-logfire-token'"