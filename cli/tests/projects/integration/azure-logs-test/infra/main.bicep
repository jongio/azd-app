targetScope = 'subscription'

@minLength(1)
@maxLength(64)
@description('Name of the environment')
param environmentName string

@minLength(1)
@description('Primary location for all resources')
param location string

// Tags applied to all resources
var tags = {
  'azd-env-name': environmentName
}

// Generate unique suffix for globally unique names
var resourceToken = toLower(uniqueString(subscription().id, environmentName, location))

// Organize resources in a resource group
resource rg 'Microsoft.Resources/resourceGroups@2024-03-01' = {
  name: 'rg-${environmentName}'
  location: location
  tags: tags
}

// =============================================================================
// SHARED MONITORING - Log Analytics Workspace for all services
// =============================================================================

module monitoring './modules/monitoring.bicep' = {
  name: 'monitoring'
  scope: rg
  params: {
    name: 'log-${resourceToken}'
    location: location
    tags: tags
  }
}

// =============================================================================
// CONTAINER REGISTRY - Shared ACR for Container Apps
// =============================================================================

module acr './modules/acr.bicep' = {
  name: 'acr'
  scope: rg
  params: {
    name: 'cr${resourceToken}'
    location: location
    tags: tags
    logAnalyticsWorkspaceId: monitoring.outputs.logAnalyticsWorkspaceId
  }
}

// =============================================================================
// AZURE CONTAINER APPS - containerapp-api service
// =============================================================================

module containerApps './modules/containerapp.bicep' = {
  name: 'containerapp-api'
  scope: rg
  params: {
    name: 'ca-${resourceToken}'
    serviceName: 'containerapp-api'
    location: location
    tags: tags
    containerRegistryName: acr.outputs.name
    logAnalyticsWorkspaceId: monitoring.outputs.logAnalyticsWorkspaceId
    targetPort: 3000
  }
}

// =============================================================================
// AZURE APP SERVICE - appservice-web service
// =============================================================================

module appService './modules/appservice.bicep' = {
  name: 'appservice-web'
  scope: rg
  params: {
    name: 'appservice-web-${resourceToken}'
    location: location
    tags: tags
    logAnalyticsWorkspaceId: monitoring.outputs.logAnalyticsWorkspaceId
  }
}

// =============================================================================
// AZURE FUNCTIONS - functions-worker service
// =============================================================================

module functions './modules/functions.bicep' = {
  name: 'functions-worker'
  scope: rg
  params: {
    name: 'func-${resourceToken}'
    location: location
    tags: tags
    logAnalyticsWorkspaceId: monitoring.outputs.logAnalyticsWorkspaceId
    appInsightsConnectionString: monitoring.outputs.appInsightsConnectionString
  }
}

// =============================================================================
// OUTPUTS - Used by azd for service discovery and log streaming
// =============================================================================

output AZURE_LOCATION string = location
output AZURE_RESOURCE_GROUP_NAME string = rg.name
output AZURE_CONTAINER_REGISTRY_ENDPOINT string = acr.outputs.loginServer

// Log Analytics workspace - required for Azure log streaming
output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = monitoring.outputs.logAnalyticsWorkspaceId
output AZURE_LOG_ANALYTICS_WORKSPACE_NAME string = monitoring.outputs.logAnalyticsWorkspaceName
output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = monitoring.outputs.logAnalyticsWorkspaceGuid

// Application Insights - for Functions telemetry
output AZURE_APPLICATION_INSIGHTS_CONNECTION_STRING string = monitoring.outputs.appInsightsConnectionString

// Service URLs for azd
output SERVICE_CONTAINERAPP_API_URL string = containerApps.outputs.uri
output SERVICE_CONTAINERAPP_API_NAME string = containerApps.outputs.name

output SERVICE_APPSERVICE_WEB_URL string = appService.outputs.uri
output SERVICE_APPSERVICE_WEB_NAME string = appService.outputs.name

output SERVICE_FUNCTIONS_WORKER_URL string = functions.outputs.uri
output SERVICE_FUNCTIONS_WORKER_NAME string = functions.outputs.name
