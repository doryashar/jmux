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

# Determine release version
if [[ "$CURRENT_VERSION" == *"-dev" ]]; then
    # Extract version without -dev suffix
    RELEASE_VERSION="${CURRENT_VERSION%-dev}"
elif [[ "$CURRENT_VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    # Already a release version, ask for confirmation or increment
    echo -e "${YELLOW}Current version is already a release version.${NC}"
    read -p "$(echo -e "${YELLOW}Use current version ${CURRENT_VERSION} or specify new version (current/new): ${NC}")" VERSION_CHOICE
    if [[ "$VERSION_CHOICE" == "new" ]]; then
        read -p "$(echo -e "${YELLOW}Enter new version (e.g., v1.1.1): ${NC}")" RELEASE_VERSION
    else
        RELEASE_VERSION="$CURRENT_VERSION"
    fi
else
    echo -e "${RED}❌ Invalid version format: ${CURRENT_VERSION}${NC}"
    echo -e "${YELLOW}Expected format: vX.Y.Z or vX.Y.Z-dev${NC}"
    exit 1
fi
echo -e "${BLUE}Release version: ${RELEASE_VERSION}${NC}"

# Prompt for confirmation
read -p "$(echo -e "${YELLOW}Create release ${RELEASE_VERSION}? (y/N): ${NC}")" -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Release cancelled${NC}"
    exit 0
fi

echo -e "${BLUE}📝 Updating version for release...${NC}"

# Update version file for release
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
GIT_COMMIT=$(git rev-parse HEAD)

# Backup original version file
cp "$VERSION_FILE" "${VERSION_FILE}.backup"

# Generate new version file with variable substitution
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

# Verify the generated file is valid Go
if ! go fmt "$VERSION_FILE" >/dev/null 2>&1; then
    echo -e "${RED}❌ Generated version file has syntax errors. Restoring backup...${NC}"
    mv "${VERSION_FILE}.backup" "$VERSION_FILE"
    exit 1
fi

echo -e "${GREEN}✅ Version file updated successfully${NC}"
echo -e "${BLUE}🔨 Building release binary...${NC}"

# Setup Go environment
export GOPATH="$HOME/go-workspace"
export GOMODCACHE="$HOME/go-workspace/pkg/mod"
export PATH="$HOME/go/bin:$PATH"
export GOROOT="$HOME/go"

# Verify Go is available
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go not found. Please install Go first.${NC}"
    exit 1
fi

echo -e "${BLUE}Go version: $(go version)${NC}"

# Build the release binary
cd "$SRC_DIR"
mkdir -p "$BIN_DIR"

echo -e "${BLUE}Building portable static binary...${NC}"
# Build with static linking for maximum portability
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a \
    -ldflags '-w -s -extldflags "-static"' \
    -tags 'netgo osusergo static_build' \
    -installsuffix netgo \
    -trimpath \
    -o "$BIN_DIR/dmux" .

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Build successful${NC}"

# Verify binary is static
echo -e "${BLUE}📦 Binary information:${NC}"
ls -lh "$BIN_DIR/dmux"
file "$BIN_DIR/dmux"

# Check if binary is truly static
if ldd "$BIN_DIR/dmux" 2>/dev/null | grep -q "not a dynamic executable"; then
    echo -e "${GREEN}✅ Binary is statically linked (portable)${NC}"
elif ldd "$BIN_DIR/dmux" 2>/dev/null | grep -q "statically linked"; then
    echo -e "${GREEN}✅ Binary is statically linked (portable)${NC}"
else
    echo -e "${YELLOW}⚠️  Binary may have dynamic dependencies:${NC}"
    ldd "$BIN_DIR/dmux" 2>/dev/null || echo "ldd check failed"
fi

# Test the binary
echo -e "${BLUE}Testing binary...${NC}"
if JMUX_SHARED_DIR="/tmp/test-dmux" "$BIN_DIR/dmux" version --verbose; then
    echo -e "${GREEN}✅ Binary test successful${NC}"
else
    echo -e "${RED}❌ Binary test failed${NC}"
    exit 1
fi

# Generate checksum
echo -e "${BLUE}Generating checksum...${NC}"
CHECKSUM=$(sha256sum "$BIN_DIR/dmux" | cut -d' ' -f1)
echo -e "${BLUE}SHA256: ${CHECKSUM}${NC}"

# Create checksum file
echo "$CHECKSUM  dmux" > "$BIN_DIR/dmux.sha256"
echo -e "${GREEN}✅ Checksum file created: ${BIN_DIR}/dmux.sha256${NC}"

echo -e "${BLUE}📤 Committing release version...${NC}"

# Commit the version change
cd "$PROJECT_ROOT"
git add "$VERSION_FILE"
git commit -m "Release ${RELEASE_VERSION}

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>"

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
    
    # Generate release notes
    RELEASE_NOTES="## dmux ${RELEASE_VERSION}

### What's New:
$(git log --oneline $(git describe --tags --abbrev=0 HEAD^ 2>/dev/null || echo "HEAD~10")..HEAD | sed 's/^/- /')

### Binary Information:
- **Type**: Statically linked, portable across Linux distributions
- **Size**: $(ls -lh "$BIN_DIR/dmux" | awk '{print $5}')
- **Architecture**: Linux x86_64
- **Dependencies**: None (static binary)
- **Compatibility**: All Linux systems with kernel 2.6+
- **SHA256**: \`$CHECKSUM\`

### Installation:
\`\`\`bash
# Download and install
curl -L https://github.com/doryashar/jmux/releases/download/${RELEASE_VERSION}/dmux -o dmux
curl -L https://github.com/doryashar/jmux/releases/download/${RELEASE_VERSION}/dmux.sha256 -o dmux.sha256

# Verify checksum (optional but recommended)
sha256sum -c dmux.sha256

# Install
chmod +x dmux
sudo mv dmux /usr/local/bin/
\`\`\`

### Features:
- 🔄 Real-time tmux session sharing
- 💬 Live messaging and notifications  
- 🚀 Built-in jcat networking (no socat dependency)
- 🔄 Auto-update functionality
- 📱 Cross-platform portability

🤖 Generated with [Claude Code](https://claude.ai/code)"

    # Create GitHub release with enhanced notes
    gh release create "$RELEASE_VERSION" \
        "$BIN_DIR/dmux" \
        "$BIN_DIR/dmux.sha256" \
        --title "dmux ${RELEASE_VERSION}" \
        --notes "$RELEASE_NOTES" \
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
git commit -m "Bump version to ${NEXT_VERSION} for development

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>"

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