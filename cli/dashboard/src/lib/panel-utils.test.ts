/**
 * Tests for panel-utils helper functions
 */
import { describe, it, expect } from 'vitest'
import { 
  isAzureLogViewingSupported, 
  getUnsupportedResourceTypeInfo,
  SUPPORTED_AZURE_LOG_RESOURCE_TYPES,
  formatResourceType,
} from './panel-utils'

describe('panel-utils', () => {
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
