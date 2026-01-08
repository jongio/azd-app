# Azure Key Vault Reference Resolution

## Overview

azd-app now supports automatic Azure Key Vault reference resolution for environment variables across all commands. This allows you to store secrets in Azure Key Vault and reference them using standardized formats, with automatic resolution at runtime.

## Supported Reference Formats

### Format 1: SecretUri (Full URI)

```bash
@Microsoft.KeyVault(SecretUri=https://vault.vault.azure.net/secrets/name[/version])
```

Example:
```yaml
services:
  api:
    environment:
      DB_PASSWORD: "@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/db-password)"
```

### Format 2: VaultName + SecretName

```bash
@Microsoft.KeyVault(VaultName=vault;SecretName=name[;SecretVersion=version])
```

Example:
```yaml
services:
  api:
    environment:
      API_KEY: "@Microsoft.KeyVault(VaultName=myvault;SecretName=api-key)"
```

### Format 3: azd akvs Format

```bash
akvs://<guid>/<vault>/<secret>[/<version>]
```

Example:
```yaml
services:
  api:
    environment:
      SECRET_KEY: "akvs://00000000-0000-0000-0000-000000000000/myvault/secret-key"
```

**Note:** The GUID is informational only; the vault name is used to construct the actual vault URL.

## Authentication

Key Vault resolution uses **DefaultAzureCredential**, which tries the following methods in order:

1. Environment variables (`AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_CLIENT_SECRET`)
2. Workload Identity (Kubernetes)
3. Managed Identity (Azure VMs/services)
4. Azure CLI (`az login`)
5. Azure PowerShell
6. Interactive browser

**This is the same authentication mechanism used by azd** - no additional setup is required if you're already authenticated with Azure!

## Usage Examples

### In azure.yaml

```yaml
name: my-app

services:
  api:
    language: python
    host: appservice
    environment:
      # Regular environment variable
      LOG_LEVEL: "debug"
      
      # Key Vault references (automatically resolved at runtime)
      DB_PASSWORD: "@Microsoft.KeyVault(VaultName=myvault;SecretName=db-password)"
      API_KEY: "@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/api-key)"
```

### In .env files

```.env
# .env.local
LOG_LEVEL=debug
DB_PASSWORD=@Microsoft.KeyVault(VaultName=myvault;SecretName=db-password)
API_KEY=@Microsoft.KeyVault(VaultName=myvault;SecretName=api-key)
```

Then use with:
```bash
azd app run --env-file .env.local
```

### Using azd env

```bash
# Set a Key Vault reference in azd environment
azd env set DB_PASSWORD "@Microsoft.KeyVault(VaultName=myvault;SecretName=db-password)"

# Run your application - the reference will be automatically resolved
azd app run
```

## Error Handling

### Graceful Degradation (Default)

By default, Key Vault resolution uses **graceful degradation**:

- If Azure credentials are not available, a warning is logged and the original reference value is used
- If a vault or secret is not found, a warning is logged and the original reference value is used
- If resolution fails for any reason, execution continues with warnings

This ensures your application can still run during development or in offline scenarios.

Example warning output:
```
Warning: failed to resolve Key Vault reference for DB_PASSWORD: failed to get secret db-password from vault myvault: vault not found
```

### Fail-Fast Mode

Currently, azd-app uses graceful degradation mode only. Fail-fast mode (stopping execution on Key Vault errors) is not exposed but can be added if needed.

## Performance

- **Client Caching**: Key Vault clients are cached per vault URL to avoid repeated authentication
- **Early Return**: If no Key Vault references are detected, overhead is <1ms
- **First Resolution**: ~100-500ms (includes authentication + HTTPS)
- **Cached Client**: ~50-100ms (HTTPS only, client reused)

## Security Considerations

### Strengths ✅

- No secrets stored in code or configuration files
- Azure RBAC for access control
- Audit trail in Key Vault audit logs
- Industry-standard DefaultAzureCredential
- Graceful failure without exposing secrets

### Considerations ⚠️

- Warning messages may reveal vault/secret names to stderr
- Ensure stderr is secure in production environments
- Monitor Key Vault audit logs for access patterns
- Consider environment variable visibility in process listings

## Azure Key Vault Setup

### 1. Create a Key Vault

```bash
# Create a resource group (if needed)
az group create --name my-rg --location eastus

# Create a Key Vault
az keyvault create --name myvault --resource-group my-rg --location eastus
```

### 2. Add Secrets

```bash
# Add a secret
az keyvault secret set --vault-name myvault --name db-password --value "MySecurePassword123!"

# Add another secret
az keyvault secret set --vault-name myvault --name api-key --value "sk-1234567890abcdef"
```

### 3. Grant Access

```bash
# Get your current user's object ID
OBJECT_ID=$(az ad signed-in-user show --query id -o tsv)

# Grant yourself access to secrets
az keyvault set-policy --name myvault \
  --object-id $OBJECT_ID \
  --secret-permissions get list

# For a service principal (e.g., in CI/CD)
az keyvault set-policy --name myvault \
  --spn <service-principal-id> \
  --secret-permissions get list
```

## PowerShell Examples

### Setting up Key Vault

```powershell
# Create a Key Vault
az keyvault create --name myvault --resource-group my-rg --location eastus

# Add secrets
az keyvault secret set --vault-name myvault --name db-password --value "MySecurePassword123!"
az keyvault secret set --vault-name myvault --name api-key --value "sk-1234567890abcdef"

# Grant access
$objectId = (az ad signed-in-user show --query id -o tsv)
az keyvault set-policy --name myvault --object-id $objectId --secret-permissions get list
```

### Using with azd-app

```powershell
# Set environment variable with Key Vault reference
azd env set DB_PASSWORD "@Microsoft.KeyVault(VaultName=myvault;SecretName=db-password)"

# Run application
azd app run
```

## Bash Examples

### Setting up Key Vault

```bash
#!/bin/bash

# Create a Key Vault
az keyvault create --name myvault --resource-group my-rg --location eastus

# Add secrets
az keyvault secret set --vault-name myvault --name db-password --value "MySecurePassword123!"
az keyvault secret set --vault-name myvault --name api-key --value "sk-1234567890abcdef"

# Grant access
OBJECT_ID=$(az ad signed-in-user show --query id -o tsv)
az keyvault set-policy --name myvault \
  --object-id $OBJECT_ID \
  --secret-permissions get list
```

### Using with azd-app

```bash
# Set environment variable with Key Vault reference
azd env set DB_PASSWORD "@Microsoft.KeyVault(VaultName=myvault;SecretName=db-password)"

# Run application
azd app run
```

## Troubleshooting

### "failed to create Key Vault resolver"

**Cause**: No Azure credentials available.

**Solution**: Authenticate with Azure:
```bash
az login
```

### "failed to get secret X from vault Y"

**Possible causes**:
1. Secret doesn't exist in the vault
2. You don't have permissions to access the secret
3. Vault name is incorrect

**Solutions**:
1. Verify the secret exists:
   ```bash
   az keyvault secret show --vault-name myvault --name secret-name
   ```

2. Check your permissions:
   ```bash
   az keyvault show --name myvault --query properties.accessPolicies
   ```

3. Grant yourself access:
   ```bash
   OBJECT_ID=$(az ad signed-in-user show --query id -o tsv)
   az keyvault set-policy --name myvault \
     --object-id $OBJECT_ID \
     --secret-permissions get list
   ```

### "Warning: failed to resolve Key Vault reference"

This is a graceful degradation warning. Your application will continue running with the unresolved reference value. Check:

1. Azure authentication is configured
2. The vault and secret exist
3. You have appropriate permissions

## Best Practices

1. **Use Managed Identity in Production**: Configure Managed Identity for Azure VMs or App Services instead of using credentials

2. **Principle of Least Privilege**: Grant only `get` and `list` permissions for secrets (not `set`, `delete`, etc.)

3. **Use Secret Versions**: For production, reference specific secret versions to ensure consistency:
   ```yaml
   API_KEY: "@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/api-key/abc123)"
   ```

4. **Monitor Access**: Enable Azure Key Vault logging and monitor access patterns

5. **Rotate Secrets Regularly**: Use Azure Key Vault's secret rotation features

6. **Test Graceful Degradation**: Ensure your application handles Key Vault unavailability appropriately

## Implementation Details

### Integration Points

Key Vault resolution is integrated at the environment resolution layer in `ResolveEnvironment()`. This ensures:

- All services automatically get Key Vault resolution
- Works with `azd app run`, `azd app start`, `azd app test`, and hooks
- Environment variables from all sources (azure.yaml, .env files, azd env) are resolved

### Value Normalization

Key Vault references are normalized to handle:
- Wrapper quotes from `azd env export`: `"@Microsoft.KeyVault(...)"`
- Leading/trailing whitespace
- Single and double quotes

This ensures references work correctly regardless of how they're defined.

## Related Documentation

- [Azure Key Vault Documentation](https://docs.microsoft.com/azure/key-vault/)
- [DefaultAzureCredential Documentation](https://docs.microsoft.com/dotnet/api/azure.identity.defaultazurecredential)
- [Azure RBAC for Key Vault](https://docs.microsoft.com/azure/key-vault/general/rbac-guide)
