// =============================================================================
// CONTAINER REGISTRY MODULE
// =============================================================================

@description('Name of the container registry')
param name string

@description('Location for the resources')
param location string = resourceGroup().location

@description('Tags to apply to resources')
param tags object = {}

@description('Log Analytics workspace ID for diagnostics')
param logAnalyticsWorkspaceId string

// =============================================================================
// Azure Container Registry
// =============================================================================

resource registry 'Microsoft.ContainerRegistry/registries@2025-11-01' = {
  name: name
  location: location
  tags: tags
  sku: {
    name: 'Basic'
  }
  properties: {
    adminUserEnabled: true
  }
}

// =============================================================================
// Diagnostic Settings
// =============================================================================

resource acrDiagnostics 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = {
  name: 'acr-diagnostics'
  scope: registry
  properties: {
    workspaceId: logAnalyticsWorkspaceId
    logs: [
      {
        categoryGroup: 'allLogs'
        enabled: true
      }
    ]
    metrics: [
      {
        category: 'AllMetrics'
        enabled: true
      }
    ]
  }
}

// =============================================================================
// OUTPUTS
// =============================================================================

@description('The name of the container registry')
output name string = registry.name

@description('The login server URL')
output loginServer string = registry.properties.loginServer

@description('The resource ID of the container registry')
output resourceId string = registry.id
