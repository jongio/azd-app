# Quick Release Guide

## Prerequisites

1. Ensure you're on the `main` branch with latest changes
2. All tests must pass
3. Update version and changelog

## Steps

### 1. Update Version Files

```powershell
# Update version number
Set-Content cli/version.txt "0.1.0"

# Update CHANGELOG.md with release notes
code cli/CHANGELOG.md
```

### 2. Test Locally

```powershell
cd cli

# Run tests
go test ./...

# Build locally
mage build

# Test installation
./install-local.ps1
azd app version
```

### 3. Create Release Tag

```powershell
# Set version (without 'v' prefix in variable, but with 'cli-v' in tag)
$VERSION = "0.1.0"

# Create and push tag
git add cli/version.txt cli/CHANGELOG.md
git commit -m "Prepare release v$VERSION"
git push origin main

git tag "cli-v$VERSION"
git push origin "cli-v$VERSION"
```

### 4. Monitor Release Workflow

1. Go to: https://github.com/jongio/azd-app/actions
2. Watch the "Release" workflow
3. Wait for completion (~5-10 minutes)

### 5. Verify Release

1. Check release page: https://github.com/jongio/azd-app/releases/tag/cli-v$VERSION
2. Verify all artifacts are present:
   - ✅ app-windows-amd64.zip
   - ✅ app-linux-amd64.tar.gz
   - ✅ app-darwin-amd64.tar.gz
   - ✅ app-darwin-arm64.tar.gz
   - ✅ registry.json
   - ✅ checksums.txt
3. Download and test one artifact

### 6. Merge Registry Update PR

1. Check for auto-created PR titled "Update registry.json for cli-v$VERSION"
2. Review changes (should only update checksums and URLs)
3. Merge the PR

## Post-Release

- Announce the release
- Update any dependent projects
- Monitor for issues

## Rollback (if needed)

```powershell
# Delete the tag
git tag -d "cli-v$VERSION"
git push origin --delete "cli-v$VERSION"

# Delete the release from GitHub UI
# This won't automatically clean up registry.json - do that manually if needed
```

## Common Issues

### Tag already exists
```powershell
# Delete local and remote tag
git tag -d "cli-v$VERSION"
git push origin --delete "cli-v$VERSION"
# Then recreate
```

### Workflow doesn't trigger
- Verify tag format: Must be `cli-v*.*.*` (e.g., `cli-v0.1.0`)
- Check GitHub Actions are enabled
- Review workflow file for syntax errors

### Build fails
- Check Go version (requires 1.22)
- Verify all dependencies in go.mod
- Test build locally first

### Checksums missing or wrong
- The workflow automatically calculates checksums
- If wrong, the issue is in the release.yml workflow
- Check the "Calculate checksums" step output
