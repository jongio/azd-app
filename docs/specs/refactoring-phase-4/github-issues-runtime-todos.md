# GitHub Issues for Runtime TODO Comments

This document outlines the GitHub issues that should be created for TODO comments in the runtime-related code.

## Issue #1001: Support Multiple Port Mappings for Container Services

**Location**: `cli/src/internal/service/container_runner.go:201`

**Summary**:
Enhance container port mapping functionality to support multiple ports beyond the primary port.

**Current Behavior**:
The `buildContainerPortMappings` function only maps a single port from `runtime.Port` to Docker port mappings. This assumes services only expose one port.

**Problem**:
Many containerized services expose multiple ports for different purposes:
- Primary HTTP/API port
- Debug ports (e.g., Node.js debugger on port 9229)
- Metrics endpoints (e.g., Prometheus on port 9090)
- Additional service ports (e.g., gRPC on a separate port)

**Proposed Solution**:
1. Add a `Ports []int` field to `ServiceRuntime` struct to store multiple port numbers
2. Update `buildContainerPortMappings` to iterate over all ports in the slice
3. Update service detection logic to parse and store multiple ports from:
   - Docker Compose `ports:` configuration
   - Dockerfile `EXPOSE` directives
   - Service configuration in azure.yaml
4. Maintain backward compatibility with single `runtime.Port` field

**Example Use Case**:
```yaml
services:
  api:
    docker:
      image: myapi:latest
      ports:
        - 8080:8080  # API port
        - 9229:9229  # Debug port
```

**Acceptance Criteria**:
- [ ] ServiceRuntime can store multiple port numbers
- [ ] All ports are correctly mapped to Docker container
- [ ] Service detection parses multiple ports from various sources
- [ ] Tests cover multi-port scenarios
- [ ] Backward compatibility maintained for single-port services

---

## Issue #1002: Add Dedicated Image Field to ServiceRuntime

**Location**: `cli/src/internal/service/detector.go:182`

**Summary**:
Add a dedicated `Image` field to the `ServiceRuntime` struct for container-based services instead of misusing the `Command` field.

**Current Behavior**:
Container images are stored in the `ServiceRuntime.Command` field, which is semantically incorrect since a command is not an image reference.

**Problem**:
- **Semantic confusion**: The `Command` field should represent the command to execute, not a container image
- **Type ambiguity**: Code must infer whether `Command` contains an image or an actual command
- **Maintenance issues**: Makes the codebase harder to understand and maintain
- **Future extensibility**: Limits ability to have both a custom command AND an image for container services

**Proposed Solution**:
1. Add a new `Image string` field to the `ServiceRuntime` struct
2. Update container service detection to populate `runtime.Image` instead of `runtime.Command`
3. Update container runner logic to use `runtime.Image` when available
4. Keep `Command` field for its original purpose (overriding the container's default command)
5. Update all tests to use the new field

**Example Structure**:
```go
type ServiceRuntime struct {
    // ... existing fields ...
    Image   string   // Container image (e.g., "nginx:latest", "myapp:v1.2.3")
    Command string   // Optional: Override container's default command
    Args    []string // Optional: Arguments to the command
    // ... other fields ...
}
```

**Migration Path**:
1. Add `Image` field (non-breaking change)
2. Update code to populate both `Image` and `Command` during transition period
3. Update consumers to check `Image` first, fall back to `Command` for compatibility
4. Remove `Command` usage for images in a future version

**Acceptance Criteria**:
- [ ] `Image` field added to `ServiceRuntime` struct
- [ ] Container detection populates `Image` field
- [ ] Container runner uses `Image` field for image reference
- [ ] `Command` field can still be used for custom commands
- [ ] All tests updated and passing
- [ ] Documentation updated to explain the fields

---

## Notes

These issues were identified during refactoring phase 4 (Task 15). The TODOs have been updated in the code with references to these issue numbers (#1001, #1002) to track them until implementation.
