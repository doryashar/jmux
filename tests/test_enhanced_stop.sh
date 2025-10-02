#!/bin/bash

set -e

echo "Testing enhanced stop command functionality..."

# Set up test environment
TEST_HOME="/tmp/test_jmux_stop_$(date +%s)"
mkdir -p "$TEST_HOME"
export HOME="$TEST_HOME"
export JMUX_SHARED_DIR="$TEST_HOME/shared"
export JMUX_CONFIG_DIR="$HOME/.config/jmux"
export JMUX_SESSIONS_DIR="$JMUX_SHARED_DIR/sessions"

echo "Test HOME: $TEST_HOME"

cd /home/yashar/projects/jmux

# Initialize the environment
mkdir -p "$JMUX_SESSIONS_DIR"

# Create mock session files for testing
echo "Creating mock session files..."

# Session 1: test-session1
cat > "$JMUX_SESSIONS_DIR/${USER}_test-session1.session" << EOF
USER=${USER}
SESSION=test-session1
PORT=12345
STARTED=$(date +%s)
PID=$$
PRIVATE=false
ALLOWED_USERS=
EOF

# Session 2: test-session2  
cat > "$JMUX_SESSIONS_DIR/${USER}_test-session2.session" << EOF
USER=${USER}
SESSION=test-session2
PORT=12346
STARTED=$(date +%s)
PID=$$
PRIVATE=false
ALLOWED_USERS=
EOF

# Session 3: test-session3
cat > "$JMUX_SESSIONS_DIR/${USER}_test-session3.session" << EOF
USER=${USER}
SESSION=test-session3
PORT=12347
STARTED=$(date +%s)
PID=$$
PRIVATE=false
ALLOWED_USERS=
EOF

echo "Created mock sessions:"
ls -la "$JMUX_SESSIONS_DIR/"

# Test 1: Stop single specific session
echo -e "\n=== Test 1: Stop single specific session ==="
HOME="$TEST_HOME" JMUX_SHARED_DIR="$JMUX_SHARED_DIR" ./jmux stop test-session1 2>/dev/null

# Verify session1 was removed
if [[ -f "$JMUX_SESSIONS_DIR/${USER}_test-session1.session" ]]; then
    echo "❌ ERROR: Session 1 file still exists"
    exit 1
fi

# Verify other sessions still exist
if [[ ! -f "$JMUX_SESSIONS_DIR/${USER}_test-session2.session" ]]; then
    echo "❌ ERROR: Session 2 file was incorrectly removed"
    exit 1
fi

echo "✅ Single session stop works correctly"

# Test 2: Stop multiple specific sessions
echo -e "\n=== Test 2: Stop multiple specific sessions ==="
HOME="$TEST_HOME" JMUX_SHARED_DIR="$JMUX_SHARED_DIR" ./jmux stop test-session2 test-session3 2>/dev/null

# Verify both sessions were removed
if [[ -f "$JMUX_SESSIONS_DIR/${USER}_test-session2.session" ]]; then
    echo "❌ ERROR: Session 2 file still exists"
    exit 1
fi

if [[ -f "$JMUX_SESSIONS_DIR/${USER}_test-session3.session" ]]; then
    echo "❌ ERROR: Session 3 file still exists"
    exit 1
fi

echo "✅ Multiple sessions stop works correctly"

# Test 3: Try to stop non-existent session
echo -e "\n=== Test 3: Stop non-existent session ==="
if HOME="$TEST_HOME" JMUX_SHARED_DIR="$JMUX_SHARED_DIR" ./jmux stop non-existent-session 2>/dev/null; then
    echo "❌ ERROR: Stop should fail for non-existent session"
    exit 1
fi

echo "✅ Non-existent session handling works correctly"

# Test 4: Stop with no arguments (should work even without tmux)
echo -e "\n=== Test 4: Stop without arguments ==="

# Create a session with hostname
cat > "$JMUX_SESSIONS_DIR/${USER}_$(hostname).session" << EOF
USER=${USER}
SESSION=$(hostname)
PORT=12348
STARTED=$(date +%s)
PID=$$
PRIVATE=false
ALLOWED_USERS=
EOF

HOME="$TEST_HOME" JMUX_SHARED_DIR="$JMUX_SHARED_DIR" ./jmux stop 2>/dev/null

# Verify hostname session was removed
if [[ -f "$JMUX_SESSIONS_DIR/${USER}_$(hostname).session" ]]; then
    echo "❌ ERROR: Hostname session file still exists"
    exit 1
fi

echo "✅ Stop without arguments works correctly"

# Cleanup
rm -rf "$TEST_HOME"

echo -e "\n✅ All enhanced stop command tests passed!"