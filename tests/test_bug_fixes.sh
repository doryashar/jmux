#!/usr/bin/env bash

# Test script for bug fixes: tmux PATH detection and saved_settings variable

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}Testing jmux bug fixes...${NC}"
echo

# Test environment
export JMUX_SHARED_DIR="/tmp/jmux_test_fixes"
mkdir -p "$JMUX_SHARED_DIR"

test_setsize_script_generation() {
    echo -e "${YELLOW}Testing setsize script generation with tmux path detection...${NC}"
    
    local test_config_dir="/tmp/jmux_test_fixes/.config/jmux"
    mkdir -p "$test_config_dir"
    
    # Test with custom config dir
    local original_home="$HOME"
    export HOME="/tmp/jmux_test_fixes"
    
    # Run jmux to trigger setsize script creation
    ./jmux status > /dev/null 2>&1
    
    # Check if setsize script was created with improved tmux detection
    local setsize_script="$test_config_dir/setsize.sh"
    
    if [[ -f "$setsize_script" ]]; then
        echo -e "${GREEN}✓ Setsize script created${NC}"
        
        if grep -q "Try multiple common tmux locations" "$setsize_script"; then
            echo -e "${GREEN}✓ Script has improved tmux detection${NC}"
        else
            echo -e "${RED}✗ Script missing improved tmux detection${NC}"
            return 1
        fi
        
        if grep -q "/bin/tmux" "$setsize_script" && grep -q "/usr/bin/tmux" "$setsize_script"; then
            echo -e "${GREEN}✓ Script checks multiple tmux paths${NC}"
        else
            echo -e "${RED}✗ Script missing multiple path checks${NC}"
            return 1
        fi
        
        if grep -q "exec.*tmux" "$setsize_script"; then
            echo -e "${GREEN}✓ Script uses exec for proper process handling${NC}"
        else
            echo -e "${RED}✗ Script missing exec command${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ Setsize script not created${NC}"
        return 1
    fi
    
    # Restore HOME
    export HOME="$original_home"
}

test_saved_settings_variable() {
    echo -e "${YELLOW}Testing saved_settings variable fix...${NC}"
    
    # Check that the variable is declared without 'local' and uses safe expansion
    if grep -q "saved_settings=.*stty.*-g" "./jmux" && grep -q '\${saved_settings:-}' "./jmux"; then
        echo -e "${GREEN}✓ saved_settings variable uses safe expansion${NC}"
    else
        echo -e "${RED}✗ saved_settings variable not properly fixed${NC}"
        return 1
    fi
    
    # Check that it's not declared as local in join_session context
    if ! grep -A 20 -B 5 "saved_settings.*stty" "./jmux" | grep -q "local saved_settings"; then
        echo -e "${GREEN}✓ saved_settings not declared as local in function scope${NC}"
    else
        echo -e "${RED}✗ saved_settings still declared as local${NC}"
        return 1
    fi
}

test_tmux_detection_robustness() {
    echo -e "${YELLOW}Testing tmux detection robustness...${NC}"
    
    # Check that setsize script has proper error handling
    local test_config_dir="/tmp/jmux_test_fixes/.config/jmux"
    local setsize_script="$test_config_dir/setsize.sh"
    
    if [[ -f "$setsize_script" ]]; then
        if grep -q "Error: tmux not found" "$setsize_script" && grep -q "Available paths:" "$setsize_script"; then
            echo -e "${GREEN}✓ Setsize script has proper error messages${NC}"
        else
            echo -e "${RED}✗ Setsize script missing error handling${NC}"
            return 1
        fi
        
        # Test script syntax
        if bash -n "$setsize_script"; then
            echo -e "${GREEN}✓ Setsize script has valid syntax${NC}"
        else
            echo -e "${RED}✗ Setsize script has syntax errors${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ Cannot test - setsize script not found${NC}"
        return 1
    fi
}

test_version_update_mechanism() {
    echo -e "${YELLOW}Testing setsize script version update mechanism...${NC}"
    
    local test_config_dir="/tmp/jmux_test_fixes/.config/jmux"
    local setsize_script="$test_config_dir/setsize.sh"
    
    # Create an old version of the script
    cat > "$setsize_script" << 'EOF'
stty rows 50 cols 254
tmux new -A -s $HOSTNAME
exit
EOF
    
    # Run jmux to trigger update
    local original_home="$HOME"
    export HOME="/tmp/jmux_test_fixes"
    ./jmux status > /dev/null 2>&1
    export HOME="$original_home"
    
    # Check if script was updated
    if grep -q "Try multiple common tmux locations" "$setsize_script"; then
        echo -e "${GREEN}✓ Old setsize script was automatically updated${NC}"
    else
        echo -e "${RED}✗ Old setsize script was not updated${NC}"
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
    if test_setsize_script_generation; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    echo
    
    if test_saved_settings_variable; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    echo
    
    if test_tmux_detection_robustness; then
        tests_passed=$((tests_passed + 1))
    fi
    tests_run=$((tests_run + 1))
    echo
    
    if test_version_update_mechanism; then
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
        echo -e "${GREEN}All bug fix tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    fi
}

main "$@"