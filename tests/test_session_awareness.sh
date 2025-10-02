#!/usr/bin/env bash

# Test script for session awareness features

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}Testing jmux session awareness features...${NC}"
echo

# Test environment
export JMUX_SHARED_DIR="/tmp/jmux_test_awareness"
mkdir -p "$JMUX_SHARED_DIR"

test_session_context_functions() {
    echo -e "${YELLOW}Testing session context detection functions...${NC}"
    
    # Source the jmux script to test internal functions
    source ./jmux
    
    # Test outside tmux
    echo -e "${BLUE}Test 1: Context detection outside tmux...${NC}"
    local context_info=$(get_session_context)
    local context=$(echo "$context_info" | grep "^context=" | cut -d'=' -f2)
    
    if [[ "$context" == "none" ]]; then
        echo -e "${GREEN}✓ Correctly detected no tmux session${NC}"
    else
        echo -e "${RED}✗ Expected 'none', got '$context'${NC}"
        return 1
    fi
    
    # Test helper functions
    if ! in_shared_session; then
        echo -e "${GREEN}✓ Correctly detected not in shared session${NC}"
    else
        echo -e "${RED}✗ Should not be in shared session${NC}"
        return 1
    fi
    
    if ! hosting_session; then
        echo -e "${GREEN}✓ Correctly detected not hosting session${NC}"
    else
        echo -e "${RED}✗ Should not be hosting session${NC}"
        return 1
    fi
}

test_context_command() {
    echo -e "${YELLOW}Testing context command...${NC}"
    
    # Test context command outside tmux
    echo -e "${BLUE}Test: Context command outside tmux...${NC}"
    local output=$(./jmux context 2>&1)
    
    if echo "$output" | grep -q "Not in a tmux session"; then
        echo -e "${GREEN}✓ Context command works outside tmux${NC}"
    else
        echo -e "${RED}✗ Context command failed outside tmux${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_enhanced_status() {
    echo -e "${YELLOW}Testing enhanced status command...${NC}"
    
    echo -e "${BLUE}Test: Enhanced status command...${NC}"
    local output=$(./jmux status 2>&1)
    
    if echo "$output" | grep -q "jmux Status" && echo "$output" | grep -q "Not in tmux"; then
        echo -e "${GREEN}✓ Enhanced status command works${NC}"
    else
        echo -e "${RED}✗ Enhanced status command failed${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_enhanced_users_command() {
    echo -e "${YELLOW}Testing enhanced users command...${NC}"
    
    echo -e "${BLUE}Test: Users command outside tmux...${NC}"
    local output=$(./jmux users 2>&1)
    
    if echo "$output" | grep -q "Not in a tmux session"; then
        echo -e "${GREEN}✓ Users command handles no tmux session correctly${NC}"
    else
        echo -e "${RED}✗ Users command failed outside tmux${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_help_documentation() {
    echo -e "${YELLOW}Testing updated help documentation...${NC}"
    
    echo -e "${BLUE}Test: Help includes new context command...${NC}"
    local output=$(./jmux help 2>&1)
    
    if echo "$output" | grep -q "context.*Show current session context"; then
        echo -e "${GREEN}✓ Help documentation includes context command${NC}"
    else
        echo -e "${RED}✗ Help documentation missing context command${NC}"
        echo "Searching for: context.*Show current session context"
        return 1
    fi
    
    if echo "$output" | grep -q "session-aware"; then
        echo -e "${GREEN}✓ Help documentation mentions session-aware features${NC}"
    else
        echo -e "${RED}✗ Help documentation missing session-aware mentions${NC}"
        return 1
    fi
}

# Main test runner
main() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Session Awareness Test Suite${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    local tests_run=0
    local tests_passed=0
    
    # Run tests
    if test_session_context_functions; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    echo
    
    if test_context_command; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    echo
    
    if test_enhanced_status; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    echo
    
    if test_enhanced_users_command; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    echo
    
    if test_help_documentation; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    
    # Cleanup
    rm -rf "$JMUX_SHARED_DIR"
    
    # Report results
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Test Results${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "  Tests run: ${tests_run}"
    echo -e "  Passed: ${GREEN}${tests_passed}${NC}"
    echo -e "  Failed: ${RED}$((tests_run - tests_passed))${NC}"
    
    if [[ $tests_passed -eq $tests_run ]]; then
        echo -e "${GREEN}All session awareness tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    fi
}

main "$@"