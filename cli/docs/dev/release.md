# release process

## workflow overview

releases are created through a manual two-step process:

1. **prepare release** - create a PR with version bump and changelog
2. **publish release** - build binaries and create github release

## complete workflow

### 1. user creates new branch
**user does:**
```bash
git checkout -b feature/my-feature
```

**what happens:**
- nothing automated, just local git operation

---

### 2. user modifies files in that branch
**user does:**
```bash
# make changes
git add .
git commit -m "feat: add new command"
git push origin feature/my-feature
```

**what happens:**
- nothing automated, just commits to feature branch
- no workflows run on feature branches

---

### 3. user creates PR to main
**user does:**
```bash
gh pr create --title "Add new command" --body "Description"
```
or uses github web ui

**what happens:**
- **ci workflow triggers** (`ci.yml`)
  - runs preflight checks
  - runs tests on ubuntu/windows/macos
  - runs linting
  - runs build
- all 6 checks must pass (branch protection)
- codecov checks run (not required)

---

### 4. PR is merged
**user does:**
```bash
gh pr merge 123
```
or clicks "squash and merge" on github

**what happens:**
- changes merge to main branch
- **ci workflow runs again** on main (push event)
- **no version changes**
- **no release created**
- **no pr created**

---

### 5. admin wants to do a release

#### step 5a: prepare release
**admin does:**
```bash
gh workflow run release-please.yml -f release-type=minor
```
or: github ui → actions → "prepare release" → run workflow → choose patch/minor/major

**what happens:**
- **prepare release workflow runs** (`release-please.yml`)
  - reads current version from `.release-please-manifest.json` (e.g., 0.2.1)
  - calculates next version based on choice:
    - patch: 0.2.1 → 0.2.2
    - minor: 0.2.1 → 0.3.0
    - major: 0.2.1 → 1.0.0
  - gets all commits since last release tag (e.g., `v0.2.1..HEAD`)
  - creates new branch `release/0.3.0`
  - updates files:
    - `cli/extension.yaml` - sets version to 0.3.0
    - `.release-please-manifest.json` - sets cli to "0.3.0"
    - `cli/CHANGELOG.md` - adds section with commits
  - creates PR with title "release 0.3.0"
  - **ci workflow triggers** on the release PR
    - runs all 6 checks (preflight, tests, lint, build)

---

#### step 5b: review and merge release PR
**admin does:**
1. review PR (check version, changelog)
2. wait for ci to pass
3. merge PR:
```bash
gh pr merge <pr-number> --squash
```

**what happens:**
- release PR merges to main
- **ci workflow runs** on main (push event)
- version files now updated on main branch
- **still no github release created**
- **no binaries built yet**

---

#### step 5c: publish release
**admin does:**
```bash
gh workflow run release.yml -f version=0.3.0
```
or: github ui → actions → "release" → run workflow → enter version number

**what happens:**
- **release workflow runs** (`release.yml`)
  - checks out main branch
  - creates git tag `v0.3.0`
  - pushes tag to github
  - builds dashboard:
    - `npm ci` in cli/dashboard
    - `npm run build`
  - runs goreleaser:
    - builds binaries for 6 platforms (linux/darwin/windows on amd64/arm64)
    - creates checksums
    - creates github release with tag `v0.3.0`
    - uploads all binaries as release assets
    - uses changelog from `cli/CHANGELOG.md` for release notes
  - attempts to update `registry.json` (if azd available)
  - commits and pushes registry.json to main

**result:**
- github release v0.3.0 exists with binaries
- users can download/install the release
- registry updated

---

## summary table

| step | user action | workflow | creates pr? | creates release? |
|------|-------------|----------|-------------|------------------|
| 1-2 | create branch, commit | none | no | no |
| 3 | create pr | ci runs | no | no |
| 4 | merge pr | ci runs | no | no |
| 5a | run "prepare release" | prepare release + ci | yes (release pr) | no |
| 5b | merge release pr | ci runs | no | no |
| 5c | run "release" | release | no | yes (with binaries) |

**key insight:** merging regular prs does nothing special. only the admin manually triggers release preparation and publishing.

---

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
