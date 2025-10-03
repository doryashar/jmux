#!/bin/bash
# Test auto display method detection

echo "🧪 Testing auto display method detection"

export JMUX_SHARED_DIR="$HOME/.jmux/shared"
export DMUX_DEBUG=1
export DMUX_MESSAGE_DISPLAY="auto"

# Clean up any existing monitors
pkill -f "_internal_messaging_monitor" 2>/dev/null
rm -f /tmp/dmux-monitor-$(whoami).pid
mkdir -p "$HOME/.jmux/shared/messages"

echo ""
echo "1. 🔍 Checking available display methods..."
echo -n "kdialog: "
which kdialog >/dev/null 2>&1 && echo "✅ Available" || echo "❌ Not available"
echo -n "notify-send: "
which notify-send >/dev/null 2>&1 && echo "✅ Available" || echo "❌ Not available"
echo -n "tmux: "
which tmux >/dev/null 2>&1 && echo "✅ Available" || echo "❌ Not available"

echo ""
echo "2. 🚀 Starting monitor with auto-detection..."
./bin/dmux monitor restart
sleep 2

echo ""
echo "3. 📨 Sending test message (auto-detection)..."
./bin/dmux msg "$(whoami)" "Auto-detection test message!"
sleep 3

echo ""
echo "4. 📨 Sending urgent message..."
cat > "$HOME/.jmux/shared/messages/$(whoami)_urgent_auto_$(date +%s).msg" << EOF
FROM=autotest
TYPE=URGENT
TIMESTAMP=$(date +%s)
DATA=Auto-detection urgent message!
PRIORITY=high
EOF

echo "Waiting for processing..."
sleep 5

echo ""
echo "5. 🛑 Stopping monitor..."
./bin/dmux monitor stop

echo ""
echo "✅ Auto-detection test complete!"
echo ""
echo "💡 Expected behavior:"
echo "   - Monitor auto-detected best available display method"
echo "   - Messages displayed using the best available option"
echo ""
echo "🔧 Display method priority:"
echo "   1. kdialog (KDE)"
echo "   2. notify-send (Desktop notifications)"
echo "   3. tmux (if sessions available)"
echo "   4. terminal (fallback)"