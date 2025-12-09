<!-- NEXT: -->
# Reqs Install URL Tasks

## Done

### DONE: Add installUrl to schema {#add-installurl-schema}
**Assigned**: Developer
**File**: `schemas/v1.1/azure.yaml.json`

Added `installUrl` property to the `requirement` definition with type: string, format: uri.

### DONE: Add install URL registry {#add-install-url-registry}
**Assigned**: Developer
**File**: `cli/src/cmd/app/commands/reqs.go`

Created `installURLRegistry` map with built-in URLs for 21 tools.

### DONE: Update Prerequisite struct {#update-prerequisite-struct}
**Assigned**: Developer
**File**: `cli/src/cmd/app/commands/reqs.go`

Added `InstallUrl` field to `Prerequisite` struct with YAML tag.

### DONE: Update ReqResult struct {#update-reqresult-struct}
**Assigned**: Developer
**File**: `cli/src/cmd/app/commands/reqs.go`

Added `InstallUrl` field to `ReqResult` struct with JSON tag.

### DONE: Update Check method output {#update-check-method}
**Assigned**: Developer
**File**: `cli/src/cmd/app/commands/reqs.go`

Modified `PrerequisiteChecker.Check()` to resolve install URL and display on failure.

### DONE: Consolidate pathutil suggestions {#consolidate-pathutil}
**Assigned**: Developer
**File**: `cli/src/internal/pathutil/pathutil.go`

Updated `GetInstallSuggestion()` to use same URLs as install URL registry.

### DONE: Add unit tests {#add-unit-tests}
**Assigned**: Developer
**File**: `cli/src/cmd/app/commands/reqs_test.go`

Added tests: `TestInstallURLRegistry`, `TestGetInstallUrl`, `TestCheckPrerequisiteIncludesInstallUrl`.

### DONE: Update documentation {#update-documentation}
**Assigned**: Developer
**File**: `cli/docs/commands/reqs.md`

Documented `installUrl` field, built-in URLs table, and updated output examples.

