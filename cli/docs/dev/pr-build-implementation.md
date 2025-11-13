# PR Build Distribution - Implementation Summary

## Overview

This implementation enables automatic builds of the AZD App extension for every pull request, allowing end users to test changes before they're merged to main.

## What Was Created

### 1. GitHub Actions Workflow (`.github/workflows/pr-build.yml`)

**Triggers:**
- On pull request to `main` branch (trusted contributors only)
- When `safe-to-build` label is added (untrusted contributors)
- Manual trigger via workflow_dispatch (maintainers)

**What It Does:**
1. **Checks permissions** - Determines if contributor is trusted
2. **For untrusted:** Posts comment asking for maintainer approval
3. **For trusted:** Proceeds automatically
4. Calculates a unique version for the PR (e.g., `0.5.7-pr123`)
2. Builds the extension for all 6 platforms using `azd x build --all`
3. Packages the binaries using `azd x pack`
4. Creates a GitHub pre-release with all platform binaries
5. Generates a `pr-registry.json` file using `azd x publish`
6. Uploads the registry to the GitHub release
7. Posts/updates a comment on the PR with installation instructions

**Cleanup:**
- Automatically deletes PR pre-releases when PR is closed/merged

### 2. Documentation

**`docs/pr-builds.md`** - Complete technical specification including:
- Architecture diagrams
- Implementation details
- Alternative approaches considered
- Maintenance procedures

**`docs/testing-pr-builds.md`** - End-user guide with:
- Step-by-step installation instructions
- Cleanup procedures
- Troubleshooting tips
- Platform-specific notes

**`docs/pr-build-quick-ref.md`** - Quick reference with:
- Copy-paste commands
- Minimal explanation
- Links to full guide

## How It Works

### Build Process

```
PR Created/Updated
    ↓
Check Contributor Trust Level
    ↓
    ├─ Trusted (Owner/Member/Collaborator/Contributor)
    │   ↓
    │   Build Runs Automatically
    │
    └─ Untrusted (First-time contributor)
        ↓
        Post Comment: "Requires maintainer approval"
        ↓
        Wait for `safe-to-build` label
        ↓
        Build Runs After Approval
    ↓
Calculate Version: 0.5.7-pr123
    ↓
Build Dashboard (npm ci && npm run build)
    ↓
Build Binaries (azd x build --all)
    ↓
Package (azd x pack)
    ↓
Create GitHub Pre-release (azd x release --prerelease)
    ↓
Generate Registry (azd x publish)
    ↓
Upload pr-registry.json to Release
    ↓
Post Comment on PR with Instructions
```

### Installation Flow (End User)

```
User Opens PR
    ↓
Reads Bot Comment
    ↓
Downloads pr-registry.json
    ↓
Adds Custom Registry Source
    ↓
Installs Extension from PR Build
    ↓
Tests Changes
    ↓
Provides Feedback on PR
    ↓
Cleanup When Done
```

## Key Design Decisions

### 1. Trust-Based Approval System

**Why:**
- Prevents malicious PRs from running untrusted code
- Leverages GitHub's built-in contributor associations
- Auto-builds for trusted contributors (no friction)
- Maintains security for first-time contributors

**How:**
- `pull_request` event: Auto-run for trusted
- `pull_request_target` event: Requires `safe-to-build` label
- Manual `workflow_dispatch`: Maintainer override

### 2. Using GitHub Pre-releases

**Why:** 
- Native integration with `azd x` tooling
- Provides direct download URLs (no artifact authentication needed)
- Consistent with production release flow
- Easy cleanup

**Alternative Considered:** GitHub Actions artifacts
- Rejected because: Requires GitHub authentication to download
- More complex installation process

### 3. Version Naming: `{base}-pr{number}`

**Example:** `0.5.7-pr123`

**Why:**
- Clearly identifies it as a PR build
- Includes PR number for traceability
- Based on current version in `extension.yaml`
- Semantic versioning compliant (pre-release identifier)

### 4. Custom Registry Per PR

**Why:**
- Isolated from production registry
- Can be tested independently
- Easy to share (single file download)
- No risk of polluting main registry

### 5. Automatic Comment Updates

**Why:**
- Users always see current build status
- Avoids cluttering PR with multiple comments
- Instructions update if workflow is re-run

## Installation Commands (For End Users)

The PR comment will contain these commands (with actual values):

```bash
# Download
curl -L -o pr-registry.json https://github.com/jongio/azd-app/releases/download/pr-123-v0.5.7-pr123/pr-registry.json

# Setup
azd config set alpha.extension.enabled on
azd extension source add -n pr-123 -t file -l "$(pwd)/pr-registry.json"

# Install
azd extension install jongio.azd.app --version 0.5.7-pr123

# Test
azd app version
azd app reqs
```

## Cleanup Automation

When a PR is closed or merged:
1. Workflow detects PR closure
2. Finds all releases matching `pr-{number}-*`
3. Deletes the release
4. Deletes the associated tag
5. GitHub automatically removes release assets

## Testing the Implementation

### Before Merging This PR

1. **Create a test PR** with a small change
2. **Wait for workflow** to complete
3. **Check PR comment** appears with instructions
4. **Download and test** the PR build following the instructions
5. **Verify cleanup** by closing the test PR

### Validation Checklist

- [ ] Workflow runs successfully for trusted contributors
- [ ] Workflow posts comment for untrusted contributors
- [ ] Adding `safe-to-build` label triggers build
- [ ] Manual workflow_dispatch works
- [ ] All 6 platform binaries are built
- [ ] GitHub pre-release is created
- [ ] `pr-registry.json` is attached to release
- [ ] PR comment is posted with correct instructions
- [ ] Can install the PR build following instructions
- [ ] `azd app version` shows correct PR version
- [ ] Cleanup runs when PR is closed
- [ ] Pre-release and tag are deleted

## Rollout Plan

### Phase 1: Testing (Current)
- Merge this PR
- Test with 2-3 real PRs
- Gather feedback from early testers

### Phase 2: Documentation
- Update CONTRIBUTING.md with PR testing process
- Add link to testing guide in PR template
- Create video walkthrough (optional)

### Phase 3: Announcement
- Announce in README
- Tweet/blog about the feature
- Update release notes

## Maintenance

### Regular Tasks

**None required** - Fully automated

### Monitoring

Watch for:
- Workflow failures (check Actions tab)
- Failed cleanups (orphaned releases)
- User feedback on installation issues

### Troubleshooting

**Workflow Fails:**
1. Check build logs in Actions
2. Verify `azd` and extensions are available
3. Check GitHub permissions

**Comment Not Posted:**
1. Verify `pull-requests: write` permission
2. Check GitHub API rate limits
3. Review workflow logs for script errors

**Registry Not Working:**
1. Verify `pr-registry.json` uploaded to release
2. Check URL in PR comment is correct
3. Test download manually

## Cost Considerations

- **GitHub Actions:** Free for public repos (unlimited minutes)
- **Storage:** Release assets deleted on PR close (minimal)
- **Bandwidth:** GitHub handles release downloads (free)

**Total Additional Cost:** $0

## Security

### Trust Model

The workflow uses GitHub's built-in contributor trust levels:

**Trusted Contributors (Auto-build):**
- Repository owners
- Organization members
- Repository collaborators
- Previous contributors (have had PRs merged before)

**Untrusted Contributors (Requires Approval):**
- First-time contributors
- External users without prior contribution history

### How It Works

1. **Trusted contributors:** Build runs automatically when PR is created/updated
2. **Untrusted contributors:** 
   - Workflow posts a comment asking for maintainer approval
   - Maintainer reviews the PR code
   - Maintainer adds `safe-to-build` label to trigger build
   - Build runs using `pull_request_target` (secure)

### Manual Build Trigger

Maintainers can also trigger builds manually:
1. Go to Actions tab
2. Select "PR Build" workflow
3. Click "Run workflow"
4. Enter PR number
5. Build runs for that PR

### Security Considerations

- **Code Review First**: Always review untrusted PRs before adding `safe-to-build` label
- **Limited Scope**: PR builds only access `GITHUB_TOKEN` (read repo, write releases)
- **No Secrets**: Workflow doesn't use or expose repository secrets
- **Isolated Builds**: Each PR build is independent and cleaned up when PR closes

### Why This Matters

Malicious PRs could potentially:
- Modify workflow files to exfiltrate secrets
- Use compute resources for crypto mining
- Spam the releases page

The approval requirement prevents these attacks while still allowing easy testing for trusted contributors.

## Future Enhancements

1. **One-line installer script:**
   ```bash
   curl -s https://raw.githubusercontent.com/jongio/azd-app/main/scripts/install-pr.sh | bash -s 123
   ```

2. **Web-based installer page:**
   - Generate on GitHub Pages
   - Click to copy commands
   - Auto-detect OS

3. **Automatic smoke tests:**
   - Run `azd app reqs` after build
   - Post results in PR comment
   - Fail workflow if smoke test fails

4. **Retention policy:**
   - Keep PR builds for 30 days even if PR is closed
   - Add download count metrics

5. **Multi-PR testing:**
   - Allow installing multiple PR builds simultaneously
   - Each in its own namespace

## Success Metrics

- Number of PRs with preview builds tested
- Time from PR creation to first test
- Number of issues caught before merge
- User feedback on installation ease

## References

- **Workflow:** `.github/workflows/pr-build.yml`
- **Spec:** `cli/docs/dev/pr-builds.md`
- **User Guide:** `cli/docs/dev/testing-pr-builds.md`
- **Maintainer Guide:** `cli/docs/dev/pr-build-maintainer-guide.md`
- **Quick Ref:** `cli/docs/dev/pr-build-quick-ref.md`
- **azd x docs:** [Azure Developer CLI Extensions](https://learn.microsoft.com/azure/developer/azure-developer-cli/azd-extensions)
- **Security:** [GitHub Actions Security](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)

---

**Status:** ✅ Ready for Testing  
**Next Steps:** Merge this PR and test with a real PR
