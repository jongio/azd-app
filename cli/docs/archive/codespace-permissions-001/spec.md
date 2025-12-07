# Codespace/Container Permission Handling

## Problem

When running `azd app run` in GitHub Codespaces, users encounter a blocking error:

```
Error: insecure file permissions on azure.yaml: file /workspaces/backyahdbbqweb/thermometers2/azure.yaml is world-writable (permissions: 0666), please run: chmod 644 /workspaces/backyahdbbqweb/thermometers2/azure.yaml
```

GitHub Codespaces (and many container environments) create files with 0666 permissions by default due to the container's umask settings. This is a false positive security concern because:

1. The container is already isolated
2. The user typically owns everything in the container
3. World-writable permissions in a single-user container aren't a real attack vector

## Solution

Modify `ValidateFilePermissions` in `cli/src/internal/security/validation.go` to detect containerized environments and issue a warning instead of a hard failure.

### Environment Detection

Detect container/Codespace environments via:
- `CODESPACES=true` - GitHub Codespaces
- `REMOTE_CONTAINERS=true` - VS Code Dev Containers
- `/.dockerenv` exists - Docker container
- `KUBERNETES_SERVICE_HOST` set - Kubernetes pod

### Behavior Change

| Environment | Current Behavior | New Behavior |
|-------------|------------------|--------------|
| Windows | Skip check | Skip check (no change) |
| Normal Linux/macOS | Error on 0666 | Error on 0666 (no change) |
| Container/Codespace | Error on 0666 | **Warn** but continue |

### Implementation

1. Add `IsContainerEnvironment()` helper function
2. Modify `ValidateFilePermissions()` to return a warning type instead of error in containers
3. Update `loadAzureYaml()` in core.go to handle warnings appropriately

### Warning Message

When in container environment with insecure permissions:
```
Warning: azure.yaml has world-writable permissions (0666). This is common in container environments but consider fixing with: chmod 644 <path>
```

## Workaround (Immediate)

Users can fix this immediately by running:
```bash
chmod 644 azure.yaml
```

## Files to Modify

- `cli/src/internal/security/validation.go` - Add container detection, modify permission check
- `cli/src/cmd/app/commands/core.go` - Handle warning vs error

## Testing

- Test in Codespaces with 0666 permissions - should warn not error
- Test on normal Linux with 0666 permissions - should error
- Test on normal Linux with 0644 permissions - should pass silently
- Test on Windows - should skip check
