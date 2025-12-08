targetScope = 'subscription'

@minLength(1)
@maxLength(64)
@description('Name of the environment')
param environmentName string

@minLength(1)
@description('Primary location for all resources')
param location string

// Tags that should be applied to all resources
var tags = {
  'azd-env-name': environmentName
}

// Generate a unique suffix for globally unique names
var resourceToken = toLower(uniqueString(subscription().id, environmentName, location))

// Organize resources in a resource group
resource rg 'Microsoft.Resources/resourceGroups@2024-03-01' = {
  name: 'rg-${environmentName}'
  location: location
  tags: tags
}

// Container Registry for storing images
module acr './modules/acr.bicep' = {
  name: 'acr'
  scope: rg
  params: {
    name: 'cr${resourceToken}'
    location: location
    tags: tags
  }
}

// Container Apps environment and web service
module web './modules/web.bicep' = {
  name: 'web'
  scope: rg
  params: {
    name: 'web'
    location: location
    tags: tags
    environmentName: environmentName
    containerRegistryName: acr.outputs.name
  }
}

// Output the web service URL
output AZURE_LOCATION string = location
output AZURE_RESOURCE_GROUP_NAME string = rg.name
output AZURE_CONTAINER_REGISTRY_ENDPOINT string = acr.outputs.loginServer
output SERVICE_WEB_URL string = web.outputs.uri
