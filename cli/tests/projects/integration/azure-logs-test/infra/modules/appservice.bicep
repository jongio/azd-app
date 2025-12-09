// =============================================================================
// APP SERVICE MODULE
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
// App Service Plan
// =============================================================================

resource appServicePlan 'Microsoft.Web/serverfarms@2025-03-01' = {
  name: 'asp-${name}'
  location: location
  tags: tags
  kind: 'linux'
  sku: {
    name: 'B1'
    tier: 'Basic'
    capacity: 1
  }
  properties: {
    reserved: true // Linux
  }
}

// =============================================================================
// Web App
// =============================================================================

resource webApp 'Microsoft.Web/sites@2025-03-01' = {
  name: name
  location: location
  tags: union(tags, {
    'azd-service-name': 'appservice-web'
  })
  kind: 'app,linux'
  properties: {
    serverFarmId: appServicePlan.id
    siteConfig: {
      linuxFxVersion: 'PYTHON|3.11'
      alwaysOn: true
      httpLoggingEnabled: true
      detailedErrorLoggingEnabled: true
      requestTracingEnabled: true
      appSettings: [
        {
          name: 'SCM_DO_BUILD_DURING_DEPLOYMENT'
          value: 'true'
        }
        {
          name: 'WEBSITE_RUN_FROM_PACKAGE'
          value: '0'
        }
        {
          name: 'LOG_LEVEL'
          value: 'INFO'
        }
      ]
    }
    httpsOnly: true
  }
}

// =============================================================================
// Diagnostic Settings for log streaming
// =============================================================================

resource webAppDiagnostics 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = {
  name: 'webapp-diagnostics'
  scope: webApp
  properties: {
    workspaceId: logAnalyticsWorkspaceId
    logs: [
      {
        category: 'AppServiceHTTPLogs'
        enabled: true
      }
      {
        category: 'AppServiceConsoleLogs'
        enabled: true
      }
      {
        category: 'AppServiceAppLogs'
        enabled: true
      }
      {
        category: 'AppServicePlatformLogs'
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

@description('The default hostname of the web app')
output uri string = 'https://${webApp.properties.defaultHostName}'

@description('The name of the web app')
output name string = webApp.name

@description('The resource ID of the web app')
output resourceId string = webApp.id
