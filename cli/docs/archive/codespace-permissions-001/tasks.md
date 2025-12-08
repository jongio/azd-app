# Codespace/Container Permission Handling

## Tasks

### 1. Add container environment detection
- **File**: `cli/src/internal/security/validation.go`
- **Status**: DONE
- **Assigned**: Developer
- **Acceptance**:
  - Add `IsContainerEnvironment() bool` function
  - Detect: `CODESPACES`, `REMOTE_CONTAINERS`, `/.dockerenv`, `KUBERNETES_SERVICE_HOST`
  - Unit tests for each detection method

### 2. Modify permission validation to support warnings
- **File**: `cli/src/internal/security/validation.go`
- **Status**: DONE
- **Assigned**: Developer
- **Acceptance**:
  - `ValidateFilePermissions` returns warning (not error) in container environments
  - Existing behavior preserved for non-container environments
  - Clear, actionable warning message with fix command

### 3. Update loadAzureYaml to handle permission warnings
- **File**: `cli/src/cmd/app/commands/core.go`
- **Status**: DONE
- **Assigned**: Developer
- **Acceptance**:
  - Warnings logged but don't block execution
  - Errors still block in non-container environments
  - Warning appears once per session, not spamming

### 4. Add integration test for Codespace scenario
- **File**: `cli/src/internal/security/validation_test.go`
- **Status**: DONE
- **Assigned**: Developer
- **Acceptance**:
  - Test with mocked `CODESPACES=true` environment
  - Verify warning returned, not error
  - Test all container detection paths
