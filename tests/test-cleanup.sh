#!/bin/bash
# Test script for dmux cleanup functionality

set -e

echo "🧪 Testing dmux cleanup functionality"
echo "===================================="

export JMUX_SHARED_DIR="$HOME/.jmux/shared"
DMUX_BIN="$(pwd)/bin/dmux"

if [ ! -f "$DMUX_BIN" ]; then
    echo "❌ dmux binary not found at $DMUX_BIN"
    exit 1
fi

# Test 1: Help text validation
echo "📋 Test 1: Validating help text..."

if $DMUX_BIN cleanup --help | grep -q "Clean up stale session files, orphaned processes"; then
    echo "  ✅ Cleanup help text includes comprehensive description"
else
    echo "  ❌ Cleanup help text missing comprehensive description"
    exit 1
fi

if $DMUX_BIN cleanup --help | grep -q "terminal.*Fix terminal settings"; then
    echo "  ✅ Terminal flag documented"
else
    echo "  ❌ Terminal flag not documented"
    exit 1
fi

if $DMUX_BIN cleanup --help | grep -q "processes.*Kill orphaned processes"; then
    echo "  ✅ Processes flag documented"
else
    echo "  ❌ Processes flag not documented"
    exit 1
fi

if $DMUX_BIN cleanup --help | grep -q "sessions.*Clean session files"; then
    echo "  ✅ Sessions flag documented"
else
    echo "  ❌ Sessions flag not documented"
    exit 1
fi

# Test 2: Flag functionality (non-interactive tests)
echo "📋 Test 2: Testing individual cleanup flags..."

echo "  Testing --sessions flag..."
if $DMUX_BIN cleanup --sessions 2>&1 | grep -q "Cleaning up stale session files"; then
    echo "  ✅ Sessions cleanup runs correctly"
else
    echo "  ❌ Sessions cleanup not working"
    exit 1
fi

echo "  Testing --processes flag..."
if $DMUX_BIN cleanup --processes 2>&1 | grep -q "Cleaning up orphaned processes"; then
    echo "  ✅ Process cleanup runs correctly"
else
    echo "  ❌ Process cleanup not working"
    exit 1
fi

echo "  Testing --terminal flag..."
if $DMUX_BIN cleanup --terminal 2>&1 | grep -q "Fixing terminal settings"; then
    echo "  ✅ Terminal cleanup runs correctly"
else
    echo "  ❌ Terminal cleanup not working"
    exit 1
fi

# Test 3: Default behavior (all flags)
echo "📋 Test 3: Testing default cleanup behavior..."

output=$($DMUX_BIN cleanup 2>&1)

if echo "$output" | grep -q "Fixing terminal settings"; then
    echo "  ✅ Default cleanup includes terminal fixing"
else
    echo "  ❌ Default cleanup missing terminal fixing"
    exit 1
fi

if echo "$output" | grep -q "Cleaning up orphaned processes"; then
    echo "  ✅ Default cleanup includes process cleanup"
else
    echo "  ❌ Default cleanup missing process cleanup"
    exit 1
fi

if echo "$output" | grep -q "Cleaning up stale session files"; then
    echo "  ✅ Default cleanup includes session cleanup"
else
    echo "  ❌ Default cleanup missing session cleanup"
    exit 1
fi

echo ""
echo "🎉 All cleanup tests passed!"
echo ""
echo "📝 Manual testing scenarios:"
echo "   1. Corrupt your terminal: 'cat /bin/ls' (makes terminal unusable)"
echo "   2. Run 'dmux cleanup --terminal' to fix it"
echo "   3. Start dmux sessions and kill them improperly"
echo "   4. Run 'dmux cleanup --processes' to clean up orphaned processes"
echo ""
echo "💡 The cleanup command now provides comprehensive system maintenance:"
echo "   • Terminal restoration (stty sane, reset)"
echo "   • Orphaned process cleanup"
echo "   • Stale session file removal"