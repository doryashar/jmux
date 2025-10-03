#!/bin/bash
# Test script for dmux messaging

echo "Testing dmux messaging functionality..."

# Set environment for testing
export JMUX_SHARED_DIR="$HOME/.jmux/shared"
export DMUX_DEBUG=1

# Ensure directories exist
mkdir -p "$HOME/.jmux/shared/messages"

echo "1. Testing status command (should show no messages initially):"
./bin/dmux status

echo ""
echo "2. Creating a test message file..."
cat > "$HOME/.jmux/shared/messages/$(whoami)_test_$(date +%s).msg" << EOF
FROM=testuser
TYPE=MESSAGE
TIMESTAMP=$(date +%s)
DATA=This is a test message
PRIORITY=normal
EOF

echo ""
echo "3. Testing status command (should show the test message):"
./bin/dmux status

echo ""
echo "4. Testing message reading:"
./bin/dmux messages

echo ""
echo "5. Testing status again (messages should be cleared):"
./bin/dmux status

echo ""
echo "Test complete!"