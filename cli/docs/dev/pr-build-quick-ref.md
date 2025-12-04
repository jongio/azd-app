# PR Build Quick Reference

> **Copy-paste these commands to test PR builds**

## One-Line Install (Recommended)

### PowerShell (Windows)
```powershell
iex "& { $(irm https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/install-pr.ps1) } -PrNumber PR_NUM -Version VERSION"
```

### Bash (macOS/Linux)
```bash
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/install-pr.sh | bash -s PR_NUM VERSION
```

## Manual Installation

```bash
# 1. Download registry (replace URL from PR comment)
curl -L -o pr-registry.json https://github.com/jongio/azd-app/releases/download/PR_TAG/pr-registry.json

# 2. Enable extensions (one-time setup)
azd config set alpha.extension.enabled on

# 3. Add registry source (replace PR_NUM)
azd extension source add -n pr-PR_NUM -t file -l "$(pwd)/pr-registry.json"

# 4. Install (replace VERSION)
azd extension install jongio.azd.app --version VERSION

# 5. Verify
azd app version
```

## One-Line Restore (Recommended)

### PowerShell (Windows)
```powershell
iex "& { $(irm https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/restore-stable.ps1) }"
```

### Bash (macOS/Linux)
```bash
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/restore-stable.sh | bash
```

## Manual Cleanup

```bash
# Uninstall preview
azd extension uninstall jongio.azd.app

# Remove registry source (replace PR_NUM)
azd extension source remove pr-PR_NUM

# Delete downloaded file
rm pr-registry.json

# Reinstall stable
azd extension source add -n app -t url -l https://raw.githubusercontent.com/jongio/azd-app/main/registry.json
azd extension install jongio.azd.app
```

---

**Full guide:** [Testing PR Builds](./testing-pr-builds.md)  
**Scripts:** [PR Install Scripts](./pr-install-scripts.md)
