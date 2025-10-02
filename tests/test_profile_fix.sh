#!/bin/bash

set -e

echo "Testing profile script creation on remote machine simulation..."

# Simulate a fresh remote machine by removing config
TEST_HOME="/tmp/test_jmux_$(date +%s)"
mkdir -p "$TEST_HOME"

# Set up test environment
export HOME="$TEST_HOME"
export JMUX_CONFIG_DIR="$HOME/.config/jmux"
export JMUX_SETSIZE_SCRIPT="$JMUX_CONFIG_DIR/setsize.sh"

echo "Test HOME: $TEST_HOME"

# Test 1: Verify profile script doesn't exist initially
if [[ -f "$HOME/.config/jmux/profile.sh" ]]; then
    echo "❌ ERROR: Profile script should not exist initially"
    exit 1
fi

# Test 2: Run jmux help command to trigger init_config
echo "Simulating jmux command to trigger init_config..."
cd /home/yashar/projects/jmux

# Run jmux help which should trigger init_config
echo "Running: HOME=\"$TEST_HOME\" JMUX_SHARED_DIR=\"$TEST_HOME/shared\" ./jmux --help"
HOME="$TEST_HOME" JMUX_SHARED_DIR="$TEST_HOME/shared" ./jmux --help

echo "Checking if directories were created:"
ls -la "$TEST_HOME/.config/" 2>/dev/null || echo "No .config directory"

# Test 3: Verify profile script was created
if [[ ! -f "$HOME/.config/jmux/profile.sh" ]]; then
    echo "❌ ERROR: Profile script was not created"
    exit 1
fi

echo "✅ Profile script created successfully"

# Test 4: Verify setsize script was created with profile sourcing
if [[ ! -f "$HOME/.config/jmux/setsize.sh" ]]; then
    echo "❌ ERROR: Setsize script was not created"
    exit 1
fi

if ! grep -q "Source jmux profile" "$HOME/.config/jmux/setsize.sh"; then
    echo "❌ ERROR: Setsize script doesn't source profile"
    exit 1
fi

echo "✅ Setsize script created with profile sourcing"

# Test 5: Verify profile script content
if ! grep -q "export PATH=.*/.local/bin" "$HOME/.config/jmux/profile.sh"; then
    echo "❌ ERROR: Profile script doesn't set PATH correctly"
    exit 1
fi

echo "✅ Profile script has correct PATH setup"

# Cleanup
rm -rf "$TEST_HOME"

echo "✅ All tests passed! Profile script fix should work."