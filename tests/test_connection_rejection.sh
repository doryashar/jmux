#!/bin/bash

echo "🧪 Testing connection rejection when limit is reached..."

# Start two background connections that will maintain the connection
echo "🔗 Starting first connection..."
(echo "Connection 1 established" && sleep 30) | /projects/common/work/dory/jmux/bin/jcat -c xsrl8-emp-166:12347 &
CONN1_PID=$!

sleep 2

echo "🔗 Starting second connection..."  
(echo "Connection 2 established" && sleep 30) | /projects/common/work/dory/jmux/bin/jcat -c xsrl8-emp-166:12347 &
CONN2_PID=$!

sleep 2

echo "🚫 Attempting third connection (should be rejected)..."
timeout 5 /projects/common/work/dory/jmux/bin/jcat -c xsrl8-emp-166:12347 < /dev/null &
CONN3_PID=$!

sleep 6

echo "📊 Checking connection results..."
if kill -0 $CONN1_PID 2>/dev/null; then
    echo "   ✅ Connection 1: Still active"
else
    echo "   ❌ Connection 1: Terminated"
fi

if kill -0 $CONN2_PID 2>/dev/null; then
    echo "   ✅ Connection 2: Still active"
else
    echo "   ❌ Connection 2: Terminated"
fi

if kill -0 $CONN3_PID 2>/dev/null; then
    echo "   ❌ Connection 3: Unexpectedly active (should have been rejected)"
else
    echo "   ✅ Connection 3: Properly rejected or failed"
fi

echo "🧹 Cleaning up connections..."
kill $CONN1_PID $CONN2_PID $CONN3_PID 2>/dev/null
wait 2>/dev/null

echo "✅ Connection rejection test completed!"