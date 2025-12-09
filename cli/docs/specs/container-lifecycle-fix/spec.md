# Container Service Lifecycle Fix

## Problem Statement

Container services (those with `image` field in azure.yaml) don't properly restart when using "Restart All" or individual restart operations from the dashboard. The services stop but don't come back to life.

## Root Cause Analysis

### Issue 1: Registry Doesn't Track Container ID

The `ServiceRegistryEntry` struct doesn't have a `ContainerID` field:

```go
// registry/registry.go
type ServiceRegistryEntry struct {
    Name        string
    PID         int       // Only for native processes
    Port        int
    // Missing: ContainerID for container services
}
```

When a container service is started, the `ContainerID` is stored in `ServiceProcess.ContainerID`, but this is never persisted to the registry. When we try to stop/restart, we have no way to reference the container.

### Issue 2: Service Operations Don't Distinguish Container Types

The dashboard's `service_operations.go` uses the same logic for all service types:

```go
// performStop - uses PID-based stopping
func (h *serviceOperationHandler) stopService(entry *registry.ServiceRegistryEntry, serviceName string) error {
    // First, try to stop by the registered PID
    if entry.PID > 0 {
        process, err := os.FindProcess(entry.PID)
        // ...tries to stop native process
    }
    // Falls back to port-based killing
}

// performStartBulk - uses StartService (native process only)
process, err := service.StartService(runtime, envVars, h.server.projectDir, functionsParser)
```

Container services have `PID=0` and require `docker stop` / `docker rm` / `docker run` commands instead.

### Issue 3: CLI Service Controller Has Same Problem

The `service_control.go` file's `performStop` method also only handles native processes:

```go
func (c *ServiceController) performStop(ctx context.Context, entry *registry.ServiceRegistryEntry, serviceName string) error {
    // Only tries PID-based stopping
    if entry.PID > 0 {
        process, err := os.FindProcess(entry.PID)
        // ...
    }
    // No container handling
}
```

## Current Container Support Location

The orchestrator (`orchestrator.go`) correctly handles containers:

```go
// StopAllServices - correctly checks Type
if proc.Runtime.Type == ServiceTypeContainer {
    stopErr = StopContainerService(proc, DefaultStopTimeout)
} else {
    stopErr = StopServiceGraceful(proc, DefaultStopTimeout)
}

// startSingleService - correctly checks Type  
if rt.Type == ServiceTypeContainer {
    process, err = StartContainerService(rt, projectDir, restartContainers)
} else {
    process, err = StartService(rt, serviceEnv, projectDir, functionsParser)
}
```

But this logic is only used during initial `azd app run` orchestration, not for individual service operations.

## Proposed Solution

### Phase 1: Add ContainerID to Registry

Add `ContainerID` field to track container services:

```go
type ServiceRegistryEntry struct {
    // ... existing fields
    ContainerID string `json:"containerId,omitempty"` // Container ID for container services
}
```

Update orchestrator to persist ContainerID when registering container services.

### Phase 2: Update Service Operations to Handle Containers

Modify `service_operations.go` to check service type and use appropriate stop/start methods:

```go
func (h *serviceOperationHandler) stopService(entry *registry.ServiceRegistryEntry, serviceName string) error {
    // Check if this is a container service
    if entry.Type == service.ServiceTypeContainer && entry.ContainerID != "" {
        return h.stopContainerService(entry, serviceName)
    }
    
    // Existing native process logic
    // ...
}

func (h *serviceOperationHandler) stopContainerService(entry *registry.ServiceRegistryEntry, serviceName string) error {
    client := docker.NewClient()
    
    if err := client.Stop(entry.ContainerID, 10); err != nil {
        log.Printf("Warning: failed to stop container: %v", err)
    }
    
    // Remove container for ephemeral dev containers
    if err := client.Remove(entry.ContainerID); err != nil {
        log.Printf("Warning: failed to remove container: %v", err)
    }
    
    return nil
}
```

Similarly for start operations:

```go
func (h *serviceOperationHandler) performStartBulk(...) error {
    // ...detect runtime...
    
    if runtime.Type == service.ServiceTypeContainer {
        process, err = service.StartContainerService(runtime, h.server.projectDir, true) // restartContainers=true
        if err == nil && process.ContainerID != "" {
            // Store ContainerID in registry
        }
    } else {
        process, err = service.StartService(runtime, envVars, h.server.projectDir, functionsParser)
    }
}
```

### Phase 3: Update CLI Service Controller

Apply same changes to `service_control.go` for CLI commands (azd app start, azd app stop, azd app restart).

### Phase 4: Container State Reconciliation

Add logic to reconcile registry state with actual Docker container state on dashboard startup:

```go
func (h *serviceOperationHandler) reconcileContainerState(serviceName string, entry *registry.ServiceRegistryEntry) {
    if entry.Type != service.ServiceTypeContainer {
        return
    }
    
    client := docker.NewClient()
    containerName := fmt.Sprintf("azd-%s", serviceName)
    
    // Check if container exists and is running
    if container, err := client.InspectByName(containerName); err == nil && container != nil {
        if client.IsRunning(container.ID) {
            // Container is running - update registry
            entry.ContainerID = container.ID
            entry.Status = constants.StatusRunning
        } else {
            // Container exists but stopped
            entry.Status = constants.StatusStopped
        }
    }
}
```

## Files to Modify

1. `cli/src/internal/registry/registry.go` - Add ContainerID field
2. `cli/src/internal/dashboard/service_operations.go` - Container-aware stop/start
3. `cli/src/cmd/app/commands/service_control.go` - Container-aware stop/start
4. `cli/src/internal/service/orchestrator.go` - Persist ContainerID to registry

## Testing

1. Start services with `azd app run` where one service is a container (e.g., Azurite)
2. Click "Restart" on the container service - should stop and restart
3. Click "Restart All" - all services including containers should restart
4. Click "Stop" on container - should stop the Docker container
5. Click "Start" on stopped container - should start a new container

## Success Criteria

- Container services can be individually started, stopped, and restarted from dashboard
- "Restart All" properly handles mixed native and container services
- CLI commands (`azd app stop`, `azd app start`, `azd app restart`) work for containers
- Container state is properly tracked in registry
- No orphaned containers after operations

---

## Design Review: Issues Identified

### Issue A: ContainerID Storage is Unnecessary

The original proposal to add `ContainerID` to registry is not needed:
- Container name is deterministic: `azd-{serviceName}`
- We can find/stop/restart containers by name using `docker.InspectByName("azd-{serviceName}")`
- No state to lose or synchronize

**Resolution**: Use container name convention instead of storing ContainerID.

### Issue B: `Type` Field Already Exists in Registry

The `ServiceRegistryEntry.Type` field already exists and the orchestrator already populates it with `"container"` for container services. The fix is simpler than originally stated - just check `entry.Type`.

### Issue C: Process Validation Breaks for Containers

In `performStartBulk` and `performStart`:
```go
if process == nil || process.Process == nil {
    return fmt.Errorf("service process not created")
}
```

Container services have `process.Process = nil` (they use `process.ContainerID`). This validation must be updated to handle both cases.

### Issue D: Registry Entry Loses Type/Mode on Restart

When creating `updatedEntry` after start, `Type` and `Mode` fields are not preserved. This loses the `"container"` type information, breaking subsequent operations.

### Issue E: Port Killing is Dangerous for Containers

`stopService()` falls back to `pm.KillProcessOnPort(entry.Port)`. For Docker, this might kill Docker's port forwarding proxy instead of the container. Container services should use `docker stop` by name, not port killing.

### Issue F: No Container Log Collection on Restart

The orchestrator calls `StartContainerLogCollection()` but `performStartBulk`/`performStart` don't. Restarted containers won't have logs visible in the dashboard.

### Issue G: restartContainers Parameter Not Specified

`StartContainerService(runtime, projectDir, restartContainers)` has a boolean parameter. The spec should specify: use `true` for restart/start operations to ensure fresh containers.

### Issue H: No State Reconciliation Timing

The spec mentions reconciliation but doesn't specify when it happens. If a container crashes externally, the registry shows stale state. Need to decide: reconcile on dashboard load? On health check? On demand?

---

## Revised Solution

### Phase 1: Update Stop Logic for Containers

Modify `stopService()` in both `service_operations.go` and `service_control.go`:

```go
func (h *serviceOperationHandler) stopService(entry *registry.ServiceRegistryEntry, serviceName string) error {
    // Container services: use Docker stop by name
    if entry.Type == service.ServiceTypeContainer {
        return h.stopContainerByName(serviceName)
    }
    
    // Native processes: existing PID + port killing logic
    // ...existing code...
}

func (h *serviceOperationHandler) stopContainerByName(serviceName string) error {
    client := docker.NewClient()
    containerName := fmt.Sprintf("azd-%s", serviceName)
    
    if err := client.Stop(containerName, 10); err != nil {
        log.Printf("Warning: failed to stop container: %v", err)
    }
    if err := client.Remove(containerName); err != nil {
        log.Printf("Warning: failed to remove container: %v", err)
    }
    return nil
}
```

### Phase 2: Update Start Logic for Containers

Modify `performStartBulk()` and `performStart()`:

```go
func (h *serviceOperationHandler) performStartBulk(...) error {
    // ...existing setup...
    
    var process *service.ServiceProcess
    var err error
    
    if runtime.Type == service.ServiceTypeContainer {
        process, err = service.StartContainerService(runtime, h.server.projectDir, true)
        if err == nil {
            // Start log collection for container
            if logErr := service.StartContainerLogCollection(process, h.server.projectDir); logErr != nil {
                log.Printf("Warning: failed to start container log collection: %v", logErr)
            }
        }
    } else {
        process, err = service.StartService(runtime, envVars, h.server.projectDir, functionsParser)
    }
    
    // Updated validation - check ContainerID for containers
    if process == nil {
        return fmt.Errorf("service process not created")
    }
    if runtime.Type != service.ServiceTypeContainer && process.Process == nil {
        return fmt.Errorf("native service process not created")
    }
    if runtime.Type == service.ServiceTypeContainer && process.ContainerID == "" {
        return fmt.Errorf("container not created")
    }
    
    // Update registry - preserve Type and Mode
    updatedEntry := &registry.ServiceRegistryEntry{
        // ...existing fields...
        Type:        runtime.Type,
        Mode:        runtime.Mode,
    }
    // Set PID only for native processes
    if process.Process != nil {
        updatedEntry.PID = process.Process.Pid
    }
    
    return reg.Register(updatedEntry)
}
```

### Phase 3: Apply Same Changes to CLI service_control.go

Apply identical logic updates to `performStart()` and `performStop()` in CLI.

### Phase 4: Optional State Reconciliation

Add reconciliation on service list retrieval:

```go
func (h *serviceOperationHandler) reconcileContainerStates() {
    client := docker.NewClient()
    reg := registry.GetRegistry(h.server.projectDir)
    
    for _, entry := range reg.ListAll() {
        if entry.Type != service.ServiceTypeContainer {
            continue
        }
        
        containerName := fmt.Sprintf("azd-%s", entry.Name)
        container, err := client.InspectByName(containerName)
        
        if err != nil || container == nil {
            // Container doesn't exist
            if entry.Status == constants.StatusRunning {
                reg.UpdateStatus(entry.Name, constants.StatusStopped)
            }
        } else if client.IsRunning(container.ID) {
            if entry.Status != constants.StatusRunning {
                reg.UpdateStatus(entry.Name, constants.StatusRunning)
            }
        } else {
            // Container exists but not running
            if entry.Status == constants.StatusRunning {
                reg.UpdateStatus(entry.Name, constants.StatusStopped)
            }
        }
    }
}
```

## Updated Files to Modify

1. `cli/src/internal/dashboard/service_operations.go` - Container-aware stop/start
2. `cli/src/cmd/app/commands/service_control.go` - Container-aware stop/start  
3. ~~`cli/src/internal/registry/registry.go`~~ - NOT NEEDED (Type field exists)
4. ~~`cli/src/internal/service/orchestrator.go`~~ - NOT NEEDED (already sets Type)
