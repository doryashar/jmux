#!/bin/bash
# Release script for jmux-go

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SRC_DIR="$PROJECT_ROOT/src/jmux-go"
BIN_DIR="$PROJECT_ROOT/bin"
VERSION_FILE="$SRC_DIR/internal/version/version.go"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 dmux Release Script${NC}"
echo "================================"

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo -e "${RED}❌ Not in a git repository${NC}"
    exit 1
fi

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    echo -e "${RED}❌ You have uncommitted changes. Please commit or stash them first.${NC}"
    exit 1
fi

# Get current version
CURRENT_VERSION=$(grep 'Version.*=' "$VERSION_FILE" | sed 's/.*"\(.*\)".*/\1/')
echo -e "${BLUE}Current version: ${CURRENT_VERSION}${NC}"

# Check if current version is dev
if [[ ! "$CURRENT_VERSION" == *"-dev" ]]; then
    echo -e "${RED}❌ Current version is not a dev version. Expected format: vX.Y.Z-dev${NC}"
    exit 1
fi

# Extract version without -dev suffix
RELEASE_VERSION="${CURRENT_VERSION%-dev}"
echo -e "${BLUE}Release version: ${RELEASE_VERSION}${NC}"

# Prompt for confirmation
read -p "$(echo -e "${YELLOW}Create release ${RELEASE_VERSION}? (y/N): ${NC}")" -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Release cancelled${NC}"
    exit 0
fi

echo -e "${BLUE}📝 Updating version for release...${NC}"

# Update version file for release (remove -dev)
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
GIT_COMMIT=$(git rev-parse HEAD)

cat > "$VERSION_FILE" << EOF
package version

import (
	"fmt"
	"runtime"
)

// Version information - will be set at build time
var (
	Version   = "${RELEASE_VERSION}"
	GitCommit = "${GIT_COMMIT}"
	BuildTime = "${BUILD_TIME}"
	GoVersion = runtime.Version()
)

// GetVersion returns the current version
func GetVersion() string {
	return Version
}

// GetFullVersion returns detailed version information
func GetFullVersion() string {
	return fmt.Sprintf("dmux %s\\nGit commit: %s\\nBuild time: %s\\nGo version: %s",
		Version, GitCommit, BuildTime, GoVersion)
}

// IsDevVersion returns true if this is a development version
func IsDevVersion() bool {
	return len(Version) > 4 && Version[len(Version)-4:] == "-dev"
}
EOF

echo -e "${BLUE}🔨 Building release binary...${NC}"

# Build the release binary
cd "$SRC_DIR"
mkdir -p "$BIN_DIR"

# Build with release flags
CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-w -s -extldflags "-static"' -tags netgo -installsuffix netgo -o "$BIN_DIR/jmux-go" .

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Build successful${NC}"

# Show binary info
echo -e "${BLUE}📦 Binary information:${NC}"
ls -lh "$BIN_DIR/jmux-go"
file "$BIN_DIR/jmux-go"
"$BIN_DIR/jmux-go" version --verbose

echo -e "${BLUE}📤 Committing release version...${NC}"

# Commit the version change
cd "$PROJECT_ROOT"
git add "$VERSION_FILE"
git commit -m "Release ${RELEASE_VERSION}"

# Create git tag
echo -e "${BLUE}🏷️  Creating git tag...${NC}"
git tag -a "$RELEASE_VERSION" -m "Release ${RELEASE_VERSION}"

echo -e "${BLUE}🚀 Creating GitHub release...${NC}"

# Check if gh CLI is available
if ! command -v gh &> /dev/null; then
    echo -e "${YELLOW}⚠️  GitHub CLI (gh) not found. Please install it to create GitHub releases automatically.${NC}"
    echo -e "${BLUE}Manual steps:${NC}"
    echo "1. Push the tag: git push origin ${RELEASE_VERSION}"
    echo "2. Create release on GitHub with the binary"
else
    # Push the tag
    git push origin "$RELEASE_VERSION"
    
    # Create GitHub release
    gh release create "$RELEASE_VERSION" \
        "$BIN_DIR/dmux" \
        --title "dmux ${RELEASE_VERSION}" \
        --notes "Release ${RELEASE_VERSION}" \
        --latest
    
    echo -e "${GREEN}✅ GitHub release created successfully${NC}"
fi

echo -e "${BLUE}📈 Incrementing version for next development cycle...${NC}"

# Calculate next dev version (increment patch version)
IFS='.' read -ra VERSION_PARTS <<< "${RELEASE_VERSION#v}"
MAJOR=${VERSION_PARTS[0]}
MINOR=${VERSION_PARTS[1]}
PATCH=${VERSION_PARTS[2]}

NEXT_PATCH=$((PATCH + 1))
NEXT_VERSION="v${MAJOR}.${MINOR}.${NEXT_PATCH}-dev"

echo -e "${BLUE}Next dev version: ${NEXT_VERSION}${NC}"

# Update version file for next development cycle
cat > "$VERSION_FILE" << EOF
package version

import (
	"fmt"
	"runtime"
)

// Version information - will be set at build time
var (
	Version   = "${NEXT_VERSION}"
	GitCommit = "unknown"
	BuildTime = "unknown"
	GoVersion = runtime.Version()
)

// GetVersion returns the current version
func GetVersion() string {
	return Version
}

// GetFullVersion returns detailed version information
func GetFullVersion() string {
	return fmt.Sprintf("dmux %s\\nGit commit: %s\\nBuild time: %s\\nGo version: %s",
		Version, GitCommit, BuildTime, GoVersion)
}

// IsDevVersion returns true if this is a development version
func IsDevVersion() bool {
	return len(Version) > 4 && Version[len(Version)-4:] == "-dev"
}
EOF

# Commit the dev version
git add "$VERSION_FILE"
git commit -m "Bump version to ${NEXT_VERSION} for development"

echo
echo -e "${GREEN}🎉 Release ${RELEASE_VERSION} completed successfully!${NC}"
echo -e "${BLUE}📋 Summary:${NC}"
echo "  • Released version: ${RELEASE_VERSION}"
echo "  • Next dev version: ${NEXT_VERSION}"
echo "  • Binary location: ${BIN_DIR}/dmux"
echo
echo -e "${YELLOW}📝 Next steps:${NC}"
echo "  • Push the development version: git push origin main"
echo "  • Test the auto-update: dmux update"
echo "  • Verify the release: https://github.com/doryashar/jmux/releases"