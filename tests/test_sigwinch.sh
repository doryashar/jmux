#!/bin/bash

echo "🧪 Testing SIGWINCH propagation in jcat reverse connection..."
echo

# Kill existing test session
tmux kill-session -t sigwinch_test 2>/dev/null || true
sleep 1

# Create test logs directory
mkdir -p /tmp/sigwinch-test

echo "1️⃣ Starting jcat reverse-listen server..."
tmux new-session -d -s sigwinch_test
tmux send-keys -t sigwinch_test '/projects/common/work/dory/jmux/bin/jcat reverse-listen :12399' Enter

sleep 2

echo "2️⃣ Testing connection from client..."
echo "   You should manually run: /projects/common/work/dory/jmux/bin/jcat reverse-connect localhost:12399"
echo "   Then try resizing your terminal window to test SIGWINCH propagation"
echo
echo "3️⃣ To stop the test:"
echo "   tmux kill-session -t sigwinch_test"
echo
echo "✅ Server is running. Connect from another terminal and resize to test."