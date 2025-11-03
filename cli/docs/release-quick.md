# Quick Release Guide

## TL;DR - Release in 3 Steps

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

## Commands Reference

### Create Release
```powershell
.\release.ps1 -Version 1.2.3          # Create draft release
.\release.ps1 -Version 1.2.3 -DryRun  # Preview without executing
```

### Monitor
```powershell
gh run watch                          # Watch current workflow
gh run list --workflow=release-draft.yml --limit 5  # Recent runs
```

### Hotfix
```powershell
git checkout cli-v1.2.3              # Start from release tag
git checkout -b hotfix/v1.2.4        # Create hotfix branch
# ... make fixes ...
git checkout main
git merge hotfix/v1.2.4              # Merge to main
.\release.ps1 -Version 1.2.4         # Release hotfix
```

## Version Bumping Guide

| Change Type | Old → New | Example |
|-------------|-----------|---------|
| Bug fix | 1.2.3 → 1.2.4 | Fixed crash on startup |
| New feature | 1.2.3 → 1.3.0 | Added `deps` command |
| Breaking change | 1.2.3 → 2.0.0 | Changed CLI interface |

## Pre-Release Checklist

- [ ] All features merged to `main`
- [ ] CI passing on `main`
- [ ] CHANGELOG.md updated (optional)
- [ ] Version number decided (semver)
- [ ] No uncommitted changes

## Full Documentation

See [release-process.md](./release-process.md) for complete details.
