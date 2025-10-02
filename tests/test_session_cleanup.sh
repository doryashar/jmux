#!/bin/bash

set -e

echo "Testing session cleanup functionality..."

# Set up test environment
TEST_HOME="/tmp/test_jmux_cleanup_$(date +%s)"
mkdir -p "$TEST_HOME"
export HOME="$TEST_HOME"
export JMUX_SHARED_DIR="$TEST_HOME/shared"
export JMUX_SESSIONS_DIR="$JMUX_SHARED_DIR/sessions"

echo "Test HOME: $TEST_HOME"

cd /home/yashar/projects/jmux

# Initialize directories
mkdir -p "$JMUX_SESSIONS_DIR"

# Create a mock stale session file (session that doesn't exist in tmux)
cat > "$JMUX_SESSIONS_DIR/${USER}_stale-session.session" << EOF
USER=${USER}
SESSION=stale-session
PORT=19999
STARTED=$(date +%s)
PID=$$
PRIVATE=false
ALLOWED_USERS=
EOF

echo "✅ Created stale session file"

# Test that sessions command cleans up stale sessions
echo "Testing stale session cleanup..."
echo "Before cleanup:"
ls -la "$JMUX_SESSIONS_DIR/"

# Run sessions command which should clean up stale sessions
HOME="$TEST_HOME" JMUX_SHARED_DIR="$JMUX_SHARED_DIR" ./jmux sessions >/dev/null 2>&1

echo "After cleanup:"
ls -la "$JMUX_SESSIONS_DIR/" 2>/dev/null || echo "Sessions directory empty"

# Verify stale session was cleaned up
if [[ ! -f "$JMUX_SESSIONS_DIR/${USER}_stale-session.session" ]]; then
    echo "✅ Stale session cleanup works correctly"
else
    echo "❌ ERROR: Stale session was not cleaned up"
    exit 1
fi

# Test status command also cleans up stale sessions
cat > "$JMUX_SESSIONS_DIR/${USER}_another-stale.session" << EOF
USER=${USER}
SESSION=another-stale
PORT=19998
STARTED=$(date +%s)
PID=$$
PRIVATE=false
ALLOWED_USERS=
EOF

echo "Testing status command cleanup..."

# Run status command which should also clean up stale sessions
HOME="$TEST_HOME" JMUX_SHARED_DIR="$JMUX_SHARED_DIR" ./jmux status >/dev/null 2>&1

# Verify stale session was cleaned up
if [[ ! -f "$JMUX_SESSIONS_DIR/${USER}_another-stale.session" ]]; then
    echo "✅ Status command cleanup works correctly"
else
    echo "❌ ERROR: Status command did not clean up stale session"
    exit 1
fi

# Cleanup
rm -rf "$TEST_HOME"

echo "✅ All session cleanup tests passed!"