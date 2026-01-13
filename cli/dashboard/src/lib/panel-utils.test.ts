/**
 * Tests for panel-utils helper functions
 */
import { afterEach, beforeEach, describe, it, expect, vi } from 'vitest'
import type { AzureServiceInfo, Service } from '@/types'
import { 
  buildAzurePortalUrl,
  formatCheckType,
  isAzureLogViewingSupported, 
  isSensitiveKey,
  maskValue,
  formatTimestamp,
  formatUptime,
  getHealthColor,
  getHealthDisplay,
  getStatusColor,
  getStatusDisplay,
  getUnsupportedResourceTypeInfo,
  hasAzureDeployment,
  SUPPORTED_AZURE_LOG_RESOURCE_TYPES,
  formatResourceType,
} from './panel-utils'

describe('panel-utils', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('formatUptime', () => {
    it('should return N/A for undefined start time', () => {
      expect(formatUptime(undefined)).toBe('N/A')
    })

    it('should return N/A for future timestamps', () => {
      vi.setSystemTime(new Date('2025-01-01T00:00:00.000Z'))
      const future = new Date('2025-01-01T00:01:00.000Z')
      expect(formatUptime(future.toISOString())).toBe('N/A')
    })

    it('should format seconds-only uptime', () => {
      vi.setSystemTime(new Date('2025-01-01T00:00:10.000Z'))
      const start = new Date('2025-01-01T00:00:00.000Z')
      expect(formatUptime(start.toISOString())).toBe('10s')
    })

    it('should format minutes uptime', () => {
      vi.setSystemTime(new Date('2025-01-01T00:02:10.000Z'))
      const start = new Date('2025-01-01T00:00:00.000Z')
      expect(formatUptime(start.toISOString())).toBe('2m 10s')
    })

    it('should format hours uptime', () => {
      vi.setSystemTime(new Date('2025-01-01T01:02:03.000Z'))
      const start = new Date('2025-01-01T00:00:00.000Z')
      expect(formatUptime(start.toISOString())).toBe('1h 2m 3s')
    })
  })

  describe('formatTimestamp', () => {
    it('should return N/A for undefined timestamp', () => {
      expect(formatTimestamp(undefined)).toBe('N/A')
    })

    it('should return N/A for invalid timestamp', () => {
      expect(formatTimestamp('not-a-date')).toBe('N/A')
    })

    it('should format valid timestamp using locale string', () => {
      const ts = '2025-01-01T00:00:00.000Z'
      const expected = new Date(ts).toLocaleString()
      expect(formatTimestamp(ts)).toBe(expected)
    })
  })

  describe('getStatusColor', () => {
    it('should return the correct colors for known statuses', () => {
      expect(getStatusColor('running')).toBe('text-green-500')
      expect(getStatusColor('ready')).toBe('text-green-500')
      expect(getStatusColor('starting')).toBe('text-yellow-500')
      expect(getStatusColor('stopping')).toBe('text-yellow-500')
      expect(getStatusColor('stopped')).toBe('text-gray-500')
      expect(getStatusColor('error')).toBe('text-red-500')
      expect(getStatusColor('not-running')).toBe('text-gray-500')
    })

    it('should fall back to not-running color for unknown status', () => {
      expect(getStatusColor('something-else')).toBe('text-gray-500')
      expect(getStatusColor(undefined)).toBe('text-gray-500')
    })
  })

  describe('getHealthColor', () => {
    it('should return the correct colors for known health values', () => {
      expect(getHealthColor('healthy')).toBe('text-green-500')
      expect(getHealthColor('degraded')).toBe('text-yellow-500')
      expect(getHealthColor('unhealthy')).toBe('text-red-500')
      expect(getHealthColor('starting')).toBe('text-yellow-500')
      expect(getHealthColor('unknown')).toBe('text-gray-500')
    })

    it('should fall back to unknown color for unknown health', () => {
      expect(getHealthColor('mystery')).toBe('text-gray-500')
      expect(getHealthColor(undefined)).toBe('text-gray-500')
    })
  })

  describe('buildAzurePortalUrl', () => {
    it('should return null when required azure fields are missing', () => {
      const service: Service = { name: 'svc' }
      expect(buildAzurePortalUrl(service)).toBeNull()

      const azurePartial1: AzureServiceInfo = { subscriptionId: 'sub' }
      expect(buildAzurePortalUrl({ ...service, azure: azurePartial1 })).toBeNull()

      const azurePartial2: AzureServiceInfo = { subscriptionId: 'sub', resourceGroup: 'rg' }
      expect(
        buildAzurePortalUrl({ ...service, azure: azurePartial2 })
      ).toBeNull()
    })

    it('should build a Container Apps portal url by default', () => {
      const service: Service = {
        name: 'svc',
        azure: {
          subscriptionId: 'sub',
          resourceGroup: 'rg',
          resourceName: 'app',
          resourceType: 'containerapp',
        },
      }
      const url = buildAzurePortalUrl(service)
      expect(url).toContain('/providers/Microsoft.App/containerApps/app')
    })

    it('should build an App Service portal url for appservice/webapp/function', () => {
      const base: Service = {
        name: 'svc',
        azure: { subscriptionId: 'sub', resourceGroup: 'rg', resourceName: 'site' },
      }
      expect(buildAzurePortalUrl({ ...base, azure: { ...base.azure, resourceType: 'appservice' } })).toContain(
        '/providers/Microsoft.Web/sites/site'
      )
      expect(buildAzurePortalUrl({ ...base, azure: { ...base.azure, resourceType: 'webapp' } })).toContain(
        '/providers/Microsoft.Web/sites/site'
      )
      expect(buildAzurePortalUrl({ ...base, azure: { ...base.azure, resourceType: 'function' } })).toContain(
        '/providers/Microsoft.Web/sites/site'
      )
    })

    it('should build an AKS portal url for aks', () => {
      const service: Service = {
        name: 'svc',
        azure: {
          subscriptionId: 'sub',
          resourceGroup: 'rg',
          resourceName: 'cluster',
          resourceType: 'aks',
        },
      }
      const url = buildAzurePortalUrl(service)
      expect(url).toContain('/providers/Microsoft.ContainerService/managedClusters/cluster')
    })
  })

  describe('isSensitiveKey', () => {
    it('should detect sensitive keys case-insensitively', () => {
      expect(isSensitiveKey('PASSWORD')).toBe(true)
      expect(isSensitiveKey('clientSecret')).toBe(true)
      expect(isSensitiveKey('apiKey')).toBe(true)
      expect(isSensitiveKey('authToken')).toBe(true)
      expect(isSensitiveKey('connectionString')).toBe(true)
      expect(isSensitiveKey('privateKey')).toBe(true)
      expect(isSensitiveKey('credential')).toBe(true)
    })

    it('should return false for non-sensitive keys', () => {
      expect(isSensitiveKey('name')).toBe(false)
      expect(isSensitiveKey('port')).toBe(false)
      expect(isSensitiveKey('replicas')).toBe(false)
    })
  })

  describe('maskValue', () => {
    it('should mask short values with the same length', () => {
      expect(maskValue('')).toBe('')
      expect(maskValue('a')).toBe('•')
      expect(maskValue('abcd')).toBe('••••')
    })

    it('should mask longer values up to 16 characters', () => {
      expect(maskValue('abcde')).toBe('•••••')
      expect(maskValue('0123456789')).toBe('••••••••••')
      expect(maskValue('0123456789abcdefg')).toBe('••••••••••••••••')
    })
  })

  describe('getStatusDisplay', () => {
    it('should return defaults for undefined/unknown', () => {
      expect(getStatusDisplay(undefined)).toEqual({ text: 'Not Running', indicator: '○' })
      expect(getStatusDisplay('unknown-status')).toEqual({ text: 'Not Running', indicator: '○' })
    })

    it('should return known displays', () => {
      expect(getStatusDisplay('running')).toEqual({ text: 'Running', indicator: '●' })
      expect(getStatusDisplay('starting')).toEqual({ text: 'Starting', indicator: '◐' })
      expect(getStatusDisplay('stopping')).toEqual({ text: 'Stopping', indicator: '◑' })
      expect(getStatusDisplay('stopped')).toEqual({ text: 'Stopped', indicator: '◉' })
      expect(getStatusDisplay('error')).toEqual({ text: 'Error', indicator: '⚠' })
    })
  })

  describe('getHealthDisplay', () => {
    it('should return defaults for undefined/unknown', () => {
      expect(getHealthDisplay(undefined)).toEqual({ text: 'Unknown', indicator: '○' })
      expect(getHealthDisplay('unknown-health')).toEqual({ text: 'Unknown', indicator: '○' })
    })

    it('should return known displays', () => {
      expect(getHealthDisplay('healthy')).toEqual({ text: 'Healthy', indicator: '●' })
      expect(getHealthDisplay('degraded')).toEqual({ text: 'Degraded', indicator: '◐' })
      expect(getHealthDisplay('unhealthy')).toEqual({ text: 'Unhealthy', indicator: '●' })
      expect(getHealthDisplay('starting')).toEqual({ text: 'Starting', indicator: '◐' })
    })
  })

  describe('formatCheckType', () => {
    it('should format known types', () => {
      expect(formatCheckType('http')).toBe('HTTP')
      expect(formatCheckType('port')).toBe('Port')
      expect(formatCheckType('process')).toBe('Process')
      expect(formatCheckType('tcp')).toBe('TCP')
    })

    it('should fall back for unknown/undefined', () => {
      expect(formatCheckType(undefined)).toBe('Unknown')
      expect(formatCheckType('grpc')).toBe('grpc')
    })
  })

  describe('hasAzureDeployment', () => {
    it('should return true if resourceName is present', () => {
      const service: Service = { name: 'svc', azure: { resourceName: 'x' } }
      expect(hasAzureDeployment(service)).toBe(true)
    })

    it('should return true if url is present', () => {
      const service: Service = { name: 'svc', azure: { url: 'https://example.com' } }
      expect(hasAzureDeployment(service)).toBe(true)
    })

    it('should return true if customUrl is present', () => {
      const service: Service = { name: 'svc', azure: { customUrl: 'https://custom.example.com' } }
      expect(hasAzureDeployment(service)).toBe(true)
    })

    it('should return true if customDomain is present', () => {
      const service: Service = { name: 'svc', azure: { customDomain: 'myapp.example.com' } }
      expect(hasAzureDeployment(service)).toBe(true)
    })

    it('should return false otherwise', () => {
      expect(hasAzureDeployment({ name: 'svc' })).toBe(false)
      expect(hasAzureDeployment({ name: 'svc', azure: {} })).toBe(false)
    })

    it('should return true if customDomainSource is present', () => {
      const service: Service = { name: 'svc', azure: { customDomainSource: 'user' } }
      expect(hasAzureDeployment(service)).toBe(true)
    })
  })

  describe('isAzureLogViewingSupported', () => {
    it('should return true for Container Apps', () => {
      expect(isAzureLogViewingSupported('containerapp')).toBe(true)
      expect(isAzureLogViewingSupported('ContainerApp')).toBe(true)
      expect(isAzureLogViewingSupported('CONTAINERAPP')).toBe(true)
    })

    it('should return true for App Service', () => {
      expect(isAzureLogViewingSupported('appservice')).toBe(true)
      expect(isAzureLogViewingSupported('AppService')).toBe(true)
      expect(isAzureLogViewingSupported('webapp')).toBe(true)
    })

    it('should return true for Azure Functions', () => {
      expect(isAzureLogViewingSupported('function')).toBe(true)
      expect(isAzureLogViewingSupported('Function')).toBe(true)
    })

    it('should return false for AKS', () => {
      expect(isAzureLogViewingSupported('aks')).toBe(false)
      expect(isAzureLogViewingSupported('AKS')).toBe(false)
    })

    it('should return false for ACI', () => {
      expect(isAzureLogViewingSupported('aci')).toBe(false)
      expect(isAzureLogViewingSupported('containerinstance')).toBe(false)
    })

    it('should return false for Static Web Apps', () => {
      expect(isAzureLogViewingSupported('staticwebapp')).toBe(false)
    })

    it('should return false for undefined or empty', () => {
      expect(isAzureLogViewingSupported(undefined)).toBe(false)
      expect(isAzureLogViewingSupported('')).toBe(false)
    })

    it('should return false for local services', () => {
      expect(isAzureLogViewingSupported('local')).toBe(false)
      expect(isAzureLogViewingSupported('localhost')).toBe(false)
    })
  })

  describe('getUnsupportedResourceTypeInfo', () => {
    it('should return coming soon true for AKS', () => {
      const info = getUnsupportedResourceTypeInfo('aks')
      expect(info.comingSoon).toBe(true)
      expect(info.displayName).toBe('Azure Kubernetes Service (AKS)')
    })

    it('should return coming soon true for ACI', () => {
      const info = getUnsupportedResourceTypeInfo('aci')
      expect(info.comingSoon).toBe(true)
      expect(info.displayName).toBe('Azure Container Instances')
    })

    it('should return coming soon true for containerinstance', () => {
      const info = getUnsupportedResourceTypeInfo('containerinstance')
      expect(info.comingSoon).toBe(true)
      expect(info.displayName).toBe('Azure Container Instances')
    })

    it('should return coming soon true for Static Web Apps', () => {
      const info = getUnsupportedResourceTypeInfo('staticwebapp')
      expect(info.comingSoon).toBe(true)
      expect(info.displayName).toBe('Static Web Apps')
    })

    it('should return coming soon true for Spring Apps', () => {
      const info = getUnsupportedResourceTypeInfo('springapp')
      expect(info.comingSoon).toBe(true)
      expect(info.displayName).toBe('Azure Spring Apps')
    })

    it('should return coming soon false for unknown types', () => {
      const info = getUnsupportedResourceTypeInfo('someunknowntype')
      expect(info.comingSoon).toBe(false)
      expect(info.displayName).toBe('someunknowntype')
    })

    it('should return unknown for undefined', () => {
      const info = getUnsupportedResourceTypeInfo(undefined)
      expect(info.comingSoon).toBe(false)
      expect(info.displayName).toBe('Unknown')
    })
  })

  describe('SUPPORTED_AZURE_LOG_RESOURCE_TYPES', () => {
    it('should include containerapp', () => {
      expect(SUPPORTED_AZURE_LOG_RESOURCE_TYPES).toContain('containerapp')
    })

    it('should include appservice', () => {
      expect(SUPPORTED_AZURE_LOG_RESOURCE_TYPES).toContain('appservice')
    })

    it('should include function', () => {
      expect(SUPPORTED_AZURE_LOG_RESOURCE_TYPES).toContain('function')
    })

    it('should include webapp', () => {
      expect(SUPPORTED_AZURE_LOG_RESOURCE_TYPES).toContain('webapp')
    })

    it('should NOT include aks', () => {
      expect(SUPPORTED_AZURE_LOG_RESOURCE_TYPES).not.toContain('aks')
    })

    it('should NOT include aci', () => {
      expect(SUPPORTED_AZURE_LOG_RESOURCE_TYPES).not.toContain('aci')
    })
  })

  describe('formatResourceType', () => {
    it('should format container app', () => {
      expect(formatResourceType('containerapp')).toBe('Container App')
    })

    it('should format app service', () => {
      expect(formatResourceType('appservice')).toBe('App Service')
    })

    it('should format function', () => {
      expect(formatResourceType('function')).toBe('Function App')
    })

    it('should format aks', () => {
      expect(formatResourceType('aks')).toBe('Kubernetes Service')
    })

    it('should return original for unknown types', () => {
      expect(formatResourceType('unknowntype')).toBe('unknowntype')
    })

    it('should return Unknown for undefined', () => {
      expect(formatResourceType(undefined)).toBe('Unknown')
    })
  })
})
