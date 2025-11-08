# Recommended Bicep Outputs for Azure Integration

This document describes the bicep outputs needed to enable full Azure integration in the azd-app dashboard.

## Overview

The dashboard automatically extracts Azure resource information from environment variables set by azd. To enable all features, configure your bicep templates to output the following values.

## Global Azure Metadata

These outputs provide context about the Azure environment:

```bicep
// Resource group information
output AZURE_RESOURCE_GROUP string = resourceGroup().name
output AZURE_LOCATION string = resourceGroup().location
output AZURE_SUBSCRIPTION_ID string = subscription().subscriptionId

// Optional: For Azure log streaming via Log Analytics
output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = logAnalyticsWorkspace.id

// Optional: For Container Apps
output AZURE_CONTAINER_APP_ENV_ID string = containerAppEnvironment.id
```

## Per-Service Outputs

For each service in your application, output the following:

### Container Apps

```bicep
// Example for an API service deployed to Container Apps
output SERVICE_API_NAME string = apiContainerApp.name
output SERVICE_API_URL string = 'https://${apiContainerApp.properties.configuration.ingress.fqdn}'
output SERVICE_API_RESOURCE_TYPE string = 'containerapp'
output SERVICE_API_IMAGE_NAME string = apiContainerApp.properties.template.containers[0].image
```

### App Service

```bicep
// Example for a web service deployed to App Service
output SERVICE_WEB_NAME string = webAppService.name
output SERVICE_WEB_URL string = 'https://${webAppService.properties.defaultHostName}'
output SERVICE_WEB_RESOURCE_TYPE string = 'appservice'
```

### Azure Functions

```bicep
// Example for a function app
output SERVICE_FUNCTIONS_NAME string = functionApp.name
output SERVICE_FUNCTIONS_URL string = 'https://${functionApp.properties.defaultHostName}'
output SERVICE_FUNCTIONS_RESOURCE_TYPE string = 'function'
```

## Naming Convention

The dashboard uses the following pattern to match services:

1. `SERVICE_{SERVICE_NAME}_{PROPERTY}` - Highest priority
   - Example: `SERVICE_API_URL`, `SERVICE_WEB_NAME`

2. `{SERVICE_NAME}_{PROPERTY}` - Fallback
   - Example: `API_URL`, `WEB_NAME`

**Important:** Service names should match those defined in your `azure.yaml` file.

## Complete Example

Here's a complete example bicep template that outputs all necessary values:

```bicep
param location string = resourceGroup().location
param environmentName string

// Log Analytics for monitoring
resource logAnalytics 'Microsoft.OperationalInsights/workspaces@2022-10-01' = {
  name: '${environmentName}-logs'
  location: location
  properties: {
    sku: {
      name: 'PerGB2018'
    }
  }
}

// Container Apps Environment
resource containerAppEnv 'Microsoft.App/managedEnvironments@2023-05-01' = {
  name: '${environmentName}-env'
  location: location
  properties: {
    appLogsConfiguration: {
      destination: 'log-analytics'
      logAnalyticsConfiguration: {
        customerId: logAnalytics.properties.customerId
        sharedKey: logAnalytics.listKeys().primarySharedKey
      }
    }
  }
}

// API Container App
resource apiContainerApp 'Microsoft.App/containerApps@2023-05-01' = {
  name: '${environmentName}-api'
  location: location
  properties: {
    managedEnvironmentId: containerAppEnv.id
    configuration: {
      ingress: {
        external: true
        targetPort: 8000
      }
    }
    template: {
      containers: [
        {
          name: 'api'
          image: 'myregistry.azurecr.io/api:latest'
        }
      ]
    }
  }
}

// Web App Service
resource webAppServicePlan 'Microsoft.Web/serverfarms@2022-09-01' = {
  name: '${environmentName}-plan'
  location: location
  sku: {
    name: 'B1'
  }
}

resource webAppService 'Microsoft.Web/sites@2022-09-01' = {
  name: '${environmentName}-web'
  location: location
  properties: {
    serverFarmId: webAppServicePlan.id
  }
}

// Global outputs
output AZURE_RESOURCE_GROUP string = resourceGroup().name
output AZURE_LOCATION string = location
output AZURE_SUBSCRIPTION_ID string = subscription().subscriptionId
output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = logAnalytics.id
output AZURE_CONTAINER_APP_ENV_ID string = containerAppEnv.id

// API service outputs
output SERVICE_API_NAME string = apiContainerApp.name
output SERVICE_API_URL string = 'https://${apiContainerApp.properties.configuration.ingress.fqdn}'
output SERVICE_API_RESOURCE_TYPE string = 'containerapp'
output SERVICE_API_IMAGE_NAME string = apiContainerApp.properties.template.containers[0].image

// Web service outputs
output SERVICE_WEB_NAME string = webAppService.name
output SERVICE_WEB_URL string = 'https://${webAppService.properties.defaultHostName}'
output SERVICE_WEB_RESOURCE_TYPE string = 'appservice'
```

## How It Works

1. **azd provision** runs your bicep template and collects outputs
2. **azd** sets environment variables from bicep outputs
3. **azd app** reads these environment variables
4. **Dashboard** extracts Azure metadata and displays it

## Environment Variable Mapping

| Bicep Output | Environment Variable | Dashboard Field |
|--------------|---------------------|-----------------|
| `AZURE_RESOURCE_GROUP` | `AZURE_RESOURCE_GROUP` | Azure.resourceGroup |
| `AZURE_LOCATION` | `AZURE_LOCATION` | Azure.location |
| `AZURE_SUBSCRIPTION_ID` | `AZURE_SUBSCRIPTION_ID` | Azure.subscriptionId |
| `SERVICE_API_URL` | `SERVICE_API_URL` | Azure.url |
| `SERVICE_API_NAME` | `SERVICE_API_NAME` | Azure.resourceName |
| `SERVICE_API_RESOURCE_TYPE` | `SERVICE_API_RESOURCE_TYPE` | Azure.resourceType |

## Features Enabled

With these outputs configured, the dashboard will:

- ✅ Show Azure resource metadata in service details
- ✅ Display Azure URLs alongside local URLs
- ✅ Enable Azure log streaming (requires Azure CLI authenticated)
- ✅ Show resource group, location, and subscription info
- ✅ Group services by deployment type

## Azure Log Streaming Requirements

To stream logs from Azure services:

1. **Azure CLI must be installed and authenticated**
   ```bash
   az login
   ```

2. **User must have permissions to read logs**
   - For Container Apps: `Microsoft.App/containerApps/logs/read`
   - For App Service: `Microsoft.Web/sites/logs/read`

3. **Optional:** Configure Log Analytics workspace ID for advanced queries

## Troubleshooting

### Logs not streaming

- Verify Azure CLI is installed: `az --version`
- Check authentication: `az account show`
- Verify resource group and resource name are correct
- Check Azure resource exists: `az containerapp show -n {name} -g {rg}`

### Metadata not showing

- Check bicep outputs are defined
- Verify `azd provision` completed successfully
- Check environment variables: `azd env get-values`
- Ensure service names match between `azure.yaml` and bicep outputs

### Resource type not detected

- Add explicit `SERVICE_{NAME}_RESOURCE_TYPE` output
- Use one of: `containerapp`, `appservice`, `function`

## Reference

- [Azure Container Apps documentation](https://learn.microsoft.com/azure/container-apps/)
- [Azure App Service documentation](https://learn.microsoft.com/azure/app-service/)
- [azd documentation](https://learn.microsoft.com/azure/developer/azure-developer-cli/)
- [Bicep documentation](https://learn.microsoft.com/azure/azure-resource-manager/bicep/)
