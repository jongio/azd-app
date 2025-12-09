// =============================================================================
// CONTAINER APPS MODULE
// Uses Azure Verified Modules for Container Apps Environment and Container App
// Configured for log streaming via Log Analytics
// =============================================================================

@description('Name of the container app')
param name string

@description('Location for the resources')
param location string = resourceGroup().location

@description('Tags to apply to resources')
param tags object = {}

@description('Environment name for naming')
param environmentName string

@description('Name of the container registry')
param containerRegistryName string

@description('Log Analytics workspace ID for diagnostics')
param logAnalyticsWorkspaceId string

@description('Target port for the container')
param targetPort int = 3000

// Reference existing container registry
resource acr 'Microsoft.ContainerRegistry/registries@2023-07-01' existing = {
  name: containerRegistryName
}

// =============================================================================
// Container Apps Environment - Using Azure Verified Module
// =============================================================================

module containerAppsEnvironment 'br/public:avm/res/app/managed-environment:0.10.0' = {
  name: 'cae-deployment'
  params: {
    name: 'cae-${environmentName}'
    location: location
    tags: tags
    logAnalyticsWorkspaceResourceId: logAnalyticsWorkspaceId
    // Configure console log streaming to Log Analytics
    appLogsConfiguration: {
      destination: 'log-analytics'
      logAnalyticsConfiguration: {
        customerId: reference(logAnalyticsWorkspaceId, '2023-09-01').customerId
        sharedKey: listKeys(logAnalyticsWorkspaceId, '2023-09-01').primarySharedKey
      }
    }
  }
}

// =============================================================================
// Container App - Using Azure Verified Module
// =============================================================================

module containerApp 'br/public:avm/res/app/container-app:0.13.0' = {
  name: 'containerapp-deployment'
  params: {
    name: '${name}-${environmentName}'
    location: location
    tags: union(tags, {
      'azd-service-name': name
    })
    environmentResourceId: containerAppsEnvironment.outputs.resourceId
    // Ingress configuration
    ingressExternal: true
    ingressTargetPort: targetPort
    ingressTransport: 'http'
    // Container configuration - placeholder image, replaced by azd deploy
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
            value: name
          }
        ]
      }
    ]
    // Scale settings
    scaleMinReplicas: 0
    scaleMaxReplicas: 3
    // Registry configuration
    registries: [
      {
        server: acr.properties.loginServer
        username: acr.listCredentials().username
        passwordSecretRef: 'acr-password'
      }
    ]
    secrets: {
      secureList: [
        {
          name: 'acr-password'
          value: acr.listCredentials().passwords[0].value
        }
      ]
    }
  }
}

// =============================================================================
// OUTPUTS
// =============================================================================

@description('The URI of the container app')
output uri string = 'https://${containerApp.outputs.fqdn}'

@description('The name of the container app')
output name string = containerApp.outputs.name

@description('The resource ID of the container app')
output resourceId string = containerApp.outputs.resourceId
