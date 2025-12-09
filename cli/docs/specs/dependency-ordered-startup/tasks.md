<!-- NEXT: -->
# Dependency-Ordered Service Startup Tasks

All tasks complete.

## Done

### DONE: Write Integration Tests {#write-integration-tests}

Added tests in `orchestrator_test.go`:
- `TestTopologicalSort_NoDependencies` - services with no deps start in level 0
- `TestTopologicalSort_LinearDependency` - linear chain (frontend → api → db)
- `TestTopologicalSort_DiamondDependency` - diamond pattern dependencies
- `TestTopologicalSort_ContainerDependencies` - container-test pattern (api uses 4 containers)
- `TestTopologicalSort_MixedDependencies` - mix of dependency depths
- `TestTopologicalSort_EmptyServices` - edge case handling
- `TestWaitForServiceHealthy_HealthCheckDisabled` - disabled health check returns immediately
- `TestWaitForServiceHealthy_HealthCheckTypeNone` - type=none returns immediately
- `TestGetServiceDependencies` - verify dependency retrieval
- `TestGetDependents` - verify dependent service retrieval
- `TestFilterGraphByServices` - transitive dependency inclusion

### DONE: Update OrchestrateServices Signature {#update-orchestrateservices-signature}

Added `services map[string]Service` parameter to `OrchestrateServices` function.
Resources parameter not needed since container dependencies are services in azure.yaml.

### DONE: Implement Level-Based Service Startup {#implement-level-based-startup}

Modified `OrchestrateServices` to:
- Build dependency graph using `BuildDependencyGraph(services, nil)`
- Get startup levels via `TopologicalSort(graph)`
- Start services level by level (parallel within each level)
- Wait for all services in level N to be healthy before starting level N+1

### DONE: Add waitForServiceHealthy Function {#add-wait-healthy-function}

Created `waitForServiceHealthy(name, process, svc, timeout)` function that:
- Returns immediately if health check is disabled for the service
- Uses existing `PerformHealthCheck` with exponential backoff
- Returns error on timeout or health check failure

### DONE: Update Registry Status Progression {#update-registry-status}

Kept existing status progression (starting → running). Health is determined dynamically
by the health check system. No new "healthy" status needed since `process.Ready = true`
indicates health check passed.

### DONE: Update Run Command {#update-run-command}

Updated `executeAndMonitorServices` in run.go to pass `azureYaml.Services` to
`OrchestrateServices`.
