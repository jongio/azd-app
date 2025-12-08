@description('Name of the container registry')
param name string

@description('Location for resources')
param location string = resourceGroup().location

@description('Tags to apply to resources')
param tags object = {}

resource acr 'Microsoft.ContainerRegistry/registries@2023-07-01' = {
  name: name
  location: location
  tags: tags
  sku: {
    name: 'Basic'
  }
  properties: {
    adminUserEnabled: true
  }
}

output name string = acr.name
output loginServer string = acr.properties.loginServer
