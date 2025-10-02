#!/usr/bin/env bash

# Test script for tmux pass-through functionality

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}Testing jmux tmux pass-through functionality...${NC}"
echo

# Test environment
export JMUX_SHARED_DIR="/tmp/jmux_test_passthrough"
mkdir -p "$JMUX_SHARED_DIR"

test_basic_tmux_commands() {
    echo -e "${YELLOW}Testing basic tmux command forwarding...${NC}"
    
    # Test ls command
    echo -e "${BLUE}Test 1: jmux ls (list sessions)...${NC}"
    local output=$(./jmux ls 2>&1)
    
    if echo "$output" | grep -q "jmux-enhanced" && echo "$output" | grep -q "Tip: Use 'jmux sessions'"; then
        echo -e "${GREEN}✓ jmux ls works with enhancements${NC}"
    else
        echo -e "${RED}✗ jmux ls failed or missing enhancements${NC}"
        echo "Output: $output"
        return 1
    fi
    
    # Test list-commands
    echo -e "${BLUE}Test 2: jmux list-commands...${NC}"
    local output=$(./jmux list-commands 2>&1 | head -3)
    
    if echo "$output" | grep -q "attach-session" && echo "$output" | grep -q "Forwarding to tmux"; then
        echo -e "${GREEN}✓ jmux list-commands forwards correctly${NC}"
    else
        echo -e "${RED}✗ jmux list-commands failed${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_enhanced_tmux_commands() {
    echo -e "${YELLOW}Testing enhanced tmux commands...${NC}"
    
    # Test attach command when already in tmux
    echo -e "${BLUE}Test 1: Enhanced attach behavior...${NC}"
    # This test requires being in tmux, so we'll skip for now
    echo -e "${YELLOW}Skipping attach test (requires tmux session)${NC}"
    
    # Test new command
    echo -e "${BLUE}Test 2: Enhanced new command...${NC}"
    # Since we can't actually create a session in tests, we'll test the detection
    if grep -q "Creating new tmux session" "./jmux" && grep -q "Tip: Use 'jmux share'" "./jmux"; then
        echo -e "${GREEN}✓ Enhanced new command logic present${NC}"
    else
        echo -e "${RED}✗ Enhanced new command logic missing${NC}"
        return 1
    fi
}

test_error_handling() {
    echo -e "${YELLOW}Testing error handling for unknown commands...${NC}"
    
    # Test completely unknown command
    echo -e "${BLUE}Test 1: Unknown command handling...${NC}"
    local output=$(./jmux nonexistentcommand 2>&1 || true)
    
    if echo "$output" | grep -q "Unknown command" && echo "$output" | grep -q "jmux commands" && echo "$output" | grep -q "tmux commands"; then
        echo -e "${GREEN}✓ Unknown command error handling works${NC}"
    else
        echo -e "${RED}✗ Unknown command error handling failed${NC}"
        echo "Output: $output"
        return 1
    fi
    
    # Test invalid tmux command
    echo -e "${BLUE}Test 2: Invalid tmux command handling...${NC}"
    local output=$(./jmux invalidtmuxcmd 2>&1 || true)
    
    if echo "$output" | grep -q "Unknown command" || echo "$output" | grep -q "unknown command"; then
        echo -e "${GREEN}✓ Invalid tmux command handled correctly${NC}"
    else
        echo -e "${RED}✗ Invalid tmux command not handled properly${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_command_detection() {
    echo -e "${YELLOW}Testing tmux command detection logic...${NC}"
    
    # Test that known tmux commands are in the list
    echo -e "${BLUE}Test 1: Known command detection...${NC}"
    if grep -q '"ls"' "./jmux" && grep -q '"list-sessions"' "./jmux" && grep -q '"attach"' "./jmux" && grep -q '"new"' "./jmux"; then
        echo -e "${GREEN}✓ Known tmux commands defined${NC}"
    else
        echo -e "${RED}✗ Known tmux commands missing${NC}"
        return 1
    fi
    
    # Test function exists
    echo -e "${BLUE}Test 2: Forward function exists...${NC}"
    if grep -q "forward_to_tmux" "./jmux"; then
        echo -e "${GREEN}✓ forward_to_tmux function exists${NC}"
    else
        echo -e "${RED}✗ forward_to_tmux function missing${NC}"
        return 1
    fi
}

test_help_documentation() {
    echo -e "${YELLOW}Testing help documentation updates...${NC}"
    
    echo -e "${BLUE}Test 1: Tmux integration section...${NC}"
    local output=$(./jmux help 2>&1)
    
    if echo "$output" | grep -q "TMUX INTEGRATION" && echo "$output" | grep -q "extends tmux"; then
        echo -e "${GREEN}✓ Help includes tmux integration section${NC}"
    else
        echo -e "${RED}✗ Help missing tmux integration section${NC}"
        return 1
    fi
    
    echo -e "${BLUE}Test 2: Tmux examples...${NC}"
    if echo "$output" | grep -q "jmux ls" && echo "$output" | grep -q "Enhanced session listing"; then
        echo -e "${GREEN}✓ Help includes tmux pass-through examples${NC}"
    else
        echo -e "${RED}✗ Help missing tmux examples${NC}"
        return 1
    fi
}

# Main test runner
main() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Tmux Pass-Through Test Suite${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    local tests_run=0
    local tests_passed=0
    
    # Run tests
    if test_basic_tmux_commands; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    echo
    
    if test_enhanced_tmux_commands; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    echo
    
    if test_error_handling; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    echo
    
    if test_command_detection; then
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
        echo -e "${GREEN}All tmux pass-through tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    fi
}

main "$@"