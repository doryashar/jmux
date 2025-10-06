#!/bin/bash

echo "🧪 Testing dmux reverse sharing process persistence..."

# Set up test environment
TEST_HOME="/tmp/test_reverse_persistence_$(date +%s)"
mkdir -p "$TEST_HOME"
export HOME="$TEST_HOME"
export JMUX_SHARED_DIR="$TEST_HOME/shared"

DMUX_BINARY="bin/dmux"

echo "Test HOME: $TEST_HOME"

# Initialize directories
mkdir -p "$JMUX_SHARED_DIR/messages"
mkdir -p "$JMUX_SHARED_DIR/sessions"

echo "🚀 Starting dmux ask-share in background..."

# Start ask-share in background
$DMUX_BINARY ask-share testuser &
ASK_SHARE_PID=$!

echo "📝 ask-share started with PID: $ASK_SHARE_PID"

# Give it time to start up
sleep 3

echo "🔍 Checking if dmux process is still running..."
if ps -p $ASK_SHARE_PID > /dev/null; then
    echo "✅ SUCCESS: dmux ask-share process is still running (PID: $ASK_SHARE_PID)"
else
    echo "❌ FAIL: dmux ask-share process has exited"
    exit 1
fi

echo "🔍 Checking if jcat server is listening on a port..."
JCAT_PORT=$(ss -tlnp | grep ":123[0-9][0-9]" | head -1 | awk '{print $4}' | cut -d: -f2)

if [ -n "$JCAT_PORT" ]; then
    echo "✅ SUCCESS: jcat server is listening on port $JCAT_PORT"
else
    echo "❌ FAIL: jcat server not found listening on any port"
    kill $ASK_SHARE_PID 2>/dev/null
    exit 1
fi

echo "🔍 Checking if invitation message was created..."
if find "$JMUX_SHARED_DIR/messages" -name "*reverse_invite*" | grep -q .; then
    echo "✅ SUCCESS: Invitation message was created"
else
    echo "❌ FAIL: No invitation message found"
    kill $ASK_SHARE_PID 2>/dev/null
    exit 1
fi

echo "🛑 Stopping dmux ask-share process..."
kill $ASK_SHARE_PID
wait $ASK_SHARE_PID 2>/dev/null

echo "🧹 Cleaning up..."
rm -rf "$TEST_HOME"

echo "🎉 All tests passed! Reverse sharing process persistence works correctly."