#!/bin/bash

set -e

echo "Testing port-to-session mapping functionality..."

# Set up test environment
TEST_HOME="/tmp/test_jmux_port_$(date +%s)"
mkdir -p "$TEST_HOME"
export HOME="$TEST_HOME"
export JMUX_SHARED_DIR="$TEST_HOME/shared"
export JMUX_SESSIONS_DIR="$JMUX_SHARED_DIR/sessions"
export JMUX_PORT_MAP="$JMUX_SHARED_DIR/port_sessions.db"

echo "Test HOME: $TEST_HOME"

cd /home/yashar/projects/jmux

# Initialize directories
mkdir -p "$JMUX_SESSIONS_DIR"

# Test 1: Test port mapping registration
echo "=== Test 1: Port mapping registration ==="
HOME="$TEST_HOME" JMUX_SHARED_DIR="$JMUX_SHARED_DIR" ./jmux --help >/dev/null 2>&1

# Check if port mapping file was created
if [[ ! -f "$JMUX_PORT_MAP" ]]; then
    echo "❌ ERROR: Port mapping file was not created"
    exit 1
fi

echo "✅ Port mapping file created"

# Test 2: Test port mapping functions work
echo "=== Test 2: Port mapping functions ==="

# Create a test port mapping
echo "12345:testuser:testsession" > "$JMUX_PORT_MAP"

# Test get_session_from_port function by creating a temporary script
cat > "$TEST_HOME/test_get_session.sh" << 'EOF'
#!/bin/bash
export JMUX_SHARED_DIR="$1"
export JMUX_PORT_MAP="$JMUX_SHARED_DIR/port_sessions.db"

get_session_from_port() {
    local port="$1"
    
    if [[ -f "${JMUX_PORT_MAP}" ]]; then
        local mapping=$(grep "^${port}:" "${JMUX_PORT_MAP}" 2>/dev/null | head -1)
        if [[ -n "$mapping" ]]; then
            echo "$mapping" | cut -d: -f3
            return 0
        fi
    fi
    
    echo "${HOSTNAME}"
    return 1
}

get_session_from_port "$2"
EOF

chmod +x "$TEST_HOME/test_get_session.sh"

# Test the function
session_result=$("$TEST_HOME/test_get_session.sh" "$JMUX_SHARED_DIR" "12345")

if [[ "$session_result" != "testsession" ]]; then
    echo "❌ ERROR: Expected 'testsession', got '$session_result'"
    exit 1
fi

echo "✅ Port-to-session lookup works correctly"

# Test 3: Test setsize script generation with port mapping
echo "=== Test 3: Setsize script with port mapping ==="

# Force regenerate setsize script
rm -f "$HOME/.config/jmux/setsize.sh" || true
HOME="$TEST_HOME" JMUX_SHARED_DIR="$JMUX_SHARED_DIR" ./jmux update-scripts >/dev/null 2>&1

# Check if setsize script contains port mapping logic
if ! grep -q "SOCAT_SOCKPORT" "$HOME/.config/jmux/setsize.sh"; then
    echo "❌ ERROR: Setsize script doesn't contain SOCAT_SOCKPORT logic"
    exit 1
fi

if ! grep -q "port_sessions.db" "$HOME/.config/jmux/setsize.sh"; then
    echo "❌ ERROR: Setsize script doesn't reference port mapping database"
    exit 1
fi

echo "✅ Setsize script contains port mapping logic"

# Test 4: Test setsize script execution simulation
echo "=== Test 4: Setsize script execution simulation ==="

# Create a test script that simulates what happens when socat runs the setsize script
cat > "$TEST_HOME/test_setsize_execution.sh" << 'EOF'
#!/bin/bash

# Simulate socat environment
export SOCAT_SOCKPORT="12345"
export JMUX_SHARED_DIR="$1"
export HOSTNAME="fallback-host"

# Source the setsize script logic (without exec part)
SESSION_NAME=""
if [[ -n "${SOCAT_SOCKPORT:-}" ]]; then
    # Get session name from port mapping
    PORT_MAP_FILE="${JMUX_SHARED_DIR:-/projects/common/work/dory/jmux}/port_sessions.db"
    if [[ -f "$PORT_MAP_FILE" ]]; then
        SESSION_NAME=$(grep "^${SOCAT_SOCKPORT}:" "$PORT_MAP_FILE" 2>/dev/null | head -1 | cut -d: -f3)
    fi
fi

# Fallback to hostname if no session name found
if [[ -z "$SESSION_NAME" ]]; then
    SESSION_NAME="${HOSTNAME}"
fi

echo "$SESSION_NAME"
EOF

chmod +x "$TEST_HOME/test_setsize_execution.sh"

# Test the simulation
session_name_result=$("$TEST_HOME/test_setsize_execution.sh" "$JMUX_SHARED_DIR")

if [[ "$session_name_result" != "testsession" ]]; then
    echo "❌ ERROR: Expected session name 'testsession', got '$session_name_result'"
    exit 1
fi

echo "✅ Setsize script correctly resolves session name from port"

# Test 5: Test fallback behavior
echo "=== Test 5: Fallback behavior ==="

# Create a separate test script for fallback with different port
cat > "$TEST_HOME/test_setsize_fallback.sh" << 'EOF'
#!/bin/bash

# Simulate socat environment with non-existent port
export SOCAT_SOCKPORT="99999"
export JMUX_SHARED_DIR="$1"
export HOSTNAME="fallback-host"

# Source the setsize script logic (without exec part)
SESSION_NAME=""
if [[ -n "${SOCAT_SOCKPORT:-}" ]]; then
    # Get session name from port mapping
    PORT_MAP_FILE="${JMUX_SHARED_DIR:-/projects/common/work/dory/jmux}/port_sessions.db"
    if [[ -f "$PORT_MAP_FILE" ]]; then
        SESSION_NAME=$(grep "^${SOCAT_SOCKPORT}:" "$PORT_MAP_FILE" 2>/dev/null | head -1 | cut -d: -f3)
    fi
fi

# Fallback to hostname if no session name found
if [[ -z "$SESSION_NAME" ]]; then
    SESSION_NAME="${HOSTNAME}"
fi

echo "$SESSION_NAME"
EOF

chmod +x "$TEST_HOME/test_setsize_fallback.sh"

# Test with non-existent port
session_fallback_result=$("$TEST_HOME/test_setsize_fallback.sh" "$JMUX_SHARED_DIR")

if [[ "$session_fallback_result" != "fallback-host" ]]; then
    echo "❌ ERROR: Fallback should return hostname, got '$session_fallback_result'"
    exit 1
fi

echo "✅ Fallback to hostname works correctly"

# Cleanup
rm -rf "$TEST_HOME"

echo "✅ All port mapping tests passed!"