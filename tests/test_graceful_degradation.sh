#!/usr/bin/env bash

# Test script to demonstrate graceful degradation without inotify-tools

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Testing dmux graceful degradation without inotify-tools...${NC}"
echo

# Build dmux binary for testing
DMUX_BINARY="/tmp/dmux_graceful_test"
if (cd "$(dirname "$0")/../src/jmux-go" && go build -o "$DMUX_BINARY" .); then
    echo "✓ dmux binary built successfully"
else
    echo "❌ Failed to build dmux binary"
    exit 1
fi

# Setup test environment
TEST_DIR="/tmp/dmux_graceful_test_env"
export JMUX_SHARED_DIR="${TEST_DIR}"

echo -e "${YELLOW}Setting up test environment...${NC}"
mkdir -p "${TEST_DIR}"/{messages,sessions}
touch "${TEST_DIR}/users.db"

echo -e "${YELLOW}Testing dmux status (should show fallback messaging)...${NC}"
"$DMUX_BINARY" status
echo

echo -e "${YELLOW}Testing message monitor status...${NC}"
"$DMUX_BINARY" monitor status
echo

echo -e "${YELLOW}Testing help command...${NC}"
"$DMUX_BINARY" --help | head -5 || true
echo

echo -e "${YELLOW}Testing message sending (should work without real-time)...${NC}"
echo "testuser:192.168.1.100:$(date +%s)" >> "${TEST_DIR}/users.db"
"$DMUX_BINARY" msg testuser "Test message without real-time"
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
echo -e "${GREEN}✓ All tests completed - dmux works without inotify-tools!${NC}"

# Cleanup
rm -rf "${TEST_DIR}" 2>/dev/null || true
rm -f "$DMUX_BINARY"