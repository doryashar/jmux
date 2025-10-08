#!/bin/bash

# Test script for connection limiting functionality
echo "🧪 Testing connection limit functionality..."

HOST_USER="dory"
SESSION_NAME="test-limit-session"
HOST="xsrl8-emp-166"
PORT="12347"

echo "📡 Testing connection to ${HOST_USER}'s session: ${SESSION_NAME}"

# Test function to attempt connection
test_connection() {
    local conn_id=$1
    echo "🔗 Attempting connection #${conn_id}..."
    
    # Use timeout to prevent hanging
    timeout 10 /projects/common/work/dory/jmux/bin/dmux join "${HOST_USER}" "${SESSION_NAME}" 2>&1 | tee "/tmp/connection_${conn_id}.log" &
    
    local pid=$!
    echo "   Started connection #${conn_id} with PID: ${pid}"
    
    # Give it a moment to establish connection
    sleep 2
    
    # Check if process is still running (successful connection)
    if kill -0 "${pid}" 2>/dev/null; then
        echo "   ✅ Connection #${conn_id} established successfully"
        return 0
    else
        echo "   ❌ Connection #${conn_id} failed or rejected"
        return 1
    fi
}

# Test multiple connections
echo "🚀 Starting connection tests..."

# First connection
test_connection 1
conn1_result=$?

# Second connection
test_connection 2  
conn2_result=$?

# Third connection (should be rejected due to limit of 2)
test_connection 3
conn3_result=$?

echo ""
echo "📊 Test Results:"
echo "   Connection 1: $([ $conn1_result -eq 0 ] && echo "✅ Success" || echo "❌ Failed")"
echo "   Connection 2: $([ $conn2_result -eq 0 ] && echo "✅ Success" || echo "❌ Failed")"  
echo "   Connection 3: $([ $conn3_result -eq 0 ] && echo "❌ Unexpected Success (should be rejected)" || echo "✅ Properly Rejected")"

echo ""
echo "📋 Connection logs:"
for i in 1 2 3; do
    if [ -f "/tmp/connection_${i}.log" ]; then
        echo "--- Connection ${i} log ---"
        cat "/tmp/connection_${i}.log"
        echo ""
    fi
done

# Clean up background processes
echo "🧹 Cleaning up test connections..."
pkill -f "dmux join ${HOST_USER} ${SESSION_NAME}" 2>/dev/null || true

echo "✅ Connection limit test completed!"