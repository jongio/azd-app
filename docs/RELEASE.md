# Release Guide

This guide explains how to publish a new release of the App Extension.

## Release Process

The release process is fully automated through GitHub Actions. Follow these steps:

### 1. Update Version Information

Before creating a release, update the following files:

#### extension.yaml
```yaml
version: 0.2.0  # Update to your new version
```

#### CHANGELOG.md
- Move items from `[Unreleased]` to a new version section
- Add release date
- Create new `[Unreleased]` section for future changes

Example:
```markdown
## [Unreleased]

## [0.2.0] - 2025-11-01

### Added
- New feature X
...
```

### 2. Commit and Push Changes

```bash
git add extension.yaml CHANGELOG.md
git commit -m "Prepare release v0.2.0"
git push origin main
```

### 3. Create and Push a Git Tag

```bash
# Create annotated tag
git tag -a v0.2.0 -m "Release v0.2.0"

# Push tag to GitHub
git push origin v0.2.0
```

### 4. Automated Release Process

Once the tag is pushed, GitHub Actions will automatically:

1. **Build binaries** for all platforms:
   - Windows AMD64
   - Linux AMD64
   - macOS AMD64
   - macOS ARM64

2. **Create platform packages**:
   - `app-windows-amd64.zip`
   - `app-linux-amd64.tar.gz`
   - `app-darwin-amd64.tar.gz`
   - `app-darwin-arm64.tar.gz`

3. **Calculate SHA256 checksums** for all packages

4. **Update registry.json** with:
   - New version number
   - Download URLs
   - SHA256 checksums

5. **Create GitHub Release** with:
   - Release notes from CHANGELOG
   - All platform packages
   - Updated registry.json
   - Checksums file

6. **Commit registry.json** back to the main branch

### 5. Verify Release

After the workflow completes:

1. Check the [Releases page](https://github.com/jongio/azd-app-extension/releases)
2. Verify all artifacts are attached
3. Verify checksums match
4. Test installation:
   ```bash
   # Add the registry
   azd config set extension.registry https://raw.githubusercontent.com/jongio/azd-app-extension/main/registry.json
   
   # Install the extension
   azd extension install app --version 0.2.0
   
   # Test it
   azd app hi
   ```

## Manual Release (Not Recommended)

If you need to create a release manually:

### Build Binaries

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/windows-amd64/app.exe ./src/cmd/app

# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/linux-amd64/app ./src/cmd/app

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/darwin-amd64/app ./src/cmd/app

# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/darwin-arm64/app ./src/cmd/app
```

### Create Packages

```bash
# Copy extension.yaml to each platform
for dir in bin/*/; do
  cp extension.yaml "$dir"
done

# Windows
cd bin/windows-amd64 && zip -r ../../app-windows-amd64.zip . && cd ../..

# Linux
cd bin/linux-amd64 && tar -czf ../../app-linux-amd64.tar.gz . && cd ../..

# macOS AMD64
cd bin/darwin-amd64 && tar -czf ../../app-darwin-amd64.tar.gz . && cd ../..

# macOS ARM64
cd bin/darwin-arm64 && tar -czf ../../app-darwin-arm64.tar.gz . && cd ../..
```

### Calculate Checksums

```bash
sha256sum app-windows-amd64.zip
sha256sum app-linux-amd64.tar.gz
sha256sum app-darwin-amd64.tar.gz
sha256sum app-darwin-arm64.tar.gz
```

### Update registry.json

Update the `registry.json` file with the new version and checksums, then upload it with the release.

## Hotfix Release

For hotfix releases (e.g., v0.2.1):

1. Create a hotfix branch from the release tag:
   ```bash
   git checkout -b hotfix/v0.2.1 v0.2.0
   ```

2. Make your fixes and commit them

3. Update version in `extension.yaml` and `CHANGELOG.md`

4. Create and push the hotfix tag:
   ```bash
   git tag -a v0.2.1 -m "Hotfix v0.2.1"
   git push origin v0.2.1
   ```

5. Merge hotfix back to main:
   ```bash
   git checkout main
   git merge hotfix/v0.2.1
   git push origin main
   ```

## Troubleshooting

### Release workflow fails

1. Check the [Actions tab](https://github.com/jongio/azd-app-extension/actions)
2. Review error logs
3. Common issues:
   - Build failures: Check Go version compatibility
   - Checksum issues: Ensure files are created correctly
   - Permission errors: Verify GITHUB_TOKEN has write permissions

### Registry update fails

If the automatic registry.json update fails:

1. Manually update `registry.json` with correct version and checksums
2. Create a PR to update it
3. Merge after verification

### Users Can't Install the Extension

1. Verify the release is published (not a draft)
2. Check that all artifacts are attached
3. Verify checksums match the actual files
4. Ensure users have added the registry:
   ```bash
   azd config set extension.registry https://raw.githubusercontent.com/jongio/azd-app-extension/main/registry.json
   ```
5. Test installation yourself:
   ```bash
   azd extension install app --version X.Y.Z
   ```

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR** (X.0.0): Breaking changes
- **MINOR** (0.X.0): New features, backwards compatible
- **PATCH** (0.0.X): Bug fixes, backwards compatible

Examples:
- `v0.1.0` → `v0.2.0`: Added new commands (minor)
- `v0.2.0` → `v0.2.1`: Bug fix (patch)
- `v0.2.1` → `v1.0.0`: Breaking API change (major)
