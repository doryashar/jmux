#!/bin/bash
# Test executable path detection

echo "Testing dmux executable path detection"
echo ""

echo "Current working directory:"
pwd
echo ""

echo "dmux binary location:"
ls -la ./bin/dmux
echo ""

echo "Testing dmux version:"
./bin/dmux --version
echo ""

echo "Testing executable path detection (debug):"
export JMUX_SHARED_DIR="$HOME/.jmux/shared"
export DMUX_DEBUG=1

# Check if monitor is running and stop it
./bin/dmux monitor stop 2>/dev/null

echo ""
echo "Starting monitor with debug..."
./bin/dmux monitor start

echo ""
echo "Monitor status:"
./bin/dmux monitor status

echo ""
echo "Process list for dmux:"
ps aux | grep dmux | grep -v grep

echo ""
echo "Recent monitor logs:"
./bin/dmux monitor logs -n 5