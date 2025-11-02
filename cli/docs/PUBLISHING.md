# Publishing Checklist for azd-app-extension

This document provides a comprehensive checklist for publishing the App Extension to GitHub and making it available for end users.

## ✅ Pre-Publishing Checklist

### Repository Setup
- [x] Repository URL: `github.com/jongio/azd-app-extension`
- [x] All devstack references removed
- [x] LICENSE file present (MIT - Jon Gallant)
- [x] README updated with clear installation instructions
- [x] CHANGELOG properly formatted
- [x] .gitignore properly configured

### Code Quality
- [ ] All tests pass: `go test ./...`
- [ ] Code coverage meets minimum 80%
- [ ] Linter passes: `golangci-lint run` or `mage lint`
- [ ] Code is properly formatted: `go fmt ./...`

### Documentation
- [x] README.md includes:
  - [x] Installation instructions for end users
  - [x] Installation instructions for developers
  - [x] Usage examples for all commands
  - [x] Build badges (CI, Go Report Card, License)
  - [x] Clear project description
  - [x] Links to documentation
- [x] CHANGELOG.md is up to date
- [x] docs/ folder contains comprehensive guides
- [x] RELEASE.md guide created

### Extension Configuration
- [x] extension.yaml properly configured:
  - [x] Correct namespace: `app`
  - [x] Correct ID: `jongio.azd.app`
  - [x] Version matches release plan
  - [x] All commands listed in examples
  - [x] Platform executables defined
- [x] registry.json properly configured:
  - [x] Correct extension ID: `jongio.azd.app`
  - [x] Correct artifact URLs
  - [x] Checksum placeholders
  - [x] All platforms listed (windows, linux, darwin-amd64, darwin-arm64)

### CI/CD Pipeline
- [x] .github/workflows/ci.yml:
  - [x] Tests on all platforms
  - [x] Multiple Go versions tested
  - [x] Linting configured
  - [x] Coverage reporting setup
- [x] .github/workflows/release.yml:
  - [x] Automated binary builds
  - [x] Platform packaging (zip/tar.gz)
  - [x] Checksum calculation
  - [x] Registry.json auto-update
  - [x] GitHub Release creation

## 📦 Publishing Steps

### 1. Initial Repository Setup (One-time)

```bash
# Push to GitHub (if not already done)
git remote add origin https://github.com/jongio/azd-app-extension.git
git branch -M main
git push -u origin main
```

### 2. Configure GitHub Repository Settings

#### Repository Settings
- **Visibility**: Public
- **Description**: "Azure Developer CLI extension for automated development environment setup"
- **Topics**: Add relevant topics:
  - `azure`
  - `azd`
  - `developer-tools`
  - `cli`
  - `golang`
  - `productivity`

#### Actions Settings
- Enable GitHub Actions
- Allow all actions and reusable workflows
- Set workflow permissions to "Read and write permissions"

#### Secrets (Optional)
- `CODECOV_TOKEN`: For code coverage reporting (if using Codecov)

### 3. Test CI Pipeline

```bash
# Push a change to trigger CI
git add .
git commit -m "Test CI pipeline"
git push origin main
```

Verify:
- CI workflow runs successfully
- All tests pass on all platforms
- Linting passes
- Artifacts are created

### 4. Prepare for First Release

#### Update Version Information
1. **extension.yaml**: Set version to `0.1.0`
2. **CHANGELOG.md**: 
   - Move unreleased changes to `[0.1.0]` section
   - Add release date
   - Keep `[Unreleased]` section for future changes

```bash
git add extension.yaml CHANGELOG.md
git commit -m "Prepare release v0.1.0"
git push origin main
```

### 5. Create First Release

```bash
# Create and push tag
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

### 6. Monitor Release Process

1. Go to: https://github.com/jongio/azd-app-extension/actions
2. Watch the "Release" workflow
3. Verify it completes successfully
4. Check the release page: https://github.com/jongio/azd-app-extension/releases

The automated release workflow will:
- ✅ Build binaries for all platforms
- ✅ Create platform packages with extension.yaml
- ✅ Calculate SHA256 checksums
- ✅ Update registry.json with version and checksums
- ✅ Create GitHub Release with all artifacts
- ✅ Attach release notes from CHANGELOG

### 7. Verify Release

#### Check Release Artifacts
- [ ] `app-windows-amd64.zip` present
- [ ] `app-linux-amd64.tar.gz` present
- [ ] `app-darwin-amd64.tar.gz` present
- [ ] `app-darwin-arm64.tar.gz` present
- [ ] `registry.json` present and updated
- [ ] `checksums.txt` present
- [ ] Release notes populated from CHANGELOG

#### Test Installation
```bash
# Add the registry
azd config set extension.registry https://raw.githubusercontent.com/jongio/azd-app-extension/main/registry.json

# Install the extension
azd extension install app

# Test commands
azd app hi
azd app reqs
azd app deps
azd app run
```

## 🌐 Publishing to azd Extension Registry

To make the extension available via `azd extension install app`, you need to submit a PR to the official Azure Developer CLI registry.

### Option 1: Official Registry (Recommended)

1. Fork the Azure Developer CLI repository:
   ```
   https://github.com/Azure/azure-dev
   ```

2. Add your extension to `cli/azd/extensions/registry.json`:
   ```json
   {
     "id": "app.azd.app",
     "namespace": "app",
     "displayName": "App Extension",
     "description": "A collection of developer productivity commands for Azure Developer CLI",
     "publisher": "jongio",
     "tags": ["developer", "productivity", "app"],
     "versions": [...]
   }
   ```

3. Submit a Pull Request with:
   - Title: "Add App Extension to registry"
   - Description: Extension overview and features
   - Link to your repository and releases

4. After PR is merged, users can install with:
   ```bash
   azd extension install app
   ```

### Option 2: Self-Hosted Registry (Current Approach)

Users install by adding your registry first:

```bash
# Users add your registry
azd config set extension.registry https://raw.githubusercontent.com/jongio/azd-app-extension/main/registry.json

# Then install
azd extension install app
```

This approach gives you full control over the extension distribution without waiting for PRs to be merged.

## 📢 Promotion

### GitHub
- [x] Add repository description
- [x] Add topics/tags
- [ ] Create comprehensive README.md
- [ ] Add GitHub badges
- [ ] Pin important issues/discussions

### Documentation
- [ ] Create usage examples
- [ ] Add troubleshooting guide
- [ ] Create video demonstration (optional)
- [ ] Write blog post (optional)

### Community
- [ ] Share on Twitter/X
- [ ] Share on LinkedIn
- [ ] Post in Azure Developer CLI discussions
- [ ] Share in relevant Discord/Slack communities

## 🔄 Ongoing Maintenance

### For Each New Release
1. Update version in `extension.yaml`
2. Update `CHANGELOG.md`
3. Commit changes
4. Create and push tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`
5. Push tag: `git push origin vX.Y.Z`
6. Verify release workflow completes
7. Test installation
8. Update registry PR (if using official registry)

### Monitoring
- [ ] Watch GitHub issues
- [ ] Monitor CI/CD pipeline health
- [ ] Review pull requests
- [ ] Update documentation as needed

## 📋 Post-Publishing Checklist

After first successful release:
- [ ] Release v0.1.0 created successfully
- [ ] All artifacts uploaded correctly
- [ ] Checksums verified
- [ ] Installation tested on Windows
- [ ] Installation tested on Linux
- [ ] Installation tested on macOS
- [ ] All commands working
- [ ] Documentation is accurate
- [ ] CI badge shows passing
- [ ] Announced to community

## 🆘 Troubleshooting

### Release Workflow Fails
1. Check Actions tab for error logs
2. Verify all files are committed
3. Ensure tag follows semver format (vX.Y.Z)
4. Check GitHub token permissions

### Installation Fails
1. Verify release artifacts are complete
2. Check checksums match
3. Verify extension.yaml is in each package
4. Test with `--verbose` flag for details

### Users Can't Find Extension
- Ensure registry.json is accessible
- Verify URLs in registry.json are correct
- Check if PR to official registry is merged

## 📚 Additional Resources

- [azd Extension Framework](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Semantic Versioning](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/)

## ✅ Ready to Publish!

Once you've completed the above checklist, your extension is ready to be published and used by the community!

**First Release Command:**
```bash
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

**Watch it happen:**
```
https://github.com/jongio/azd-app-extension/actions
```

**Share it:**
```
🎉 Just released App Extension v0.1.0 for Azure Developer CLI!
Automate your dev environment setup for Node.js, Python, and .NET projects.

Install:
azd config set extension.registry https://raw.githubusercontent.com/jongio/azd-app-extension/main/registry.json
azd extension install app

Learn more: https://github.com/jongio/azd-app-extension
```
