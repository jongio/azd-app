# Listen Command - Extension Framework Integration

## Overview

The `listen` command is a hidden command that enables proper integration with the Azure Developer CLI (azd) extension framework. It's automatically invoked by azd during various lifecycle operations and prevents capability conflicts.

## Purpose

When you install an azd extension, azd may attempt to invoke it during operations like `azd provision`, `azd deploy`, etc. The `listen` command:

1. **Establishes a gRPC connection** with azd's extension framework
2. **Registers the extension's actual capabilities** to prevent misuse
3. **Prevents azd from trying to use the extension for unsupported operations**

## Why It's Needed

Without the `listen` command, azd may try to invoke your extension for capabilities it doesn't support. For example:

- Extension declares only `custom-commands` capability in `extension.yaml`
- User runs `azd provision` (which needs infrastructure deployment)
- azd tries to invoke ALL installed extensions as potential service-target-providers
- **Error**: "extension does not support service-target-provider capability"

## Implementation

The `listen` command creates an `ExtensionHost` with NO service targets or framework services registered:

```go
// Create a context with the AZD access token
ctx := azdext.WithAccessToken(cmd.Context())

// Create a new AZD client
azdClient, err := azdext.NewAzdClient()
if err != nil {
    return fmt.Errorf("failed to create azd client: %w", err)
}
defer azdClient.Close()

// Create an extension host with NO capabilities registered
// This tells azd we only support custom-commands (declared in extension.yaml)
// and prevents azd from trying to invoke us for service-target-provider
host := azdext.NewExtensionHost(azdClient)

// Start the extension host - blocks until azd closes the connection
if err := host.Run(ctx); err != nil {
    return fmt.Errorf("failed to run extension: %w", err)
}
```

## Command Characteristics

- **Hidden**: Not shown in `azd app --help` (uses `Hidden: true` in Cobra)
- **Invoked by azd**: Never called directly by users
- **Blocking**: Runs until azd closes the connection
- **Required**: Must exist for extensions declaring any lifecycle capabilities

## Extension Capabilities

The App extension declares only `custom-commands` in `extension.yaml`:

```yaml
capabilities:
  - custom-commands
```

By creating an empty `ExtensionHost` in the `listen` command, we ensure azd respects this capability declaration and doesn't try to use our extension for:

- `service-target-provider` (deploying services to Azure)
- `framework-service-provider` (building/packaging projects)
- `lifecycle-events` (hooking into azd's provision/deploy lifecycle)

## Testing the Fix

Before the `listen` command was added:

```bash
$ azd provision
ERROR: failed to register service target 'demo': extension does not support service-target-provider capability
```

After adding the `listen` command:

```bash
$ azd provision
Provisioning Azure resources...
# Works correctly - no capability conflict
```

## References

- [Azure Developer CLI Extension Framework Documentation](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md)
- [ExtensionHost Builder Pattern](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md#complete-extension-host-builder-pattern)
- [Extension Capabilities](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md#extension-capabilities)
