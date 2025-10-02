#!/usr/bin/env bash

# Test simulating the exact permission scenario the user experienced

set -euo pipefail

echo "Testing scenario where realpath fails due to permissions..."

# Create a test environment
TEST_DIR="/tmp/jmux_permission_test"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"

# Create a script that simulates the jmux symlink function
cat > "$TEST_DIR/test_jmux_symlink.sh" << 'EOF'
#!/usr/bin/env bash

# Simulated version of the improved setup_jmux_symlink function
setup_jmux_symlink() {
    local jmux_script=""
    local local_bin="${HOME}/.local/bin"
    local jmux_link="${local_bin}/jmux"
    
    echo "Attempting path resolution for: $0"
    
    # Try different methods to get absolute path
    if command -v realpath &> /dev/null; then
        echo "Trying realpath..."
        jmux_script="$(realpath "$0" 2>/dev/null)" || jmux_script=""
        if [[ -n "$jmux_script" ]]; then
            echo "✓ realpath succeeded: $jmux_script"
        else
            echo "! realpath failed"
        fi
    fi
    
    # Fallback to readlink if realpath failed
    if [[ -z "$jmux_script" ]] && command -v readlink &> /dev/null; then
        echo "Trying readlink fallback..."
        jmux_script="$(readlink -f "$0" 2>/dev/null)" || jmux_script=""
        if [[ -n "$jmux_script" ]]; then
            echo "✓ readlink succeeded: $jmux_script"
        else
            echo "! readlink failed"
        fi
    fi
    
    # Fallback to manual resolution
    if [[ -z "$jmux_script" ]]; then
        echo "Using manual resolution..."
        if [[ "$0" = /* ]]; then
            # Already absolute path
            jmux_script="$0"
            echo "✓ Already absolute: $jmux_script"
        else
            # Make relative path absolute
            jmux_script="$(pwd)/$0"
            echo "✓ Manual resolution: $jmux_script"
        fi
    fi
    
    # Verify the script path exists and is readable
    if [[ ! -r "$jmux_script" ]]; then
        echo "Warning: Cannot resolve jmux script path, skipping symlink creation"
        return 0
    fi
    
    # Skip if we're already running from ~/.local/bin
    if [[ "$jmux_script" == "$jmux_link" ]]; then
        echo "Already running from ~/.local/bin, skipping"
        return 0
    fi
    
    # Ensure ~/.local/bin exists
    mkdir -p "$local_bin"
    
    # Create symlink if it doesn't exist or points to wrong location
    if [[ ! -L "$jmux_link" ]] || [[ "$(readlink "$jmux_link" 2>/dev/null)" != "$jmux_script" ]]; then
        echo "Creating jmux symlink in ${local_bin}..."
        if ln -sf "$jmux_script" "$jmux_link" 2>/dev/null; then
            echo "✓ jmux symlink created: ${jmux_link} -> ${jmux_script}"
        else
            echo "Warning: Failed to create symlink (permission denied)"
        fi
    else
        echo "Symlink already exists and is correct"
    fi
}

# Test both working and failing scenarios
echo "=== Test 1: Normal scenario ==="
setup_jmux_symlink

echo
echo "=== Test 2: Simulating realpath failure ==="
# Temporarily disable realpath
realpath() {
    echo "realpath: $1: Permission denied" >&2
    return 1
}
export -f realpath

setup_jmux_symlink
EOF

chmod +x "$TEST_DIR/test_jmux_symlink.sh"

# Run the test
echo "Running symlink test..."
cd "$TEST_DIR"
HOME="$TEST_DIR" ./test_jmux_symlink.sh

# Check results
echo
echo "Test results:"
if [[ -L "$TEST_DIR/.local/bin/jmux" ]]; then
    echo "✓ Symlink was created successfully"
    echo "Target: $(readlink "$TEST_DIR/.local/bin/jmux")"
else
    echo "✗ No symlink was created"
fi

# Cleanup
rm -rf "$TEST_DIR"

echo "✓ Permission scenario test completed"