# Podman Docker Version Detection

## Overview

Support detecting Docker version when Podman is aliased to Docker. When users have Podman redirected to Docker (common in rootless container setups), running `docker --version` outputs a multi-line Podman format instead of Docker's single-line format.

## Problem

When Podman is used as a Docker drop-in replacement:
1. `azd app reqs` needs to extract the version from Podman's multi-line output format
2. **Version schemes are incompatible**: Docker uses versions like `20.10.0`, `28.5.1` while Podman uses `4.x`, `5.x`
3. A typical azure.yaml might specify `minVersion: "20.10.0"` which Podman's `5.7.0` would fail

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

## Solution

### Version Check Behavior

When Podman is detected aliased to Docker:
1. **Version extraction**: Parse Podman version from "Version:" line
2. **Version comparison**: **SKIPPED** - Podman and Docker versions are not comparable
3. **Result**: Mark as satisfied if Podman is installed and running (when `checkRunning: true`)

This means:
- `docker` requirement with `minVersion: "20.10.0"` will pass if Podman 5.7.0 is installed
- The output will show: `docker: 5.7.0 via Podman (version check skipped)`
- JSON output includes `"isPodman": true` field

### Rationale

Podman is designed as a Docker-compatible container runtime. Rather than maintain a complex version mapping table or require users to specify separate `podman` requirements, we trust that any reasonably recent Podman version provides Docker-compatible functionality.

## Requirements

### Version Extraction
- Detect Podman output format when `docker --version` is called
- Extract the Client Version from multi-line Podman output
- Continue supporting native Docker version format
- Extract semantic version (X.Y.Z) from "Version: X.Y.Z" line

### Output Handling
- When Podman is detected, skip version comparison (versions are incompatible)
- Display indicates Docker is satisfied via Podman
- JSON output includes `isPodman: true` flag

### Test Coverage
- Unit tests for Podman multi-line version output parsing
- Unit tests for Docker native version output parsing
- Ensure both formats extract version correctly

## Implementation

### Files Modified
- `cli/src/cmd/app/commands/reqs.go` - `extractVersion`, `Check`, `getInstalledVersion`
- `cli/src/cmd/app/commands/reqs_test.go` - Tests for Podman output format

### Approach
1. In `getInstalledVersion`, detect "Podman Engine" in output and return `isPodman` flag
2. In `Check`, when `isPodman` is true for `docker` requirement, skip version comparison
3. Continue to `checkRunning` verification if configured

## Acceptance Criteria
- `azd app reqs` correctly reports Docker satisfied when Podman is aliased
- Version comparison skipped for Podman (avoids false negatives)
- Native Docker version detection and comparison continues to work
- Unit tests pass for both Docker and Podman formats
- JSON output includes `isPodman` field when applicable
