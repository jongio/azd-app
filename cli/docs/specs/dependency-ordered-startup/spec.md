# Dependency-Ordered Service Startup

## Overview

Update the `azd app run` command to honor the `uses` field in azure.yaml, starting services in dependency order and waiting for dependencies to become healthy before starting dependent services.

## Problem

Currently, `OrchestrateServices` starts all services in parallel without respecting the `uses` dependencies. This causes issues when:
- An API service depends on a database container
- The API starts before the database is ready to accept connections
- Connection errors occur during startup

Example from azure.yaml:
```yaml
services:
  postgres:
    image: postgres:16-alpine
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      
  api:
    project: ./api
    uses:
      - postgres  # API should wait for postgres to be healthy
```

## Solution

Use the existing `BuildDependencyGraph` and `TopologicalSort` functions to:
1. Build dependency graph from services and their `uses` fields
2. Group services into startup levels (level 0 = no deps, level 1 = depends on level 0, etc.)
3. Start services level by level
4. Wait for all services in a level to become **healthy** before starting the next level

### Healthy vs Started

Wait for **healthy** status rather than just **started** because:
- A database being "started" doesn't mean it's accepting connections
- Container services may take time after process start to become functional
- Health checks are already implemented and configurable per-service

### Implementation Changes

#### 1. Update OrchestrateServices signature

Add `services` and `resources` parameters to build the dependency graph:
```go
func OrchestrateServices(
    runtimes []*ServiceRuntime,
    services map[string]Service,    // NEW: for dependency graph
    resources map[string]Resource,  // NEW: for dependency graph  
    envVars map[string]string,
    logger *ServiceLogger,
    restartContainers bool,
) (*OrchestrationResult, error)
```

#### 2. Modify orchestration logic

```go
// Build dependency graph
graph, err := BuildDependencyGraph(services, resources)
if err != nil {
    return nil, fmt.Errorf("failed to build dependency graph: %w", err)
}

// Get services grouped by startup level
levels := TopologicalSort(graph)

// Start services level by level
for levelNum, serviceNames := range levels {
    // Start all services in this level in parallel
    for _, name := range serviceNames {
        runtime := runtimeMap[name]
        wg.Add(1)
        go startService(runtime, ...)
    }
    wg.Wait()
    
    // Wait for all services in this level to become healthy
    for _, name := range serviceNames {
        if err := waitForHealthy(name); err != nil {
            return nil, fmt.Errorf("dependency %s failed to become healthy: %w", name, err)
        }
    }
}
```

#### 3. Add health waiting function

Create a function to wait for a service to become healthy with timeout:
```go
func waitForServiceHealthy(name string, processes map[string]*ServiceProcess, timeout time.Duration) error
```

#### 4. Update run command

Pass `azureYaml.Services` and `azureYaml.Resources` to `OrchestrateServices`.

### Edge Cases

1. **No dependencies**: Services without `uses` start in level 0 (parallel)
2. **Missing dependency**: Error during graph building (already handled)
3. **Circular dependency**: Error during graph building (already handled)
4. **Health check disabled**: Service considered healthy when started
5. **Container services**: Use Docker health check status
6. **Service filter**: Only include filtered services and their transitive dependencies

### Registry Status Updates

Update registry status progression:
- `starting` - Process launched
- `running` - Process running, health checks in progress
- `healthy` - Health check passed (new status)
- `error` - Failed to start or health check failed

## Success Criteria

1. Services with no dependencies start immediately (parallel)
2. Services with dependencies wait for dependencies to be healthy
3. Clear error messages when a dependency fails health check
4. Timeout handling when a dependency never becomes healthy
5. Existing behavior preserved when no `uses` fields present

## Out of Scope

- Dynamic dependency resolution at runtime
- Circular dependency breaking strategies
- Resource provisioning (only service startup order)
