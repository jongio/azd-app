# release process

## tl;dr - one-click release!

```bash
# just choose bump type and click:
github actions → release → patch/minor/major → run
```

**that's it!** everything is automated - zero manual steps.

---

## what happens automatically

when you run the release workflow:

1. ✅ calculate next version (patch: 0.4.2→0.4.3, minor: 0.4.2→0.5.0, major: 0.4.2→1.0.0)
2. ✅ update `cli/extension.yaml` with new version
3. ✅ extract commits since last release
4. ✅ update `cli/changelog.md` with new version section
5. ✅ commit version bump to main
6. ✅ build dashboard (npm ci && npm run build)
7. ✅ build binaries for 6 platforms (azd x build --all)
8. ✅ package into archives (azd x pack)
9. ✅ create git tag (e.g., v0.4.3)
10. ✅ create github release with changelog
11. ✅ upload all 6 platform binaries
12. ✅ update registry.json (azd x publish)
13. ✅ commit registry.json to main

**time**: ~3-5 minutes  
**manual steps**: 0

---

## quick start

### run release workflow

**github ui:**
1. go to actions tab
2. click "release" workflow
3. click "run workflow"
4. choose bump type:
   - **patch**: bug fixes (0.4.2 → 0.4.3)
   - **minor**: new features (0.4.2 → 0.5.0)
   - **major**: breaking changes (0.4.2 → 1.0.0)
5. click green "run workflow" button

**or via cli:**
```bash
gh workflow run release.yml -f bump=patch   # or minor, major
```

### verify release

```bash
# check the release
gh release view v0.4.3

# test installation
azd extension install jongio.azd.app --version 0.4.3

# verify it works
azd app version
```

---

## complete workflow details

### why azd x commands?

#### previous approach (goreleaser)
- ❌ required separate tool installation
- ❌ tag format mismatch with azd extension registry (`v0.4.3` vs `azd-ext-jongio-azd-app_0.4.3`)
- ❌ manual changelog extraction needed
- ❌ two separate packaging systems (goreleaser + azd)

#### new approach (azd x)
- ✅ native azd tooling - designed for azd extensions
- ✅ automatic tag creation and format handling
- ✅ auto-extracts release notes from changelog.md
- ✅ seamless integration with extension registry
- ✅ single toolchain for build, package, release, publish

---

## what each azd x command does

### `azd x build --all --cwd cli`

builds go binaries for all platforms:
- windows/amd64, windows/arm64
- darwin/amd64, darwin/arm64  
- linux/amd64, linux/arm64

uses environment variables:
- `extension_id=jongio.azd.app`
- `extension_version=0.4.3`

embeds version info during compilation and outputs to `cli/bin/`

### `azd x pack --cwd cli`

creates deployment archives:
- windows: `.zip` files
- macos/linux: `.tar.gz` files

each archive contains:
- binary for that platform
- `extension.yaml` metadata

outputs to: `~/.azd/registry/jongio.azd.app/0.4.3/`

### `azd x release`

creates the github release:
- creates git tag (e.g., `v0.4.3`)
- pushes tag to github
- parses `cli/changelog.md` for version `[0.4.3]` section
- creates github release with those notes
- uploads all packaged archives as release assets

### `azd x publish`

updates the extension registry:
- reads packaged artifacts from local cache
- calculates sha256 checksums
- updates `registry.json` with new version entry
- maps each platform to its github release url

---

## detailed workflow steps

### automatic version calculation

the workflow automatically calculates the next version:

```bash
# read current version from cli/extension.yaml
current=$(grep '^version:' cli/extension.yaml | awk '{print $2}')

# split into major.minor.patch
major=$(echo $current | cut -d. -f1)
minor=$(echo $current | cut -d. -f2)
patch=$(echo $current | cut -d. -f3)

# bump based on choice
if bump == "patch":
  patch = patch + 1
elif bump == "minor":
  minor = minor + 1
  patch = 0
elif bump == "major":
  major = major + 1
  minor = 0
  patch = 0

next_version = "$major.$minor.$patch"
```

### automatic changelog generation

the workflow generates changelog from git commits:

```bash
# get commits since last tag (or last version bump commit if no tag exists)
if git rev-parse "v0.4.2" >/dev/null 2>&1; then
  # tag exists, get commits since that tag
  git log v0.4.2..HEAD --pretty=format:"- %s (%h)" --no-merges
else
  # no tag exists, try to find the last version bump commit
  LAST_VERSION_COMMIT=$(git log --grep="^chore: bump version to" --format="%H" -n 1)
  if [ -n "$LAST_VERSION_COMMIT" ]; then
    # get commits since the last version bump (excluding it)
    git log ${LAST_VERSION_COMMIT}..HEAD --pretty=format:"- %s (%h)" --no-merges
  else
    # this is likely the first release, get all commits
    git log --pretty=format:"- %s (%h)" --no-merges
  fi
fi

# output example:
- feat: add new command (abc123)
- fix: resolve crash (def456)
- docs: update readme (789ghi)

# prepend to cli/changelog.md
cat > cli/changelog.md <<eof
## [0.4.3] - 2025-11-06

- feat: add new command (abc123)
- fix: resolve crash (def456)
- docs: update readme (789ghi)

$(cat cli/changelog.md)
eof

# after committing the version bump, create and push a git tag
git tag -a "v0.4.3" -m "Release version 0.4.3"
git push origin "v0.4.3"
```

**note:** the workflow now creates git tags after each release. this ensures future releases can accurately determine which commits to include by using `git log v<previous>..HEAD`. if no tag exists (e.g., for older releases before this fix), the workflow falls back to finding the last "chore: bump version to" commit instead.

### automatic file updates

workflow updates these files automatically:

```yaml
# cli/extension.yaml
version: 0.4.3  # auto-updated

# cli/changelog.md
## [0.4.3] - 2025-11-06  # auto-generated

- feat: add new command (abc123)
- fix: resolve crash (def456)
```

then commits:
```bash
git add cli/extension.yaml cli/changelog.md
git commit -m "chore: bump version to 0.4.3"
git push origin main
```

### build process

```bash
# 1. install dependencies
curl -fssl https://aka.ms/install-azd.sh | bash
azd extension install microsoft.azd.extensions

# 2. build dashboard
cd cli/dashboard
npm ci
npm run build
# outputs to cli/src/internal/dashboard/dist

# 3. build extension binaries
cd ..
export extension_id="jongio.azd.app"
export extension_version="0.4.3"
azd x build --all --cwd cli
# creates 6 binaries in cli/bin/
```

### packaging process

```bash
azd x pack --cwd cli

# creates archives in ~/.azd/registry/jongio.azd.app/0.4.3/:
# - jongio-azd-app-windows-amd64.zip
# - jongio-azd-app-windows-arm64.zip
# - jongio-azd-app-darwin-amd64.tar.gz
# - jongio-azd-app-darwin-arm64.tar.gz
# - jongio-azd-app-linux-amd64.tar.gz
# - jongio-azd-app-linux-arm64.tar.gz
```

### release creation

```bash
azd x release \
  --cwd cli \
  --repo "jongio/azd-app" \
  --version "0.4.3" \
  --confirm

# automatic operations:
# - creates git tag: v0.4.3
# - pushes tag to github
# - extracts release notes from cli/changelog.md [0.4.3] section
# - creates github release
# - uploads all 6 platform archives as assets
```

### registry update

```bash
azd x publish \
  --cwd cli \
  --registry ../registry.json \
  --version "0.4.3"

# local mode (no --repo flag):
# - reads packaged artifacts from ~/.azd/registry/jongio.azd.app/0.4.3/
# - calculates sha256 checksums
# - updates registry.json with new version entry

# then commits:
git add registry.json
git commit -m "chore: update registry.json for v0.4.3"
git push
```

---

## workflow comparison

| task | old (goreleaser) | new (azd x) |
|------|------------------|-------------|
| install tools | goreleaser + azd | azd only |
| build binaries | `goreleaser build` | `azd x build --all` |
| package archives | goreleaser | `azd x pack` |
| create git tag | manual `git tag` | `azd x release` |
| extract changelog | manual script | `azd x release` (automatic) |
| create github release | `goreleaser release` | `azd x release` |
| upload artifacts | goreleaser | `azd x release` |
| update registry | `azd x publish --repo` | `azd x publish` (local) |
| tag format | `v0.4.3` | compatible with azd |
| version bumping | release-please | bash script in workflow |
| changelog generation | release-please | git log in workflow |

---

## local testing

### test build

```bash
cd cli
export extension_id="jongio.azd.app"
export extension_version="0.4.3-test"

# build dashboard first
cd dashboard
npm ci
npm run build
cd ..

# build binaries
azd x build --all

# verify
ls bin/
# should see 6 binaries
```

### test package

```bash
cd cli
azd x pack

# verify packages
ls ~/.azd/registry/jongio.azd.app/0.4.3-test/
# should see 6 archives (.zip and .tar.gz)
```

### test registry update (safe - uses test version)

```bash
cd cli
azd x publish --registry ../registry.json --version "0.4.3-test"

# check registry
cat ../registry.json | jq '.extensions."jongio.azd.app".versions[] | select(.version=="0.4.3-test")'
```

### test release (dry run)

```bash
cd cli
azd x release --version "0.4.3-test" --repo "jongio/azd-app" --draft

# creates draft github release (can be deleted after testing)
# verify on github then delete:
gh release delete v0.4.3-test
```

---

## example timeline

```
monday: merge pr #15 "feat: add new command"
        → main has new feature

tuesday: merge pr #16 "fix: resolve crash"
         → main has bug fix

wednesday: time to release!
           
           1. run "release" workflow → choose "minor"
              → workflow automatically:
                 - bumps version 0.4.2 → 0.5.0
                 - updates cli/extension.yaml
                 - generates changelog from commits
                 - updates cli/changelog.md
                 - commits version bump to main
                 - builds dashboard
                 - builds 6 platform binaries
                 - packages into archives
                 - creates tag v0.5.0
                 - creates github release
                 - uploads binaries
                 - updates registry.json
                 - commits registry.json
           
           2. verify release
              → gh release view v0.5.0
              → azd extension install jongio.azd.app --version 0.5.0
              → users can now install v0.5.0
```

---

## troubleshooting

### workflow fails on version bump

**symptoms:** "git push failed" or "nothing to commit"

```bash
# check if version was already bumped
git log --oneline -5
# look for recent "chore: bump version" commit

# if already bumped, just re-run with different bump type
# or manually edit version and re-run
```

### `azd x build` fails

**symptoms:** compilation errors or missing dashboard

```bash
# fix 1: build dashboard first
cd cli/dashboard
npm ci
npm run build

# fix 2: check go version
go version  # should be 1.23+

# fix 3: clear cache
rm -rf cli/bin/

# fix 4: check environment variables
echo $extension_id
echo $extension_version
```

### `azd x pack` can't find binaries

**symptoms:** "no binaries found in bin/"

```bash
# run build first
azd x build --all --cwd cli

# check binaries exist
ls cli/bin/
# should see 6 files
```

### `azd x release` fails to find changelog.md

**symptoms:** release created without notes

```bash
# ensure changelog.md exists with correct format
cat cli/changelog.md | head -20

# should have format:
## [0.4.3] - 2025-11-06

- changes here
```

### `azd x publish` registry.json issues

**symptoms:** invalid json or missing artifacts

```bash
# validate registry.json
cat registry.json | jq .

# check artifacts were packaged
ls ~/.azd/registry/jongio.azd.app/0.4.3/

# verify checksums
cat registry.json | jq '.extensions."jongio.azd.app".versions[] | select(.version=="0.4.3")'
```

### tag already exists

**symptoms:** "tag already exists" error

```bash
# delete local tag
git tag -d v0.4.3

# delete remote tag
git push --delete origin v0.4.3

# delete github release
gh release delete v0.4.3 --yes

# re-run workflow
```

### release workflow fails

**symptoms:** various errors in workflow logs

```bash
# check workflow logs
gh run list --workflow=release.yml
gh run view <run-id>

# common issues:
# - goreleaser config error (not used anymore)
# - missing dependencies (npm, go, azd)
# - network issues (github api rate limit)
# - permission issues (github token)

# fix the issue and re-run workflow
gh workflow run release.yml -f bump=patch
```

### need to republish release

```bash
# 1. delete the github release
gh release delete v0.4.3 --yes

# 2. delete the git tag
git push --delete origin v0.4.3

# 3. re-run the "release" workflow
gh workflow run release.yml -f bump=patch
```

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

breaking changes: add `!` or `breaking change:` in commit body

examples:
```bash
git commit -m "feat: add new command for deployments"
git commit -m "fix: resolve crash when parsing yaml"
git commit -m "docs: update readme with new examples"
git commit -m "feat!: change config file format (breaking)"
```

---

## release checklist

### before running release workflow
- [ ] all desired prs merged to main
- [ ] main branch ci passing
- [ ] ready to create version bump
- [ ] decided on bump type (patch/minor/major)

### during workflow execution
- [ ] version calculated correctly
- [ ] changelog generated from commits
- [ ] all files updated properly
- [ ] version bump committed to main

### after workflow completes
- [ ] github release created successfully
- [ ] all 6 platform binaries attached
- [ ] changelog shows in release notes
- [ ] registry.json updated on main
- [ ] test installation works

### verification commands
```bash
# check release
gh release view v0.4.3

# check binaries
gh release download v0.4.3 --dir /tmp/test-release
ls /tmp/test-release

# test install
azd extension install jongio.azd.app --version 0.4.3

# verify version
azd app version

# check registry
cat registry.json | jq '.extensions."jongio.azd.app"'
```

---

## files reference

### workflow files
- `.github/workflows/release.yml` - automated release workflow (azd x only)
- `.github/workflows/ci.yml` - pr checks (not used for releases)

### version files (auto-updated)
- `cli/extension.yaml` - extension metadata including version
- `cli/changelog.md` - release history with version sections

### registry
- `registry.json` - extension registry (auto-updated by azd x publish)

### removed files (no longer needed)
- `cli/.goreleaser.yml` - removed (replaced by azd x pack)
- `cli/scripts/test-release.ps1` - removed (not needed)
- `.github/workflows/release-please.yml` - removed (integrated into release.yml)
- `.github/workflows/release-simple.yml` - removed (duplicate)
- `.release-please-manifest.json` - removed (version now in extension.yaml)
- `release-please-config.json` - removed (not using release-please)

### generated files (temporary)
- `cli/bin/*` - built binaries (gitignored)
- `~/.azd/registry/jongio.azd.app/*/` - packaged archives

---

## key benefits

### 1. zero manual steps
```yaml
# no manual version editing
# no manual changelog updates
# no manual file commits
# just click and release!
```

### 2. automatic changelog handling
```yaml
# old: need to extract changelog section manually
- name: extract release notes
  run: |
    sed -n '/^## \[0.4.3\]/,/^## \[/p' changelog.md > notes.md

# new: azd x release does it automatically
- name: create release
  run: azd x release --version "0.4.3"
  # ✅ automatically finds and uses cli/changelog.md
```

### 3. no tag format issues
```yaml
# old: goreleaser creates v0.4.3, but azd x publish expects azd-ext-jongio-azd-app_0.4.3
# solution: use local mode to avoid tag lookup

# new: azd x release creates correct tag format
# azd x publish in local mode doesn't need tag lookup at all
```

### 4. simplified configuration
```yaml
# old: maintain .goreleaser.yml with complex build matrix
# new: azd x knows how to build extensions natively
```

### 5. single source of truth
```yaml
# everything driven by cli/extension.yaml:
id: jongio.azd.app
version: 0.4.3
language: go
# azd x commands automatically respect this
```

---

## complete release example

```bash
# week 1-2: feature development
git checkout -b feat/new-command
# ... code changes ...
git commit -m "feat: add deployment command"
gh pr create
# ci runs, tests pass, merge pr

# week 3: bug fixes
git checkout -b fix/bug-123
# ... bug fix ...
git commit -m "fix: resolve crash in logs"
gh pr create
# ci runs, tests pass, merge pr

# week 4: time to release!

# step 1: trigger release workflow
# github ui → actions → "release" → run → minor
gh workflow run release.yml -f bump=minor

# wait ~3 minutes for workflow to:
# 1. bump version 0.4.2 → 0.5.0
# 2. update extension.yaml and changelog.md
# 3. commit version bump
# 4. build dashboard
# 5. build 6 binaries
# 6. package archives
# 7. create tag v0.5.0
# 8. create github release
# 9. upload binaries
# 10. update registry.json
# 11. commit registry

# step 2: verify
gh release view v0.5.0
# ✅ release v0.5.0 published with binaries
# ✅ registry.json updated
# ✅ users can now: azd extension install jongio.azd.app --version 0.5.0

# test install
azd extension install jongio.azd.app --version 0.5.0
azd app version  # shows: azd app extension version 0.5.0
```

---

## summary

### what you need to know

1. **one workflow**: just run "release" and choose bump type
2. **automatic everything**: version, changelog, build, release, registry
3. **zero manual steps**: no file editing needed
4. **native tooling**: uses azd x commands only
5. **fast**: ~3-5 minutes from click to release

### what you don't need

- ❌ goreleaser
- ❌ release-please
- ❌ manual version editing
- ❌ manual changelog editing
- ❌ manual tagging
- ❌ manual registry updates
- ❌ complex scripts

### tools used

- ✅ `azd x build` - build binaries
- ✅ `azd x pack` - package archives
- ✅ `azd x release` - create github release
- ✅ `azd x publish` - update registry

### that's it! 🎉

**just click and release!**

---

## recent fixes (november 2025)

### release notes accuracy fix

**problem**: release notes were showing the full changelog instead of only commits for that specific version.

**root causes**:
1. no git tags were being created during releases
2. commit extraction always failed to find tags, falling back to "last 10 commits"
3. each changelog version section accumulated the full commit history
4. release notes extraction had a bug where the awk pattern matched both start and end on the same line

**fix applied**:
1. **git tag creation**: workflow now creates and pushes annotated tags (`v0.4.3`) after each version bump
   - future releases can use `git log v<previous>..HEAD` to get accurate commit ranges
   - fallback logic uses last "chore: bump version to" commit if tag doesn't exist
   
2. **improved commit extraction**: three-tier approach
   ```bash
   # tier 1: use version tag if it exists (preferred)
   git log v0.4.2..HEAD
   
   # tier 2: use last version bump commit if no tag
   git log <last-version-commit>..HEAD
   
   # tier 3: use all commits (first release only)
   git log --pretty=format:"- %s (%h)" --no-merges
   ```

3. **fixed release notes extraction**: replaced broken awk range pattern with proper state machine
   ```bash
   # old (broken): matched start and end on same line
   awk "/^## \[$VERSION\]/,/^## \[/" CHANGELOG.md
   
   # new (fixed): proper state tracking
   awk 'BEGIN { in_section = 0 }
        /^## \['$VERSION'\]/ { in_section = 1; next }
        /^## \[/ { in_section = 0 }
        in_section && NF > 0 { print }'
   ```

**result**: future releases will have accurate, version-specific release notes showing only commits since the previous version.

**migration**: no action needed. the fix applies automatically to all future releases. existing malformed changelog entries will remain as-is but won't affect future releases.
