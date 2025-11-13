# PR Build Installation Scripts

Automated scripts to install and test PR builds of the azd app extension.

## Scripts

### `install-pr.ps1` / `install-pr.sh`
Installs a specific PR build.

**Usage:**
```powershell
# PowerShell
.\install-pr.ps1 -PrNumber 123 -Version 0.5.7-pr123

# Or one-liner
iex "& { $(irm https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/install-pr.ps1) } -PrNumber 123 -Version 0.5.7-pr123"
```

```bash
# Bash
./install-pr.sh 123 0.5.7-pr123

# Or one-liner
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/install-pr.sh | bash -s 123 0.5.7-pr123
```

**What it does:**
1. Enables azd extensions
2. Uninstalls existing extension (if any)
3. Downloads PR registry from GitHub release
4. Adds PR registry as source
5. Installs specified version
6. Verifies installation

### `restore-stable.ps1` / `restore-stable.sh`
Restores the stable version of the extension.

**Usage:**
```powershell
# PowerShell
.\restore-stable.ps1

# Or one-liner
iex "& { $(irm https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/restore-stable.ps1) }"
```

```bash
# Bash
./restore-stable.sh

# Or one-liner
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/restore-stable.sh | bash
```

**What it does:**
1. Uninstalls current extension
2. Removes all PR registry sources
3. Cleans up pr-registry.json files
4. Adds stable registry source
5. Installs latest stable version
6. Verifies installation

## Security

These scripts:
- ✅ Only interact with official GitHub releases
- ✅ Don't require or use any credentials
- ✅ Don't modify system files outside azd directories
- ✅ Can be reviewed before running (they're in this repo)

**Recommended:** Review the script content before running with `iex` or piping to bash.

## Examples

### Test a PR build

```bash
# Get PR number and version from PR comment
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/install-pr.sh | bash -s 123 0.5.7-pr123

# Test it
azd app version
azd app reqs
azd app run

# Done testing
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/restore-stable.sh | bash
```

### Switch between PR builds

```bash
# Test PR #123
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/install-pr.sh | bash -s 123 0.5.7-pr123

# Switch to PR #456
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/install-pr.sh | bash -s 456 0.5.7-pr456

# Back to stable
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/restore-stable.sh | bash
```

## Troubleshooting

### Script fails to download registry

**Error:** `Failed to download registry from ...`

**Cause:** The PR build doesn't exist yet or has been deleted.

**Fix:** 
- Wait for the PR build workflow to complete
- Check the PR comment for the correct version
- Verify the PR is still open (builds are deleted when PR closes)

### Wrong version installed

**Error:** `Version mismatch - expected X.X.X-prNNN`

**Cause:** An older version might be cached.

**Fix:**
```bash
azd extension uninstall jongio.azd.app
rm -rf ~/.azd/extensions/jongio.azd.app
# Run install script again
```

### Permission denied (bash)

**Error:** `Permission denied: ./install-pr.sh`

**Fix:**
```bash
chmod +x install-pr.sh
./install-pr.sh 123 0.5.7-pr123
```

Or use the curl one-liner which doesn't require local file permissions.

## Manual Installation

If you prefer not to use scripts, see [testing-pr-builds.md](../cli/docs/dev/testing-pr-builds.md) for manual installation instructions.

## Documentation

- [Testing PR Builds Guide](../cli/docs/dev/testing-pr-builds.md)
- [PR Build Quick Reference](../cli/docs/dev/pr-build-quick-ref.md)
- [PR Install Scripts Overview](../cli/docs/dev/pr-install-scripts.md)
