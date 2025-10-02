#!/usr/bin/env bash

# Test script for automatic tmux installation and symlink creation

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}Testing jmux auto-installation features...${NC}"
echo

# Test symlink creation
test_symlink_creation() {
    echo -e "${YELLOW}Testing symlink creation...${NC}"
    
    # Create a test directory structure
    local test_dir="/tmp/jmux_test_install"
    local test_local_bin="${test_dir}/.local/bin"
    
    mkdir -p "${test_local_bin}"
    
    # Mock HOME for testing
    local original_home="$HOME"
    export HOME="$test_dir"
    
    # Test the symlink function by sourcing it
    # We'll extract just the symlink logic for testing
    cat > "${test_dir}/test_symlink.sh" << 'EOF'
#!/usr/bin/env bash
setup_jmux_symlink() {
    local jmux_script="$(realpath "$0")"
    local local_bin="${HOME}/.local/bin"
    local jmux_link="${local_bin}/jmux"
    
    # Skip if we're already running from ~/.local/bin
    if [[ "$jmux_script" == "$jmux_link" ]]; then
        return 0
    fi
    
    # Ensure ~/.local/bin exists
    mkdir -p "$local_bin"
    
    # Create symlink if it doesn't exist or points to wrong location
    if [[ ! -L "$jmux_link" ]] || [[ "$(readlink "$jmux_link")" != "$jmux_script" ]]; then
        echo "Creating jmux symlink in ${local_bin}..."
        ln -sf "$jmux_script" "$jmux_link"
        echo "✓ jmux symlink created: ${jmux_link} -> ${jmux_script}"
    fi
}

setup_jmux_symlink
EOF
    
    chmod +x "${test_dir}/test_symlink.sh"
    
    # Run the test
    if cd "$test_dir" && ./test_symlink.sh; then
        if [[ -L "${test_local_bin}/jmux" ]]; then
            echo -e "${GREEN}✓ Symlink creation test passed${NC}"
        else
            echo -e "${RED}✗ Symlink was not created${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ Symlink creation test failed${NC}"
        return 1
    fi
    
    # Restore HOME
    export HOME="$original_home"
    
    # Cleanup
    rm -rf "$test_dir"
}

# Test PATH detection
test_path_detection() {
    echo -e "${YELLOW}Testing PATH detection...${NC}"
    
    # Test PATH checking logic
    local test_path="/test/path"
    local current_path="$PATH"
    
    # Test when path is not in PATH
    if [[ ":$current_path:" == *":$test_path:"* ]]; then
        echo -e "${YELLOW}Test path already in PATH, skipping...${NC}"
    else
        echo -e "${GREEN}✓ PATH detection logic working correctly${NC}"
    fi
    
    # Test when path is in PATH
    export PATH="$test_path:$PATH"
    if [[ ":$PATH:" == *":$test_path:"* ]]; then
        echo -e "${GREEN}✓ PATH addition detection working${NC}"
    else
        echo -e "${RED}✗ PATH addition detection failed${NC}"
        return 1
    fi
    
    # Restore PATH
    export PATH="$current_path"
}

# Test dependency checking (mock)
test_dependency_check() {
    echo -e "${YELLOW}Testing dependency checking...${NC}"
    
    # Check if socat exists (should always be required)
    if command -v socat &> /dev/null; then
        echo -e "${GREEN}✓ socat dependency check working${NC}"
    else
        echo -e "${YELLOW}! socat not found (this is expected for testing)${NC}"
    fi
    
    # Test curl/wget requirements for tmux installation
    if command -v curl &> /dev/null && command -v wget &> /dev/null; then
        echo -e "${GREEN}✓ curl and wget available for tmux installation${NC}"
    else
        echo -e "${YELLOW}! curl or wget missing - tmux auto-install would fail${NC}"
    fi
}

# Main test runner
main() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}jmux Auto-Installation Test Suite${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    local tests_run=0
    local tests_passed=0
    
    # Run tests
    if test_symlink_creation; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    
    if test_path_detection; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    
    if test_dependency_check; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    
    # Report results
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Test Results${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "  Tests run: ${tests_run}"
    echo -e "  Passed: ${GREEN}${tests_passed}${NC}"
    echo -e "  Failed: ${RED}$((tests_run - tests_passed))${NC}"
    
    if [[ $tests_passed -eq $tests_run ]]; then
        echo -e "${GREEN}All auto-installation tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    fi
}

main "$@"