#!/bin/bash
# Debug messaging issue - comprehensive test

echo "=== Debugging messaging issue in tmux sessions ==="

export JMUX_SHARED_DIR="$HOME/.jmux/shared"
export DMUX_DEBUG=1

# Clean up any existing
pkill -f "_internal_messaging_monitor" 2>/dev/null
mkdir -p "$HOME/.jmux/shared/messages"

echo "1. Testing messaging monitor functionality..."
echo "Starting messaging monitor in background..."
./bin/dmux _internal_messaging_monitor &
MONITOR_PID=$!
echo "Monitor PID: $MONITOR_PID"

sleep 2

echo "2. Creating test message while monitor is running..."
MSG_FILE="$HOME/.jmux/shared/messages/$(whoami)_debug_$(date +%s).msg"
cat > "$MSG_FILE" << EOF
FROM=debuguser
TYPE=MESSAGE
TIMESTAMP=$(date +%s)
DATA=Debug message - should appear in real-time
PRIORITY=normal
EOF

echo "Created message file: $MSG_FILE"
sleep 3

echo "3. Checking if monitor processed the message..."
if [[ -f "$MSG_FILE" ]]; then
    echo "❌ Message file still exists - not processed"
    cat "$MSG_FILE"
else
    echo "✅ Message file was processed and removed"
fi

echo "4. Testing tmux display directly..."
if tmux has-session -t test-session 2>/dev/null; then
    tmux kill-session -t test-session
fi
tmux new-session -d -s test-session
sleep 1
tmux display-message -t test-session -d 2000 "💬 Direct tmux test message"
echo "Check if you see a message in the tmux session"

echo "5. Testing tmux message via dmux monitor (if still running)..."
if kill -0 $MONITOR_PID 2>/dev/null; then
    echo "Monitor still running, creating another message..."
    cat > "$HOME/.jmux/shared/messages/$(whoami)_debug2_$(date +%s).msg" << EOF
FROM=debuguser2
TYPE=URGENT
TIMESTAMP=$(date +%s)
DATA=Urgent debug message
PRIORITY=high
EOF
    sleep 3
else
    echo "Monitor stopped"
fi

echo "6. Cleanup..."
kill $MONITOR_PID 2>/dev/null
tmux kill-session -t test-session 2>/dev/null
rm -f "$HOME/.jmux/shared/messages/$(whoami)_debug"*.msg

echo "=== Debug complete ==="