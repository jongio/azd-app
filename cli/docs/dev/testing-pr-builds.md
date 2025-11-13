# Testing PR Builds

Want to test changes from a pull request before they're merged? Follow this guide!

## Prerequisites

- [Azure Developer CLI (azd)](https://learn.microsoft.com/azure/developer/azure-developer-cli/install-azd) installed
- Extensions enabled: `azd config set alpha.extension.enabled on`

## When Builds Are Available

### For Trusted Contributors
If you're a repository collaborator, org member, or have contributed before, builds are generated automatically when you create a PR.

### For First-Time Contributors
A maintainer must review and approve your PR by adding the `safe-to-build` label before a build is generated.

### For Maintainers
You can manually trigger a build for any PR from the Actions tab.

## Quick Install (Recommended)

The easiest way to test a PR build is using the one-line install script from the PR comment.

### PowerShell (Windows)

```powershell
iex "& { $(irm https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/install-pr.ps1) } -PrNumber 123 -Version 0.5.7-pr123"
```

### Bash (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/install-pr.sh | bash -s 123 0.5.7-pr123
```

**What it does:**
1. Uninstalls existing extension
2. Downloads PR registry
3. Adds registry source
4. Installs PR version
5. Verifies installation

## Manual Installation

If you prefer not to run remote scripts:

### Step 1: Download the Registry

```bash
curl -L -o pr-registry.json https://github.com/jongio/azd-app/releases/download/pr-123-v0.5.7-pr123/pr-registry.json
```

### Step 2: Add Registry Source

```bash
azd extension source add -n pr-123 -t file -l "$(pwd)/pr-registry.json"
```

### Step 3: Install the Preview

```bash
azd extension install jongio.azd.app --version 0.5.7-pr123
```

### Step 4: Verify Installation

```bash
azd app version
# Should show: azd app extension version 0.5.7-pr123

azd app hi
azd app reqs
```

## Testing

Now you can test the changes! Try out the commands or features mentioned in the PR description.

## Quick Restore (Recommended)

When you're done testing, restore the stable version with one command:

### PowerShell (Windows)

```powershell
iex "& { $(irm https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/restore-stable.ps1) }"
```

### Bash (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/restore-stable.sh | bash
```

**What it does:**
1. Uninstalls PR version
2. Removes all PR registry sources
3. Cleans up PR registry files
4. Adds stable registry source
5. Installs latest stable version
6. Verifies installation

## Manual Cleanup

When you're done testing:

### Remove the Preview Build

```bash
azd extension uninstall jongio.azd.app
azd extension source remove pr-123  # Use your PR number
rm pr-registry.json
```

### Reinstall Stable Version

```bash
azd extension source add -n app -t url -l "https://raw.githubusercontent.com/jongio/azd-app/refs/heads/main/registry.json"
azd extension install jongio.azd.app
```

## Troubleshooting

### "Extension already installed"

```bash
azd extension uninstall jongio.azd.app
# Then try installing again
```

### "Registry not found"

Make sure you're in the same directory where you downloaded `pr-registry.json`, or use the full path:

```bash
azd extension source add -n pr-123 -t file -l "/full/path/to/pr-registry.json"
```

### Wrong Version Shows Up

```bash
# Uninstall and clear cache
azd extension uninstall jongio.azd.app
rm -rf ~/.azd/extensions/jongio.azd.app

# Try installing again
azd extension install jongio.azd.app --version 0.5.7-pr123
```

### Can't Download Registry File

Make sure you have access to the repository and the release exists. Check the PR comment for the correct download URL.

## Platform-Specific Notes

### Windows (PowerShell)

Use the same commands, but if you need to specify the current directory:

```powershell
azd extension source add -n pr-123 -t file -l "$PWD\pr-registry.json"
```

### macOS/Linux

The commands work as shown above. On some systems you may need to use `$PWD` instead of `$(pwd)`:

```bash
azd extension source add -n pr-123 -t file -l "$PWD/pr-registry.json"
```

## FAQ

**Q: Why isn't a build available for my PR?**  
A: If you're a first-time contributor, a maintainer needs to review and approve the build by adding the `safe-to-build` label.

**Q: How long are PR builds available?**  
A: PR builds are automatically deleted when the PR is closed or merged.

**Q: Can I have both stable and PR versions installed?**  
A: No, you can only have one version installed at a time. Uninstall one before installing the other.

**Q: What if the PR is updated with new commits?**  
A: The workflow runs again and updates the preview build. Uninstall and reinstall to get the latest changes.

**Q: Is this safe?**  
A: PR builds are from the same repository but from an unmerged branch. Review the PR changes before installing if you have concerns.

## Example: Complete Flow Using Scripts

```bash
# Test PR #123 (macOS/Linux)
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/install-pr.sh | bash -s 123 0.5.7-pr123

# Try it out
azd app version
azd app reqs
azd app run

# Done testing - restore stable
curl -fsSL https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/restore-stable.sh | bash
```

```powershell
# Test PR #123 (Windows)
iex "& { $(irm https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/install-pr.ps1) } -PrNumber 123 -Version 0.5.7-pr123"

# Try it out
azd app version
azd app reqs
azd app run

# Done testing - restore stable
iex "& { $(irm https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/restore-stable.ps1) }"
```

## Example: Complete Manual Flow

```bash
# 1. Download registry (from PR comment)
curl -L -o pr-registry.json https://github.com/jongio/azd-app/releases/download/pr-456-v0.5.7-pr456/pr-registry.json

# 2. Enable extensions (one-time)
azd config set alpha.extension.enabled on

# 3. Add registry
azd extension source add -n pr-456 -t file -l "$(pwd)/pr-registry.json"

# 4. Install
azd extension install jongio.azd.app --version 0.5.7-pr456

# 5. Test
azd app version
azd app reqs
azd app run

# 6. When done, cleanup
azd extension uninstall jongio.azd.app
azd extension source remove pr-456
rm pr-registry.json

# 7. Back to stable
azd extension source add -n app -t url -l "https://raw.githubusercontent.com/jongio/azd-app/refs/heads/main/registry.json"
azd extension install jongio.azd.app
```

---

**Need Help?** Post a comment on the PR or open an issue.
