#!/bin/bash
# Build script for jcat binary

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SRC_DIR="$PROJECT_ROOT/src/jcat"
BIN_DIR="$PROJECT_ROOT/bin"

echo "Building jcat..."

# Ensure bin directory exists
mkdir -p "$BIN_DIR"

# Build jcat as a static binary for portability
cd "$SRC_DIR"
CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-w -s -extldflags "-static"' -o "$BIN_DIR/jcat-binary" ./jcat.go

echo "✓ jcat built successfully at $BIN_DIR/jcat-binary"

# Make it executable
chmod +x "$BIN_DIR/jcat-binary"

# Show binary info
echo "Binary info:"
ls -lh "$BIN_DIR/jcat-binary"
file "$BIN_DIR/jcat-binary"