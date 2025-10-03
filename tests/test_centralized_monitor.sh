#!/bin/bash
# Test centralized messaging monitor

echo "🧪 Testing centralized messaging monitor with kdialog"

export JMUX_SHARED_DIR="$HOME/.jmux/shared"
export DMUX_DEBUG=1
export DMUX_MESSAGE_DISPLAY="kdialog"

# Clean up any existing monitors
pkill -f "_internal_messaging_monitor" 2>/dev/null
rm -f /tmp/dmux-monitor-$(whoami).pid
mkdir -p "$HOME/.jmux/shared/messages"

echo ""
echo "1. 📊 Checking monitor status..."
./bin/dmux monitor status

echo ""
echo "2. 🚀 Starting monitor..."
./bin/dmux monitor start
sleep 2

echo ""
echo "3. 📊 Checking monitor status again..."
./bin/dmux monitor status

echo ""
echo "4. 📨 Sending test message (should trigger kdialog)..."
./bin/dmux msg "$(whoami)" "Test message for kdialog display!"
sleep 3

echo ""
echo "5. 📨 Sending urgent message..."
cat > "$HOME/.jmux/shared/messages/$(whoami)_urgent_$(date +%s).msg" << EOF
FROM=testuser
TYPE=URGENT
TIMESTAMP=$(date +%s)
DATA=This is an urgent test message!
PRIORITY=high
EOF

echo "Waiting for message processing..."
sleep 5

echo ""
echo "6. 📨 Sending invite message..."
cat > "$HOME/.jmux/shared/messages/$(whoami)_invite_$(date +%s).msg" << EOF
FROM=inviter
TYPE=INVITE
TIMESTAMP=$(date +%s)
DATA=test-session
PRIORITY=normal
EOF

echo "Waiting for invite processing..."
sleep 5

echo ""
echo "7. 🛑 Stopping monitor..."
./bin/dmux monitor stop

echo ""
echo "8. 📊 Final status check..."
./bin/dmux monitor status

echo ""
echo "✅ Test complete!"
echo ""
echo "💡 Expected behavior:"
echo "   - Three kdialog popups should have appeared"
echo "   - Monitor should start/stop cleanly"
echo "   - PID file management should work"
echo ""
echo "🔧 To switch back to terminal display:"
echo "   export DMUX_MESSAGE_DISPLAY=terminal"
echo ""
echo "🔧 To switch to tmux display:"
echo "   export DMUX_MESSAGE_DISPLAY=tmux"