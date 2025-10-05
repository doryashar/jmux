#!/bin/bash

# Test script to reproduce the dmux hang issue when run from within tmux sessions
# This demonstrates the blocking behavior caused by fmt.Scanln in background update check

echo "Testing dmux hang issue when run from within tmux sessions..."

# Start a tmux session for testing
echo "Creating test tmux session..."
tmux new-session -d -s dmux-hang-test

# Set TMUX environment variable to simulate being inside tmux
export TMUX="/tmp/tmux-$(id -u)/default,123,0"

echo "TMUX environment set to: $TMUX"

# Test dmux command with timeout to see if it hangs
echo "Testing dmux version command (should not hang)..."
timeout 10s /home/yashar/projects/jmux/bin/dmux version
if [ $? -eq 124 ]; then
    echo "❌ HANG DETECTED: dmux version command timed out after 10 seconds"
else
    echo "✅ dmux version completed successfully"
fi

# Test dmux help command
echo "Testing dmux help command..."
timeout 10s /home/yashar/projects/jmux/bin/dmux help
if [ $? -eq 124 ]; then
    echo "❌ HANG DETECTED: dmux help command timed out after 10 seconds"
else
    echo "✅ dmux help completed successfully"
fi

# Test dmux status command (this is more likely to trigger the hang)
echo "Testing dmux status command..."
timeout 10s /home/yashar/projects/jmux/bin/dmux status
if [ $? -eq 124 ]; then
    echo "❌ HANG DETECTED: dmux status command timed out after 10 seconds"
else
    echo "✅ dmux status completed successfully"
fi

# Clean up test session
echo "Cleaning up test tmux session..."
tmux kill-session -t dmux-hang-test 2>/dev/null

echo "Test completed."