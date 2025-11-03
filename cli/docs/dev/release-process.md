# Release Process

This document describes the automated release process for the azd-app CLI extension.

## Overview

The release process is automated through GitHub Actions and consists of two main workflows:

1. **release.yml** - Builds binaries, creates release packages, and publishes to GitHub Releases
2. **update-registry.yml** - Updates the registry.json file in the main branch after a release

## Triggering a Release

To trigger a release, push a tag with the format `cli-v*.*.*`:

```bash
# Example: Release version 0.1.0
git tag cli-v0.1.0
git push origin cli-v0.1.0
```

## Release Workflow Steps

### 1. Build and Release (release.yml)

When a tag matching `cli-v*.*.*` is pushed:

1. **Checkout code** - Fetches the repository with full history
2. **Set up Go** - Installs Go 1.22
3. **Get version from tag** - Extracts version number (e.g., `0.1.0` from `cli-v0.1.0`)
4. **Build for all platforms** - Compiles binaries for:
   - Windows AMD64 (`app.exe`)
   - Linux AMD64 (`app`)
   - macOS AMD64 (`app`)
   - macOS ARM64 (`app`)
5. **Create platform packages**:
   - Copies `extension.yaml` to each platform directory
   - Creates `.zip` archive for Windows
   - Creates `.tar.gz` archives for Linux and macOS
6. **Calculate checksums** - Generates SHA256 checksums for all packages
7. **Update registry.json** - Updates the registry file with:
   - Current version number
   - Download URLs pointing to the release assets
   - SHA256 checksums for verification
8. **Extract changelog** - Pulls release notes from CHANGELOG.md
9. **Create GitHub Release** - Publishes release with:
   - Platform-specific packages (zip/tar.gz files)
   - Updated registry.json
   - Checksums file
   - Release notes from changelog

### 2. Update Registry (update-registry.yml)

After a release is published:

1. **Download registry.json** - Fetches the updated registry.json from the release assets
2. **Create Pull Request** - Opens a PR to merge the updated registry.json into main branch

This two-step process ensures:
- The registry.json attached to the release has correct checksums
- The main branch gets updated via PR for review
- No conflicts from pushing to a detached HEAD state

## File Structure in Release

Each release includes:

```
cli-v0.1.0/
├── app-windows-amd64.zip          # Windows binary + extension.yaml
├── app-linux-amd64.tar.gz         # Linux binary + extension.yaml
├── app-darwin-amd64.tar.gz        # macOS Intel binary + extension.yaml
├── app-darwin-arm64.tar.gz        # macOS Apple Silicon binary + extension.yaml
├── registry.json                   # Updated registry with checksums
└── checksums.txt                   # SHA256 checksums for verification
```

## Registry.json Updates

The registry.json is automatically updated with:

```json
{
  "extensions": [{
    "versions": [{
      "version": "0.1.0",
      "artifacts": {
        "windows-amd64": {
          "url": "https://github.com/jongio/azd-app/releases/download/cli-v0.1.0/app-windows-amd64.zip",
          "checksum": {
            "algorithm": "sha256",
            "value": "<actual-sha256-checksum>"
          }
        },
        // ... other platforms
      }
    }]
  }]
}
```

## Pre-Release Checklist

Before creating a release tag:

1. Update `cli/version.txt` with the new version number
2. Update `cli/CHANGELOG.md` with release notes
3. Ensure all tests pass: `go test ./...`
4. Build locally to verify: `mage build` or `./build.ps1`
5. Test the extension: `./install-local.ps1`

## Manual Testing of Release Artifacts

After a release is published, you can test the installation:

```bash
# Download and test Windows version
curl -L -o app.zip https://github.com/jongio/azd-app/releases/download/cli-v0.1.0/app-windows-amd64.zip
unzip app.zip
./app.exe version

# Download and test Linux/macOS version
curl -L -o app.tar.gz https://github.com/jongio/azd-app/releases/download/cli-v0.1.0/app-linux-amd64.tar.gz
tar -xzf app.tar.gz
./app version
```

## Troubleshooting

### Release workflow fails during build
- Check Go version compatibility
- Verify all dependencies are available
- Review build logs for compilation errors

### Checksums don't match
- Ensure the checksum step has `id: checksums`
- Verify file paths are correct (working-directory is `cli`)
- Check that SHA256 calculation uses correct file paths

### Registry update fails
- Verify registry.json is attached to the release
- Check that the tag matches the pattern `cli-v*.*.*`
- Ensure the update-registry workflow has proper permissions

### PR not created for registry update
- Verify the release was published (not draft)
- Check workflow permissions for PR creation
- Review update-registry.yml workflow logs
