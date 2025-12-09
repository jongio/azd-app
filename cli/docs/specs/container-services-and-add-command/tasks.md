# Tasks: Container Services and `azd app add` Command

## Progress: 12/12 tasks complete ✅

---

## Phase 1: Docker Integration Foundation

### Task 1: Create Docker client package

**Agent**: Developer  
**Status**: ✅ DONE

**Description**:
Create new package `cli/src/internal/docker/` with Docker client abstraction for container management.

**Changes Made**:
- Created `types.go` with ContainerConfig, PortMapping, Container types
- Created `client.go` with Client interface (IsAvailable, Pull, Run, Stop, Remove, Logs, Inspect, IsRunning)
- Created `exec.go` with ExecClient implementation using docker CLI
- Created `client_test.go` with unit tests for port mapping, argument building, validation

---

### Task 2: Add container service detection

**Agent**: Developer  
**Status**: ✅ DONE

**Description**:
Update service detection to identify container services (has `image` field) vs native process services.

**Changes Made**:
- Added `ServiceTypeContainer` constant to types.go
- Added `IsContainerService()` method to Service type
- Added `GetContainerImage()` method to Service type
- Added `detectContainerRuntime()` function to detector.go
- Updated `DetectServiceRuntime()` to check for container services first
- Added unit tests for container service detection

---

### Task 3: Update orchestrator for containers

**Agent**: Developer  
**Status**: ✅ DONE

**Description**:
Update service orchestrator to start, monitor, and stop Docker containers alongside native processes.

**Changes Made**:
- Created `container_runner.go` with StartContainerService, StopContainerService, StartContainerLogCollection
- Updated `orchestrator.go` to detect container services and use container runner
- Updated `StopAllServices` to handle container shutdown
- Container logs streamed to unified log system via LogBuffer

---

### Task 4: Add Docker to reqs validation

**Agent**: Developer  
**Status**: � IN PROGRESS

**Description**:
When container services are defined, automatically validate Docker is installed and running.

**Acceptance Criteria**:
- [ ] Detect if any services are container services
- [ ] If yes, add implicit `docker` requirement with `checkRunning: true`
- [ ] Clear error message if Docker not available
- [ ] Skip Docker check if no container services defined

**Files**:
- `cli/src/cmd/app/commands/reqs.go`
- `cli/src/cmd/app/commands/run.go`

---

## Phase 2: Dashboard Integration

### Task 5: Display container services in dashboard

**Agent**: Developer  
**Status**: 🔲 TODO

**Description**:
Update dashboard to display container services with appropriate indicators and information.

**Acceptance Criteria**:
- [ ] Container services show container icon/indicator
- [ ] Display image name in service details
- [ ] Show port mappings (host:container)
- [ ] Health status from TCP check
- [ ] Differentiate from native process services

**Files**:
- `cli/dashboard/src/components/ServiceCard.tsx`
- `cli/dashboard/src/types.ts`
- `cli/src/internal/dashboard/api.go`

---

### Task 6: Stream container logs to dashboard

**Agent**: Developer  
**Status**: 🔲 TODO

**Description**:
Stream container stdout/stderr to the unified log system for dashboard display.

**Acceptance Criteria**:
- [ ] Container logs appear in service log stream
- [ ] Logs properly attributed to container service
- [ ] Log filtering works for container services
- [ ] Historical logs available on dashboard load

**Files**:
- `cli/src/internal/service/container_runner.go`
- `cli/src/internal/service/logger.go`

---

## Phase 3: Well-Known Services Registry

### Task 7: Create well-known services registry

**Agent**: Developer  
**Status**: 🔲 TODO

**Description**:
Create registry of well-known Azure emulators and common development services with their configurations.

**Acceptance Criteria**:
- [ ] `wellknown/services.go` with service definitions
- [ ] Azurite configuration (image, ports, connection string)
- [ ] Cosmos DB emulator configuration
- [ ] Redis configuration
- [ ] PostgreSQL configuration
- [ ] Connection string templates for each service
- [ ] Unit tests for service lookup

**Files**:
- `cli/src/internal/wellknown/services.go`
- `cli/src/internal/wellknown/services_test.go`

---

## Phase 4: `azd app add` Command

### Task 8: Implement `azd app add` command

**Agent**: Developer  
**Status**: 🔲 TODO

**Description**:
Create new `azd app add` command to add well-known services to azure.yaml.

**Acceptance Criteria**:
- [ ] `add.go` command implementation
- [ ] Accepts service type as argument (e.g., `azurite`, `cosmos`)
- [ ] `--name` flag for custom service name
- [ ] `--port` flag for port override
- [ ] `--dry-run` flag to preview changes
- [ ] Validates azure.yaml exists
- [ ] Validates service doesn't already exist
- [ ] Unit tests for command logic

**Files**:
- `cli/src/cmd/app/commands/add.go`
- `cli/src/cmd/app/commands/add_test.go`

---

### Task 9: YAML modification for add command

**Agent**: Developer  
**Status**: 🔲 TODO

**Description**:
Implement azure.yaml modification to add new service while preserving existing content and formatting.

**Acceptance Criteria**:
- [ ] Add service to existing azure.yaml
- [ ] Preserve comments and formatting where possible
- [ ] Handle missing `services` section
- [ ] Create azure.yaml if doesn't exist (with confirmation)
- [ ] Backup original file before modification

**Files**:
- `cli/src/internal/yamlutil/modify.go`
- `cli/src/internal/yamlutil/modify_test.go`

---

### Task 10: Connection string display

**Agent**: Developer  
**Status**: 🔲 TODO

**Description**:
Display connection string information after adding a service so users know how to configure dependent services.

**Acceptance Criteria**:
- [ ] Show environment variable name for connection string
- [ ] Show full connection string value
- [ ] Show example environment config for azure.yaml
- [ ] Include any special notes (e.g., HTTPS cert for Cosmos)
- [ ] Copy-friendly output format

**Files**:
- `cli/src/cmd/app/commands/add.go`

---

## Phase 5: Documentation and Testing

### Task 11: Update schema and documentation

**Agent**: Developer  
**Status**: 🔲 TODO

**Description**:
Update JSON schema and documentation to reflect container service support and add command.

**Acceptance Criteria**:
- [ ] Update `schemas/v1.1/azure.yaml.json` with container examples
- [ ] Add `cli/docs/commands/add.md` documentation
- [ ] Update `cli/docs/schema/azure.yaml.md` with container service section
- [ ] Update `cli/docs/commands/run.md` with container info
- [ ] Add examples with Azurite and Cosmos DB

**Files**:
- `schemas/v1.1/azure.yaml.json`
- `cli/docs/commands/add.md` (new)
- `cli/docs/schema/azure.yaml.md`
- `cli/docs/commands/run.md`

---

### Task 12: Integration tests

**Agent**: Tester  
**Status**: 🔲 TODO

**Description**:
Create integration tests for container services and add command.

**Acceptance Criteria**:
- [ ] Test `azd app add azurite` creates correct yaml
- [ ] Test `azd app add cosmos` creates correct yaml
- [ ] Test `azd app run` with container service (requires Docker)
- [ ] Test container service health checks
- [ ] Test container service shutdown
- [ ] Test error handling when Docker unavailable

**Files**:
- `cli/src/cmd/app/commands/add_integration_test.go`
- `cli/src/internal/service/container_integration_test.go`

---

## Dependency Graph

```
Task 1 (Docker client) 
    └── Task 2 (Service detection) 
        └── Task 3 (Orchestrator)
            └── Task 4 (Reqs validation)
                └── Task 5 (Dashboard)
                    └── Task 6 (Container logs)

Task 7 (Registry) 
    └── Task 8 (Add command)
        └── Task 9 (YAML modify)
            └── Task 10 (Connection strings)

Task 11 (Docs) - can run in parallel after Task 8
Task 12 (Tests) - after all implementation tasks
```
