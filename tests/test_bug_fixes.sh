#!/usr/bin/env bash

# Test script for bug fixes: dmux functionality validation

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}Testing dmux bug fixes...${NC}"
echo

# Test environment
export JMUX_SHARED_DIR="/tmp/dmux_test_fixes"
mkdir -p "$JMUX_SHARED_DIR"

# Build dmux binary for testing
DMUX_BINARY="/tmp/dmux_bugfix_test"
if (cd "$(dirname "$0")/../src/jmux-go" && go build -o "$DMUX_BINARY" .); then
    echo -e "${GREEN}✓ dmux binary built successfully${NC}"
else
    echo -e "${RED}✗ Failed to build dmux binary${NC}"
    exit 1
fi

test_session_creation_fixes() {
    echo -e "${YELLOW}Testing session creation bug fixes...${NC}"
    
    local test_config_dir="/tmp/dmux_test_fixes/.config/jmux"
    mkdir -p "$test_config_dir"
    
    # Test with custom config dir
    local original_home="$HOME"
    export HOME="/tmp/dmux_test_fixes"
    
    # Test session creation with argument parsing fix
    local session_name="bug-fix-test-$(date +%s)"
    local output=$("$DMUX_BINARY" share positional_arg --name "$session_name" 2>&1 || true)
    
    if echo "$output" | grep -q "$session_name"; then
        echo -e "${GREEN}✓ Argument parsing fix working (--name takes precedence)${NC}"
    else
        echo -e "${RED}✗ Argument parsing fix not working${NC}"
        echo "Output: $output"
        return 1
    fi
    
    # Test that session creation doesn't hang
    timeout 10s "$DMUX_BINARY" status > /dev/null 2>&1
    if [[ $? -eq 0 ]]; then
        echo -e "${GREEN}✓ Status command doesn't hang${NC}"
    else
        echo -e "${RED}✗ Status command hangs or fails${NC}"
        return 1
    fi
    
    # Restore HOME
    export HOME="$original_home"
}

test_terminal_handling() {
    echo -e "${YELLOW}Testing terminal handling improvements...${NC}"
    
    # Test that dmux handles non-terminal environments gracefully
    local output
    output=$(TERM="" "$DMUX_BINARY" share test-terminal 2>&1 || true)
    
    if echo "$output" | grep -q "background"; then
        echo -e "${GREEN}✓ dmux handles non-terminal environment gracefully${NC}"
    else
        echo -e "${YELLOW}Note: Terminal handling test - output: $output${NC}"
        # This is not necessarily a failure
    fi
    
    # Test cleanup functionality
    local cleanup_output
    cleanup_output=$("$DMUX_BINARY" cleanup 2>&1 || true)
    
    if echo "$cleanup_output" | grep -q "Cleanup complete"; then
        echo -e "${GREEN}✓ Cleanup functionality working${NC}"
    else
        echo -e "${RED}✗ Cleanup functionality not working${NC}"
        echo "Output: $cleanup_output"
        return 1
    fi
}

test_port_management() {
    echo -e "${YELLOW}Testing port management robustness...${NC}"
    
    # Test that dmux properly handles port assignment
    local port_output
    port_output=$("$DMUX_BINARY" share test-port-1 2>&1 || true)
    
    if echo "$port_output" | grep -q "port"; then
        echo -e "${GREEN}✓ Port assignment working${NC}"
    else
        echo -e "${RED}✗ Port assignment not working${NC}"
        echo "Output: $port_output"
        return 1
    fi
    
    # Test session listing
    local session_list
    session_list=$("$DMUX_BINARY" sessions 2>&1 || true)
    
    if echo "$session_list" | grep -q -E "(sessions|Session:)" || echo "$session_list" | grep -q "No active"; then
        echo -e "${GREEN}✓ Session listing working${NC}"
    else
        echo -e "${RED}✗ Session listing not working${NC}"
        echo "Output: $session_list"
        return 1
    fi
}

test_version_command() {
    echo -e "${YELLOW}Testing version command functionality...${NC}"
    
    # Test version command
    local version_output
    version_output=$("$DMUX_BINARY" version 2>&1 || true)
    
    if echo "$version_output" | grep -q -E "v[0-9]+\.[0-9]+\.[0-9]+"; then
        echo -e "${GREEN}✓ Version command returns valid version${NC}"
    else
        echo -e "${RED}✗ Version command not working properly${NC}"
        echo "Output: $version_output"
        return 1
    fi
    
    # Test help command
    local help_output
    help_output=$("$DMUX_BINARY" --help 2>&1 || true)
    
    if echo "$help_output" | grep -q "dmux"; then
        echo -e "${GREEN}✓ Help command working${NC}"
    else
        echo -e "${RED}✗ Help command not working${NC}"
        echo "Output: $help_output"
        return 1
    fi
}

# Main test runner
main() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Bug Fixes Test Suite${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    local tests_run=0
    local tests_passed=0
    
    # Run tests
    if test_session_creation_fixes; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    echo
    
    if test_terminal_handling; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    echo
    
    if test_port_management; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    echo
    
    if test_version_command; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    
    # Cleanup
    rm -rf "$JMUX_SHARED_DIR"
    rm -f "$DMUX_BINARY"
    
    # Report results
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Test Results${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "  Tests run: ${tests_run}"
    echo -e "  Passed: ${GREEN}${tests_passed}${NC}"
    echo -e "  Failed: ${RED}$((tests_run - tests_passed))${NC}"
    
    if [[ $tests_passed -eq $tests_run ]]; then
        echo -e "${GREEN}All bug fix tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    fi
}

main "$@"