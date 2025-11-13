# PR Build Security Flow

## Trust Decision Tree

```
                        PR Created/Updated
                               │
                               ▼
                    Check Author Association
                               │
                ┌──────────────┴──────────────┐
                ▼                             ▼
         TRUSTED                        UNTRUSTED
    (Owner/Member/                  (First-time or
     Collaborator/                   External User)
     Contributor)                          │
         │                                 ▼
         │                        Post Comment:
         │                        "Requires approval"
         │                                 │
         │                                 ▼
         │                     Maintainer Reviews Code
         │                                 │
         │                      ┌──────────┴──────────┐
         │                      ▼                     ▼
         │                  Suspicious             Looks Safe
         │                      │                     │
         │                      ▼                     ▼
         │                 Close PR            Add 'safe-to-build'
         │                                          label
         │                                            │
         └────────────────┬───────────────────────────┘
                          ▼
                    Build Starts
                          │
           ┌──────────────┼──────────────┐
           ▼              ▼              ▼
      Build         Package         Release
    Dashboard       Binaries       Pre-release
           │              │              │
           └──────────────┴──────────────┘
                          │
                          ▼
                 Generate Registry
                          │
                          ▼
              Post Comment with Install
                   Instructions
```

## Contributor Trust Levels

| Association | Auto-Build | Requires Approval | Manual Trigger |
|-------------|------------|-------------------|----------------|
| `OWNER` | ✅ Yes | ❌ No | ✅ Yes |
| `MEMBER` | ✅ Yes | ❌ No | ✅ Yes |
| `COLLABORATOR` | ✅ Yes | ❌ No | ✅ Yes |
| `CONTRIBUTOR` | ✅ Yes | ❌ No | ✅ Yes |
| `FIRST_TIME_CONTRIBUTOR` | ❌ No | ✅ Yes | ✅ Yes |
| `NONE` (external) | ❌ No | ✅ Yes | ✅ Yes |

## Approval Workflow

### For Maintainers

```
1. New PR from unknown contributor
   │
   ▼
2. Bot comments: "Requires maintainer approval"
   │
   ▼
3. You review the PR code
   │
   ├─ Looks suspicious
   │  │
   │  └─> Close PR / Request changes
   │
   └─ Looks safe
      │
      ▼
   Add 'safe-to-build' label
      │
      ▼
   Build runs automatically
      │
      ▼
   Bot posts install instructions
```

### Manual Override

```
Any PR (trusted or untrusted)
   │
   ▼
Go to Actions tab
   │
   ▼
Select "PR Build" workflow
   │
   ▼
Click "Run workflow"
   │
   ▼
Enter PR number
   │
   ▼
Click "Run workflow" button
   │
   ▼
Build runs regardless of trust level
```

## Security Events

### GitHub Action Events Used

1. **`pull_request`** - For trusted contributors
   - Runs in PR context
   - Has limited permissions
   - Safe for trusted code

2. **`pull_request_target`** - For labeled PRs
   - Triggered by `safe-to-build` label
   - Runs in base branch context
   - Has write permissions
   - Only runs after manual approval

3. **`workflow_dispatch`** - For manual triggers
   - Runs in base branch context
   - Requires maintainer action
   - Full control over which PR to build

## Permission Model

```yaml
permissions:
  contents: write       # Create releases and tags
  pull-requests: write  # Post comments on PRs
```

**What the workflow CAN do:**
- ✅ Read PR code
- ✅ Build binaries
- ✅ Create GitHub pre-releases
- ✅ Post comments on PRs
- ✅ Upload release assets

**What the workflow CANNOT do:**
- ❌ Access repository secrets (for untrusted PRs)
- ❌ Modify main branch
- ❌ Push to protected branches
- ❌ Change repository settings
- ❌ Access other organization resources

## Attack Vectors & Mitigations

### Attack: Malicious Workflow Modification

**Scenario:** First-time contributor modifies `.github/workflows/pr-build.yml` to exfiltrate secrets

**Mitigation:**
- Workflow runs in `pull_request_target` mode for untrusted PRs
- Uses base branch version of workflow (not PR version)
- Maintainer reviews code before approval
- No secrets are exposed to the workflow

### Attack: Resource Abuse

**Scenario:** Contributor submits PR with infinite loop to waste compute resources

**Mitigation:**
- Requires manual approval via `safe-to-build` label
- GitHub Actions has timeout limits (default 6 hours)
- Maintainer reviews code before approval
- Can set lower timeout in workflow config

### Attack: Spam Releases

**Scenario:** Creating many PRs to spam the releases page

**Mitigation:**
- Requires manual approval for each build
- Pre-releases auto-cleanup when PR closes
- Maintainers can delete releases manually
- Can block repeat offenders

## Best Practices for Maintainers

### ✅ DO

- Review code changes before adding `safe-to-build` label
- Check for workflow file modifications
- Verify no suspicious external calls
- Monitor build logs for unusual activity
- Remove label if PR is updated with new commits

### ❌ DON'T

- Add `safe-to-build` label without reviewing code
- Approve PRs that modify workflow files (unless you trust the contributor)
- Leave `safe-to-build` label on PRs that get updated
- Approve PRs with obvious malicious code

## Example: Safe PR Review Checklist

Before adding `safe-to-build` label:

- [ ] Reviewed all code changes
- [ ] No modifications to `.github/workflows/`
- [ ] No modifications to build scripts (unless expected)
- [ ] No suspicious external network calls
- [ ] No obvious resource abuse patterns
- [ ] Contributor has valid reason for PR
- [ ] Changes align with PR description

If all checked, safe to add `safe-to-build` label.

## Monitoring

### Check Build Status

```bash
# List recent workflow runs
gh run list --workflow=pr-build.yml

# View specific run
gh run view <run-id>

# View logs
gh run view <run-id> --log
```

### Check Pre-releases

```bash
# List all releases (including pre-releases)
gh release list --limit 50

# View specific release
gh release view pr-123-v0.5.7-pr123
```

### Cleanup Old Pre-releases

```bash
# List PR pre-releases
gh release list | grep "^pr-"

# Delete specific PR release
gh release delete pr-123-v0.5.7-pr123 --yes

# Delete associated tag
gh api repos/:owner/:repo/git/refs/tags/pr-123-v0.5.7-pr123 -X DELETE
```

## Resources

- [GitHub Actions Security](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)
- [pull_request_target](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#pull_request_target)
- [Preventing pwn requests](https://securitylab.github.com/research/github-actions-preventing-pwn-requests/)
