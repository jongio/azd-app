# Release Process

This document describes the release process for azd-app CLI.

## Overview

We use a **semi-automated release process** that balances automation with manual control:

1. ✅ **CI validates every push to main** - Tests, linting, builds
2. 🚀 **Manual trigger creates draft release** - One command
3. 👀 **Human review before publishing** - Click one button
4. 🏷️ **Auto-tagging for hotfixes** - Tag created automatically

## Branching Strategy

### GitHub Flow (Simplified)

```
main (always deployable)
  ├── feature/add-new-command
  ├── fix/dashboard-bug
  └── hotfix/v1.2.1 (from tag cli-v1.2.0)
```

**Key Points:**
- **`main` branch**: Protected, always deployable
- **Feature branches**: Develop here, merge via PR when ready
- **Batching features**: Merge multiple features to main, then release when ready
- **Hotfixes**: Branch from release tag, fix, bump patch version, release

### Why Not GitFlow?

For a CLI tool, GitFlow's separate `develop` and `main` branches add unnecessary complexity. GitHub Flow is simpler and works well when you want to:
- Batch multiple features before releasing
- Keep main always deployable
- Use tags for version tracking

## Release Process

### Step 1: Prepare for Release

1. **Ensure main is ready:**
   ```powershell
   git checkout main
   git pull origin main
   ```

2. **Verify CI passes:**
   - Check that all tests pass: https://github.com/jongio/azd-app/actions

3. **Update CHANGELOG.md** (optional but recommended):
   ```markdown
   ## [1.2.3] - 2025-11-03
   
   ### Added
   - New feature X
   - New command Y
   
   ### Fixed
   - Bug Z
   
   ### Changed
   - Improved performance of ABC
   ```

### Step 2: Create Draft Release

**Option A: Using PowerShell Script (Recommended)**

```powershell
cd cli
.\release.ps1 -Version 1.2.3
```

The script will:
- Validate version format
- Check for existing tags
- Show you a summary
- Ask for confirmation
- Trigger the GitHub Actions workflow

**Option B: Using GitHub CLI Directly**

```powershell
gh workflow run release-draft.yml -f "version=1.2.3"
```

**Option C: Via GitHub UI**

1. Go to: https://github.com/jongio/azd-app/actions/workflows/release-draft.yml
2. Click "Run workflow"
3. Enter version (e.g., `1.2.3`)
4. Click "Run workflow"

### Step 3: Monitor Workflow

Watch the workflow progress:

```powershell
gh run watch
```

Or view in browser:
https://github.com/jongio/azd-app/actions

The workflow will:
- ✅ Update `version.txt`
- ✅ Build binaries for all platforms (Windows, Linux, macOS AMD64/ARM64)
- ✅ Create checksums
- ✅ Update `registry.json` with version, URLs, and checksums
- ✅ Commit and push `registry.json` changes
- ✅ Create git tag `cli-v1.2.3`
- ✅ Create **DRAFT** release with all assets

### Step 4: Review Draft Release

1. Go to: https://github.com/jongio/azd-app/releases
2. Find your draft release
3. Review:
   - Release notes (auto-generated from CHANGELOG.md)
   - Attached binaries (4 platform archives)
   - Checksums
   - registry.json

### Step 5: Publish Release

When ready, click **"Publish release"** in the GitHub UI.

This triggers:
- ✅ Release becomes public
- ✅ Tag is already created, so hotfixes can branch from it
- ✅ Registry.json is already updated in main (no PR needed)

### Step 6: Done!

That's it! The release is published and `registry.json` in the main branch already has the correct checksums and URLs for users to install the extension.

## Hotfix Process

If you need to fix a critical bug in a released version:

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

## Version Numbering

We use [Semantic Versioning](https://semver.org/):

- **Major** (1.0.0): Breaking changes
- **Minor** (1.1.0): New features, backward compatible
- **Patch** (1.1.1): Bug fixes, backward compatible

Examples:
- New command added: `1.2.0` → `1.3.0`
- Bug fix: `1.2.0` → `1.2.1`
- Breaking change: `1.2.0` → `2.0.0`

## FAQ

### Q: Can I delete a draft release?

**A:** Yes! Draft releases can be deleted without consequence. The tag will remain, so delete it too if you want to recreate:

```powershell
# Delete draft in GitHub UI, then:
git tag -d cli-v1.2.3
git push origin :refs/tags/cli-v1.2.3
```

### Q: What if I need to update a published release?

**A:** You can edit release notes, but **don't change binaries**. Instead, create a new patch version (e.g., 1.2.4).

### Q: How do I test before releasing?

**A:** CI builds on every push. You can also:
- Use `mage build` locally
- Test with `azd extension install ./cli` (local path)
- Create a draft, download artifacts, test, then publish or delete

### Q: Can I automate the publish step too?

**A:** Yes, but not recommended. Manual review prevents accidental releases. If you want full automation, change `draft: true` to `draft: false` in `release-draft.yml`.

### Q: How do I roll back a release?

**A:** You can't delete a published release (GitHub policy), but you can:
1. Mark it as pre-release in GitHub UI
2. Quickly release a fixed version
3. Update release notes with warnings

## Workflows Reference

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `ci.yml` | Push to main, PR | Run tests, lint, build |
| `release-draft.yml` | Manual dispatch | Create draft release with binaries |
| `release-publish.yml` | Release published | Update registry, post-release tasks |

## Best Practices

1. **Always run CI before releasing** - Don't release broken code
2. **Update CHANGELOG.md** - Makes release notes meaningful
3. **Use semantic versioning** - Be consistent
4. **Test drafts before publishing** - Download and verify binaries
5. **Tag hotfixes appropriately** - Use patch version bumps
6. **Keep main clean** - Only merge PR'd, reviewed code

## Troubleshooting

### Workflow Fails

Check the Actions tab for error logs. Common issues:
- Version already exists
- Build failures (fix in code, push to main, retry)
- Permission issues (check GitHub token)

### Tag Already Exists

Delete and recreate:
```powershell
git tag -d cli-v1.2.3
git push origin :refs/tags/cli-v1.2.3
# Then retry release
```

### Binary Size Too Large

GitHub has a 2GB release asset limit. If binaries grow:
- Use `-ldflags="-s -w"` (already in use)
- Consider UPX compression
- Split large assets

## Summary

**Simple workflow:**
1. Develop features → Merge to main via PR
2. When ready to release → `.\release.ps1 -Version X.Y.Z`
3. Review draft → Click "Publish"
4. Done! ✅

**For hotfixes:**
1. Branch from tag → Fix → Merge to main
2. Release patch version → Publish
3. Done! ✅
