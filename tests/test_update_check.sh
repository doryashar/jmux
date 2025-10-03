#!/bin/bash
# Test script for update checking

echo "Testing automatic update check functionality..."

# Set environment for testing
export JMUX_SHARED_DIR="$HOME/.jmux/shared"
export JMUX_DEBUG_UPDATES=1

# Remove any existing update check file to force a check
rm -f "$HOME/.config/jmux/last_update_check"

echo "Running dmux with forced update check..."
echo "This should trigger an update check since no check file exists."

# Run a simple command that will initialize the system
timeout 10s ./bin/dmux version || echo "Command completed or timed out"

echo ""
echo "Check if update check file was created:"
if [ -f "$HOME/.config/jmux/last_update_check" ]; then
    echo "✅ Update check file created:"
    ls -la "$HOME/.config/jmux/last_update_check"
    echo "Last check time: $(cat "$HOME/.config/jmux/last_update_check")"
else
    echo "❌ Update check file not found"
fi

echo ""
echo "Running again (should NOT check for updates this time):"
timeout 5s ./bin/dmux version || echo "Command completed"