#!/bin/bash
# Test robust terminal launching for dmux invitations

set -e

echo "🧪 Testing robust terminal launching..."

# Test setup
TEST_DIR="/tmp/jmux-terminal-test"
MESSAGES_DIR="$TEST_DIR/messages"
USER1="testuser1"
USER2="testuser2"

# Setup test environment
rm -rf "$TEST_DIR"
mkdir -p "$MESSAGES_DIR"

echo "📋 Testing terminal and dmux detection..."

# Test dmux executable detection
echo "✅ Testing dmux path detection:"
DMUX_PATH=$(cd /home/yashar/projects/jmux/src/jmux-go && ./bin/dmux --help >/dev/null 2>&1 && echo "/home/yashar/projects/jmux/src/jmux-go/bin/dmux" || echo "dmux not found")
echo "   Dmux executable: $DMUX_PATH"

# Test terminal detection
echo "✅ Testing terminal emulator detection:"
AVAILABLE_TERMINALS=""
for term in konsole gnome-terminal xfce4-terminal mate-terminal xterm urxvt terminator tilix alacritty kitty x-terminal-emulator; do
    if command -v "$term" >/dev/null 2>&1; then
        AVAILABLE_TERMINALS="$AVAILABLE_TERMINALS $term"
        echo "   Found: $term"
    fi
done

if [ -z "$AVAILABLE_TERMINALS" ]; then
    echo "   ⚠️ No terminal emulators found - this may cause issues"
else
    echo "   ✅ Available terminals:$AVAILABLE_TERMINALS"
fi

echo ""
echo "🔧 Testing PATH resolution..."

# Test PATH enhancement
echo "Current PATH: $PATH"
ENHANCED_PATH="$PATH:/usr/local/bin:/usr/bin:$HOME/.local/bin:$HOME/bin"
echo "Enhanced PATH: $ENHANCED_PATH"

# Test dmux availability in common locations
echo ""
echo "📍 Testing dmux in common locations:"
COMMON_PATHS="/usr/local/bin/dmux /usr/bin/dmux $HOME/.local/bin/dmux $HOME/bin/dmux"
for path in $COMMON_PATHS; do
    if [ -f "$path" ]; then
        echo "   ✅ Found: $path"
    else
        echo "   ❌ Not found: $path"
    fi
done

echo ""
echo "🎭 Testing invitation terminal launch..."

# Create test invitation
INVITATION="{\"from\":\"$USER1\",\"type\":\"INVITE\",\"timestamp\":$(date +%s),\"data\":\"Terminal launch test session\",\"priority\":\"high\"}"
echo "$INVITATION" > "$MESSAGES_DIR/$USER2.invites"

echo "📨 Created test invitation - you should see a notification dialog"
echo "   Click 'Join Session' to test terminal launching"
echo "   The terminal should open with the dmux join command"
echo ""

# Test with debug enabled
echo "USER=\"$USER2\" JMUX_SHARED_DIR=\"$TEST_DIR\" DMUX_MESSAGE_DISPLAY=\"zenity\" DMUX_DEBUG=1 ./bin/dmux messages"

# Run the test (this will show the zenity dialog)
USER="$USER2" JMUX_SHARED_DIR="$TEST_DIR" DMUX_MESSAGE_DISPLAY="zenity" DMUX_DEBUG=1 ./bin/dmux messages &

# Wait a moment for the dialog to appear
sleep 2

echo ""
echo "🔍 Debug information that should appear:"
echo "   [DEBUG] Using dmux path: <path_to_dmux>"
echo "   [DEBUG] Using terminal: <terminal_command> <args>"
echo "   [DEBUG] Terminal command: <full_command>"
echo "   [DEBUG] User clicked Join Session (if user clicks Join)"
echo ""

# Cleanup
sleep 5
rm -rf "$TEST_DIR"

echo ""
echo "🎉 Terminal launching test completed!"
echo ""
echo "Summary of improvements:"
echo "✅ Robust dmux executable detection (PATH + common locations)"
echo "✅ Automatic terminal emulator detection (11 different terminals)"
echo "✅ Enhanced PATH in spawned terminal environment"
echo "✅ Absolute paths to prevent 'command not found' errors"
echo "✅ Debug output for troubleshooting"
echo "✅ Fallback mechanisms for edge cases"
echo ""
echo "Supported terminals (in priority order):"
echo "  konsole, gnome-terminal, xfce4-terminal, mate-terminal,"
echo "  xterm, urxvt, terminator, tilix, alacritty, kitty,"
echo "  x-terminal-emulator (fallback)"
echo ""
echo "🔧 Troubleshooting:"
echo "  - Set DMUX_DEBUG=1 to see detailed launch information"
echo "  - Ensure dmux is in PATH or installed in common locations"
echo "  - Check that a terminal emulator is installed"