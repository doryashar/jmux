#!/bin/bash
# Test real-time messaging in tmux sessions

echo "Testing real-time messaging in tmux sessions..."

export DMUX_DEBUG=1
export JMUX_SHARED_DIR="$HOME/.jmux/shared"

# Ensure directories exist
mkdir -p "$HOME/.jmux/shared/messages"

echo "1. Starting messaging monitor manually to test..."
./bin/dmux _internal_messaging_monitor &
MONITOR_PID=$!

sleep 2

echo "2. Creating a test message..."
cat > "$HOME/.jmux/shared/messages/$(whoami)_realtime_$(date +%s).msg" << EOF
FROM=testuser
TYPE=MESSAGE
TIMESTAMP=$(date +%s)
DATA=Real-time message test
PRIORITY=normal
EOF

echo "3. Waiting for message processing..."
sleep 3

echo "4. Checking if monitor is still running..."
if kill -0 $MONITOR_PID 2>/dev/null; then
    echo "Monitor is running"
    kill $MONITOR_PID
else
    echo "Monitor stopped"
fi

echo "5. Testing tmux display message directly..."
if command -v tmux >/dev/null 2>&1; then
    echo "Testing tmux display-message..."
    tmux display-message -d 2000 "💬 Test message: Direct tmux display"
    echo "If you see a message at the bottom of the terminal, tmux messaging works!"
else
    echo "tmux not available for testing"
fi

echo "Test complete!"