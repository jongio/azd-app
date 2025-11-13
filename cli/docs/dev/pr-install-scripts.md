# PR Build Install Scripts

Quick one-line scripts to install and test PR builds.

## Install PR Build

### PowerShell (Windows)

```powershell
# Replace PR_NUMBER and VERSION with values from PR comment
iex "& { $(irm https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/install-pr.ps1) } -PrNumber PR_NUMBER -Version VERSION"
```

**Example:**
```powershell
iex "& { $(irm https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/install-pr.ps1) } -PrNumber 123 -Version 0.5.7-pr123"
```

### Bash (macOS/Linux)

```bash
# Replace PR_NUMBER and VERSION with values from PR comment
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/install-pr.sh | bash -s PR_NUMBER VERSION
```

**Example:**
```bash
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/install-pr.sh | bash -s 123 0.5.7-pr123
```

## Restore Stable Version

### PowerShell (Windows)

```powershell
iex "& { $(irm https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/restore-stable.ps1) }"
```

### Bash (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/restore-stable.sh | bash
```

## What These Scripts Do

### Install PR Build Script
1. Uninstalls existing `jongio.azd.app` extension
2. Downloads `pr-registry.json` for the specified PR
3. Adds the PR registry as a source
4. Installs the PR version
5. Verifies installation

### Restore Stable Script
1. Uninstalls PR version
2. Removes PR registry sources
3. Cleans up `pr-registry.json` files
4. Adds stable registry source
5. Installs latest stable version
6. Verifies installation

## Manual Installation (Without Scripts)

If you prefer not to run remote scripts, see [testing-pr-builds.md](./testing-pr-builds.md) for manual instructions.
