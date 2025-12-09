// =============================================================================
// MONITORING MODULE - Log Analytics Workspace + Application Insights
// Uses Azure Verified Modules for consistent, production-ready configuration
// =============================================================================

@description('Name for the Log Analytics workspace')
param name string

@description('Location for the resources')
param location string = resourceGroup().location

@description('Tags to apply to resources')
param tags object = {}

// =============================================================================
// Log Analytics Workspace - Using Azure Verified Module
// This is the central hub for all Azure service logs
// =============================================================================

module logAnalyticsWorkspace 'br/public:avm/res/operational-insights/workspace:0.10.0' = {
  name: 'log-analytics'
  params: {
    name: name
    location: location
    tags: tags
    // Retain logs for 30 days (default for test environments)
    dataRetention: 30
    // Enable all relevant solutions for log streaming
    gallerySolutions: [
      {
        name: 'ContainerInsights'
        product: 'OMSGallery/ContainerInsights'
        publisher: 'Microsoft'
      }
    ]
  }
}

// =============================================================================
// Application Insights - Using Azure Verified Module
// Required for Azure Functions telemetry and logging
// =============================================================================

module appInsights 'br/public:avm/res/insights/component:0.6.0' = {
  name: 'app-insights'
  params: {
    name: 'appi-${name}'
    location: location
    tags: tags
    workspaceResourceId: logAnalyticsWorkspace.outputs.resourceId
    // Application type for general web apps and functions
    applicationType: 'web'
  }
}

// =============================================================================
// OUTPUTS
// =============================================================================

@description('The resource ID of the Log Analytics workspace')
output logAnalyticsWorkspaceId string = logAnalyticsWorkspace.outputs.resourceId

@description('The name of the Log Analytics workspace')
output logAnalyticsWorkspaceName string = logAnalyticsWorkspace.outputs.name

@description('The Application Insights connection string')
output appInsightsConnectionString string = appInsights.outputs.connectionString

@description('The Application Insights instrumentation key')
output appInsightsInstrumentationKey string = appInsights.outputs.instrumentationKey
