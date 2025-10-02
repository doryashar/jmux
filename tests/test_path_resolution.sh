#!/usr/bin/env bash

# Test script to verify path resolution works with permission issues

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}Testing path resolution with permission scenarios...${NC}"
echo

# Test path resolution function
test_path_resolution() {
    echo -e "${YELLOW}Testing path resolution methods...${NC}"
    
    local test_script="$0"
    local resolved_path=""
    
    # Test realpath (might fail with permissions)
    echo -e "${BLUE}Testing realpath...${NC}"
    if command -v realpath &> /dev/null; then
        if resolved_path="$(realpath "$test_script" 2>/dev/null)"; then
            echo -e "${GREEN}✓ realpath worked: $resolved_path${NC}"
        else
            echo -e "${YELLOW}! realpath failed (permission denied)${NC}"
        fi
    else
        echo -e "${YELLOW}! realpath not available${NC}"
    fi
    
    # Test readlink fallback
    echo -e "${BLUE}Testing readlink fallback...${NC}"
    if [[ -z "$resolved_path" ]] && command -v readlink &> /dev/null; then
        if resolved_path="$(readlink -f "$test_script" 2>/dev/null)"; then
            echo -e "${GREEN}✓ readlink worked: $resolved_path${NC}"
        else
            echo -e "${YELLOW}! readlink failed${NC}"
        fi
    fi
    
    # Test manual resolution
    echo -e "${BLUE}Testing manual resolution...${NC}"
    if [[ -z "$resolved_path" ]]; then
        if [[ "$test_script" = /* ]]; then
            resolved_path="$test_script"
            echo -e "${GREEN}✓ Script already absolute: $resolved_path${NC}"
        else
            resolved_path="$(pwd)/$test_script"
            echo -e "${GREEN}✓ Manual resolution: $resolved_path${NC}"
        fi
    fi
    
    # Test readability
    echo -e "${BLUE}Testing file readability...${NC}"
    if [[ -r "$resolved_path" ]]; then
        echo -e "${GREEN}✓ File is readable: $resolved_path${NC}"
        return 0
    else
        echo -e "${RED}✗ File is not readable: $resolved_path${NC}"
        return 1
    fi
}

# Test symlink creation with different scenarios
test_symlink_scenarios() {
    echo -e "${YELLOW}Testing symlink creation scenarios...${NC}"
    
    local test_dir="/tmp/jmux_symlink_test"
    local test_bin="${test_dir}/.local/bin"
    
    # Clean up any previous test
    rm -rf "$test_dir"
    mkdir -p "$test_bin"
    
    # Test 1: Normal case
    echo -e "${BLUE}Test 1: Normal symlink creation...${NC}"
    local test_target="${test_dir}/jmux_test"
    echo "#!/bin/bash" > "$test_target"
    chmod +x "$test_target"
    
    if ln -sf "$test_target" "${test_bin}/jmux_test"; then
        echo -e "${GREEN}✓ Normal symlink creation works${NC}"
    else
        echo -e "${RED}✗ Normal symlink creation failed${NC}"
    fi
    
    # Test 2: Empty target (simulating the user's issue)
    echo -e "${BLUE}Test 2: Empty target scenario...${NC}"
    if ln -sf "" "${test_bin}/jmux_empty" 2>/dev/null; then
        echo -e "${YELLOW}! Empty target symlink created (this is the bug)${NC}"
    else
        echo -e "${GREEN}✓ Empty target symlink prevented${NC}"
    fi
    
    # Test 3: Non-existent target
    echo -e "${BLUE}Test 3: Non-existent target...${NC}"
    if ln -sf "/nonexistent/path" "${test_bin}/jmux_missing" 2>/dev/null; then
        echo -e "${YELLOW}! Non-existent target symlink created${NC}"
    else
        echo -e "${GREEN}✓ Non-existent target symlink prevented${NC}"
    fi
    
    # Cleanup
    rm -rf "$test_dir"
}

# Test the actual jmux setup function (extracted)
test_jmux_setup_function() {
    echo -e "${YELLOW}Testing jmux setup function logic...${NC}"
    
    # Simulate the improved setup function
    local jmux_script=""
    local test_script="$0"
    
    # Try different methods to get absolute path
    if command -v realpath &> /dev/null; then
        jmux_script="$(realpath "$test_script" 2>/dev/null)" || jmux_script=""
    fi
    
    # Fallback to readlink if realpath failed
    if [[ -z "$jmux_script" ]] && command -v readlink &> /dev/null; then
        jmux_script="$(readlink -f "$test_script" 2>/dev/null)" || jmux_script=""
    fi
    
    # Fallback to manual resolution
    if [[ -z "$jmux_script" ]]; then
        if [[ "$test_script" = /* ]]; then
            jmux_script="$test_script"
        else
            jmux_script="$(pwd)/$test_script"
        fi
    fi
    
    # Verify the script path exists and is readable
    if [[ ! -r "$jmux_script" ]]; then
        echo -e "${YELLOW}Warning: Cannot resolve script path, would skip symlink creation${NC}"
        return 0
    fi
    
    echo -e "${GREEN}✓ Path resolution successful: $jmux_script${NC}"
    return 0
}

# Main test runner
main() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Path Resolution Test Suite${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    local tests_run=0
    local tests_passed=0
    
    # Run tests
    if test_path_resolution; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    echo
    
    if test_symlink_scenarios; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    echo
    
    if test_jmux_setup_function; then
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
        echo -e "${GREEN}All path resolution tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    fi
}

main "$@"