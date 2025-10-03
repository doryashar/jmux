#!/bin/bash
# Build script for dmux Go version

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SRC_DIR="$PROJECT_ROOT/src/jmux-go"
BIN_DIR="$PROJECT_ROOT/bin"

echo "Building dmux Go version..."

# Ensure bin directory exists
mkdir -p "$BIN_DIR"

# Build jmux as a static binary for portability
cd "$SRC_DIR"

# Get dependencies
echo "Getting Go dependencies..."
go mod tidy

# Build the binary (fully static for portability)
echo "Building static binary..."

# Get build information
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")

# Build with version information
CGO_ENABLED=0 GOOS=linux go build -a \
  -ldflags "-w -s -extldflags '-static' -X 'jmux/internal/version.BuildTime=${BUILD_TIME}' -X 'jmux/internal/version.GitCommit=${GIT_COMMIT}'" \
  -tags netgo -installsuffix netgo \
  -o "$BIN_DIR/dmux" .

echo "✓ dmux built successfully at $BIN_DIR/dmux"

# Make it executable
chmod +x "$BIN_DIR/dmux"

# Show binary info
echo "Binary info:"
ls -lh "$BIN_DIR/dmux"
file "$BIN_DIR/dmux"

echo ""
echo "Usage:"
echo "  $BIN_DIR/dmux --help"