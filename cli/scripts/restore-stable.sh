#!/bin/bash
# Restore the stable version of azd app extension
# Usage: ./restore-stable.sh
# Or: curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/scripts/restore-stable.sh | bash

set -e

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
GRAY='\033[0;90m'
WHITE='\033[0;37m'
NC='\033[0m' # No Color

REPO="jongio/azd-app"
EXTENSION_ID="jongio.azd.app"
STABLE_REGISTRY_URL="https://raw.githubusercontent.com/${REPO}/refs/heads/main/registry.json"

echo -e "${CYAN}🔄 Restoring stable azd app extension${NC}"
echo ""

# Step 1: Uninstall current extension
echo -e "${GRAY}🗑️  Uninstalling current extension...${NC}"
azd extension uninstall $EXTENSION_ID 2>/dev/null || true

# Step 2: Remove all PR registry sources
echo -e "${GRAY}🧹 Removing PR registry sources...${NC}"
SOURCES=$(azd extension source list --output json 2>/dev/null || echo "[]")
if [ "$SOURCES" != "[]" ]; then
    echo "$SOURCES" | grep -o '"name":"pr-[0-9]*"' | sed 's/"name":"\(.*\)"/\1/' | while read -r source; do
        if [ ! -z "$source" ]; then
            echo -e "${GRAY}   Removing: $source${NC}"
            azd extension source remove "$source" 2>/dev/null || true
        fi
    done
fi

# Step 3: Clean up pr-registry.json files
echo -e "${GRAY}🧹 Cleaning up pr-registry.json files...${NC}"
rm -f ./pr-registry.json
rm -f ~/pr-registry.json
rm -f $HOME/pr-registry.json

# Step 4: Add stable registry source
echo -e "${GRAY}🔗 Adding stable registry source...${NC}"
azd extension source remove "app" 2>/dev/null || true
azd extension source add -n "app" -t url -l "$STABLE_REGISTRY_URL"
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Failed to add stable registry source${NC}"
    exit 1
fi

# Step 5: Install latest stable version
echo -e "${GRAY}📦 Installing latest stable version...${NC}"
azd extension install $EXTENSION_ID
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Failed to install stable extension${NC}"
    exit 1
fi

# Step 6: Verify installation
echo ""
echo -e "${GREEN}✅ Restoration complete!${NC}"
echo ""
echo -e "${GRAY}🔍 Verifying installation...${NC}"
INSTALLED_VERSION=$(azd app version 2>&1)
if [ $? -eq 0 ]; then
    echo -e "${GRAY}   $INSTALLED_VERSION${NC}"
    echo ""
    echo -e "${GREEN}✨ Success! Stable version is installed.${NC}"
else
    echo -e "${YELLOW}⚠️  Could not verify version${NC}"
fi

echo ""
echo -e "${CYAN}You can now use azd app normally:${NC}"
echo -e "${WHITE}  azd app run${NC}"
echo -e "${WHITE}  azd app reqs${NC}"
echo -e "${WHITE}  azd app run${NC}"
