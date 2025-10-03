#!/bin/bash
# Final test to demonstrate messaging works in tmux sessions

echo "🧪 Final messaging test - real-time messaging in tmux sessions"

export JMUX_SHARED_DIR="$HOME/.jmux/shared"
export DMUX_DEBUG=1

# Cleanup
pkill -f "_internal_messaging_monitor" 2>/dev/null
tmux kill-session -t dmux-main 2>/dev/null
mkdir -p "$HOME/.jmux/shared/messages"

echo ""
echo "1. 🚀 Starting dmux messaging monitor (simulates real dmux session)..."
./bin/dmux _internal_messaging_monitor &
MONITOR_PID=$!
echo "   Monitor PID: $MONITOR_PID"

echo ""
echo "2. 📺 Creating tmux session 'dmux-main' (simulates dmux session)..."
tmux new-session -d -s dmux-main "echo 'This is a dmux tmux session. Messaging should work here.'; sleep 30"

echo ""
echo "3. 📩 Sending messages and testing real-time display..."

# Send regular message
echo "   → Sending regular message..."
./bin/dmux msg "$(whoami)" "This is a test message - should appear in tmux!"

sleep 2

# Send urgent message  
echo "   → Sending urgent message..."
cat > "$HOME/.jmux/shared/messages/$(whoami)_urgent_$(date +%s).msg" << EOF
FROM=urgent-sender
TYPE=URGENT
TIMESTAMP=$(date +%s)
DATA=URGENT: This should appear immediately in tmux!
PRIORITY=high
EOF

sleep 2

# Send invite message
echo "   → Sending invite message..."
cat > "$HOME/.jmux/shared/messages/$(whoami)_invite_$(date +%s).msg" << EOF
FROM=inviter
TYPE=INVITE
TIMESTAMP=$(date +%s)
DATA=shared-session
PRIORITY=normal
EOF

echo ""
echo "4. ⏱️  Waiting for message processing and auto-cleanup..."
sleep 6

echo ""
echo "5. 🧹 Checking cleanup - messages should be auto-removed..."
MSG_COUNT=$(ls "$HOME/.jmux/shared/messages/"$(whoami)"_"* 2>/dev/null | wc -l)
echo "   Remaining messages: $MSG_COUNT"

echo ""
echo "6. 🧪 Testing status command with messages..."
./bin/dmux status

echo ""
echo "7. 🧼 Cleanup..."
kill $MONITOR_PID 2>/dev/null
tmux kill-session -t dmux-main 2>/dev/null

echo ""
echo "✅ Test complete!"
echo ""
echo "💡 Expected behavior:"
echo "   - You should see tmux display messages at the bottom of your terminal"
echo "   - Messages should auto-disappear after 5 seconds"  
echo "   - Different message types should have different icons (💬, 🚨, 📨)"
echo "   - Messages should be auto-cleaned after display"
echo ""
echo "🎯 If you saw the tmux messages, real-time messaging is working!"