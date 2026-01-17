@description('Main infrastructure template for e2e test project')

param location string = resourceGroup().location

output location string = location
