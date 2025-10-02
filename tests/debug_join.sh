#!/bin/bash

echo "=== Debug Join Process ==="

# Simulate the join process
export JMUX_SHARED_DIR="/projects/common/work/dory/jmux"
PORT_MAP_FILE="$JMUX_SHARED_DIR/port_sessions.db"

echo "1. Checking port mapping file: $PORT_MAP_FILE"

if [[ -f "$PORT_MAP_FILE" ]]; then
    echo "✅ Port mapping file exists"
    echo "Contents:"
    cat "$PORT_MAP_FILE"
    echo ""
else
    echo "❌ Port mapping file does not exist"
fi

echo "2. Simulating setsize script execution with SOCAT_SOCKPORT=12345"

# Simulate what the setsize script does
export SOCAT_SOCKPORT="12345"
SESSION_NAME=""

if [[ -n "${SOCAT_SOCKPORT:-}" ]]; then
    echo "  SOCAT_SOCKPORT is set to: $SOCAT_SOCKPORT"
    if [[ -f "$PORT_MAP_FILE" ]]; then
        echo "  Searching for port mapping..."
        SESSION_NAME=$(grep "^${SOCAT_SOCKPORT}:" "$PORT_MAP_FILE" 2>/dev/null | head -1 | cut -d: -f3)
        echo "  Found session name: '$SESSION_NAME'"
    else
        echo "  Port mapping file not found"
    fi
fi

if [[ -z "$SESSION_NAME" ]]; then
    SESSION_NAME="${HOSTNAME}"
    echo "  Falling back to hostname: $SESSION_NAME"
fi

echo "3. Final SESSION_NAME: '$SESSION_NAME'"

if [[ -z "$SESSION_NAME" ]]; then
    echo "❌ ERROR: SESSION_NAME is empty!"
else
    echo "✅ SESSION_NAME is valid"
fi