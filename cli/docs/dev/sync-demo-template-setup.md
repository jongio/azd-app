# Sync Demo Template Workflow Setup

This document explains how to set up and maintain the `sync-demo-template.yml` GitHub Actions workflow, which automatically syncs the demo project from `cli/demo/` to a separate repository.

## Overview

The workflow:
1. Triggers on pushes to `main` that modify files in `cli/demo/**`
2. Syncs those files to a target repository (default: `jongio/azd-app-demo`)
3. Runs smoke tests to validate the synced demo works correctly

## Prerequisites

### 1. Target Repository

Create the target repository where the demo will be synced:

1. Go to https://github.com/new
2. Create a new repository (e.g., `azd-app-demo`)
3. Initialize it as **empty** (no README, no .gitignore, no license)
4. The workflow will populate it automatically on first run

### 2. Personal Access Token (PAT)

Create a PAT with write access to the target repository:

#### Option A: Fine-Grained PAT (Recommended)

1. Go to https://github.com/settings/tokens?type=beta
2. Click **Generate new token**
3. Configure:
   - **Token name**: `azd-app-demo-sync` (or similar)
   - **Expiration**: Choose based on your security requirements
   - **Repository access**: Select **Only select repositories** and choose your target repo (e.g., `azd-app-demo`)
   - **Permissions**:
     - **Contents**: Read and write
     - **Metadata**: Read-only (automatically selected)
4. Click **Generate token**
5. Copy the token immediately (you won't see it again)

#### Option B: Classic PAT

1. Go to https://github.com/settings/tokens
2. Click **Generate new token** → **Generate new token (classic)**
3. Configure:
   - **Note**: `azd-app-demo-sync`
   - **Expiration**: Choose based on your security requirements
   - **Scopes**: Select `repo` (Full control of private repositories)
4. Click **Generate token**
5. Copy the token immediately

### 3. Repository Secret

Add the PAT as a repository secret:

1. Go to your source repository's settings: `https://github.com/{owner}/{repo}/settings/secrets/actions`
2. Click **New repository secret**
3. Configure:
   - **Name**: `DEMO_REPO_PAT`
   - **Secret**: Paste the PAT you created
4. Click **Add secret**

## Configuration

### Environment Variables

The workflow uses these environment variables (defined at the top of the workflow file):

| Variable | Description | Default |
|----------|-------------|---------|
| `DEMO_SOURCE_PATH` | Path to demo files in source repo | `cli/demo` |
| `TARGET_REPO` | Target repository (owner/repo format) | `jongio/azd-app-demo` |

To change the target repository, edit `.github/workflows/sync-demo-template.yml`:

```yaml
env:
  DEMO_SOURCE_PATH: cli/demo
  TARGET_REPO: your-org/your-demo-repo  # Change this
```

### Required Demo Files

The workflow validates these files exist in the demo source:

- `azure.yaml` - Azure Developer CLI configuration
- `.vscode/mcp.json` - VS Code MCP server configuration

## Workflow Triggers

### Automatic (Push)

The workflow runs automatically when:
- A push is made to the `main` branch
- The push includes changes to files in `cli/demo/**`

### Manual (workflow_dispatch)

You can trigger the workflow manually:

1. Go to **Actions** → **Sync Demo Template**
2. Click **Run workflow**
3. Optionally enable **Dry run** to preview changes without pushing

## Troubleshooting

### Error: DEMO_REPO_PAT HAS EXPIRED

**Symptom**: Workflow fails with "401 Unauthorized"

**Solution**:
1. Generate a new PAT following the steps above
2. Update the `DEMO_REPO_PAT` secret with the new token

### Error: DEMO_REPO_PAT LACKS WRITE ACCESS

**Symptom**: Workflow fails with "403 Forbidden" or "Permission denied"

**Solution**:
1. For fine-grained PATs: Edit the token and ensure **Contents: Read and write** is enabled
2. For classic PATs: Ensure the `repo` scope is selected
3. Verify the PAT has access to the correct target repository

### Error: TARGET REPOSITORY NOT FOUND

**Symptom**: Workflow fails with "404 Not Found"

**Solution**:
1. Verify the target repository exists
2. Verify the repository name is correct in the workflow file
3. For private repos, ensure the PAT has access

### Error: Permission denied during push

**Symptom**: Checkout succeeds but push fails with "Permission to {repo} denied"

**Cause**: The PAT has read access but not write access.

**Solution**:
1. Check that the PAT has write permissions:
   - Fine-grained: **Contents: Read and write**
   - Classic: `repo` scope
2. If using fine-grained PAT, ensure the target repo is in the allowed repository list
3. Check for branch protection rules that might block pushes

### Workflow skipped with warning

**Symptom**: Workflow shows warning "DEMO_REPO_PAT secret is not configured"

**Solution**: Add the `DEMO_REPO_PAT` secret as described above

## Testing Changes

### Dry Run

To test workflow changes without affecting the target repository:

1. Go to **Actions** → **Sync Demo Template**
2. Click **Run workflow**
3. Check **Dry run (do not push changes)**
4. Click **Run workflow**

The workflow will show what changes would be made without actually pushing.

### Local Testing

To verify the demo files locally:

```bash
# Navigate to demo directory
cd cli/demo

# Check required files exist
ls azure.yaml .vscode/mcp.json

# Test with azd (if installed)
azd app reqs
azd app deps
```

## Maintenance

### Rotating the PAT

PATs should be rotated periodically for security:

1. Create a new PAT (following steps above)
2. Update the `DEMO_REPO_PAT` secret
3. Trigger the workflow manually to verify it works
4. Delete the old PAT

### Monitoring

- Check the **Actions** tab for workflow run status
- Failed workflows will show annotations explaining the error
- Enable notifications for workflow failures in repository settings

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Source Repository                        │
│                   (jongio/azd-app)                          │
│                                                             │
│  cli/demo/                                                  │
│  ├── azure.yaml                                             │
│  ├── .vscode/mcp.json                                       │
│  ├── api/                                                   │
│  └── ...                                                    │
│                                                             │
│  .github/workflows/sync-demo-template.yml                   │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ Push to main (cli/demo/**)
                      │ or manual trigger
                      ▼
┌─────────────────────────────────────────────────────────────┐
│              GitHub Actions Workflow                        │
│                                                             │
│  1. Validate PAT permissions                                │
│  2. Checkout source repo                                    │
│  3. Checkout target repo                                    │
│  4. Sync files (rsync)                                      │
│  5. Commit and push changes                                 │
│  6. Run smoke tests                                         │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ Push via DEMO_REPO_PAT
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                   Target Repository                         │
│                 (jongio/azd-app-demo)                       │
│                                                             │
│  / (root)                                                   │
│  ├── azure.yaml                                             │
│  ├── .vscode/mcp.json                                       │
│  ├── api/                                                   │
│  └── ...                                                    │
│                                                             │
│  Ready for users to clone and use!                          │
└─────────────────────────────────────────────────────────────┘
```

## Security Considerations

1. **Minimal permissions**: Use fine-grained PATs with only the necessary permissions
2. **Repository scoping**: Limit PAT access to only the target repository
3. **Secret rotation**: Rotate PATs periodically
4. **Audit logs**: Review GitHub audit logs for unexpected access patterns
5. **Branch protection**: Consider branch protection on the target repo to require reviews

## Related Files

- [.github/workflows/sync-demo-template.yml](../../../.github/workflows/sync-demo-template.yml) - The workflow file
- [cli/demo/](../../demo/) - The demo source files
- [cli/demo/README.md](../../demo/README.md) - Demo documentation
