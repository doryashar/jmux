#!/bin/bash
# Test monitor logging functionality

echo "🧪 Testing monitor logging functionality"

export JMUX_SHARED_DIR="$HOME/.jmux/shared"
export DMUX_DEBUG=1

# Clean up
pkill -f "_internal_messaging_monitor" 2>/dev/null
rm -f /tmp/dmux-monitor-$(whoami).pid
rm -f "$HOME/.config/jmux/monitor.log"
mkdir -p "$HOME/.jmux/shared/messages"
mkdir -p "$HOME/.config/jmux"

echo ""
echo "1. 📊 Checking logs (should be empty)..."
./bin/dmux monitor logs -n 10

echo ""
echo "2. 🚀 Starting monitor with logging..."
./bin/dmux monitor start
sleep 2

echo ""
echo "3. 📨 Sending test messages to generate logs..."
./bin/dmux msg "$(whoami)" "Test message for logging!"
sleep 2

cat > "$HOME/.jmux/shared/messages/$(whoami)_urgent_$(date +%s).msg" << EOF
FROM=logger-test
TYPE=URGENT
TIMESTAMP=$(date +%s)
DATA=Urgent test message for logging
PRIORITY=high
EOF

sleep 3

echo ""
echo "4. 📜 Checking monitor logs..."
./bin/dmux monitor logs -n 20

echo ""
echo "5. 📍 Showing log file location..."
echo "Log file: $HOME/.config/jmux/monitor.log"
echo "Available commands:"
echo "  dmux monitor logs           # Show last 50 lines"
echo "  dmux monitor logs -n 100    # Show last 100 lines"
echo "  dmux monitor logs -f        # Follow logs (like tail -f)"

echo ""
echo "6. 🛑 Stopping monitor..."
./bin/dmux monitor stop

echo ""
echo "7. 📜 Final logs after stopping..."
./bin/dmux monitor logs -n 10

echo ""
echo "✅ Monitor logging test complete!"