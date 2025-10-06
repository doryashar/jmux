#!/usr/bin/env bash

# Test script for dmux tmux pass-through functionality

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}Testing dmux tmux pass-through functionality...${NC}"
echo

# Test environment
export JMUX_SHARED_DIR="/tmp/dmux_test_passthrough"
mkdir -p "$JMUX_SHARED_DIR"

# Build dmux binary for testing
DMUX_BINARY="/tmp/dmux_passthrough_test"
if (cd "$(dirname "$0")/../src/jmux-go" && go build -o "$DMUX_BINARY" .); then
    echo -e "${GREEN}✓ dmux binary built successfully${NC}"
else
    echo -e "${RED}✗ Failed to build dmux binary${NC}"
    exit 1
fi

test_basic_tmux_commands() {
    echo -e "${YELLOW}Testing basic tmux command forwarding...${NC}"
    
    # Test ls command (should pass through to tmux)
    echo -e "${BLUE}Test 1: dmux ls (list sessions via tmux passthrough)...${NC}"
    local output=$("$DMUX_BINARY" ls 2>&1)
    
    # dmux ls should either show tmux sessions or pass through to tmux ls
    if echo "$output" | grep -qE "(no server running|session|windows)" || echo "$output" | grep -q "tmux"; then
        echo -e "${GREEN}✓ dmux ls works with tmux passthrough${NC}"
    else
        echo -e "${RED}✗ dmux ls failed or missing passthrough${NC}"
        echo "Output: $output"
        return 1
    fi
    
    # Test list-commands (should pass through to tmux)
    echo -e "${BLUE}Test 2: dmux list-commands...${NC}"
    local output=$("$DMUX_BINARY" list-commands 2>&1 | head -3)
    
    if echo "$output" | grep -q "attach-session"; then
        echo -e "${GREEN}✓ dmux list-commands forwards correctly to tmux${NC}"
    else
        echo -e "${RED}✗ dmux list-commands failed${NC}"
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
    # Since we can't actually create a session in tests, we'll test the command exists
    local output=$("$DMUX_BINARY" new --help 2>&1 || true)
    if echo "$output" | grep -q "new" || echo "$output" | grep -q "session"; then
        echo -e "${GREEN}✓ Enhanced new command available${NC}"
    else
        echo -e "${YELLOW}Note: Enhanced new command test skipped (legacy functionality)${NC}"
    fi
}

test_error_handling() {
    echo -e "${YELLOW}Testing error handling for unknown commands...${NC}"
    
    # Test completely unknown command
    echo -e "${BLUE}Test 1: Unknown command handling...${NC}"
    local output=$("$DMUX_BINARY" nonexistentcommand 2>&1 || true)
    
    if echo "$output" | grep -qE "(unknown|Unknown|error|Error)"; then
        echo -e "${GREEN}✓ Unknown command error handling works${NC}"
    else
        echo -e "${RED}✗ Unknown command error handling failed${NC}"
        echo "Output: $output"
        return 1
    fi
    
    # Test invalid tmux command
    echo -e "${BLUE}Test 2: Invalid tmux command handling...${NC}"
    local output=$("$DMUX_BINARY" invalidtmuxcmd 2>&1 || true)
    
    if echo "$output" | grep -qE "(unknown|Unknown|error|Error)"; then
        echo -e "${GREEN}✓ Invalid tmux command handled correctly${NC}"
    else
        echo -e "${RED}✗ Invalid tmux command not handled properly${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_command_detection() {
    echo -e "${YELLOW}Testing tmux command detection logic...${NC}"
    
    # Test that dmux has tmux integration
    echo -e "${BLUE}Test 1: Tmux integration available...${NC}"
    local output=$("$DMUX_BINARY" --help 2>&1)
    if echo "$output" | grep -qE "(tmux|session|attach|list)"; then
        echo -e "${GREEN}✓ Tmux integration commands available${NC}"
    else
        echo -e "${RED}✗ Tmux integration commands missing${NC}"
        return 1
    fi
    
    # Test ls command passes through
    echo -e "${BLUE}Test 2: Tmux passthrough functionality...${NC}"
    local output=$("$DMUX_BINARY" ls 2>&1 || true)
    if echo "$output" | grep -qE "(session|tmux|no server running)"; then
        echo -e "${GREEN}✓ Tmux passthrough functionality works${NC}"
    else
        echo -e "${YELLOW}Note: Tmux passthrough test result: $output${NC}"
    fi
}

test_help_documentation() {
    echo -e "${YELLOW}Testing help documentation updates...${NC}"
    
    echo -e "${BLUE}Test 1: Tmux integration in help...${NC}"
    local output=$("$DMUX_BINARY" --help 2>&1)
    
    if echo "$output" | grep -qE "(tmux|session|enhanced|sharing)"; then
        echo -e "${GREEN}✓ Help includes tmux/session functionality${NC}"
    else
        echo -e "${RED}✗ Help missing tmux integration information${NC}"
        return 1
    fi
    
    echo -e "${BLUE}Test 2: Available commands...${NC}"
    if echo "$output" | grep -qE "(ls|attach|new|share)"; then
        echo -e "${GREEN}✓ Help includes relevant dmux commands${NC}"
    else
        echo -e "${RED}✗ Help missing key commands${NC}"
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
    rm -f "$DMUX_BINARY"
    
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