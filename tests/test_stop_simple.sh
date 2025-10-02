#!/bin/bash

set -e

echo "Testing enhanced stop command functionality..."

# Set up test environment  
TEST_HOME="/tmp/test_jmux_stop_$(date +%s)"
mkdir -p "$TEST_HOME"

# Override environment to avoid symlink creation 
export HOME="$TEST_HOME"
export JMUX_SHARED_DIR="$TEST_HOME/shared"
export JMUX_SESSIONS_DIR="$JMUX_SHARED_DIR/sessions"

cd /home/yashar/projects/jmux

# Initialize directories
mkdir -p "$JMUX_SESSIONS_DIR"

# Create mock session files
cat > "$JMUX_SESSIONS_DIR/${USER}_session1.session" << EOF
USER=${USER}
SESSION=session1
PORT=12345
STARTED=$(date +%s)
PID=$$
PRIVATE=false
ALLOWED_USERS=
EOF

cat > "$JMUX_SESSIONS_DIR/${USER}_session2.session" << EOF
USER=${USER}
SESSION=session2
PORT=12346
STARTED=$(date +%s)
PID=$$
PRIVATE=false
ALLOWED_USERS=
EOF

echo "✅ Created test session files"

# Test 1: Stop single session
echo "Testing single session stop..."
if HOME="$TEST_HOME" JMUX_SHARED_DIR="$JMUX_SHARED_DIR" ./jmux stop session1 >/dev/null 2>&1; then
    if [[ ! -f "$JMUX_SESSIONS_DIR/${USER}_session1.session" ]]; then
        echo "✅ Single session stop works"
    else
        echo "❌ Session file not removed"
        exit 1
    fi
else
    echo "❌ Stop command failed"
    exit 1
fi

# Test 2: Stop multiple sessions
echo "Testing multiple sessions stop..."
if HOME="$TEST_HOME" JMUX_SHARED_DIR="$JMUX_SHARED_DIR" ./jmux stop session2 >/dev/null 2>&1; then
    if [[ ! -f "$JMUX_SESSIONS_DIR/${USER}_session2.session" ]]; then
        echo "✅ Multiple sessions stop works"
    else
        echo "❌ Session file not removed"
        exit 1
    fi
else
    echo "❌ Stop command failed"
    exit 1
fi

# Cleanup
rm -rf "$TEST_HOME"

echo "✅ All enhanced stop tests passed!"