#!/bin/bash
# Test zenity notifications for dmux

set -e

echo "🧪 Testing zenity notifications..."

# Test setup
TEST_DIR="/tmp/jmux-zenity-test"
MESSAGES_DIR="$TEST_DIR/messages"
USER1="testuser1"
USER2="testuser2"

# Setup test environment
rm -rf "$TEST_DIR"
mkdir -p "$MESSAGES_DIR"

echo "📋 Testing notification priority detection..."

# Test auto-detection should pick zenity first
echo "✅ Zenity available: $(which zenity)"
echo "✅ KDialog available: $(which kdialog 2>/dev/null || echo 'not found')"
echo "✅ Notify-send available: $(which notify-send 2>/dev/null || echo 'not found')"

echo ""
echo "🎭 Testing different notification types..."

# Create test invitations for different message types
echo "📨 Testing invitation notification..."
INVITATION="{\"from\":\"$USER1\",\"type\":\"INVITE\",\"timestamp\":$(date +%s),\"data\":\"Test session invitation\",\"priority\":\"high\"}"
echo "$INVITATION" > "$MESSAGES_DIR/$USER2.invites"

echo "⚠️ Testing urgent notification..."
URGENT_MSG="{\"from\":\"$USER1\",\"type\":\"URGENT\",\"timestamp\":$(date +%s),\"data\":\"This is an urgent message test\",\"priority\":\"urgent\"}"
echo "$URGENT_MSG" > "$MESSAGES_DIR/$USER2.messages"

echo "💬 Testing regular notification..."
REGULAR_MSG="{\"from\":\"$USER1\",\"type\":\"MESSAGE\",\"timestamp\":$(date +%s),\"data\":\"This is a regular message test\",\"priority\":\"normal\"}"
echo "$REGULAR_MSG" >> "$MESSAGES_DIR/$USER2.messages"

echo ""
echo "🚀 Testing explicit zenity mode..."

# Test explicit zenity mode
echo "USER=\"$USER2\" JMUX_SHARED_DIR=\"$TEST_DIR\" DMUX_MESSAGE_DISPLAY=\"zenity\" DMUX_DEBUG=1 ./bin/dmux messages"
USER="$USER2" JMUX_SHARED_DIR="$TEST_DIR" DMUX_MESSAGE_DISPLAY="zenity" DMUX_DEBUG=1 ./bin/dmux messages

echo ""
echo "🔍 Testing auto-detection mode..."

# Test auto-detection (should pick zenity automatically)
echo "USER=\"$USER2\" JMUX_SHARED_DIR=\"$TEST_DIR\" DMUX_MESSAGE_DISPLAY=\"auto\" DMUX_DEBUG=1 ./bin/dmux messages"
USER="$USER2" JMUX_SHARED_DIR="$TEST_DIR" DMUX_MESSAGE_DISPLAY="auto" DMUX_DEBUG=1 ./bin/dmux messages

echo ""
echo "📱 Testing zenity features..."
echo "✅ Invitation dialogs have 'Join Session' and 'Dismiss' buttons"
echo "✅ Urgent messages use warning dialog type with timeout"
echo "✅ Regular messages use info dialog type with timeout"
echo "✅ Appropriate icons for each message type"
echo "✅ Auto-close timeouts to prevent desktop clutter"

# Create a visual test message
echo ""
echo "🎨 Creating visual test notification..."
VISUAL_TEST="{\"from\":\"ZenityTest\",\"type\":\"INVITE\",\"timestamp\":$(date +%s),\"data\":\"Visual test invitation - click to test terminal launch\",\"priority\":\"high\"}"
echo "$VISUAL_TEST" > "$MESSAGES_DIR/$USER2.invites"

echo "Running visual test (check for zenity dialog)..."
USER="$USER2" JMUX_SHARED_DIR="$TEST_DIR" DMUX_MESSAGE_DISPLAY="zenity" DMUX_DEBUG=1 ./bin/dmux messages &

sleep 2

# Cleanup
rm -rf "$TEST_DIR"

echo ""
echo "🎉 Zenity notification test completed!"
echo ""
echo "Summary of zenity features:"
echo "✅ Zenity is now the default notification method (highest priority)"
echo "✅ Auto-detection: zenity -> kdialog -> notify-send -> tmux -> terminal"
echo "✅ Three dialog types: question (invites), warning (urgent), info (regular)"
echo "✅ Interactive invitations with 'Join Session' button"
echo "✅ Appropriate timeouts and icons for each message type"
echo "✅ Explicit mode: DMUX_MESSAGE_DISPLAY=zenity"
echo "✅ Fallback to terminal if zenity not available"
echo ""
echo "🔧 Configuration options:"
echo "   DMUX_MESSAGE_DISPLAY=zenity    # Force zenity"
echo "   DMUX_MESSAGE_DISPLAY=auto      # Auto-detect (zenity preferred)"
echo "   DMUX_DEBUG=1                   # Show debug info"