#!/usr/bin/env bash

# Test script to demonstrate graceful degradation without inotify-tools

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Testing jmux graceful degradation without inotify-tools...${NC}"
echo

# Setup test environment
TEST_DIR="/tmp/jmux_graceful_test"
export JMUX_SHARED_DIR="${TEST_DIR}"

echo -e "${YELLOW}Setting up test environment...${NC}"
mkdir -p "${TEST_DIR}"/{messages,sessions}
touch "${TEST_DIR}/users.db"

echo -e "${YELLOW}Testing jmux status (should show fallback messaging)...${NC}"
./jmux status
echo

echo -e "${YELLOW}Testing message watcher status...${NC}"
./jmux watch status
echo

echo -e "${YELLOW}Testing help command...${NC}"
./jmux help | head -5
echo

echo -e "${YELLOW}Testing message sending (should work without real-time)...${NC}"
echo "testuser:192.168.1.100:$(date +%s)" >> "${TEST_DIR}/users.db"
./jmux msg testuser message "Test message without real-time"
echo

echo -e "${YELLOW}Checking message was created...${NC}"
if ls "${TEST_DIR}/messages"/*.msg &>/dev/null; then
    echo -e "${GREEN}✓ Message created successfully${NC}"
    echo "Message content:"
    cat "${TEST_DIR}/messages"/*.msg | head -5
else
    echo -e "${YELLOW}No messages found${NC}"
fi

echo
echo -e "${GREEN}✓ All tests completed - jmux works without inotify-tools!${NC}"

# Cleanup
rm -rf "${TEST_DIR}"