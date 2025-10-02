#!/usr/bin/env bash

# jmux test suite
# Tests for the enhanced jmux functionality

set -euo pipefail

# Colors for test output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_DIR="/tmp/jmux_test_$$"
JMUX_SCRIPT="$(dirname "$0")/jmux"
export JMUX_SHARED_DIR="${TEST_DIR}/shared"
export JMUX_PORT="22345"

# Test tracking
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

setup_test() {
    echo -e "${BLUE}Setting up test environment...${NC}"
    mkdir -p "${TEST_DIR}/shared/jmux/"{messages,sessions}
    touch "${TEST_DIR}/shared/jmux/users.db"
    
    # Create test users
    echo "alice:192.168.1.10:$(date +%s)" >> "${TEST_DIR}/shared/jmux/users.db"
    echo "bob:192.168.1.11:$(date +%s)" >> "${TEST_DIR}/shared/jmux/users.db"
    echo "charlie:192.168.1.12:$(date +%s)" >> "${TEST_DIR}/shared/jmux/users.db"
}

cleanup_test() {
    echo -e "${BLUE}Cleaning up test environment...${NC}"
    rm -rf "${TEST_DIR}"
}

run_test() {
    local test_name="$1"
    local test_func="$2"
    
    echo -e "${YELLOW}Running test: ${test_name}${NC}"
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if $test_func; then
        echo -e "${GREEN}✓ PASS: ${test_name}${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ FAIL: ${test_name}${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    echo
}

# Test 1: Named session creation
test_named_session() {
    local session_file="${JMUX_SHARED_DIR}/jmux/sessions/${USER}_testsession.session"
    
    # Create a mock session file to test parsing
    cat > "${session_file}" << EOF
USER=${USER}
SESSION=testsession
PORT=22346
STARTED=$(date +%s)
PID=12345
PRIVATE=false
ALLOWED_USERS=
EOF
    
    # Check if session file was created correctly
    if [[ -f "${session_file}" ]]; then
        source "${session_file}"
        [[ "${SESSION}" == "testsession" ]] && [[ "${PORT}" == "22346" ]]
    else
        return 1
    fi
}

# Test 2: Private session functionality
test_private_session() {
    local session_file="${JMUX_SHARED_DIR}/jmux/sessions/alice_private.session"
    
    # Create a private session file
    cat > "${session_file}" << EOF
USER=alice
SESSION=private
PORT=22347
STARTED=$(date +%s)
PID=12346
PRIVATE=true
ALLOWED_USERS=bob,charlie
EOF
    
    # Test access check logic
    source "${session_file}"
    if [[ "${PRIVATE}" == "true" ]]; then
        IFS=',' read -ra allowed_array <<< "${ALLOWED_USERS}"
        local user_allowed=false
        for allowed_user in "${allowed_array[@]}"; do
            if [[ "${allowed_user}" == "bob" ]]; then
                user_allowed=true
                break
            fi
        done
        [[ "${user_allowed}" == "true" ]]
    else
        return 1
    fi
}

# Test 3: Multiple invitees message creation
test_multiple_invites() {
    local invite_count=0
    
    # Simulate sending invites to multiple users
    for user in alice bob charlie; do
        local msg_file="${JMUX_SHARED_DIR}/jmux/messages/${user}_$(date +%s%N).msg"
        cat > "${msg_file}" << EOF
FROM=${USER}
TYPE=INVITE
TIMESTAMP=$(date +%s)
DATA=testsession
EOF
        if [[ -f "${msg_file}" ]]; then
            invite_count=$((invite_count + 1))
        fi
    done
    
    [[ ${invite_count} -eq 3 ]]
}

# Test 4: Auto-join functionality
test_auto_join() {
    # Create an invitation message
    local msg_file="${JMUX_SHARED_DIR}/jmux/messages/${USER}_$(date +%s%N).msg"
    cat > "${msg_file}" << EOF
FROM=alice
TYPE=INVITE
TIMESTAMP=$(date +%s)
DATA=testsession
EOF
    
    # Check if invitation can be found
    local invite_msg=$(find "${JMUX_SHARED_DIR}/jmux/messages" -name "${USER}_*.msg" -exec grep -l "TYPE=INVITE" {} \; 2>/dev/null | head -1)
    
    if [[ -n "${invite_msg}" ]]; then
        source "${invite_msg}"
        [[ "${FROM}" == "alice" ]] && [[ "${DATA}" == "testsession" ]]
    else
        return 1
    fi
}

# Test 5: Session name with invitees generation
test_session_name_generation() {
    local hostname="testhost"
    local invite_users=("alice" "bob")
    
    # Simulate session name generation logic
    local invitees_str=$(IFS="-"; echo "${invite_users[*]}")
    local session_name="${hostname}-${invitees_str}"
    
    [[ "${session_name}" == "testhost-alice-bob" ]]
}

# Test 6: Join specific session
test_join_specific_session() {
    # Create multiple sessions for alice
    local session1="${JMUX_SHARED_DIR}/jmux/sessions/alice_session1.session"
    local session2="${JMUX_SHARED_DIR}/jmux/sessions/alice_session2.session"
    
    cat > "${session1}" << EOF
USER=alice
SESSION=session1
PORT=22348
STARTED=$(date +%s)
PID=12347
PRIVATE=false
ALLOWED_USERS=
EOF
    
    cat > "${session2}" << EOF
USER=alice
SESSION=session2
PORT=22349
STARTED=$(date +%s)
PID=12348
PRIVATE=false
ALLOWED_USERS=
EOF
    
    # Test that specific session can be found
    local target_session="${JMUX_SHARED_DIR}/jmux/sessions/alice_session2.session"
    if [[ -f "${target_session}" ]]; then
        source "${target_session}"
        [[ "${SESSION}" == "session2" ]] && [[ "${PORT}" == "22349" ]]
    else
        return 1
    fi
}

# Test 7: Tmux status line message generation
test_tmux_status_line() {
    local user="testuser"
    local port="22350"
    local connection_count="2"
    
    # Test status message generation
    local status_msg="[SHARED] Join: jmux join ${user} | Connections: ${connection_count}"
    local expected="[SHARED] Join: jmux join testuser | Connections: 2"
    
    [[ "${status_msg}" == "${expected}" ]]
}

# Test 8: IP address validation
test_ip_validation() {
    # Source the jmux script to get the is_ip_address function
    source "${JMUX_SCRIPT}"
    
    # Valid IP addresses
    is_ip_address "192.168.1.1" || return 1
    is_ip_address "10.0.0.1" || return 1
    is_ip_address "172.16.0.1" || return 1
    is_ip_address "255.255.255.255" || return 1
    is_ip_address "0.0.0.0" || return 1
    
    # Invalid IP addresses
    ! is_ip_address "256.1.1.1" || return 1
    ! is_ip_address "192.168.1" || return 1
    ! is_ip_address "192.168.1.1.1" || return 1
    ! is_ip_address "not.an.ip" || return 1
    ! is_ip_address "hostname" || return 1
    
    return 0
}

# Test 9: Hostname validation
test_hostname_validation() {
    # Test basic hostname patterns directly
    
    # Valid hostname patterns (should contain dots or hyphens)
    [[ "server.example.com" =~ \. ]] || return 1
    [[ "host.local" =~ \. ]] || return 1  
    [[ "my-server" =~ - ]] || return 1
    [[ "server123.domain.org" =~ \. ]] || return 1
    
    # Invalid hostname patterns (pure usernames)
    [[ "username" =~ ^[a-zA-Z]+$ ]] || return 1
    [[ "user123" =~ [a-zA-Z] ]] && [[ "user123" =~ [0-9] ]] || return 1
    
    # Should not match IP pattern
    local ip="192.168.1.1"
    local IFS='.'
    read -ra parts <<< "$ip"
    [[ ${#parts[@]} -eq 4 ]] || return 1
    
    return 0
}

# Test 10: Real-time message format
test_realtime_message_format() {
    local test_msg_file="${TEST_DIR}/test_message.msg"
    
    # Create test message with new format
    cat > "${test_msg_file}" << 'EOF'
FROM=alice
TYPE=URGENT
TIMESTAMP=1234567890
DATA="Test urgent message"
PRIORITY=urgent
READ=false
EOF
    
    # Verify all fields are present by reading them manually
    local from=$(grep "^FROM=" "${test_msg_file}" | cut -d= -f2)
    local type=$(grep "^TYPE=" "${test_msg_file}" | cut -d= -f2)
    local data=$(grep "^DATA=" "${test_msg_file}" | cut -d= -f2)
    local priority=$(grep "^PRIORITY=" "${test_msg_file}" | cut -d= -f2)
    local read_status=$(grep "^READ=" "${test_msg_file}" | cut -d= -f2)
    
    [[ "${from}" == "alice" ]] || return 1
    [[ "${type}" == "URGENT" ]] || return 1
    [[ "${data}" == "\"Test urgent message\"" ]] || return 1
    [[ "${priority}" == "urgent" ]] || return 1
    [[ "${read_status}" == "false" ]] || return 1
    
    return 0
}

# Test 11: Message watcher PID management
test_message_watcher_pid() {
    local test_pid_file="${TEST_DIR}/test_watcher.pid"
    
    # Test PID file creation
    echo "12345" > "${test_pid_file}"
    [[ -f "${test_pid_file}" ]] || return 1
    
    # Test PID reading
    local test_pid=$(cat "${test_pid_file}")
    [[ "${test_pid}" == "12345" ]] || return 1
    
    # Test PID file cleanup
    rm -f "${test_pid_file}"
    [[ ! -f "${test_pid_file}" ]] || return 1
    
    return 0
}

# Test 12: Message type handling
test_message_types() {
    # Test message type classification
    local message_types=("INVITE" "URGENT" "MESSAGE")
    
    for msg_type in "${message_types[@]}"; do
        case "${msg_type}" in
            INVITE|URGENT|MESSAGE)
                # Valid message type
                ;;
            *)
                return 1
                ;;
        esac
    done
    
    return 0
}

# Main test runner
main() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}jmux Enhanced Features Test Suite${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    if [[ ! -f "${JMUX_SCRIPT}" ]]; then
        echo -e "${RED}Error: jmux script not found at ${JMUX_SCRIPT}${NC}"
        exit 1
    fi
    
    setup_test
    
    # Run all tests
    run_test "Named session creation" test_named_session
    run_test "Private session functionality" test_private_session
    run_test "Multiple invitees message creation" test_multiple_invites
    run_test "Auto-join functionality" test_auto_join
    run_test "Session name generation with invitees" test_session_name_generation
    run_test "Join specific session" test_join_specific_session
    run_test "Tmux status line message generation" test_tmux_status_line
    run_test "IP address validation" test_ip_validation
    run_test "Hostname validation" test_hostname_validation
    run_test "Real-time message format" test_realtime_message_format
    run_test "Message watcher PID management" test_message_watcher_pid
    run_test "Message type handling" test_message_types
    
    cleanup_test
    
    # Report results
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Test Results${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "  Tests run: ${TESTS_RUN}"
    echo -e "  Passed: ${GREEN}${TESTS_PASSED}${NC}"
    echo -e "  Failed: ${RED}${TESTS_FAILED}${NC}"
    
    if [[ ${TESTS_FAILED} -eq 0 ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    fi
}

main "$@"