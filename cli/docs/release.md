# Release Process

This guide covers how to release new versions of the azd-app CLI extension.

---

## Quick Start (Automated)

```bash
# 1. Create feature branch and make changes
git checkout -b feat/dashboard-filtering
git commit -m "feat: add dashboard filtering"
git push origin feat/dashboard-filtering

# 2. Create PR and merge to main
gh pr create --fill
# After approval, "Squash and merge" with: "feat: add dashboard filtering"

# 3. Release-Please creates a release PR automatically with:
#    - Auto-generated changelog
#    - Version bump in extension.yaml
#    - All changes since last release

# 4. Review and merge the "chore: release" PR
#    https://github.com/jongio/azd-app/pulls

# 5. Release is created automatically and binaries are published!
```

That's it! 🎉

**See [Conventional Commits Guide](../../docs/conventional-commits.md) for commit message format.**

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

We use a **fully automated release process** powered by [Release-Please](https://github.com/googleapis/release-please):

1. ✅ **Write conventional commits** - `feat:`, `fix:`, etc.
2. 🤖 **Release-Please creates PR** - Auto-generated changelog + version bump
3. 👀 **Review and merge** - Check the changes look good
4. 🚀 **Release published automatically** - Binaries built and published

**Key Points:**
- No manual changelog editing required
- Version bumps determined from commit messages
- `main` branch is always deployable
- Everything triggered by merging the release PR
- Built with GoReleaser + `azd x` commands

---

## Standard Release

### Step 1: Develop on a Feature Branch

```bash
# Create a feature branch
git checkout -b feat/add-dashboard-filtering

# Make your changes and commit using conventional format
git add .
git commit -m "feat: add dashboard filtering"

# Push and create a PR
git push origin feat/add-dashboard-filtering
gh pr create --fill
```

### Step 2: Merge PR to Main

```bash
# After approval, merge the PR
# Use "Squash and merge" with a conventional commit message:
# - feat: add dashboard filtering (minor bump)
# - fix: resolve Windows bug (patch bump)  
# - feat!: redesign API (major bump with BREAKING CHANGE)
```

**Important:** The commit message when merging to `main` is what Release-Please uses.

**See [Conventional Commits Guide](../../docs/conventional-commits.md) for details.**

### Step 3: Release-Please Creates a Release PR

After merging to `main`, Release-Please automatically:
- ✅ Analyzes your commits since last release
- ✅ Determines version bump (`feat` = minor, `fix` = patch, `!` = major)
- ✅ Generates changelog from commit messages
- ✅ Updates `extension.yaml` version
- ✅ Creates a "chore: release X.Y.Z" PR

### Step 4: Review the Release PR

1. Go to: https://github.com/jongio/azd-app/pulls
2. Find the "chore: release X.Y.Z" PR
3. Review the auto-generated changelog
4. Check the version bump is correct
5. Add any manual notes if needed

### Step 5: Merge the Release PR

Click **"Merge pull request"** - that's it!

### Step 6: Automatic Build and Publish

Once merged, the workflow automatically:
- ✅ Builds binaries with GoReleaser (all platforms)
- ✅ Calculates SHA256 checksums
- ✅ Creates GitHub release with `azd x release`
- ✅ Updates `registry.json` with `azd x publish`
- ✅ Commits registry changes to main

Done! Users can install the new version immediately.

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

```bash
git checkout main
git merge hotfix/v1.2.4
git push origin main
```

### 4. Release-Please Handles the Rest

After merging to main:
- Release-Please will create a release PR
- Review and merge it
- Release is published automatically

### 5. Clean Up

Delete hotfix branch: `git branch -d hotfix/v1.2.4`

---

## Version Numbering

We use [Semantic Versioning](https://semver.org/), determined automatically from commit messages:

| Commit Type | Example | Version Bump | Changelog Section |
|-------------|---------|--------------|-------------------|
| `feat:` | `feat: add logs command` | Minor (0.1.0 → 0.2.0) | ✅ Features |
| `fix:` | `fix: resolve crash` | Patch (0.1.0 → 0.1.1) | ✅ Bug Fixes |
| `feat!:` or `BREAKING CHANGE:` | `feat!: redesign API` | Major (0.1.0 → 1.0.0) | ✅ BREAKING CHANGES |
| `perf:` | `perf: optimize queries` | Patch | ✅ Performance |
| `docs:` | `docs: update README` | Patch | ✅ Documentation |
| `chore:`, `test:`, `refactor:` | `chore: update deps` | Patch | ❌ (hidden) |

**Release-Please automatically determines the version based on your commits since the last release.**

---

## FAQ

### Do I need to manually edit CHANGELOG.md?

**No!** Release-Please generates it automatically from your commit messages. Just use conventional commits format.

### Can I edit the changelog before release?

**Yes!** Edit the CHANGELOG.md in the release PR before merging. Your changes will be preserved.

### What if I make a typo in a commit message?

Before the release PR is created, you can amend commits. After the PR is created, you can edit CHANGELOG.md directly in the PR.

### Can I skip a release?

If Release-Please creates a PR but you don't want to release yet, just leave it open. It will accumulate more changes as you push more commits.

### How do I trigger a major version (1.0.0)?

Use `feat!:` or add `BREAKING CHANGE:` in the commit footer:

```bash
git commit -m "feat!: redesign CLI interface

BREAKING CHANGE: command structure has changed"
```

---

## Workflows Reference

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `ci.yml` | Push to main, PRs | Run tests, lint, build |
| `release-please.yml` | Push to main | Create release PR, build and publish on merge |
## Best Practices

1. ✅ **Work on feature branches** - Never commit directly to `main`
2. ✅ **Use conventional commits** - Enables automatic changelog generation
3. ✅ **Squash merge PRs** - Keep main history clean with one commit per feature
4. ✅ **Write descriptive PR merge messages** - They become your changelog
5. ✅ **Review release PRs carefully** - This is your last chance to edit
6. ✅ **Let CI validate everything** - Don't merge failing builds

**See [Conventional Commits Guide](../../docs/conventional-commits.md) for commit message examples.**
4. ✅ **Review release PRs carefully** - This is your last chance to edit
5. ✅ **Let CI validate everything** - Don't merge failing builds

**See [Conventional Commits Guide](../../docs/conventional-commits.md) for commit message examples.**

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
