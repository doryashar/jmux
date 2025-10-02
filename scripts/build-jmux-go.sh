#!/bin/bash
# Build script for jmux Go version

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SRC_DIR="$PROJECT_ROOT/src/jmux-go"
BIN_DIR="$PROJECT_ROOT/bin"

echo "Building jmux Go version..."

# Ensure bin directory exists
mkdir -p "$BIN_DIR"

# Build jmux as a static binary for portability
cd "$SRC_DIR"

# Get dependencies
echo "Getting Go dependencies..."
go mod tidy

# Build the binary
echo "Building binary..."
CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-w -s -extldflags "-static"' -o "$BIN_DIR/jmux-go" .

echo "✓ jmux-go built successfully at $BIN_DIR/jmux-go"

# Make it executable
chmod +x "$BIN_DIR/jmux-go"

# Show binary info
echo "Binary info:"
ls -lh "$BIN_DIR/jmux-go"
file "$BIN_DIR/jmux-go"

echo ""
echo "Usage:"
echo "  $BIN_DIR/jmux-go --help"