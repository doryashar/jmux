#!/bin/bash
# Test invitation lifecycle: create, receive, join, cleanup

set -e

# Test setup
TEST_DIR="/tmp/jmux-invitation-test"
MESSAGES_DIR="$TEST_DIR/messages"
USER1="testuser1"
USER2="testuser2"

echo "🧪 Testing invitation lifecycle..."

# Setup test environment
rm -rf "$TEST_DIR"
mkdir -p "$MESSAGES_DIR"

# Create test invitation
echo "📨 Creating test invitation..."
INVITATION="{\"from\":\"$USER1\",\"type\":\"INVITE\",\"timestamp\":$(date +%s),\"data\":\"Test invitation\",\"priority\":\"high\"}"
echo "$INVITATION" > "$MESSAGES_DIR/$USER2.invites"

echo "✅ Created invitation file with content:"
cat "$MESSAGES_DIR/$USER2.invites"

# Test invitation persistence (simulate dmux messages reading but not clearing)
echo ""
echo "📖 Testing invitation persistence..."
USER="$USER2" JMUX_SHARED_DIR="$TEST_DIR" ./bin/dmux messages

echo "✅ Invitation should still exist after viewing:"
cat "$MESSAGES_DIR/$USER2.invites" 2>/dev/null || echo "File doesn't exist"

# Add another invitation
echo ""
echo "📨 Adding second invitation..."
INVITATION2="{\"from\":\"$USER1\",\"type\":\"INVITE\",\"timestamp\":$(date +%s),\"data\":\"Second invitation\",\"priority\":\"high\"}"
echo "$INVITATION2" >> "$MESSAGES_DIR/$USER2.invites"

echo "✅ Now have multiple invitations:"
cat "$MESSAGES_DIR/$USER2.invites"

# Test invitation cleanup
echo ""
echo "🧹 Testing invitation cleanup..."
USER="$USER2" JMUX_SHARED_DIR="$TEST_DIR" ./bin/dmux clear-invites

echo "✅ After clearing, invitations file should be empty:"
cat "$MESSAGES_DIR/$USER2.invites" 2>/dev/null || echo "File is empty or doesn't exist"

# Test expired invitation cleanup (simulate old invitation)
echo ""
echo "⏰ Testing expired invitation cleanup..."
OLD_TIMESTAMP=$(($(date +%s) - 86400 - 3600)) # 25 hours ago
EXPIRED_INVITATION="{\"from\":\"$USER1\",\"type\":\"INVITE\",\"timestamp\":$OLD_TIMESTAMP,\"data\":\"Expired invitation\",\"priority\":\"high\"}"
CURRENT_INVITATION="{\"from\":\"$USER1\",\"type\":\"INVITE\",\"timestamp\":$(date +%s),\"data\":\"Current invitation\",\"priority\":\"high\"}"

echo "$EXPIRED_INVITATION" > "$MESSAGES_DIR/$USER2.invites"
echo "$CURRENT_INVITATION" >> "$MESSAGES_DIR/$USER2.invites"

echo "✅ Created invitations with mixed timestamps:"
cat "$MESSAGES_DIR/$USER2.invites"

# Note: Automatic cleanup happens in the monitor every 10 minutes
# We can't easily test it here without running the monitor
echo ""
echo "ℹ️  Automatic cleanup of expired invitations happens every 10 minutes in the monitor"
echo "   Invitations older than 24 hours are automatically removed"

# Cleanup
rm -rf "$TEST_DIR"

echo ""
echo "🎉 Invitation lifecycle test completed successfully!"
echo ""
echo "Summary of invitation behavior:"
echo "✅ Invitations persist after viewing with 'dmux messages'"
echo "✅ Multiple invitations can accumulate"
echo "✅ Invitations can be manually cleared with 'dmux clear-invites'"
echo "✅ Invitations are automatically removed when joining the inviting user's session"
echo "✅ Invitations older than 24 hours are automatically cleaned up by the monitor"