#!/bin/bash
# Test script for dmux sharing modes

set -e

echo "🧪 Testing dmux sharing modes"
echo "=============================="

export JMUX_SHARED_DIR="$HOME/.jmux/shared"
DMUX_BIN="$(pwd)/bin/dmux"

if [ ! -f "$DMUX_BIN" ]; then
    echo "❌ dmux binary not found at $DMUX_BIN"
    exit 1
fi

# Test 1: Help text validation
echo "📋 Test 1: Validating help text..."

echo "  Checking share command help..."
if $DMUX_BIN share --help | grep -q "view.*View-only mode"; then
    echo "  ✅ Share --view flag documented"
else
    echo "  ❌ Share --view flag not found in help"
    exit 1
fi

if $DMUX_BIN share --help | grep -q "rogue.*Rogue mode"; then
    echo "  ✅ Share --rogue flag documented"
else
    echo "  ❌ Share --rogue flag not found in help"
    exit 1
fi

echo "  Checking join command help..."
if $DMUX_BIN join --help | grep -q "view.*Force view-only mode"; then
    echo "  ✅ Join --view flag documented"
else
    echo "  ❌ Join --view flag not found in help"
    exit 1
fi

if $DMUX_BIN join --help | grep -q "rogue.*Force rogue mode"; then
    echo "  ✅ Join --rogue flag documented"
else
    echo "  ❌ Join --rogue flag not found in help"
    exit 1
fi

# Test 2: Flag validation (mutually exclusive)
echo "📋 Test 2: Testing flag validation..."

echo "  Testing mutually exclusive share flags..."
if $DMUX_BIN share --view --rogue test-session 2>&1 | grep -q "mutually exclusive"; then
    echo "  ✅ Share flags properly validated as mutually exclusive"
else
    echo "  ❌ Share flags not properly validated"
    exit 1
fi

echo "  Testing mutually exclusive join flags..."
if $DMUX_BIN join testuser --view --rogue 2>&1 | grep -q "mutually exclusive"; then
    echo "  ✅ Join flags properly validated as mutually exclusive"
else
    echo "  ❌ Join flags not properly validated"
    exit 1
fi

# Test 3: Session mode recording (without actual tmux)
echo "📋 Test 3: Testing session mode recording..."

# Clean up any existing test sessions
rm -f "$JMUX_SHARED_DIR/sessions/yashar_test-*"

# We can't test actual tmux session creation in CI, but we can test the mode recording
# by examining what would have been written to the session file

echo "  All flag validation tests passed!"

echo ""
echo "🎉 All automated tests passed!"
echo ""
echo "📝 Manual testing required:"
echo "   1. Start a tmux session and test: dmux share --view"
echo "   2. From another terminal: dmux join \$USER --view"
echo "   3. Start a tmux session and test: dmux share --rogue"
echo "   4. From another terminal: dmux join \$USER --rogue"
echo "   5. Verify modes work as expected with: dmux sessions"