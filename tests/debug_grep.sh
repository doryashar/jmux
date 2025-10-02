#!/bin/bash

echo "=== Debug Grep Behavior ==="

# Create a test port mapping file
PORT_MAP_FILE="/tmp/test_port_map.db"

# Test case 1: Normal case
echo "12345:tomere:xsrl8-emp-506-tomere" > "$PORT_MAP_FILE"

echo "Test 1: Normal case"
echo "File contents:"
cat "$PORT_MAP_FILE"

RESULT=$(grep "^12345:" "$PORT_MAP_FILE" 2>/dev/null | head -1 | cut -d: -f3)
echo "Result: '$RESULT'"
echo "Length: ${#RESULT}"

# Test case 2: Empty result
echo "" > "$PORT_MAP_FILE"

echo -e "\nTest 2: Empty file"
echo "File contents:"
cat "$PORT_MAP_FILE"

RESULT=$(grep "^12345:" "$PORT_MAP_FILE" 2>/dev/null | head -1 | cut -d: -f3)
echo "Result: '$RESULT'"
echo "Length: ${#RESULT}"

# Test case 3: Port not found
echo "54321:other:session" > "$PORT_MAP_FILE"

echo -e "\nTest 3: Port not found"
echo "File contents:"
cat "$PORT_MAP_FILE"

RESULT=$(grep "^12345:" "$PORT_MAP_FILE" 2>/dev/null | head -1 | cut -d: -f3)
echo "Result: '$RESULT'"
echo "Length: ${#RESULT}"

# Test case 4: What happens if cut gets empty input
echo -e "\nTest 4: Cut with empty input"
RESULT=$(echo "" | cut -d: -f3)
echo "Result: '$RESULT'"
echo "Length: ${#RESULT}"

rm -f "$PORT_MAP_FILE"