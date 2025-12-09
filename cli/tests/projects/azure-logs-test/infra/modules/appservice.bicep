// =============================================================================
// APP SERVICE MODULE
// Uses Azure Verified Modules for App Service Plan and Web App
// Configured for log streaming via Log Analytics
// =============================================================================

@description('Name of the web app')
param name string

@description('Location for the resources')
param location string = resourceGroup().location

@description('Tags to apply to resources')
param tags object = {}

@description('Log Analytics workspace ID for diagnostics')
param logAnalyticsWorkspaceId string

// =============================================================================
// App Service Plan - Using Azure Verified Module
// =============================================================================

module appServicePlan 'br/public:avm/res/web/serverfarm:0.4.0' = {
  name: 'asp-deployment'
  params: {
    name: 'asp-${name}'
    location: location
    tags: tags
    // Linux hosting for Python
    kind: 'linux'
    reserved: true
    // B1 tier for test environments
    skuName: 'B1'
    skuCapacity: 1
    // Diagnostic settings
    diagnosticSettings: [
      {
        name: 'asp-diagnostics'
        workspaceResourceId: logAnalyticsWorkspaceId
        metricCategories: [
          { category: 'AllMetrics' }
        ]
      }
    ]
  }
}

// =============================================================================
// Web App - Using Azure Verified Module
// =============================================================================

module webApp 'br/public:avm/res/web/site:0.13.0' = {
  name: 'webapp-deployment'
  params: {
    name: name
    location: location
    tags: union(tags, {
      'azd-service-name': 'appservice-web'
    })
    kind: 'app,linux'
    serverFarmResourceId: appServicePlan.outputs.resourceId
    // Python 3.11 runtime
    siteConfig: {
      linuxFxVersion: 'PYTHON|3.11'
      alwaysOn: true
      // Enable detailed logging
      httpLoggingEnabled: true
      detailedErrorLoggingEnabled: true
      requestTracingEnabled: true
    }
    // App settings
    appSettingsKeyValuePairs: {
      SCM_DO_BUILD_DURING_DEPLOYMENT: 'true'
      WEBSITE_RUN_FROM_PACKAGE: '0'
      LOG_LEVEL: 'INFO'
    }
    // Diagnostic settings for log streaming
    diagnosticSettings: [
      {
        name: 'webapp-diagnostics'
        workspaceResourceId: logAnalyticsWorkspaceId
        logCategoriesAndGroups: [
          { category: 'AppServiceHTTPLogs' }
          { category: 'AppServiceConsoleLogs' }
          { category: 'AppServiceAppLogs' }
          { category: 'AppServicePlatformLogs' }
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

@description('The default hostname of the web app')
output uri string = 'https://${webApp.outputs.defaultHostname}'

@description('The name of the web app')
output name string = webApp.outputs.name

@description('The resource ID of the web app')
output resourceId string = webApp.outputs.resourceId
