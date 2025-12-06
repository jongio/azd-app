# Podman Docker Version Detection

## Overview

Support detecting Docker version when Podman is aliased to Docker. When users have Podman redirected to Docker (common in rootless container setups), running `docker --version` outputs a multi-line Podman format instead of Docker's single-line format.

## Problem

Currently, `azd app reqs` fails to extract the Docker version when Podman is used as a Docker drop-in replacement.

**Docker native output:**
```
Docker version 28.5.1, build abc123
```

**Podman aliased to Docker output:**
```
Client:       Podman Engine
Version:      5.7.0
API Version:  5.7.0
Go Version:   go1.25.4
Git Commit:   0370128fc8dcae93533334324ef838db8f8da8cb
Built:        Tue Nov 11 10:57:57 2025
OS/Arch:      windows/amd64

Server:       Podman Engine
Version:      5.7.0
API Version:  5.7.0
Go Version:   go1.24.9
Git Commit:   0370128fc8dcae93533334324ef838db8f8da8cb
Built:        Mon Nov 10 16:00:00 2025
OS/Arch:      linux/amd64
```

## Requirements

### Version Extraction
- Detect Podman output format when `docker --version` is called
- Extract the Client Version from multi-line Podman output
- Continue supporting native Docker version format
- Extract semantic version (X.Y.Z) from "Version: X.Y.Z" line

### Output Handling
- When Podman is detected, report version as found
- The extracted version should work with version comparison logic
- Display should indicate Docker is satisfied (regardless of underlying engine)

### Test Coverage
- Unit tests for Podman multi-line version output parsing
- Unit tests for Docker native version output parsing
- Ensure both formats extract version correctly

## Implementation

### Files to Modify
- `cli/src/cmd/app/commands/reqs.go` - Update `extractVersion` function
- `cli/src/cmd/app/commands/reqs_test.go` - Add tests for Podman output format
- `cli/src/cmd/app/commands/generate.go` - Update `extractVersionFromOutput` if needed

### Approach
1. In `extractVersion`, check if output matches Podman multi-line format
2. If Podman format detected, parse "Version:" line from Client section
3. Fall back to existing Docker parsing logic for native Docker output

## Acceptance Criteria
- `azd app reqs` correctly reports Docker version when Podman is aliased
- Native Docker version detection continues to work
- Unit tests pass for both Docker and Podman formats
- Version comparison works correctly with extracted Podman versions
