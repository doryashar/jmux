#!/usr/bin/env bash

# dmux Regression Test Runner
# Runs all tests to validate dmux functionality

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DMUX_DIR="$(dirname "$SCRIPT_DIR")/src/jmux-go"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}dmux Regression Test Suite${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "Test directory: ${SCRIPT_DIR}"
echo -e "dmux directory: ${DMUX_DIR}"
echo

# Function to run a test and track results
run_test() {
    local test_name="$1"
    local test_command="$2"
    local test_type="${3:-bash}"
    
    echo -e "${YELLOW}Running: ${test_name}${NC}"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    local start_time=$(date +%s)
    local exit_code=0
    
    if [[ "$test_type" == "go" ]]; then
        # Run Go test
        if [[ "$test_command" == cd* ]]; then
            # Handle cd commands specially
            eval "$test_command" || exit_code=$?
        else
            (cd "$SCRIPT_DIR" && $test_command) || exit_code=$?
        fi
    else
        # Run bash test
        (cd "$SCRIPT_DIR" && bash $test_command) || exit_code=$?
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [[ $exit_code -eq 0 ]]; then
        echo -e "${GREEN}✓ PASS${NC} - ${test_name} (${duration}s)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ FAIL${NC} - ${test_name} (${duration}s)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    echo
}

# Function to skip a test
skip_test() {
    local test_name="$1"
    local reason="$2"
    
    echo -e "${YELLOW}SKIP${NC} - ${test_name} (${reason})"
    SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo
}

# Check dependencies
check_dependencies() {
    echo -e "${CYAN}Checking dependencies...${NC}"
    
    # Check for Go
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go is required but not installed${NC}"
        exit 1
    fi
    
    # Check for tmux
    if ! command -v tmux &> /dev/null; then
        echo -e "${RED}Error: tmux is required but not installed${NC}"
        exit 1
    fi
    
    # Check if dmux source exists
    if [[ ! -d "$DMUX_DIR" ]]; then
        echo -e "${RED}Error: dmux source directory not found at $DMUX_DIR${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✓ All dependencies available${NC}"
    echo
}

# Build dmux binary for testing
build_dmux() {
    echo -e "${CYAN}Building dmux binary...${NC}"
    
    if (cd "$DMUX_DIR" && go build -o "$SCRIPT_DIR/dmux_test" .); then
        echo -e "${GREEN}✓ dmux binary built successfully${NC}"
    else
        echo -e "${RED}✗ Failed to build dmux binary${NC}"
        exit 1
    fi
    echo
}

# Cleanup function
cleanup() {
    echo -e "${CYAN}Cleaning up test artifacts...${NC}"
    
    # Remove test binaries
    rm -f "$SCRIPT_DIR/dmux_test"
    rm -f /tmp/dmux_*test*
    
    # Clean up any test directories
    rm -rf /tmp/dmux_test_*
    rm -rf /tmp/jmux_test_*
    
    echo -e "${GREEN}✓ Cleanup complete${NC}"
    echo
}

# Set up signal handlers for cleanup
trap cleanup EXIT
trap 'echo -e "\n${YELLOW}Tests interrupted${NC}"; exit 130' INT TERM

# Main test execution
main() {
    echo -e "${CYAN}Starting regression test suite...${NC}"
    echo
    
    check_dependencies
    build_dmux

    # 1. Go Unit Tests
    echo -e "${BLUE}━━━ Go Unit Tests ━━━${NC}"
    
    # Security tests (existing)
    if [[ -f "$DMUX_DIR/internal/security/security_test.go" ]]; then
        run_test "Security Module Tests" "cd $DMUX_DIR && go test -v ./internal/security" go
    else
        skip_test "Security Module Tests" "security_test.go not found"
    fi
    
    # New Go tests (using proper test framework)
    if [[ -f "$SCRIPT_DIR/session_lifecycle_test.go" ]]; then
        run_test "Session Lifecycle Tests" "go test -v session_lifecycle_test.go" go
    else
        skip_test "Session Lifecycle Tests" "session_lifecycle_test.go not found"
    fi
    
    if [[ -f "$SCRIPT_DIR/port_management_test.go" ]]; then
        run_test "Port Management Tests" "go test -v port_management_test.go" go
    else
        skip_test "Port Management Tests" "port_management_test.go not found"
    fi
    
    if [[ -f "$SCRIPT_DIR/share_join_test.go" ]]; then
        run_test "Share/Join Tests" "go test -v share_join_test.go" go
    else
        skip_test "Share/Join Tests" "share_join_test.go not found"
    fi

    # 2. Integration Tests (Bash)
    echo -e "${BLUE}━━━ Integration Tests ━━━${NC}"
    
    # Core functionality tests
    if [[ -f "$SCRIPT_DIR/test_jmux.sh" ]]; then
        run_test "Core jmux Functionality" "test_jmux.sh"
    else
        skip_test "Core jmux Functionality" "test_jmux.sh not found"
    fi
    
    # Bug fix regression tests
    if [[ -f "$SCRIPT_DIR/test_bug_fixes.sh" ]]; then
        run_test "Bug Fix Regression Tests" "test_bug_fixes.sh"
    else
        skip_test "Bug Fix Regression Tests" "test_bug_fixes.sh not found"
    fi
    
    # Messaging tests
    if [[ -f "$SCRIPT_DIR/test_realtime_messaging.sh" ]]; then
        run_test "Real-time Messaging" "test_realtime_messaging.sh"
    else
        skip_test "Real-time Messaging" "test_realtime_messaging.sh not found"
    fi
    
    # Session management tests
    if [[ -f "$SCRIPT_DIR/test_session_cleanup.sh" ]]; then
        run_test "Session Cleanup" "test_session_cleanup.sh"
    else
        skip_test "Session Cleanup" "test_session_cleanup.sh not found"
    fi
    
    # Port mapping tests
    if [[ -f "$SCRIPT_DIR/test_port_mapping.sh" ]]; then
        run_test "Port Mapping" "test_port_mapping.sh"
    else
        skip_test "Port Mapping" "test_port_mapping.sh not found"
    fi
    
    # Tmux integration tests
    if [[ -f "$SCRIPT_DIR/test_tmux_passthrough.sh" ]]; then
        run_test "Tmux Passthrough" "test_tmux_passthrough.sh"
    else
        skip_test "Tmux Passthrough" "test_tmux_passthrough.sh not found"
    fi
    
    # Menu system tests
    if [[ -f "$SCRIPT_DIR/test_menu.sh" ]]; then
        run_test "Menu System" "test_menu.sh"
    else
        skip_test "Menu System" "test_menu.sh not found"
    fi

    # 3. System Integration Tests
    echo -e "${BLUE}━━━ System Integration Tests ━━━${NC}"
    
    # Monitor tests
    if [[ -f "$SCRIPT_DIR/test_monitor_logging.sh" ]]; then
        run_test "Monitor Logging" "test_monitor_logging.sh"
    else
        skip_test "Monitor Logging" "test_monitor_logging.sh not found"
    fi
    
    # Complete messaging tests
    if [[ -f "$SCRIPT_DIR/test_complete_messaging.sh" ]]; then
        run_test "Complete Messaging System" "test_complete_messaging.sh"
    else
        skip_test "Complete Messaging System" "test_complete_messaging.sh not found"
    fi
    
    # Security tests
    if [[ -f "$SCRIPT_DIR/test_security.go" ]]; then
        run_test "Security Integration" "go test -v test_security.go" go
    else
        skip_test "Security Integration" "test_security.go not found"
    fi

    # 4. New Feature Tests  
    echo -e "${BLUE}━━━ New Feature Tests ━━━${NC}"
    
    # JCAT Auto-Detection Tests
    if [[ -f "$SCRIPT_DIR/test_jcat_autodetect.sh" ]]; then
        run_test "JCAT Auto-Detection" "test_jcat_autodetect.sh" bash
    else
        skip_test "JCAT Auto-Detection" "test_jcat_autodetect.sh not found"
    fi
    
    # Reverse Sharing Tests
    if [[ -f "$SCRIPT_DIR/test_reverse_sharing.sh" ]]; then
        run_test "Reverse Sharing" "test_reverse_sharing.sh" bash
    else
        skip_test "Reverse Sharing" "test_reverse_sharing.sh not found"
    fi

    # 5. Edge Cases and Stress Tests
    echo -e "${BLUE}━━━ Edge Cases and Stress Tests ━━━${NC}"
    
    # Sharing modes tests
    if [[ -f "$SCRIPT_DIR/test-sharing-modes.sh" ]]; then
        run_test "Sharing Modes" "test-sharing-modes.sh"
    else
        skip_test "Sharing Modes" "test-sharing-modes.sh not found"
    fi
    
    # Tmux hang issue tests
    if [[ -f "$SCRIPT_DIR/test_tmux_hang_issue.sh" ]]; then
        run_test "Tmux Hang Issue" "test_tmux_hang_issue.sh"
    else
        skip_test "Tmux Hang Issue" "test_tmux_hang_issue.sh not found"
    fi
    
    # Graceful degradation tests
    if [[ -f "$SCRIPT_DIR/test_graceful_degradation.sh" ]]; then
        run_test "Graceful Degradation" "test_graceful_degradation.sh"
    else
        skip_test "Graceful Degradation" "test_graceful_degradation.sh not found"
    fi

    # Report results
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}Test Results Summary${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "Total tests:  ${TOTAL_TESTS}"
    echo -e "Passed:       ${GREEN}${PASSED_TESTS}${NC}"
    echo -e "Failed:       ${RED}${FAILED_TESTS}${NC}"
    echo -e "Skipped:      ${YELLOW}${SKIPPED_TESTS}${NC}"
    echo
    
    if [[ $FAILED_TESTS -eq 0 ]]; then
        echo -e "${GREEN}🎉 All tests passed!${NC}"
        
        if [[ $SKIPPED_TESTS -gt 0 ]]; then
            echo -e "${YELLOW}Note: ${SKIPPED_TESTS} tests were skipped${NC}"
        fi
        
        exit 0
    else
        echo -e "${RED}❌ ${FAILED_TESTS} test(s) failed!${NC}"
        exit 1
    fi
}

# Run the test suite
main "$@"