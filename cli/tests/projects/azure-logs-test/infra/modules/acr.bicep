// =============================================================================
// CONTAINER REGISTRY MODULE
// Uses Azure Verified Module for ACR
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
// Azure Container Registry - Using Azure Verified Module
// =============================================================================

module registry 'br/public:avm/res/container-registry/registry:0.8.0' = {
  name: 'acr-deployment'
  params: {
    name: name
    location: location
    tags: tags
    acrSku: 'Basic'
    // Enable admin user for simple auth in test scenarios
    acrAdminUserEnabled: true
    // Configure diagnostic settings to send logs to Log Analytics
    diagnosticSettings: [
      {
        name: 'acr-diagnostics'
        workspaceResourceId: logAnalyticsWorkspaceId
        logCategoriesAndGroups: [
          { categoryGroup: 'allLogs' }
        ]
        metricCategories: [
          { category: 'AllMetrics' }
        ]
      }
    ]
  }
}

// =============================================================================
// OUTPUTS
// =============================================================================

@description('The name of the container registry')
output name string = registry.outputs.name

@description('The login server URL')
output loginServer string = registry.outputs.loginServer

@description('The resource ID of the container registry')
output resourceId string = registry.outputs.resourceId
