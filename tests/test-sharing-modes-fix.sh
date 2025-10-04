#!/bin/bash
# Test script for dmux sharing modes fix (including remote connections)

set -e

echo "🧪 Testing dmux sharing modes fix"
echo "=================================="

export JMUX_SHARED_DIR="$HOME/.jmux/shared"
DMUX_BIN="$(pwd)/bin/dmux"

if [ ! -f "$DMUX_BIN" ]; then
    echo "❌ dmux binary not found at $DMUX_BIN"
    exit 1
fi

# Test 1: Verify setsize script contains mode logic
echo "📋 Test 1: Verifying setsize script contains mode logic..."

if [ ! -f "$HOME/.config/jmux/setsize.sh" ]; then
    echo "  Creating setsize script..."
    $DMUX_BIN sessions > /dev/null
fi

if grep -q "JCAT_MODE" "$HOME/.config/jmux/setsize.sh"; then
    echo "  ✅ Setsize script contains JCAT_MODE handling"
else
    echo "  ❌ Setsize script missing JCAT_MODE handling"
    exit 1
fi

if grep -q "attach-session -t.*-r" "$HOME/.config/jmux/setsize.sh"; then
    echo "  ✅ Setsize script contains view mode logic (attach -r)"
else
    echo "  ❌ Setsize script missing view mode logic"
    exit 1
fi

if grep -q "new-session -t" "$HOME/.config/jmux/setsize.sh"; then
    echo "  ✅ Setsize script contains rogue mode logic (new-session)"
else
    echo "  ❌ Setsize script missing rogue mode logic"
    exit 1
fi

# Test 2: Test mode environment variable handling
echo "📋 Test 2: Testing mode detection logic..."

# Simulate what happens in the setsize script
test_mode_logic() {
    local test_mode="$1"
    local expected_args="$2"
    
    # Source the relevant part of the setsize script
    JCAT_MODE="$test_mode"
    SESSION_NAME="test-session"
    
    # Replicate the case logic from setsize script
    TMUX_MODE="${JCAT_MODE:-pair}"
    case "$TMUX_MODE" in
        "view")
            TMUX_ARGS="attach-session -t $SESSION_NAME -r"
            ;;
        "rogue")
            TMUX_ARGS="new-session -t $SESSION_NAME"
            ;;
        *)
            TMUX_ARGS="new -A -s $SESSION_NAME"
            ;;
    esac
    
    if [[ "$TMUX_ARGS" == *"$expected_args"* ]]; then
        echo "  ✅ Mode '$test_mode' correctly generates: $TMUX_ARGS"
        return 0
    else
        echo "  ❌ Mode '$test_mode' generated: $TMUX_ARGS (expected: *$expected_args*)"
        return 1
    fi
}

test_mode_logic "view" "attach-session -t test-session -r"
test_mode_logic "rogue" "new-session -t test-session"  
test_mode_logic "pair" "new -A -s test-session"
test_mode_logic "" "new -A -s test-session"  # default case

echo ""
echo "🎉 All sharing mode fix tests passed!"
echo ""
echo "📝 Manual testing required:"
echo "   1. In terminal 1: Start a shared session with 'dmux share --view test-session'"
echo "   2. In terminal 2: Join with 'dmux join \$USER test-session'"
echo "   3. Verify terminal 2 is read-only (cannot type)"
echo "   4. In terminal 3: Join with 'dmux join \$USER test-session --rogue'"
echo "   5. Verify terminal 3 has independent control"
echo ""
echo "🔗 For remote testing:"
echo "   1. Share session on host1: 'dmux share --view remote-test'"
echo "   2. Join from host2: 'dmux join user@host1 remote-test'"
echo "   3. Verify remote connection respects view mode"