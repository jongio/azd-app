#!/bin/bash
# Install a PR build of the azd app extension
# Usage: ./install-pr.sh PR_NUMBER VERSION
# Example: ./install-pr.sh 123 0.5.7-pr123
# Or: curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/scripts/install-pr.sh | bash -s 123 0.5.7-pr123

set -e

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
GRAY='\033[0;90m'
WHITE='\033[0;37m'
NC='\033[0m' # No Color

# Check arguments
if [ -z "$1" ] || [ -z "$2" ]; then
    echo -e "${RED}❌ Error: Missing arguments${NC}"
    echo "Usage: $0 PR_NUMBER VERSION"
    echo "Example: $0 123 0.5.7-pr123"
    exit 1
fi

PR_NUMBER=$1
VERSION=$2
REPO="jongio/azd-app"
EXTENSION_ID="jongio.azd.app"
TAG="azd-ext-jongio-azd-app_${VERSION}"
REGISTRY_URL="https://github.com/${REPO}/releases/download/${TAG}/pr-registry.json"
REGISTRY_PATH="./pr-registry.json"

echo -e "${CYAN}🚀 Installing azd app PR #${PR_NUMBER} (version ${VERSION})${NC}"
echo ""

# Step 1: Enable extensions
echo -e "${GRAY}📋 Enabling azd extensions...${NC}"
azd config set alpha.extension.enabled on
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Failed to enable extensions${NC}"
    exit 1
fi

# Step 2: Uninstall existing extension
echo -e "${GRAY}🗑️  Uninstalling existing extension (if any)...${NC}"
azd extension uninstall $EXTENSION_ID 2>/dev/null || true
echo -e "${GRAY}   ✓${NC}"

# Step 3: Download PR registry
echo -e "${GRAY}📥 Downloading PR registry...${NC}"
if curl -fsSL -o "$REGISTRY_PATH" "$REGISTRY_URL"; then
    echo -e "${GRAY}   ✓ Downloaded to: $REGISTRY_PATH${NC}"
else
    echo -e "${RED}❌ Failed to download registry from $REGISTRY_URL${NC}"
    echo -e "${YELLOW}   Make sure the PR build exists and is accessible${NC}"
    exit 1
fi

# Step 4: Add registry source
echo -e "${GRAY}🔗 Adding PR registry source...${NC}"
azd extension source remove "pr-${PR_NUMBER}" 2>/dev/null || true
azd extension source add -n "pr-${PR_NUMBER}" -t file -l "$(pwd)/${REGISTRY_PATH}"
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Failed to add registry source${NC}"
    exit 1
fi

# Step 5: Install PR version
echo -e "${GRAY}📦 Installing version ${VERSION}...${NC}"
azd extension install $EXTENSION_ID --version $VERSION
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Failed to install extension${NC}"
    exit 1
fi

# Step 6: Verify installation
echo ""
echo -e "${GREEN}✅ Installation complete!${NC}"
echo ""
echo -e "${GRAY}🔍 Verifying installation...${NC}"
INSTALLED_VERSION=$(azd app version 2>&1)
if [ $? -eq 0 ]; then
    echo -e "${GRAY}   $INSTALLED_VERSION${NC}"
    if echo "$INSTALLED_VERSION" | grep -q "$VERSION"; then
        echo ""
        echo -e "${GREEN}✨ Success! PR build is ready to test.${NC}"
    else
        echo ""
        echo -e "${YELLOW}⚠️  Version mismatch - expected $VERSION${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Could not verify version${NC}"
fi

echo ""
echo -e "${CYAN}Try these commands:${NC}"
echo -e "${WHITE}  azd app hi${NC}"
echo -e "${WHITE}  azd app reqs${NC}"
echo ""
echo -e "${GRAY}To restore stable version, run:${NC}"
echo -e "${WHITE}  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/cli/scripts/restore-stable.sh | bash${NC}"
