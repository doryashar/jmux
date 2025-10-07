#!/bin/bash
# Quick release script - wrapper around the main release script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RELEASE_SCRIPT="$SCRIPT_DIR/release.sh"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Quick Release Helper${NC}"
echo "=========================="

# Check if main release script exists
if [[ ! -f "$RELEASE_SCRIPT" ]]; then
    echo -e "${RED}❌ Release script not found: $RELEASE_SCRIPT${NC}"
    exit 1
fi

# Parse options
PREVIEW_ONLY=false
AUTO_MODE=false

case "${1:-}" in
    "preview"|"dry-run"|"--dry-run")
        PREVIEW_ONLY=true
        echo -e "${YELLOW}🔍 Preview mode - showing what would be released${NC}"
        ;;
    "auto"|"--auto")
        AUTO_MODE=true
        echo -e "${GREEN}⚡ Auto mode - fully automated release${NC}"
        ;;
    "help"|"--help"|"-h"|"")
        echo "Usage: $0 [MODE]"
        echo ""
        echo "Modes:"
        echo "  preview    Show what would be released (dry-run)"
        echo "  auto       Fully automated release (no prompts)"
        echo "  (none)     Interactive release with prompts"
        echo ""
        echo "Examples:"
        echo "  $0 preview    # Preview the next release"
        echo "  $0 auto       # Automatically create and publish release"
        echo "  $0            # Interactive release"
        echo ""
        exit 0
        ;;
    *)
        echo -e "${BLUE}📝 Interactive mode - will prompt for confirmation${NC}"
        ;;
esac

echo ""

# Execute the appropriate mode
if [[ "$PREVIEW_ONLY" == "true" ]]; then
    exec "$RELEASE_SCRIPT" --dry-run
elif [[ "$AUTO_MODE" == "true" ]]; then
    exec "$RELEASE_SCRIPT" --auto
else
    exec "$RELEASE_SCRIPT"
fi