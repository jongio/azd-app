// =============================================================================
// AZURE FUNCTIONS MODULE
// Uses Azure Verified Modules for Function App
// Configured for log streaming via Log Analytics and Application Insights
// =============================================================================

@description('Name of the function app')
param name string

@description('Location for the resources')
param location string = resourceGroup().location

@description('Tags to apply to resources')
param tags object = {}

@description('Log Analytics workspace ID for diagnostics')
param logAnalyticsWorkspaceId string

@description('Application Insights connection string')
param appInsightsConnectionString string

// =============================================================================
// Storage Account - Required for Azure Functions
// =============================================================================

module storageAccount 'br/public:avm/res/storage/storage-account:0.15.0' = {
  name: 'func-storage-deployment'
  params: {
    name: 'st${replace(name, '-', '')}'
    location: location
    tags: tags
    skuName: 'Standard_LRS'
    kind: 'StorageV2'
    // Diagnostic settings
    diagnosticSettings: [
      {
        name: 'storage-diagnostics'
        workspaceResourceId: logAnalyticsWorkspaceId
        metricCategories: [
          { category: 'Transaction' }
        ]
      }
    ]
  }
}

// =============================================================================
// App Service Plan for Functions - Consumption plan
// =============================================================================

module functionPlan 'br/public:avm/res/web/serverfarm:0.4.0' = {
  name: 'func-asp-deployment'
  params: {
    name: 'asp-${name}'
    location: location
    tags: tags
    // Consumption plan for Functions
    skuName: 'Y1'
    skuCapacity: 0
    kind: 'functionapp'
    reserved: true
    // Diagnostic settings
    diagnosticSettings: [
      {
        name: 'func-asp-diagnostics'
        workspaceResourceId: logAnalyticsWorkspaceId
        metricCategories: [
          { category: 'AllMetrics' }
        ]
      }
    ]
  }
}

// =============================================================================
// Function App - Using Azure Verified Module
// =============================================================================

module functionApp 'br/public:avm/res/web/site:0.13.0' = {
  name: 'func-deployment'
  params: {
    name: name
    location: location
    tags: union(tags, {
      'azd-service-name': 'functions-worker'
    })
    kind: 'functionapp,linux'
    serverFarmResourceId: functionPlan.outputs.resourceId
    // Node.js 20 runtime for TypeScript functions
    siteConfig: {
      linuxFxVersion: 'NODE|20'
      // Function runtime settings
      ftpsState: 'Disabled'
      minTlsVersion: '1.2'
    }
    // App settings for Functions
    appSettingsKeyValuePairs: {
      FUNCTIONS_EXTENSION_VERSION: '~4'
      FUNCTIONS_WORKER_RUNTIME: 'node'
      AzureWebJobsStorage: 'DefaultEndpointsProtocol=https;AccountName=${storageAccount.outputs.name};EndpointSuffix=core.windows.net;AccountKey=${storageAccount.outputs.exportedSecrets['accessKey1'].value}'
      APPLICATIONINSIGHTS_CONNECTION_STRING: appInsightsConnectionString
      WEBSITE_RUN_FROM_PACKAGE: '1'
    }
    // Diagnostic settings for log streaming
    diagnosticSettings: [
      {
        name: 'func-diagnostics'
        workspaceResourceId: logAnalyticsWorkspaceId
        logCategoriesAndGroups: [
          { category: 'FunctionAppLogs' }
        ]
        metricCategories: [
          { category: 'AllMetrics' }
        ]
      }
    ]
  }
  dependsOn: [
    storageAccount
  ]
}

// =============================================================================
// OUTPUTS
// =============================================================================

@description('The default hostname of the function app')
output uri string = 'https://${functionApp.outputs.defaultHostname}'

@description('The name of the function app')
output name string = functionApp.outputs.name

@description('The resource ID of the function app')
output resourceId string = functionApp.outputs.resourceId
