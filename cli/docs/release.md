# Release Process

This guide covers how to release new versions of the azd-app CLI extension.

---

## Quick Start

```powershell
# 1. Make sure you're on main and CI passes
git checkout main && git pull

# 2. Create draft release
cd cli
.\release.ps1 -Version 1.2.3

# 3. Review and publish at:
# https://github.com/jongio/azd-app/releases
```

That's it! 🎉

---

## Table of Contents

- [Overview](#overview)
- [Standard Release](#standard-release)
- [Hotfix Release](#hotfix-release)
- [Version Numbering](#version-numbering)
- [FAQ](#faq)
- [Workflows Reference](#workflows-reference)

---

## Overview

We use a **semi-automated release process**:

1. ✅ **CI validates every push** - Tests, linting, builds automatically
2. 🚀 **Manual trigger creates draft** - One command creates everything
3. 👀 **Human review before publishing** - Click "Publish release" when ready
4. 🤖 **Checksums calculated automatically** - No manual work needed

**Key Points:**
- `main` branch is always deployable
- Features are batched and released when ready
- Draft releases can be tested before publishing
- Checksums and registry.json are updated automatically

---

## Standard Release

### Step 1: Prepare

1. **Ensure main is ready:**
   ```powershell
   git checkout main
   git pull origin main
   ```

2. **Verify CI passes:**
   - Check: https://github.com/jongio/azd-app/actions

3. **Update CHANGELOG.md** (recommended):
   ```markdown
   ## [1.2.3] - 2025-11-03
   
   ### Added
   - New feature X
   
   ### Fixed
   - Bug Z
   ```

### Step 2: Create Draft Release

**Option A: PowerShell Script (Recommended)**

```powershell
cd cli
.\release.ps1 -Version 1.2.3
```

**Option B: GitHub CLI**

```powershell
gh workflow run release-draft.yml -f "version=1.2.3"
```

**Option C: GitHub UI**

1. Go to: https://github.com/jongio/azd-app/actions/workflows/release-draft.yml
2. Click "Run workflow"
3. Enter version: `1.2.3`
4. Click "Run workflow"

### Step 3: Monitor Progress

```powershell
gh run watch
```

The workflow automatically:
- ✅ Updates `version.txt`
- ✅ Builds binaries for all platforms
- ✅ Calculates SHA256 checksums
- ✅ Updates `registry.json` with version and checksums
- ✅ Commits changes to main
- ✅ Creates git tag `cli-v1.2.3`
- ✅ Creates draft release with all assets

### Step 4: Review Draft

1. Go to: https://github.com/jongio/azd-app/releases
2. Find your draft release
3. Review release notes and attached binaries
4. Optionally download and test binaries

### Step 5: Publish

Click **"Publish release"** in the GitHub UI.

Done! Users can now install the new version.

---

## Hotfix Release

For critical bugs in a released version:

### 1. Create Hotfix Branch from Tag

```powershell
# Example: Hotfix for v1.2.3
git checkout cli-v1.2.3
git checkout -b hotfix/v1.2.4
```

### 2. Make the Fix

```powershell
# Make your changes
git add .
git commit -m "fix: critical bug description"
```

### 3. Merge to Main

```powershell
git checkout main
git merge hotfix/v1.2.4
git push origin main
```

### 4. Release Hotfix

```powershell
cd cli
.\release.ps1 -Version 1.2.4
```

### 5. Publish and Clean Up

- Publish the draft release
- Delete hotfix branch: `git branch -d hotfix/v1.2.4`

---

## Version Numbering

We use [Semantic Versioning](https://semver.org/):

| Version Type | Format | When to Use | Example |
|--------------|--------|-------------|---------|
| **Patch** | 1.2.3 → 1.2.4 | Bug fixes only | Fixed crash on startup |
| **Minor** | 1.2.3 → 1.3.0 | New features, backward compatible | Added `logs` command |
| **Major** | 1.2.3 → 2.0.0 | Breaking changes | Changed CLI interface |

---

## FAQ

### Can I delete a draft release?

**Yes!** Draft releases can be deleted without consequence. Delete the tag too if recreating:

```powershell
# Delete draft in GitHub UI, then:
git tag -d cli-v1.2.3
git push origin :refs/tags/cli-v1.2.3
```

### What if I need to update a published release?

You can edit release notes, but **don't change binaries**. Instead, create a new patch version.

### How do I test before releasing?

- CI builds on every push to main
- Create a draft release, download binaries, test locally
- Install locally: `azd extension install ./cli`
- Delete the draft if issues found

### Can I automate the publish step?

Yes, but not recommended. Manual review prevents accidental releases. To enable, change `draft: true` to `draft: false` in `release-draft.yml`.

### How do I roll back a release?

You can't delete published releases, but you can:
1. Mark it as pre-release in GitHub UI
2. Release a fixed version immediately
3. Add warnings to release notes

### Do I need to manually update checksums?

**No!** The release workflow calculates SHA256 checksums and updates `registry.json` automatically.

---

## Workflows Reference

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `ci.yml` | Push to main, PRs | Run tests, lint, build |
| `release-draft.yml` | Manual dispatch | Create draft release with binaries |
| `release-publish.yml` | Release published | Post-release tasks |

---

## Best Practices

1. ✅ **Always verify CI passes** - Don't release broken code
2. ✅ **Update CHANGELOG.md** - Makes release notes meaningful  
3. ✅ **Use semantic versioning** - Be consistent with version bumps
4. ✅ **Test drafts before publishing** - Download and verify binaries
5. ✅ **Keep main clean** - Only merge reviewed PRs

---

## Troubleshooting

### Workflow Fails

Check the Actions tab for error logs. Common issues:

- **Version already exists** - Choose a different version number
- **Build failures** - Fix in code, push to main, retry workflow
- **Permission issues** - Check repository settings

### Need Help?

- Check workflow logs: https://github.com/jongio/azd-app/actions
- Review this guide thoroughly
- File an issue if you find a problem with the release process
