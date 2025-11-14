import { describe, it, expect } from 'vitest'
import {
  getServiceStatus,
  getServiceHealth,
  isServiceHealthy,
  getStatusColor,
  getStatusLabel,
  formatUptime,
  isValidAzureIdentifier,
  groupServicesBy,
} from './serviceUtils'
import type { Service } from '@/types'

describe('serviceUtils', () => {
  const mockService: Service = {
    name: 'test-service',
    language: 'TypeScript',
    status: 'running',
    health: 'healthy',
    local: {
      status: 'running',
      health: 'healthy',
      port: 3000,
      url: 'http://localhost:3000',
    },
    environmentVariables: {},
  }

  describe('getServiceStatus', () => {
    it('should return local status when available', () => {
      expect(getServiceStatus(mockService)).toBe('running')
    })

    it('should return top-level status when local not available', () => {
      const service = { ...mockService, local: undefined }
      expect(getServiceStatus(service)).toBe('running')
    })

    it('should return not-running when no status available', () => {
      const service = { ...mockService, status: undefined, local: undefined }
      expect(getServiceStatus(service)).toBe('not-running')
    })
  })

  describe('getServiceHealth', () => {
    it('should return local health when available', () => {
      expect(getServiceHealth(mockService)).toBe('healthy')
    })

    it('should return top-level health when local not available', () => {
      const service = { ...mockService, local: undefined }
      expect(getServiceHealth(service)).toBe('healthy')
    })

    it('should return unknown when no health available', () => {
      const service = { ...mockService, health: undefined, local: undefined }
      expect(getServiceHealth(service)).toBe('unknown')
    })
  })

  describe('isServiceHealthy', () => {
    it('should return true for running and healthy service', () => {
      expect(isServiceHealthy(mockService)).toBe(true)
    })

    it('should return true for ready and healthy service', () => {
      const service = { ...mockService, local: { ...mockService.local!, status: 'ready' } }
      expect(isServiceHealthy(service)).toBe(true)
    })

    it('should return false for running but unhealthy service', () => {
      const service = { ...mockService, local: { ...mockService.local!, health: 'unhealthy' } }
      expect(isServiceHealthy(service)).toBe(false)
    })

    it('should return false for not-running service', () => {
      const service = { ...mockService, local: { ...mockService.local!, status: 'not-running' } }
      expect(isServiceHealthy(service)).toBe(false)
    })
  })

  describe('getStatusColor', () => {
    it('should return success color for healthy running service', () => {
      expect(getStatusColor(mockService)).toBe('bg-success')
    })

    it('should return success color for healthy ready service', () => {
      const service = { ...mockService, local: { ...mockService.local!, status: 'ready' } }
      expect(getStatusColor(service)).toBe('bg-success')
    })

    it('should return warning color for starting service', () => {
      const service = { ...mockService, local: { ...mockService.local!, status: 'starting' } }
      expect(getStatusColor(service)).toBe('bg-warning')
    })

    it('should return destructive color for error status', () => {
      const service = { ...mockService, local: { ...mockService.local!, status: 'error' } }
      expect(getStatusColor(service)).toBe('bg-destructive')
    })

    it('should return destructive color for unhealthy service', () => {
      const service = { ...mockService, local: { ...mockService.local!, health: 'unhealthy' } }
      expect(getStatusColor(service)).toBe('bg-destructive')
    })

    it('should return muted color for unknown status', () => {
      const service = { ...mockService, local: { ...mockService.local!, status: 'not-running', health: 'unknown' } }
      expect(getStatusColor(service)).toBe('bg-muted-foreground')
    })
  })

  describe('getStatusLabel', () => {
    it('should return Healthy for healthy running service', () => {
      expect(getStatusLabel(mockService)).toBe('Healthy')
    })

    it('should return Unhealthy for unhealthy service', () => {
      const service = { ...mockService, local: { ...mockService.local!, health: 'unhealthy' } }
      expect(getStatusLabel(service)).toBe('Unhealthy')
    })

    it('should return Starting for starting service', () => {
      const service = { ...mockService, local: { ...mockService.local!, status: 'starting' } }
      expect(getStatusLabel(service)).toBe('Starting')
    })

    it('should return Error for error status', () => {
      const service = { ...mockService, local: { ...mockService.local!, status: 'error' } }
      expect(getStatusLabel(service)).toBe('Error')
    })

    it('should return Not Running for not-running status', () => {
      const service = { ...mockService, local: { ...mockService.local!, status: 'not-running' } }
      expect(getStatusLabel(service)).toBe('Not Running')
    })

    it('should return Unknown for unknown status', () => {
      const service = { ...mockService, local: undefined, status: 'unknown', health: 'unknown' }
      expect(getStatusLabel(service)).toBe('Unknown')
    })
  })

  describe('formatUptime', () => {
    it('should return N/A for undefined uptime', () => {
      expect(formatUptime(undefined)).toBe('N/A')
    })

    it('should return N/A for zero uptime', () => {
      // Based on implementation: if (!seconds) returns N/A for 0
      expect(formatUptime(0)).toBe('N/A')
    })

    it('should format seconds only', () => {
      expect(formatUptime(45)).toBe('45s')
    })

    it('should format minutes and seconds', () => {
      expect(formatUptime(125)).toBe('2m 5s')
    })

    it('should format hours, minutes and seconds', () => {
      expect(formatUptime(3665)).toBe('1h 1m 5s')
    })
  })

  describe('isValidAzureIdentifier', () => {
    it('should return true for valid identifiers', () => {
      expect(isValidAzureIdentifier('my-resource-group')).toBe(true)
      expect(isValidAzureIdentifier('resource_123')).toBe(true)
      expect(isValidAzureIdentifier('MyApp')).toBe(true)
    })

    it('should return false for identifiers with semicolons', () => {
      expect(isValidAzureIdentifier('resource;rm -rf')).toBe(false)
    })

    it('should return false for identifiers with pipes', () => {
      expect(isValidAzureIdentifier('resource|echo')).toBe(false)
    })

    it('should return false for identifiers with ampersands', () => {
      expect(isValidAzureIdentifier('resource&cmd')).toBe(false)
    })

    it('should return false for identifiers with dollar signs', () => {
      expect(isValidAzureIdentifier('resource$var')).toBe(false)
    })

    it('should return false for identifiers with backticks', () => {
      expect(isValidAzureIdentifier('resource`cmd`')).toBe(false)
    })

    it('should return false for identifiers with newlines', () => {
      expect(isValidAzureIdentifier('resource\nrm')).toBe(false)
    })

    it('should return false for identifiers with carriage returns', () => {
      expect(isValidAzureIdentifier('resource\rrm')).toBe(false)
    })
  })

  describe('groupServicesBy', () => {
    const services: Service[] = [
      { ...mockService, name: 'api', language: 'TypeScript' },
      { ...mockService, name: 'worker', language: 'TypeScript' },
      { ...mockService, name: 'db', language: 'PostgreSQL' },
      { ...mockService, name: 'cache', language: 'Redis' },
    ]

    it('should group services by language', () => {
      const grouped = groupServicesBy(services, 'language')
      expect(Object.keys(grouped)).toEqual(['TypeScript', 'PostgreSQL', 'Redis'])
      expect(grouped['TypeScript']).toHaveLength(2)
      expect(grouped['PostgreSQL']).toHaveLength(1)
      expect(grouped['Redis']).toHaveLength(1)
    })

    it('should group services with Unknown for missing property', () => {
      const servicesWithMissing = [
        ...services,
        { ...mockService, name: 'unknown', language: undefined },
      ]
      const grouped = groupServicesBy(servicesWithMissing, 'language')
      expect(grouped['Unknown']).toHaveLength(1)
    })

    it('should return empty object for empty services array', () => {
      const grouped = groupServicesBy([], 'language')
      expect(grouped).toEqual({})
    })
  })
})
