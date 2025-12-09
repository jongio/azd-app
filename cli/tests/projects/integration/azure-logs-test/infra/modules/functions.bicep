// =============================================================================
// AZURE FUNCTIONS MODULE
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
// Using managed identity for authentication (no local auth per policy)
// =============================================================================

resource storageAccount 'Microsoft.Storage/storageAccounts@2025-06-01' = {
  name: 'st${replace(replace(name, '-', ''), '_', '')}'
  location: location
  tags: tags
  sku: {
    name: 'Standard_LRS'
  }
  kind: 'StorageV2'
  properties: {
    supportsHttpsTrafficOnly: true
    minimumTlsVersion: 'TLS1_2'
    allowBlobPublicAccess: false
    allowSharedKeyAccess: false // Disabled per corporate policy
  }
}

// Blob service for Functions
resource blobService 'Microsoft.Storage/storageAccounts/blobServices@2025-06-01' = {
  parent: storageAccount
  name: 'default'
}

// =============================================================================
// App Service Plan for Functions - Elastic Premium (EP1) for Linux support
// =============================================================================

resource functionPlan 'Microsoft.Web/serverfarms@2025-03-01' = {
  name: 'asp-${name}'
  location: location
  tags: tags
  kind: 'elastic'
  sku: {
    name: 'EP1'
    tier: 'ElasticPremium'
  }
  properties: {
    reserved: true // Linux
    maximumElasticWorkerCount: 3
  }
}

// =============================================================================
// Function App with System Assigned Managed Identity
// =============================================================================

resource functionApp 'Microsoft.Web/sites@2025-03-01' = {
  name: name
  location: location
  tags: union(tags, {
    'azd-service-name': 'functions-worker'
  })
  kind: 'functionapp,linux'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    serverFarmId: functionPlan.id
    siteConfig: {
      linuxFxVersion: 'NODE|20'
      ftpsState: 'Disabled'
      minTlsVersion: '1.2'
      appSettings: [
        {
          name: 'FUNCTIONS_EXTENSION_VERSION'
          value: '~4'
        }
        {
          name: 'FUNCTIONS_WORKER_RUNTIME'
          value: 'node'
        }
        {
          name: 'AzureWebJobsStorage__accountName'
          value: storageAccount.name
        }
        {
          name: 'APPLICATIONINSIGHTS_CONNECTION_STRING'
          value: appInsightsConnectionString
        }
        {
          name: 'WEBSITE_RUN_FROM_PACKAGE'
          value: '1'
        }
      ]
    }
    httpsOnly: true
  }
}

// =============================================================================
// Role Assignment - Storage Blob Data Owner for Function App
// =============================================================================

var storageBlobDataOwnerRoleId = 'b7e6dc6d-f1e8-4753-8033-0f276bb0955b'

resource storageRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(storageAccount.id, functionApp.id, storageBlobDataOwnerRoleId)
  scope: storageAccount
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', storageBlobDataOwnerRoleId)
    principalId: functionApp.identity.principalId
    principalType: 'ServicePrincipal'
  }
}

// =============================================================================
// Diagnostic Settings for log streaming
// =============================================================================

resource functionAppDiagnostics 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = {
  name: 'func-diagnostics'
  scope: functionApp
  properties: {
    workspaceId: logAnalyticsWorkspaceId
    logs: [
      {
        category: 'FunctionAppLogs'
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

@description('The default hostname of the function app')
output uri string = 'https://${functionApp.properties.defaultHostName}'

@description('The name of the function app')
output name string = functionApp.name

@description('The resource ID of the function app')
output resourceId string = functionApp.id
