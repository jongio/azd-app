<!-- NEXT: #test-container-lifecycle -->
# Container Lifecycle Fix Tasks

## DONE: Update Dashboard Stop Logic for Containers {#update-dashboard-stop-for-containers}

**Assignee**: Developer  
**File**: `cli/src/internal/dashboard/service_operations.go`

Modify `stopService()` to detect container services and use Docker stop by name instead of PID/port killing.

**Changes**:
- Check `entry.Type == service.ServiceTypeContainer`
- If container: call `docker.Stop("azd-{serviceName}")` and `docker.Remove()`
- If native: use existing PID + port logic

**Acceptance Criteria**:
- Dashboard "Stop" on container service stops the Docker container
- Dashboard "Restart All" properly stops container services
- No port-killing fallback for containers (prevents killing Docker proxy)

---

## DONE: Update Dashboard Start Logic for Containers {#update-dashboard-start-for-containers}

**Assignee**: Developer  
**File**: `cli/src/internal/dashboard/service_operations.go`

Modify `performStartBulk()` and `performStart()` to handle container services.

**Changes**:
- Check `runtime.Type == service.ServiceTypeContainer`
- If container: call `service.StartContainerService(runtime, projectDir, true)`
- Call `service.StartContainerLogCollection()` after starting container
- Update process validation to check `ContainerID` for containers (not `Process.Pid`)
- Preserve `Type` and `Mode` in registry entry after restart

**Acceptance Criteria**:
- Dashboard "Start" on stopped container starts new container
- Dashboard "Restart" on container restarts the container
- Container logs appear in dashboard after restart
- Registry entry preserves Type="container" after restart

---

## DONE: Update CLI Stop Logic for Containers {#update-cli-stop-for-containers}

**Assignee**: Developer  
**File**: `cli/src/cmd/app/commands/service_control.go`

Modify `performStop()` to detect container services and use Docker stop.

**Changes**:
- Check `entry.Type == service.ServiceTypeContainer`
- If container: call Docker stop by name (`azd-{serviceName}`)
- If native: use existing PID + port logic

**Acceptance Criteria**:
- `azd app stop` works for container services
- `azd app stop --all` properly stops mixed services

---

## DONE: Update CLI Start Logic for Containers {#update-cli-start-for-containers}

**Assignee**: Developer  
**File**: `cli/src/cmd/app/commands/service_control.go`

Modify `performStart()` to handle container services.

**Changes**:
- Check `runtime.Type == service.ServiceTypeContainer`
- If container: call `service.StartContainerService()`
- Call `service.StartContainerLogCollection()` after starting
- Update process validation for containers
- Preserve `Type` and `Mode` in registry entry

**Acceptance Criteria**:
- `azd app start` works for container services
- `azd app restart` works for container services
- Registry entry preserves container type after restart

---

## TODO: Test Container Lifecycle Operations {#test-container-lifecycle}

**Assignee**: Tester  
**Precondition**: Implementation tasks complete

Manual and automated testing of container lifecycle operations.

**Test Cases**:
1. `azd app run` with container service - container starts, Type="container" in registry
2. Dashboard "Stop" on container - container stops via `docker stop`
3. Dashboard "Start" on stopped container - new container starts
4. Dashboard "Restart" on container - container restarts, logs visible
5. Dashboard "Restart All" with mixed services - all services restart correctly
6. CLI `azd app restart` on container - container restarts
7. CLI `azd app stop` + `azd app start` on container - works correctly
8. Registry preserves Type="container" after all operations

---

## Done

(No completed tasks yet)
