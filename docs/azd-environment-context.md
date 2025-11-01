# Azure Developer CLI (azd) Environment Context

## Overview

When `azd` invokes an extension command, it automatically sets up an environment context that includes critical variables for interacting with the Azure Developer CLI ecosystem. This document explains how the App extension ensures this context is properly propagated to all child processes.

## Environment Variables Set by azd

According to the [Azure Developer CLI Extension Framework documentation](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md), when azd launches an extension, it sets:

### Core Extension Variables

1. **`AZD_SERVER`** - The gRPC server address and port (e.g., `localhost:12345`)
   - Used for bidirectional communication between the extension and azd core
   - Required for accessing azd services (Project, Environment, Deployment, etc.)

2. **`AZD_ACCESS_TOKEN`** - JWT authentication token
   - Signed with a randomly generated key valid for the command's lifetime
   - Includes claims identifying the extension and its capabilities
   - Must be included in gRPC requests via the `authorization` header

### Environment-Specific Variables

3. **Environment variables from the current azd environment**
   - All key-value pairs from the active azd environment (e.g., `.azure/<env-name>/.env`)
   - Includes Azure subscription IDs, resource group names, deployment outputs, etc.
   - Critical for commands that need access to deployment context

## Implementation in App Extension

### Executor Package

The `executor` package ensures all spawned commands inherit the complete environment context:

```go
// All executor functions set cmd.Env = os.Environ()
func RunWithContext(ctx context.Context, name string, args []string, dir string) error {
    cmd := exec.CommandContext(ctx, name, args...)
    cmd.Dir = dir
    cmd.Env = os.Environ() // ← Inherit all environment variables
    // ...
}
```

This pattern is applied to:
- `RunWithContext()` - Context-aware command execution
- `RunCommand()` - Standard command execution with timeout
- `StartCommand()` - Background process launching (for servers, Aspire, etc.)
- `RunCommandWithOutput()` - Command execution with output capture
- `StartCommandWithOutputMonitoring()` - Monitored background processes

### Why This Matters

Without explicit environment inheritance, child processes would:

❌ Lose access to azd gRPC services (no `AZD_SERVER` or `AZD_ACCESS_TOKEN`)
❌ Lack deployment context (missing subscription IDs, resource names, etc.)
❌ Be unable to interact with the azd ecosystem
❌ Fail to access environment-specific configuration

With proper environment inheritance, child processes can:

✅ Call back to azd services via gRPC
✅ Access deployment context and Azure resources
✅ Use azd environment variables in their operations
✅ Maintain the full execution context chain

## Use Cases in App Extension

### Running Development Servers

When `azd app run` launches a development server (Aspire, Node.js, Python, etc.), the server process needs access to:

- Azure connection strings (from azd environment)
- Service endpoints (from deployment outputs)
- Configuration values (from environment variables)

Example:
```bash
azd app run
  ↓ (inherits AZD_SERVER, AZD_ACCESS_TOKEN, AZURE_*, etc.)
  dotnet run --project ./AppHost
    ↓ (Aspire host can access azd environment)
    Services can use Azure resources configured by azd
```

### Installing Dependencies

When `azd app deps` runs package managers, they may need environment context for:

- Private package registries requiring authentication
- Build scripts that access Azure resources
- Post-install scripts that configure services

### Checking Requirements

When `azd app reqs` detects tools, environment context helps:

- Determine which tools are needed based on deployment configuration
- Validate versions match deployment requirements
- Check for Azure-specific tooling

## Testing Environment Inheritance

To verify environment variables are properly inherited:

```bash
# Set a test variable
$env:TEST_AZD_VAR = "test-value"

# Run a command that echoes environment
azd app run  # Should see TEST_AZD_VAR in child process environment
```

## Best Practices

1. **Always use executor package functions** instead of raw `exec.Command()`
   - Ensures consistent environment handling
   - Provides timeout protection
   - Includes security validation

2. **Don't manually set cmd.Env** unless adding specific variables
   - Use `os.Environ()` as the base
   - Append additional variables if needed:
     ```go
     cmd.Env = append(os.Environ(), "CUSTOM_VAR=value")
     ```

3. **Document environment dependencies** in commands
   - Note if a command requires specific azd environment variables
   - Validate critical variables are present before execution

4. **Test with azd lifecycle events**
   - Verify environment propagation during preprovision, predeploy, etc.
   - Ensure spawned processes can access deployment context

## Related Documentation

- [Azure Developer CLI Extension Framework](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md)
- [Extension Invoking Documentation](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md#invoking-extension-commands)
- [gRPC Services](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md#grpc-services)
- [Executor Package](../src/internal/executor/executor.go)

## Summary

The App extension ensures full azd context propagation by explicitly setting `cmd.Env = os.Environ()` in all command execution paths. This critical implementation detail enables child processes to maintain connectivity with azd core and access deployment context, making the extension a true member of the azd ecosystem.
