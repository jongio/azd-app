// =============================================================================
// CONTAINER APPS MODULE
// Configured for log streaming via Log Analytics
// =============================================================================

@description('Name of the container app (must be <= 32 chars)')
param name string

@description('Service name for azd tagging')
param serviceName string

@description('Location for the resources')
param location string = resourceGroup().location

@description('Tags to apply to resources')
param tags object = {}

@description('Name of the container registry')
param containerRegistryName string

@description('Log Analytics workspace ID for diagnostics')
param logAnalyticsWorkspaceId string

@description('Target port for the container')
param targetPort int = 3000

// Reference existing container registry
resource acr 'Microsoft.ContainerRegistry/registries@2025-11-01' existing = {
  name: containerRegistryName
}

// Reference Log Analytics workspace for customer ID
resource logAnalytics 'Microsoft.OperationalInsights/workspaces@2025-07-01' existing = {
  name: last(split(logAnalyticsWorkspaceId, '/'))
}

// =============================================================================
// Container Apps Environment
// =============================================================================

resource containerAppsEnvironment 'Microsoft.App/managedEnvironments@2024-03-01' = {
  name: 'cae-${name}'
  location: location
  tags: tags
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

// =============================================================================
// Container App
// =============================================================================

resource containerApp 'Microsoft.App/containerApps@2024-03-01' = {
  name: name
  location: location
  tags: union(tags, {
    'azd-service-name': serviceName
  })
  properties: {
    managedEnvironmentId: containerAppsEnvironment.id
    configuration: {
      ingress: {
        external: true
        targetPort: targetPort
        transport: 'http'
      }
      registries: [
        {
          server: acr.properties.loginServer
          username: acr.listCredentials().username
          passwordSecretRef: 'acr-password'
        }
      ]
      secrets: [
        {
          name: 'acr-password'
          value: acr.listCredentials().passwords[0].value
        }
      ]
    }
    template: {
      containers: [
        {
          name: name
          image: 'mcr.microsoft.com/azuredocs/containerapps-helloworld:latest'
          resources: {
            cpu: json('0.25')
            memory: '0.5Gi'
          }
          env: [
            {
              name: 'PORT'
              value: string(targetPort)
            }
            {
              name: 'SERVICE_NAME'
              value: serviceName
            }
          ]
        }
      ]
      scale: {
        // Keep at least 1 replica running so the app doesn't scale-to-zero and stop producing logs.
        minReplicas: 1
        maxReplicas: 3
      }
    }
  }
}

// =============================================================================
// OUTPUTS
// =============================================================================

@description('The URI of the container app')
output uri string = 'https://${containerApp.properties.configuration.ingress.fqdn}'

@description('The name of the container app')
output name string = containerApp.name

@description('The resource ID of the container app')
output resourceId string = containerApp.id
