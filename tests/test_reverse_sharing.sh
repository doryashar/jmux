#!/bin/bash

set -e

echo "🧪 Testing dmux reverse sharing functionality..."

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

# Set up test environment
TEST_HOME="/tmp/test_reverse_sharing_$(date +%s)"
mkdir -p "$TEST_HOME"
export HOME="$TEST_HOME"
export JMUX_SHARED_DIR="$TEST_HOME/shared"

echo "Test HOME: $TEST_HOME"

# Build dmux binary for testing
DMUX_BINARY="/tmp/dmux_reverse_test"
if (cd "$(dirname "$0")/../src/jmux-go" && go build -o "$DMUX_BINARY" .); then
    echo -e "${GREEN}✓ dmux binary built successfully${NC}"
else
    echo -e "${RED}❌ Failed to build dmux binary${NC}"
    exit 1
fi

# Initialize directories
mkdir -p "$JMUX_SHARED_DIR/messages"
mkdir -p "$JMUX_SHARED_DIR/sessions"

test_ask_share_help() {
    echo -e "${YELLOW}Testing ask-share help command...${NC}"
    
    local output=$("$DMUX_BINARY" ask-share --help 2>&1)
    
    if echo "$output" | grep -q "Ask other users to share their sessions" && \
       echo "$output" | grep -q "listening server" && \
       echo "$output" | grep -q "password" && \
       echo "$output" | grep -q "mode"; then
        echo -e "${GREEN}✓ ask-share help shows correct information${NC}"
        return 0
    else
        echo -e "${RED}✗ ask-share help missing expected content${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_ask_share_no_users() {
    echo -e "${YELLOW}Testing ask-share with no users...${NC}"
    
    local output=$("$DMUX_BINARY" ask-share 2>&1)
    
    if echo "$output" | grep -q "No users specified"; then
        echo -e "${GREEN}✓ ask-share correctly handles missing users${NC}"
        return 0
    else
        echo -e "${RED}✗ ask-share should require users${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_ask_share_invalid_mode() {
    echo -e "${YELLOW}Testing ask-share with invalid mode...${NC}"
    
    local output=$("$DMUX_BINARY" ask-share --mode invalid testuser 2>&1)
    
    if echo "$output" | grep -q "Invalid mode"; then
        echo -e "${GREEN}✓ ask-share correctly validates mode${NC}"
        return 0
    else
        echo -e "${RED}✗ ask-share should validate mode${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_ask_share_basic() {
    echo -e "${YELLOW}Testing basic ask-share functionality...${NC}"
    
    # Run ask-share in background with timeout
    timeout 3s "$DMUX_BINARY" ask-share testuser1 testuser2 >/dev/null 2>&1 &
    local ask_share_pid=$!
    
    # Give it time to create invitation messages
    sleep 1
    
    # Kill the background process
    kill $ask_share_pid 2>/dev/null || true
    wait $ask_share_pid 2>/dev/null || true
    
    # Check if invitation messages were created
    local message_count=$(find "$JMUX_SHARED_DIR/messages" -name "*reverse_invite*" | wc -l)
    
    if [[ $message_count -eq 2 ]]; then
        echo -e "${GREEN}✓ ask-share created invitation messages for both users${NC}"
        return 0
    else
        echo -e "${RED}✗ ask-share should create 2 invitation messages, found $message_count${NC}"
        ls -la "$JMUX_SHARED_DIR/messages/"
        return 1
    fi
}

test_ask_share_message_content() {
    echo -e "${YELLOW}Testing ask-share message content...${NC}"
    
    # Run ask-share with specific parameters
    timeout 3s "$DMUX_BINARY" ask-share --mode view --password secret testuser3 >/dev/null 2>&1 &
    local ask_share_pid=$!
    
    # Give it time to create invitation message
    sleep 1
    
    # Kill the background process
    kill $ask_share_pid 2>/dev/null || true
    wait $ask_share_pid 2>/dev/null || true
    
    # Find and check the message content
    local message_file=$(find "$JMUX_SHARED_DIR/messages" -name "testuser3_reverse_invite*" | head -1)
    
    if [[ -f "$message_file" ]]; then
        local content=$(cat "$message_file")
        
        if echo "$content" | grep -q "TYPE=REVERSE_INVITE" && \
           echo "$content" | grep -q "MODE=view" && \
           echo "$content" | grep -q "PASSWORD_PROTECTED=true" && \
           echo "$content" | grep -q "requesting to share a session"; then
            echo -e "${GREEN}✓ ask-share message contains correct content${NC}"
            return 0
        else
            echo -e "${RED}✗ ask-share message missing expected content${NC}"
            echo "Message content:"
            cat "$message_file"
            return 1
        fi
    else
        echo -e "${RED}✗ ask-share message file not found${NC}"
        return 1
    fi
}

test_ask_share_session_registration() {
    echo -e "${YELLOW}Testing ask-share session registration...${NC}"
    
    # Run ask-share in background
    timeout 3s "$DMUX_BINARY" ask-share testuser4 >/dev/null 2>&1 &
    local ask_share_pid=$!
    
    # Give it time to register session
    sleep 1
    
    # Check if session was registered
    local sessions_output=$("$DMUX_BINARY" sessions 2>&1)
    
    # Kill the background process
    kill $ask_share_pid 2>/dev/null || true
    wait $ask_share_pid 2>/dev/null || true
    
    if echo "$sessions_output" | grep -q "reverse-"; then
        echo -e "${GREEN}✓ ask-share registers reverse session${NC}"
        return 0
    else
        echo -e "${RED}✗ ask-share should register reverse session${NC}"
        echo "Sessions output: $sessions_output"
        return 1
    fi
}

test_ask_share_private_mode() {
    echo -e "${YELLOW}Testing ask-share private mode...${NC}"
    
    # Run ask-share with private flag
    timeout 3s "$DMUX_BINARY" ask-share --private testuser5 >/dev/null 2>&1 &
    local ask_share_pid=$!
    
    # Give it time to create invitation message
    sleep 1
    
    # Kill the background process
    kill $ask_share_pid 2>/dev/null || true
    wait $ask_share_pid 2>/dev/null || true
    
    # Find and check the message content for private flag
    local message_file=$(find "$JMUX_SHARED_DIR/messages" -name "testuser5_reverse_invite*" | head -1)
    
    if [[ -f "$message_file" ]] && grep -q "PRIVATE=true" "$message_file"; then
        echo -e "${GREEN}✓ ask-share correctly sets private mode${NC}"
        return 0
    else
        echo -e "${RED}✗ ask-share should set PRIVATE=true${NC}"
        [[ -f "$message_file" ]] && cat "$message_file"
        return 1
    fi
}

test_ask_share_multiple_modes() {
    echo -e "${YELLOW}Testing ask-share with different modes...${NC}"
    
    local modes=("pair" "view" "rogue")
    local success_count=0
    
    for mode in "${modes[@]}"; do
        # Clean previous messages
        rm -f "$JMUX_SHARED_DIR/messages"/testuser_${mode}_*
        
        # Run ask-share with specific mode
        timeout 3s "$DMUX_BINARY" ask-share --mode "$mode" "testuser_${mode}" >/dev/null 2>&1 &
        local ask_share_pid=$!
        
        # Give it time to create invitation message
        sleep 1
        
        # Kill the background process
        kill $ask_share_pid 2>/dev/null || true
        wait $ask_share_pid 2>/dev/null || true
        
        # Check if mode was set correctly
        local message_file=$(find "$JMUX_SHARED_DIR/messages" -name "testuser_${mode}_reverse_invite*" | head -1)
        
        if [[ -f "$message_file" ]] && grep -q "MODE=$mode" "$message_file"; then
            success_count=$((success_count + 1))
        fi
    done
    
    if [[ $success_count -eq 3 ]]; then
        echo -e "${GREEN}✓ ask-share correctly handles all modes (pair, view, rogue)${NC}"
        return 0
    else
        echo -e "${RED}✗ ask-share failed for some modes (success: $success_count/3)${NC}"
        return 1
    fi
}

test_join_share_help() {
    echo -e "${YELLOW}Testing join-share help command...${NC}"
    
    local output=$("$DMUX_BINARY" join-share --help 2>&1)
    
    if echo "$output" | grep -q "Join a reverse sharing session" && \
       echo "$output" | grep -q "respond to reverse sharing invitations" && \
       echo "$output" | grep -q "ask-share" && \
       echo "$output" | grep -q "view" && \
       echo "$output" | grep -q "rogue"; then
        echo -e "${GREEN}✓ join-share help shows correct information${NC}"
        return 0
    else
        echo -e "${RED}✗ join-share help missing expected content${NC}"
        echo "Output: $output"
        return 1
    fi
}

test_join_share_no_invitation() {
    echo -e "${YELLOW}Testing join-share with no invitation...${NC}"
    
    # Clean any existing messages first
    rm -f "$JMUX_SHARED_DIR/messages"/*reverse_invite*
    
    local output=$("$DMUX_BINARY" join-share nonexistentuser 2>&1)
    
    if echo "$output" | grep -q "no reverse sharing invitation found"; then
        echo -e "${GREEN}✓ join-share correctly handles missing invitations${NC}"
        return 0
    else
        echo -e "${RED}✗ join-share should detect missing invitations${NC}"
        echo "Output: $output"
        return 1
    fi
}

# Main test execution
main() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}DMUX Reverse Sharing Test Suite${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    local tests_run=0
    local tests_passed=0
    
    # Run all tests
    for test_func in test_ask_share_help test_ask_share_no_users test_ask_share_invalid_mode \
                     test_ask_share_basic test_ask_share_message_content test_ask_share_session_registration \
                     test_ask_share_private_mode test_ask_share_multiple_modes \
                     test_join_share_help test_join_share_no_invitation; do
        if $test_func; then
            tests_passed=$((tests_passed + 1))
        fi
        tests_run=$((tests_run + 1))
        echo
    done
    
    # Cleanup
    rm -rf "$TEST_HOME" 2>/dev/null || true
    rm -f "$DMUX_BINARY"
    
    # Report results
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Test Results${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "  Tests run: ${tests_run}"
    echo -e "  Passed: ${GREEN}${tests_passed}${NC}"
    echo -e "  Failed: ${RED}$((tests_run - tests_passed))${NC}"
    
    if [[ $tests_passed -eq $tests_run ]]; then
        echo -e "${GREEN}✅ All reverse sharing tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}❌ Some tests failed!${NC}"
        exit 1
    fi
}

main "$@"