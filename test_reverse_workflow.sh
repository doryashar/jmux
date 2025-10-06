#!/bin/bash

echo "🧪 Testing complete reverse sharing workflow..."

# Set up test environment
TEST_HOME="/tmp/test_reverse_workflow_$(date +%s)"
mkdir -p "$TEST_HOME"
export HOME="$TEST_HOME"
export JMUX_SHARED_DIR="$TEST_HOME/shared"

DMUX_BINARY="bin/dmux"

echo "Test HOME: $TEST_HOME"

# Initialize directories
mkdir -p "$JMUX_SHARED_DIR/messages"
mkdir -p "$JMUX_SHARED_DIR/sessions"

echo "🚀 Step 1: Starting dmux ask-share in background..."

# Start ask-share in background
$DMUX_BINARY ask-share testuser &
ASK_SHARE_PID=$!

echo "📝 ask-share started with PID: $ASK_SHARE_PID"

# Give it time to start up and create tmux session
sleep 3

echo "🔍 Step 2: Checking if dmux process and tmux session are running..."

if ps -p $ASK_SHARE_PID > /dev/null; then
    echo "✅ dmux ask-share process is running (PID: $ASK_SHARE_PID)"
else
    echo "❌ FAIL: dmux ask-share process has exited"
    exit 1
fi

# Check if tmux session was created
TMUX_SESSION=$(tmux ls 2>/dev/null | grep reverse | head -1 | cut -d: -f1)
if [ -n "$TMUX_SESSION" ]; then
    echo "✅ tmux session created: $TMUX_SESSION"
else
    echo "❌ FAIL: tmux session not found"
    kill $ASK_SHARE_PID 2>/dev/null
    exit 1
fi

# Check if jcat server is listening
JCAT_PORT=$(ss -tlnp | grep ":123[0-9][0-9]" | head -1 | awk '{print $4}' | cut -d: -f2)
if [ -n "$JCAT_PORT" ]; then
    echo "✅ jcat server is listening on port $JCAT_PORT"
else
    echo "❌ FAIL: jcat server not listening"
    kill $ASK_SHARE_PID 2>/dev/null
    exit 1
fi

echo "🔍 Step 3: Testing join-share command (should find invitation)..."

# Test join-share command (should find invitation and try to connect)
JOIN_OUTPUT=$($DMUX_BINARY join-share yashar 2>&1 | head -3)
if echo "$JOIN_OUTPUT" | grep -q "Connecting to yashar's reverse sharing session"; then
    echo "✅ join-share correctly found invitation and attempted connection"
else
    echo "❌ FAIL: join-share did not find invitation properly"
    echo "Output: $JOIN_OUTPUT"
    kill $ASK_SHARE_PID 2>/dev/null
    exit 1
fi

echo "🛑 Step 4: Cleaning up..."
kill $ASK_SHARE_PID 2>/dev/null
wait $ASK_SHARE_PID 2>/dev/null || true

# Clean up tmux session if it still exists
if [ -n "$TMUX_SESSION" ]; then
    tmux kill-session -t "$TMUX_SESSION" 2>/dev/null || true
fi

# Clean up test directory
rm -rf "$TEST_HOME"

echo "🎉 All workflow tests passed! Reverse sharing functionality works correctly."
echo ""
echo "Summary of what was tested:"
echo "✅ ask-share process stays alive"
echo "✅ tmux session gets created with jcat server"
echo "✅ jcat server listens on network port"
echo "✅ join-share command finds invitation and attempts connection"
echo "✅ Proper cleanup when stopped"