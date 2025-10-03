#!/bin/bash
# Complete messaging test - simulate real usage

echo "=== Complete dmux messaging test ==="

export JMUX_SHARED_DIR="$HOME/.jmux/shared"
export DMUX_DEBUG=1

# Clean up
pkill -f "_internal_messaging_monitor" 2>/dev/null
tmux kill-session -t dmux-main 2>/dev/null
mkdir -p "$HOME/.jmux/shared/messages"

echo "1. Testing dmux share command with messaging..."

# Start a background dmux monitor to simulate real usage
./bin/dmux _internal_messaging_monitor &
MONITOR_PID=$!
echo "Started messaging monitor with PID: $MONITOR_PID"

# Create a tmux session to simulate dmux usage
tmux new-session -d -s dmux-main "sleep 60"
echo "Created tmux session: dmux-main"

sleep 2

echo "2. Sending a test message to current user..."
./bin/dmux msg "$(whoami)" "Hello from dmux messaging test!"

sleep 3

echo "3. Testing share command (creates invite messages)..."
timeout 5s ./bin/dmux share test-session 2>/dev/null || echo "Share command tested"

sleep 3

echo "4. Checking for any remaining messages..."
ls -la "$HOME/.jmux/shared/messages/" | grep "$(whoami)" || echo "No messages remaining"

echo "5. Cleanup..."
kill $MONITOR_PID 2>/dev/null
tmux kill-session -t dmux-main 2>/dev/null

echo "=== Test complete ==="
echo "If you saw tmux display messages, the messaging system is working!"