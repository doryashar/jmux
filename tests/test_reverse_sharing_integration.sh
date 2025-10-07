#!/bin/bash

# Integration test for dmux reverse sharing functionality
# This test validates the complete workflow using tmux-test-setup.sh environment:
# 1. Remote machine (machine1) asks for share (dmux ask-share yashar)
# 2. Local machine (machine2) joins and shares session (dmux join-share dory) 
# 3. Remote machine should get access to local user's session (whoami should return "yashar")

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
JMUX_BIN="/projects/common/work/dory/jmux/bin/dmux"
TEST_SESSION="reverse_test_session"
LOG_DIR="/tmp/tmux-logs"
REMOTE_LOG="$LOG_DIR/machine1.log"
LOCAL_LOG="$LOG_DIR/machine2.log"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}🧪 Starting dmux reverse sharing integration test...${NC}"
# ./scripts/tmux-test-setup.sh
# Ensure tmux test session is running
if ! tmux has-session -t tmux_testing 2>/dev/null; then
    echo -e "${RED}❌ tmux_testing session not found. Please run ./scripts/tmux-test-setup.sh first${NC}"
    exit 1
fi

echo -e "${GREEN}✅ tmux_testing session found${NC}"

# Clear previous logs
> "$REMOTE_LOG"
> "$LOCAL_LOG"

echo -e "${YELLOW}🔄 Step 1: Starting ask-share on remote machine (machine1)...${NC}"

# Send ask-share command to remote machine
tmux send-keys -t tmux_testing:1.0 "echo '=== TEST START: ask-share ===' && date" C-m
sleep 1
tmux send-keys -t tmux_testing:1.0 "$JMUX_BIN ask-share yashar" C-m

# Wait for ask-share to start
echo "Waiting 5 seconds for ask-share to initialize..."
sleep 5

echo -e "${YELLOW}🔄 Step 2: Starting join-share on local machine (machine2)...${NC}"

# Send join-share command to local machine  
tmux send-keys -t tmux_testing:1.1 "echo '=== TEST START: join-share ===' && date" C-m
sleep 1
tmux send-keys -t tmux_testing:1.1 "$JMUX_BIN join-share dory" C-m

# Wait for connection to establish
echo "Waiting 8 seconds for connection to establish..."
sleep 8

echo -e "${YELLOW}🔄 Step 3: Testing remote access to local session...${NC}"

# Test 1: Check whoami from remote machine - this should show "yashar" if working correctly
echo "Testing whoami command on remote machine..."
tmux send-keys -t tmux_testing:1.0 "echo '=== TESTING whoami ===' && whoami && echo '=== END whoami ==='" C-m
sleep 3

# Test 2: Check hostname from remote machine - this should show "YasharPC" if working correctly
echo "Testing hostname command on remote machine..."
tmux send-keys -t tmux_testing:1.0 "echo '=== TESTING hostname ===' && hostname && echo '=== END hostname ==='" C-m
sleep 3

# Test 3: Check current working directory - this should show local machine's path
echo "Testing pwd command on remote machine..."
tmux send-keys -t tmux_testing:1.0 "echo '=== TESTING pwd ===' && pwd && echo '=== END pwd ==='" C-m
sleep 3

echo -e "${YELLOW}🔄 Step 4: Analyzing test results...${NC}"

# Wait a bit more for all commands to complete
sleep 2

# Check remote log for test results
echo -e "${BLUE}📋 Remote machine log (last 30 lines):${NC}"
tail -30 "$REMOTE_LOG" | cat -n
echo ""

echo -e "${BLUE}📋 Local machine log (last 30 lines):${NC}"
tail -30 "$LOCAL_LOG" | cat -n
echo ""

echo -e "${BLUE}🔍 Analyzing results...${NC}"

# Check if remote machine shows local user "yashar"
# Look for the pattern after connection is established
if strings "$REMOTE_LOG" | grep -A 3 "=== TESTING whoami ===" | grep -q "yashar"; then
    echo -e "${GREEN}✅ SUCCESS: Remote machine shows local user 'yashar' in whoami test${NC}"
    WHOAMI_TEST="PASS"
else
    echo -e "${RED}❌ FAILURE: Remote machine does not show local user 'yashar' in whoami test${NC}"
    echo -e "${YELLOW}Expected: yashar${NC}"
    echo -e "${YELLOW}Actual:${NC}"
    strings "$REMOTE_LOG" | grep -A 3 "=== TESTING whoami ===" || echo "whoami test section not found"
    WHOAMI_TEST="FAIL"
fi

# Check if remote machine shows local hostname "YasharPC"
if strings "$REMOTE_LOG" | grep -A 3 "=== TESTING hostname ===" | grep -q "YasharPC"; then
    echo -e "${GREEN}✅ SUCCESS: Remote machine shows local hostname 'YasharPC' in hostname test${NC}"
    HOSTNAME_TEST="PASS"
else
    echo -e "${RED}❌ FAILURE: Remote machine does not show local hostname 'YasharPC' in hostname test${NC}"
    echo -e "${YELLOW}Expected: YasharPC${NC}"
    echo -e "${YELLOW}Actual:${NC}"
    strings "$REMOTE_LOG" | grep -A 3 "=== TESTING hostname ===" || echo "hostname test section not found"
    HOSTNAME_TEST="FAIL"
fi

# Check for jcat reverse connection establishment
if grep -q "Connected to reverse jcat server" "$LOCAL_LOG"; then
    echo -e "${GREEN}✅ SUCCESS: jcat reverse connection established${NC}"
    CONNECTION_TEST="PASS"
else
    echo -e "${RED}❌ FAILURE: jcat reverse connection not established${NC}"
    CONNECTION_TEST="FAIL"
fi

# Check if ask-share started correctly
if grep -q "jcat reverse-listen started" "$REMOTE_LOG"; then
    echo -e "${GREEN}✅ SUCCESS: ask-share started jcat reverse-listen server${NC}"
    ASK_SHARE_TEST="PASS"
else
    echo -e "${RED}❌ FAILURE: ask-share did not start jcat reverse-listen server${NC}"
    ASK_SHARE_TEST="FAIL"
fi

# Check if join-share connected properly
if grep -q "Sharing your session with" "$LOCAL_LOG"; then
    echo -e "${GREEN}✅ SUCCESS: join-share initiated session sharing${NC}"
    JOIN_SHARE_TEST="PASS"
else
    echo -e "${RED}❌ FAILURE: join-share did not initiate session sharing${NC}"
    JOIN_SHARE_TEST="FAIL"
fi

echo ""
echo -e "${BLUE}📊 TEST SUMMARY:${NC}"
echo -e "   ask-share server start: $ASK_SHARE_TEST"
echo -e "   join-share connection: $JOIN_SHARE_TEST"
echo -e "   jcat connection: $CONNECTION_TEST"
echo -e "   whoami test: $WHOAMI_TEST (CRITICAL - should show 'yashar')"  
echo -e "   hostname test: $HOSTNAME_TEST (CRITICAL - should show 'YasharPC')"

# Cleanup
echo ""
echo -e "${YELLOW}🧹 Cleaning up...${NC}"
tmux send-keys -t tmux_testing:1.0 C-c
tmux send-keys -t tmux_testing:1.1 C-c
sleep 2

# Stop any remaining processes
tmux send-keys -t tmux_testing:1.0 C-c
tmux send-keys -t tmux_testing:1.1 C-c
sleep 1

# Clear the sessions
tmux send-keys -t tmux_testing:1.0 "clear" C-m
tmux send-keys -t tmux_testing:1.1 "clear" C-m

if [[ "$WHOAMI_TEST" == "PASS" && "$HOSTNAME_TEST" == "PASS" && "$CONNECTION_TEST" == "PASS" && "$ASK_SHARE_TEST" == "PASS" && "$JOIN_SHARE_TEST" == "PASS" ]]; then
    echo -e "${GREEN}🎉 ALL TESTS PASSED!${NC}"
    echo "The reverse sharing functionality is working correctly."
    exit 0
else
    echo -e "${RED}💥 TESTS FAILED - Issues detected with reverse sharing implementation${NC}"
    echo ""
    echo -e "${YELLOW}Expected behavior:${NC}"
    echo "1. Remote machine runs 'ask-share yashar' and starts listening"
    echo "2. Local machine runs 'join-share dory' and shares its session"
    echo "3. Remote machine should get access to local machine's session"
    echo "4. Commands run on remote should execute in local context (yashar@YasharPC)"
    echo ""
    echo -e "${YELLOW}Current issues:${NC}"
    [[ "$ASK_SHARE_TEST" != "PASS" ]] && echo "- ask-share not starting jcat reverse-listen properly"
    [[ "$JOIN_SHARE_TEST" != "PASS" ]] && echo "- join-share not initiating session sharing properly"
    [[ "$CONNECTION_TEST" != "PASS" ]] && echo "- jcat reverse connection not establishing"
    [[ "$WHOAMI_TEST" != "PASS" ]] && echo "- Remote machine not getting local user context (whoami should return yashar)"
    [[ "$HOSTNAME_TEST" != "PASS" ]] && echo "- Remote machine not getting local hostname context (hostname should return YasharPC)"
    exit 1
fi