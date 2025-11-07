# AZD Context Inheritance

This document explains how `azd app` ensures that all child processes properly inherit the azd context through environment variables.

## Overview

When running as an `azd` extension, this CLI tool receives azd context through environment variables set by the parent `azd` process. These environment variables must be inherited by all child processes (services, installers, build tools, etc.) to enable:

1. **Azure connectivity**: Access to Azure subscription, resource groups, and deployed resources
2. **Extension framework communication**: gRPC communication with azd via `AZD_SERVER` and `AZD_ACCESS_TOKEN`
3. **Environment values**: All environment-specific configuration from `azd env`

## Key Environment Variables

### Core AZD Extension Variables

- `AZD_SERVER`: gRPC server address for azd communication (e.g., `localhost:12345`)
- `AZD_ACCESS_TOKEN`: JWT token for authenticating with azd extension framework APIs

### Azure Environment Variables

- `AZURE_SUBSCRIPTION_ID`: Current Azure subscription
- `AZURE_RESOURCE_GROUP_NAME`: Target resource group
- `AZURE_ENV_NAME`: Environment name (e.g., dev, staging, prod)
- `AZURE_LOCATION`: Azure region
- `SERVICE_*_URL`: Auto-generated URLs for deployed services
- All other variables from `azd env get-values`

## Implementation Pattern

All code that spawns child processes MUST use one of the following patterns to ensure azd context is inherited:

### Pattern 1: Using executor package (Preferred)

The `executor` package automatically inherits all environment variables:

```go
import "github.com/jongio/azd-app/cli/src/internal/executor"

// All executor functions automatically call cmd.Env = os.Environ()
err := executor.StartCommand(ctx, "dotnet", []string{"run"}, projectDir)
err := executor.RunCommand(ctx, "npm", []string{"install"}, projectDir)
```

### Pattern 2: Direct exec.Command with explicit inheritance

When using `exec.Command` directly, explicitly set `cmd.Env`:

```go
import "os/exec"

cmd := exec.Command("python", "app.py")
cmd.Dir = projectDir
cmd.Env = os.Environ() // REQUIRED: Inherit azd context
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr

if err := cmd.Run(); err != nil {
    return fmt.Errorf("failed: %w", err)
}
```

### Pattern 3: Service execution with environment merging

Services receive a merged environment that includes azd context. The `OrchestrateServices` function
builds the environment in the following priority order (highest to lowest):

1. **Runtime-specific environment** (from azure.yaml service.env)
2. **Custom .env file variables** (from --env-file flag)  
3. **OS environment** (includes all azd context: AZD_*, AZURE_*)

```go
// In OrchestrateServices, each service receives:
// 1. Start with os.Environ() - includes azd context
serviceEnv := make(map[string]string)
for _, e := range os.Environ() {
    pair := strings.SplitN(e, "=", 2)
    if len(pair) == 2 {
        serviceEnv[pair[0]] = pair[1]
    }
}
// 2. Merge custom --env-file variables
for k, v := range envVars {
    serviceEnv[k] = v
}
// 3. Merge service-specific variables (highest priority)
for k, v := range rt.Env {
    serviceEnv[k] = v
}

// createServiceCommand converts the map to environment slice
process, err := service.StartService(runtime, serviceEnv, projectDir)
```

## Files Implementing Azd Context Inheritance

### Service Execution

- **`cli/src/internal/service/orchestrator.go`**
  - `OrchestrateServices()`: Builds environment for each service starting with `os.Environ()` to inherit azd context
  - Merges custom env vars from --env-file and runtime-specific variables
  - Ensures all services receive AZD_*, AZURE_* variables

- **`cli/src/internal/service/executor.go`**
  - `createServiceCommand()`: Creates commands with merged environment that includes azd context
  - All service processes receive full azd environment

- **`cli/src/internal/service/env.go`**
  - `ResolveEnvironment()`: Merges environment variables starting with `os.Environ()` which includes azd context
  - Preserves `AZD_*` and `AZURE_*` variables through all merging operations

### Dependency Installation

- **`cli/src/internal/installer/installer.go`**
  - All `exec.Command` calls include `cmd.Env = os.Environ()`
  - Ensures dependency installers (npm, pip, poetry, uv, dotnet) have azd context

### Command Execution

- **`cli/src/internal/executor/executor.go`**
  - `RunCommand()`: Sets `cmd.Env = os.Environ()` for all commands
  - `StartCommand()`: Sets `cmd.Env = os.Environ()` for long-running processes
  - `StartCommandWithOutputMonitoring()`: Sets `cmd.Env = os.Environ()` with output capture

### Runners

- **`cli/src/internal/runner/runner.go`**
  - All runner functions use the `executor` package which handles environment inheritance
  - `RunAspire()`, `RunNode()`, `RunPython()`, `RunDotnet()` all inherit azd context

## Verification

To verify azd context is properly inherited, you can:

1. **Check environment in your application**:
   ```python
   # Python example
   import os
   print(f"AZD_SERVER: {os.getenv('AZD_SERVER')}")
   print(f"AZURE_SUBSCRIPTION_ID: {os.getenv('AZURE_SUBSCRIPTION_ID')}")
   ```

2. **Enable verbose logging**:
   ```bash
   azd app run --verbose
   ```

3. **Check test files**:
   - `cli/tests/projects/aspire-test/TestAppHost/AppHost.cs` demonstrates azd context verification

## Common Pitfalls

### ❌ DON'T: Create empty environment

```go
// This loses azd context!
cmd := exec.Command("python", "app.py")
cmd.Env = []string{} // WRONG: No azd context
```

### ❌ DON'T: Forget to set Env

```go
// This uses a default environment that may not include azd context
cmd := exec.Command("python", "app.py")
// Missing: cmd.Env = os.Environ()
cmd.Run()
```

### ✅ DO: Always inherit from os.Environ()

```go
// Correct: Inherit azd context
cmd := exec.Command("python", "app.py")
cmd.Env = os.Environ()
cmd.Run()
```

### ✅ DO: Use executor package when possible

```go
// Best: Use executor which handles environment automatically
executor.StartCommand(ctx, "python", []string{"app.py"}, dir)
```

## Testing

When writing tests that spawn processes, consider:

1. Set up test environment variables for `AZD_*` and `AZURE_*` keys
2. Use `t.Setenv()` in Go tests to set environment variables
3. Verify child processes receive expected environment variables

## Reference

For more information about the azd extension framework, see:
- [Azure Developer CLI Extension Framework Documentation](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md)
- [Environment Service gRPC API](https://github.com/Azure/azure-dev/blob/main/cli/azd/grpc/proto/environment.proto)
