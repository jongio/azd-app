# Podman Docker Detection Tasks

## Status Key
- TODO: Not started
- IN PROGRESS: Currently being worked on
- DONE: Completed

## Tasks

### Task 1: Add Podman version extraction logic [Developer]
**Status:** DONE
**Description:** Updated `extractVersion` function in reqs.go to detect Podman multi-line output format and extract version from "Version:" line.

**Files:**
- `cli/src/cmd/app/commands/reqs.go`

**Criteria:**
- Detect "Podman Engine" in docker --version output
- Parse first "Version: X.Y.Z" line after Client section
- Extract semantic version (X.Y.Z) from the line
- Fall back to existing Docker logic if not Podman format

### Task 2: Update generate.go version extraction [Developer]
**Status:** DONE
**Description:** Updated `extractVersionFromOutput` function in generate.go to handle Podman output when detecting Docker during azure.yaml generation.

**Files:**
- `cli/src/cmd/app/commands/generate.go`

**Criteria:**
- Same Podman detection logic as reqs.go
- Extracted version works with normalizeVersion

### Task 3: Add unit tests for Podman output parsing [Developer]
**Status:** DONE
**Description:** Added test cases for Podman multi-line version output in reqs_test.go.

**Files:**
- `cli/src/cmd/app/commands/reqs_test.go`

**Criteria:**
- Test case for full Podman multi-line output
- Test case verifying Docker native output still works
- Test case for version extraction from "Version: 5.7.0" line
- Test case for edge cases (missing Version line, etc.)

### Task 4: Verify integration [Tester]
**Status:** DONE
**Description:** All unit tests pass for both Docker and Podman version extraction.
