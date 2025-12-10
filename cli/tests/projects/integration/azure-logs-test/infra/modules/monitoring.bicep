// =============================================================================
// MONITORING MODULE - Log Analytics Workspace + Application Insights
// =============================================================================

@description('Name for the Log Analytics workspace')
param name string

@description('Location for the resources')
param location string = resourceGroup().location

@description('Tags to apply to resources')
param tags object = {}

// =============================================================================
// Log Analytics Workspace
// This is the central hub for all Azure service logs
// =============================================================================

resource logAnalyticsWorkspace 'Microsoft.OperationalInsights/workspaces@2025-07-01' = {
  name: name
  location: location
  tags: tags
  properties: {
    sku: {
      name: 'PerGB2018'
    }
    retentionInDays: 30
  }
}

// =============================================================================
// Application Insights
// Required for Azure Functions telemetry and logging
// =============================================================================

resource appInsights 'Microsoft.Insights/components@2020-02-02' = {
  name: 'appi-${name}'
  location: location
  tags: tags
  kind: 'web'
  properties: {
    Application_Type: 'web'
    WorkspaceResourceId: logAnalyticsWorkspace.id
  }
}

// =============================================================================
// OUTPUTS
// =============================================================================

@description('The resource ID of the Log Analytics workspace')
output logAnalyticsWorkspaceId string = logAnalyticsWorkspace.id

@description('The name of the Log Analytics workspace')
output logAnalyticsWorkspaceName string = logAnalyticsWorkspace.name

@description('The workspace GUID (customerId) used for Log Analytics queries')
output logAnalyticsWorkspaceGuid string = logAnalyticsWorkspace.properties.customerId

@description('The Application Insights connection string')
output appInsightsConnectionString string = appInsights.properties.ConnectionString

@description('The Application Insights instrumentation key')
output appInsightsInstrumentationKey string = appInsights.properties.InstrumentationKey
