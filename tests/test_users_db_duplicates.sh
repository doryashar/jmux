#!/bin/bash

# Test users.db duplicate prevention
# This test validates that users.db doesn't grow with duplicate entries

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
DMUX_BIN="$PROJECT_DIR/bin/dmux"

# Test configuration
TEST_USER=$(whoami)
TEST_CONFIG_DIR="/tmp/jmux-test-users-$$"
TEST_USERS_FILE="$TEST_CONFIG_DIR/users.db"

echo "🧪 Testing users.db duplicate prevention..."

# Setup test environment
mkdir -p "$TEST_CONFIG_DIR"

# Set environment to use test config
export JMUX_CONFIG_DIR="$TEST_CONFIG_DIR"
export JMUX_SHARED_DIR="$TEST_CONFIG_DIR"

echo "📊 Initial state - users.db should not exist:"
ls -la "$TEST_USERS_FILE" 2>/dev/null || echo "users.db doesn't exist (expected)"

echo ""
echo "🔧 Running dmux version command multiple times to trigger user registration..."

cd "$PROJECT_DIR"

# Run dmux status multiple times - this triggers registerCurrentUser each time
for i in {1..5}; do
    echo "  Run $i..."
    "$DMUX_BIN" status >/dev/null 2>&1 || true
done

echo ""
echo "📊 Contents of users.db after 5 runs:"
if [ -f "$TEST_USERS_FILE" ]; then
    echo "File size: $(wc -l < "$TEST_USERS_FILE") lines"
    echo "Contents:"
    cat "$TEST_USERS_FILE"
    
    # Count lines in users.db
    line_count=$(wc -l < "$TEST_USERS_FILE")
    
    if [ "$line_count" -eq 1 ]; then
        echo "✅ SUCCESS: users.db contains exactly 1 entry (no duplicates)"
    else
        echo "❌ FAIL: users.db contains $line_count entries (should be 1)"
        echo "This indicates duplicate entries are being added"
        exit 1
    fi
    
    # Test hostname change scenario
    echo ""
    echo "🔧 Simulating hostname change..."
    
    # Manually add an entry with different hostname to simulate old entry
    echo "$TEST_USER:old-hostname" >> "$TEST_USERS_FILE"
    echo "Added old entry. File now contains:"
    cat "$TEST_USERS_FILE"
    
    # Run dmux again - should update the hostname
    "$DMUX_BIN" status >/dev/null 2>&1 || true
    
    echo ""
    echo "📊 Contents after hostname change:"
    cat "$TEST_USERS_FILE"
    
    # Should still be 1 line with updated hostname
    line_count=$(wc -l < "$TEST_USERS_FILE")
    if [ "$line_count" -eq 1 ]; then
        echo "✅ SUCCESS: Hostname updated correctly, still 1 entry"
    else
        echo "❌ FAIL: Expected 1 entry after hostname update, got $line_count"
        exit 1
    fi
    
else
    echo "❌ FAIL: users.db was not created"
    exit 1
fi

# Cleanup test environment
rm -rf "$TEST_CONFIG_DIR"

echo ""
echo "✅ Users.db duplicate prevention test completed successfully!"
echo ""
echo "Summary of changes:"
echo "- Fixed registerCurrentUser to check for existing entries before adding"
echo "- Prevents duplicate user entries that cause memory bloating"
echo "- Updates hostname if user exists with different hostname"
echo "- Maintains clean, minimal users.db file"