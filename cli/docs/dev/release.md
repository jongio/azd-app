# release process

## overview

releases are created through a manual two-step process:

1. **prepare release** - create a PR with version bump and changelog
2. **publish release** - build binaries and create github release

## step-by-step workflow

### 1. develop and merge features

**what you do:**
- create feature branches
- open PRs with descriptive titles
- use conventional commit format for commit messages (optional but recommended):
  - `feat: add new feature` - new functionality
  - `fix: resolve bug` - bug fixes
  - `docs: update readme` - documentation
  - `chore: update dependencies` - maintenance

**what happens:**
- ci runs on your PR (preflight, tests, lint, build)
- branch protection requires all 6 checks to pass
- merge to main after approval and checks pass

**result:** changes are on main branch, no version changes yet

---

### 2. prepare release

**when:** you decide it's time to bundle accumulated changes into a release

**what you do:**
1. go to github actions
2. select "prepare release" workflow
3. click "run workflow"
4. choose release type:
   - **patch** (0.2.1 → 0.2.2) - bug fixes, small changes
   - **minor** (0.2.1 → 0.3.0) - new features, backward compatible
   - **major** (0.2.1 → 1.0.0) - breaking changes

**what happens:**
- workflow calculates next version number
- creates a new branch `release/x.y.z`
- updates:
  - `cli/extension.yaml` - version field
  - `.release-please-manifest.json` - version tracking
  - `cli/CHANGELOG.md` - adds entry with commits since last release
- creates PR with title "release x.y.z"
- ci automatically runs on the PR

**result:** PR ready for review with all version/changelog updates

---

### 3. review and merge release PR

**what you do:**
- review the PR to check:
  - version number is correct
  - changelog includes all important changes
  - ci checks pass (all 6 required)
- optionally edit changelog for clarity
- merge the PR after approval

**what happens:**
- version and changelog updates are merged to main
- git history now shows the new version in files

**result:** main branch has new version number, but no github release yet

---

### 4. publish release

**when:** immediately after merging the release PR (or whenever ready)

**what you do:**
1. go to github actions
2. select "release" workflow
3. click "run workflow"
4. enter the version number (e.g., `0.2.2` - same as the PR you just merged)

**what happens:**
- workflow checks out main branch
- creates git tag `v0.2.2`
- builds dashboard (npm ci, npm run build)
- runs goreleaser to:
  - build binaries for all platforms (linux/darwin/windows on amd64/arm64)
  - create checksums
  - create github release with tag `v0.2.2`
  - upload all binaries as release assets
- attempts to update registry.json (if azd available)
- pushes registry.json back to main

**result:** 
- github release created with binaries
- users can download/install the new version
- registry updated (if azd available)

---

## example timeline

```
monday: merge PR #15 "feat: add new command"
        → main has new feature, version still 0.2.1

tuesday: merge PR #16 "fix: resolve crash"
         → main has bug fix, version still 0.2.1

wednesday: decide to release
           
           1. run "prepare release" → choose "minor"
              → creates PR #17 "release 0.3.0"
              → changelog lists PRs #15 and #16
              → ci runs on PR #17
           
           2. review PR #17 → looks good → merge
              → main now shows version 0.3.0 in files
           
           3. run "release" workflow → enter "0.3.0"
              → creates tag v0.3.0
              → builds binaries
              → creates github release
              → users can now install v0.3.0
```

---

## troubleshooting

### ci fails on release PR

- fix the failing check
- commit fix to main
- close the release PR
- re-run "prepare release" workflow (it will include your fix)

### wrong version number

- close the release PR
- re-run "prepare release" with correct release type

### need to update changelog

- edit `cli/CHANGELOG.md` directly in the release PR
- commit changes to the PR branch
- merge when ready

### release workflow fails

- check the workflow logs for specific errors
- common issues:
  - goreleaser config error
  - missing dependencies
  - network issues
- fix the issue and re-run the workflow with same version

### need to republish release

1. delete the github release
2. delete the git tag: `git push --delete origin vX.Y.Z`
3. re-run the "release" workflow

---

## files involved

- `.release-please-manifest.json` - tracks current version
- `cli/extension.yaml` - extension metadata including version
- `cli/CHANGELOG.md` - release history
- `cli/.goreleaser.yml` - goreleaser build configuration
- `registry.json` - extension registry (auto-updated)
- `.github/workflows/release-please.yml` - prepare release workflow
- `.github/workflows/release.yml` - publish release workflow

---

## conventional commits (recommended)

using conventional commit format helps generate better changelogs:

- `feat: description` → features section
- `fix: description` → bug fixes section
- `docs: description` → documentation section
- `perf: description` → performance section
- `refactor: description` → code refactoring
- `test: description` → tests
- `chore: description` → maintenance

breaking changes: add `!` or `BREAKING CHANGE:` in commit body

---

## release checklist

before running "prepare release":
- [ ] all desired PRs merged to main
- [ ] main branch ci passing
- [ ] ready to create version bump

before merging release PR:
- [ ] version number correct
- [ ] changelog accurate and clear
- [ ] all ci checks passing
- [ ] PR approved

before running "release" workflow:
- [ ] release PR merged
- [ ] version number matches PR
- [ ] ready to publish to users
