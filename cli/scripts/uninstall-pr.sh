#!/usr/bin/env bash
# Uninstall azd app PR build

set -e

EXTENSION_ID="jongio.azd.app"
PR_NUMBER="${1:-}"

echo "🗑️  Uninstalling azd app PR build"
echo ""

# Uninstall extension
echo "📦 Removing extension..."
azd extension uninstall "$EXTENSION_ID" 2>/dev/null || true
echo "   ✓"

# Remove PR registry sources
if [ -n "$PR_NUMBER" ]; then
    echo "🔗 Removing PR #$PR_NUMBER registry source..."
    azd extension source remove "pr-$PR_NUMBER" 2>/dev/null || true
    echo "   ✓"
else
    echo "🔗 Removing all PR registry sources..."
    azd extension source list 2>/dev/null | grep -E "^pr-[0-9]+" | awk '{print $1}' | while read -r source; do
        azd extension source remove "$source" 2>/dev/null || true
    done
    echo "   ✓"
fi

# Clean up local registry file if it exists
if [ -f "pr-registry.json" ]; then
    echo "🧹 Cleaning up registry file..."
    rm -f "pr-registry.json"
    echo "   ✓"
fi

echo ""
echo "✅ Uninstall complete!"
echo ""
echo "To install the stable version:"
echo "  azd extension install $EXTENSION_ID"
