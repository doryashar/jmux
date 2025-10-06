#!/bin/bash

set -e

echo "🧪 Testing jcat auto-detection functionality..."

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

# Build jcat binary for testing
JCAT_BINARY="/tmp/jcat_autodetect_test"
if (cd "$(dirname "$0")/../src/jcat" && go build -o "$JCAT_BINARY" .); then
    echo -e "${GREEN}✓ jcat binary built successfully${NC}"
else
    echo -e "${RED}❌ Failed to build jcat binary${NC}"
    exit 1
fi

test_help_command() {
    echo -e "${YELLOW}Testing help command...${NC}"
    
    local output=$("$JCAT_BINARY" help 2>&1)
    
    if echo "$output" | grep -q "jcat - TCP tunnel for terminal sharing" && \
       echo "$output" | grep -q "jcat listen" && \
       echo "$output" | grep -q "jcat connect"; then
        echo -e "${GREEN}✓ Help command shows correct usage${NC}"
        return 0
    else
        echo -e "${RED}✗ Help command missing expected content${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_version_command() {
    echo -e "${YELLOW}Testing version command...${NC}"
    
    local output=$("$JCAT_BINARY" version 2>&1)
    
    if echo "$output" | grep -q "jcat version"; then
        echo -e "${GREEN}✓ Version command works${NC}"
        return 0
    else
        echo -e "${RED}✗ Version command failed${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_listen_command() {
    echo -e "${YELLOW}Testing listen command...${NC}"
    
    # Test listen command with timeout
    local output=$(timeout 2s "$JCAT_BINARY" listen :9998 2>&1 || true)
    
    if echo "$output" | grep -q "listening on :9998"; then
        echo -e "${GREEN}✓ Listen command works correctly${NC}"
        return 0
    else
        echo -e "${RED}✗ Listen command failed${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_connect_command_validation() {
    echo -e "${YELLOW}Testing connect command validation...${NC}"
    
    # Test connect without address (should fail)
    local output=$("$JCAT_BINARY" connect 2>&1 || true)
    
    if echo "$output" | grep -q "connect command requires host:port argument"; then
        echo -e "${GREEN}✓ Connect command validation works${NC}"
        return 0
    else
        echo -e "${RED}✗ Connect command validation failed${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_invalid_command() {
    echo -e "${YELLOW}Testing invalid command handling...${NC}"
    
    local output=$("$JCAT_BINARY" invalidcommand 2>&1 || true)
    
    if echo "$output" | grep -q "Unknown command: invalidcommand"; then
        echo -e "${GREEN}✓ Invalid command handling works${NC}"
        return 0
    else
        echo -e "${RED}✗ Invalid command handling failed${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_no_arguments() {
    echo -e "${YELLOW}Testing no arguments handling...${NC}"
    
    local output=$("$JCAT_BINARY" 2>&1 || true)
    
    if echo "$output" | grep -q "Usage:" && echo "$output" | grep -q "jcat listen"; then
        echo -e "${GREEN}✓ No arguments handling works${NC}"
        return 0
    else
        echo -e "${RED}✗ No arguments handling failed${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_listen_with_custom_port() {
    echo -e "${YELLOW}Testing listen with custom port...${NC}"
    
    # Test listen command with custom port
    local output=$(timeout 2s "$JCAT_BINARY" listen :7777 2>&1 || true)
    
    if echo "$output" | grep -q "listening on :7777"; then
        echo -e "${GREEN}✓ Listen with custom port works${NC}"
        return 0
    else
        echo -e "${RED}✗ Listen with custom port failed${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_listen_default_port() {
    echo -e "${YELLOW}Testing listen with default port...${NC}"
    
    # Test listen command without port (should use default :1337)
    local output=$(timeout 2s "$JCAT_BINARY" listen 2>&1 || true)
    
    if echo "$output" | grep -q "listening on :1337"; then
        echo -e "${GREEN}✓ Listen with default port works${NC}"
        return 0
    else
        echo -e "${RED}✗ Listen with default port failed${NC}"
        echo "Output: $output"
        return 1
    fi
}

# Main test execution
main() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}JCAT Auto-Detection Test Suite${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    local tests_run=0
    local tests_passed=0
    
    # Run all tests
    for test_func in test_help_command test_version_command test_listen_command \
                     test_connect_command_validation test_invalid_command \
                     test_no_arguments test_listen_with_custom_port test_listen_default_port; do
        if $test_func; then
            tests_passed=$((tests_passed + 1))
        fi
        tests_run=$((tests_run + 1))
        echo
    done
    
    # Cleanup
    rm -f "$JCAT_BINARY"
    
    # Report results
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Test Results${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "  Tests run: ${tests_run}"
    echo -e "  Passed: ${GREEN}${tests_passed}${NC}"
    echo -e "  Failed: ${RED}$((tests_run - tests_passed))${NC}"
    
    if [[ $tests_passed -eq $tests_run ]]; then
        echo -e "${GREEN}✅ All jcat auto-detection tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}❌ Some tests failed!${NC}"
        exit 1
    fi
}

main "$@"