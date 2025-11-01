# AZD Context and Environment Variables

## Overview

When azd invokes an extension command, it automatically passes several environment variables that provide context about the current azd environment and project.

## Environment Variables Provided by AZD

According to the [azd extension framework documentation](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md#invoking-extension-commands), azd automatically sets these environment variables when invoking extension commands:

### Core Context Variables

1. **`AZD_SERVER`** - The address of the gRPC server (e.g., `localhost:12345`)
   - Used for extensions that need to communicate back to azd via gRPC
   - Format: `hostname:port`

2. **`AZD_ACCESS_TOKEN`** - JWT token for authenticating with the azd gRPC server
   - Signed with a randomly generated key
   - Valid for the lifetime of the command
   - Includes claims identifying the extension and its capabilities

3. **AZD Environment Variables** - All environment variables from the current azd environment are also passed
   - These include user-defined variables from `.env` files
   - Azure-specific variables like subscription IDs, resource group names, etc.

## How to Access AZD Context

### In Go Code

All environment variables are automatically available via `os.Getenv()`:

```go
import "os"

func someCommand() error {
    // Get AZD server address (for gRPC communication)
    azdServer := os.Getenv("AZD_SERVER")
    
    // Get AZD access token (for authentication)
    azdToken := os.Getenv("AZD_ACCESS_TOKEN")
    
    // Get any azd environment variable
    subscriptionId := os.Getenv("AZURE_SUBSCRIPTION_ID")
    resourceGroup := os.Getenv("AZURE_RESOURCE_GROUP_NAME")
    
    // Use the values...
    return nil
}
```

### Current Implementation

**The extension already has access to all azd environment variables** - no code changes are needed! When azd invokes any command in this extension (like `azd app reqs`), all environment variables are automatically available in the process.

This means:
- ✅ Commands can access `AZURE_SUBSCRIPTION_ID`
- ✅ Commands can access `AZURE_RESOURCE_GROUP_NAME`
- ✅ Commands can access any user-defined environment variables from the azd environment
- ✅ Commands can access `AZD_SERVER` and `AZD_ACCESS_TOKEN` if they need gRPC communication

## Example Use Cases

### 1. Accessing Azure Context

```go
func deployCommand() error {
    subscriptionId := os.Getenv("AZURE_SUBSCRIPTION_ID")
    if subscriptionId == "" {
        return fmt.Errorf("AZURE_SUBSCRIPTION_ID not set - run 'azd env refresh' first")
    }
    
    resourceGroup := os.Getenv("AZURE_RESOURCE_GROUP_NAME")
    location := os.Getenv("AZURE_LOCATION")
    
    fmt.Printf("Deploying to:\n")
    fmt.Printf("  Subscription: %s\n", subscriptionId)
    fmt.Printf("  Resource Group: %s\n", resourceGroup)
    fmt.Printf("  Location: %s\n", location)
    
    return nil
}
```

### 2. Using gRPC to Query AZD

For advanced scenarios where you need to query azd's internal state, you can use the gRPC client:

```go
import (
    "github.com/azure/azure-dev/cli/azd/pkg/azdext"
)

func queryAzdProject() error {
    ctx := azdext.WithAccessToken(context.Background())
    
    azdClient, err := azdext.NewAzdClient()
    if err != nil {
        return fmt.Errorf("failed to create azd client: %w", err)
    }
    defer azdClient.Close()
    
    // Query current project
    projectResp, err := azdClient.Project().Get(ctx, &azdext.EmptyRequest{})
    if err != nil {
        return err
    }
    
    fmt.Printf("Project: %s\n", projectResp.Project.Name)
    return nil
}
```

## Common AZD Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `AZURE_SUBSCRIPTION_ID` | Azure subscription ID | `12345678-1234-1234-1234-123456789abc` |
| `AZURE_RESOURCE_GROUP_NAME` | Resource group name | `rg-myapp-dev` |
| `AZURE_LOCATION` | Azure region | `eastus` |
| `AZURE_TENANT_ID` | Azure tenant ID | `87654321-4321-4321-4321-987654321cba` |
| `AZURE_ENV_NAME` | AZD environment name | `dev` |
| `AZURE_PRINCIPAL_ID` | Service principal ID (if using service principal) | `...` |

## Best Practices

1. **Check for Required Variables**: Always check if critical environment variables are set before using them
2. **Provide Helpful Error Messages**: If a variable is missing, tell the user what to do (e.g., "Run 'azd env refresh' first")
3. **Don't Assume Variables Exist**: Not all azd commands require an environment to be set up
4. **Use gRPC Sparingly**: For most use cases, environment variables are sufficient. Only use gRPC when you need to query or modify azd's internal state

## References

- [AZD Extension Framework Documentation](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md)
- [AZD gRPC Services](https://github.com/Azure/azure-dev/blob/main/cli/azd/grpc/proto)
