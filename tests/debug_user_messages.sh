#!/bin/bash
# Debug user message processing

echo "🔍 Debugging user message processing"
echo ""

echo "Environment Information:"
echo "USER: $USER"
echo "LOGNAME: $LOGNAME"
echo "whoami: $(whoami)"
echo ""

echo "Expected message prefix: ${USER}_"
echo ""

echo "Checking messages directory:"
MESSAGES_DIR="/projects/common/work/dory/jmux/messages"
if [ -d "$MESSAGES_DIR" ]; then
    echo "Messages directory: $MESSAGES_DIR"
    echo ""
    
    echo "All message files:"
    ls -la "$MESSAGES_DIR"/*.msg 2>/dev/null || echo "No message files found"
    echo ""
    
    echo "Messages for current user (${USER}):"
    ls -la "$MESSAGES_DIR"/${USER}_*.msg 2>/dev/null || echo "No messages for user $USER"
    echo ""
    
    echo "Messages for other users:"
    ls -la "$MESSAGES_DIR" | grep "\.msg" | grep -v "${USER}_" || echo "No messages for other users"
    echo ""
else
    echo "❌ Messages directory not found: $MESSAGES_DIR"
fi

echo "Monitor status:"
export JMUX_SHARED_DIR="/projects/common/work/dory/jmux"
./bin/dmux monitor status
echo ""

echo "Recent monitor logs:"
./bin/dmux monitor logs -n 5
echo ""

echo "💡 Debugging tips:"
echo "1. Check if you're running as the right user"
echo "2. Message files should be named: \${USER}_timestamp.msg"
echo "3. Monitor only processes messages for current user"
echo "4. Try: export DMUX_DEBUG=1 && dmux monitor restart"
echo "5. Create test message: echo 'test' | dmux msg \$USER"