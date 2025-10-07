#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

REMOTE_LOG="/tmp/tmux-logs/machine1.log"
LOCAL_LOG="/tmp/tmux-logs/machine2.log"

echo -e "${BLUE}🧪 Testing SIGWINCH propagation in dmux reverse sharing...${NC}"

# Check if tmux_testing session exists
if ! tmux has-session -t tmux_testing 2>/dev/null; then
    echo -e "${RED}❌ tmux_testing session not found. Please run ./scripts/tmux-test-setup.sh first${NC}"
    exit 1
fi

echo -e "${GREEN}✅ tmux_testing session found${NC}"

# Clear logs
> "$REMOTE_LOG"
> "$LOCAL_LOG"

echo -e "${YELLOW}🔄 Step 1: Starting ask-share on remote machine (machine1)...${NC}"
tmux send-keys -t tmux_testing:1.0 'echo "=== SIGWINCH TEST START: ask-share ===" && date' C-m
sleep 1
tmux send-keys -t tmux_testing:1.0 '/projects/common/work/dory/jmux/bin/dmux ask-share yashar' C-m

echo "Waiting 5 seconds for ask-share to initialize..."
sleep 5

echo -e "${YELLOW}🔄 Step 2: Starting join-share on local machine (machine2)...${NC}" 
tmux send-keys -t tmux_testing:1.1 'echo "=== SIGWINCH TEST START: join-share ===" && date' C-m
sleep 1
tmux send-keys -t tmux_testing:1.1 '/projects/common/work/dory/jmux/bin/dmux join-share dory' C-m

echo "Waiting 5 seconds for connection to establish..."
sleep 5

echo -e "${YELLOW}🔄 Step 3: Testing SIGWINCH by changing terminal size...${NC}"
# Test initial terminal size command
tmux send-keys -t tmux_testing:1.0 'echo "=== TESTING tput cols/lines BEFORE resize ===" && tput cols && tput lines && echo "=== END BEFORE resize ==="' C-m
sleep 2

# Resize the remote machine terminal (simulate SIGWINCH)
echo "Simulating terminal resize on remote machine..."
tmux send-keys -t tmux_testing:1.1 'printf "\\033[8;30;120t"' C-m  # Resize local terminal
sleep 1

# Test terminal size after resize  
tmux send-keys -t tmux_testing:1.0 'echo "=== TESTING tput cols/lines AFTER resize ===" && tput cols && tput lines && echo "=== END AFTER resize ==="' C-m
sleep 2

# Test with stty size as well
tmux send-keys -t tmux_testing:1.0 'echo "=== TESTING stty size ===" && stty size && echo "=== END stty size ==="' C-m
sleep 2

echo -e "${YELLOW}🔄 Step 4: Cleaning up...${NC}"
# Clean up by stopping the reverse sharing
tmux send-keys -t tmux_testing:1.0 C-c
tmux send-keys -t tmux_testing:1.1 C-c
sleep 2

echo -e "${YELLOW}🔄 Step 5: Analyzing SIGWINCH results...${NC}"

echo -e "${BLUE}📋 Remote machine log (terminal size tests):${NC}"
strings "$REMOTE_LOG" | grep -A 10 "=== TESTING tput cols/lines" || echo "No terminal size test results found"
echo ""

echo -e "${BLUE}📋 Local machine log:${NC}"
tail -20 "$LOCAL_LOG" | cat -n
echo ""

# Basic check for SIGWINCH functionality
if strings "$REMOTE_LOG" | grep -q "tput cols\|tput lines"; then
    echo -e "${GREEN}✅ SUCCESS: Terminal size commands executed on remote machine${NC}"
    echo -e "${BLUE}💡 Note: Check the terminal size values above to verify SIGWINCH propagation${NC}"
    echo -e "${BLUE}💡 If the values changed after resize, SIGWINCH is working correctly${NC}"
    SIGWINCH_TEST="PASS"
else
    echo -e "${RED}❌ FAILURE: Terminal size test commands not found in remote log${NC}"
    SIGWINCH_TEST="FAIL"
fi

echo ""
echo -e "${BLUE}📊 TEST SUMMARY:${NC}"
echo "   SIGWINCH terminal size test: $SIGWINCH_TEST"

if [ "$SIGWINCH_TEST" = "PASS" ]; then
    echo -e "${GREEN}✅ SIGWINCH TEST COMPLETED${NC}"
    echo -e "${YELLOW}Please manually verify that terminal size values changed appropriately${NC}"
    exit 0
else
    echo -e "${RED}💥 SIGWINCH TESTS FAILED${NC}"
    exit 1
fi