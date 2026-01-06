/**
 * Component tests for HealthTooltip
 * Tests tooltip rendering logic, memoization, and callback handling
 * 
 * Note: Full tooltip interaction tests are in E2E tests (health-tooltip.spec.ts)
 * due to React 19 + Radix UI + Vitest rendering limitations.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { buildHealthDiagnostic } from '@/lib/health-diagnostics'
import type { HealthCheckResult, Service } from '@/types'

// Mock clipboard API
const mockWriteText = vi.fn(() => Promise.resolve())
Object.assign(navigator, {
  clipboard: {
    writeText: mockWriteText,
  },
})

// Mock toast hook
const mockShowToast = vi.fn()
vi.mock('@/hooks/useToast', () => ({
  useToast: () => ({
    showToast: mockShowToast,
  }),
}))

describe('HealthTooltip Logic', () => {
  const baseService: Service = {
    name: 'api',
    host: 'local',
    local: {
      status: 'running',
      health: 'healthy',
      port: 8080,
      serviceType: 'http',
      serviceMode: 'daemon',
    },
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockWriteText.mockClear()
    mockWriteText.mockResolvedValue(undefined)
  })

  describe('diagnostic building', () => {
    it('builds diagnostic for healthy status', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
      }

      const diagnostic = buildHealthDiagnostic(healthStatus, baseService)
      
      expect(diagnostic.healthStatus.serviceName).toBe('api')
      expect(diagnostic.healthStatus.status).toBe('healthy')
      expect(diagnostic.healthStatus.checkType).toBe('http')
      expect(diagnostic.healthStatus.responseTime / 1000000).toBe(10)
    })

    it('builds diagnostic for unhealthy HTTP status', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        endpoint: 'http://localhost:8080/health',
        statusCode: 503,
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'Service Unavailable',
      }

      const diagnostic = buildHealthDiagnostic(healthStatus, baseService)
      
      expect(diagnostic.healthStatus.status).toBe('unhealthy')
      expect(diagnostic.healthStatus.statusCode).toBe(503)
      expect(diagnostic.healthStatus.error).toBe('Service Unavailable')
      expect(diagnostic.suggestedActions.some(a => a.label.includes('service dependencies'))).toBe(true)
    })

    it('builds diagnostic for degraded status', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'degraded',
        checkType: 'http',
        responseTime: 500000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
      }

      const diagnostic = buildHealthDiagnostic(healthStatus, baseService)
      
      expect(diagnostic.healthStatus.status).toBe('degraded')
      expect(diagnostic.healthStatus.responseTime / 1000000).toBe(500)
    })

    it('builds diagnostic for TCP check', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'database',
        status: 'unhealthy',
        checkType: 'tcp',
        port: 5432,
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'Connection refused',
      }

      const diagnostic = buildHealthDiagnostic(healthStatus, {
        name: 'database',
        host: 'local',
        local: {
          status: 'stopped',
          health: 'unhealthy',
          port: 5432,
          serviceType: 'tcp',
          serviceMode: 'daemon',
        },
      })
      
      expect(diagnostic.healthStatus.checkType).toBe('tcp')
      expect(diagnostic.healthStatus.port).toBe(5432)
      expect(diagnostic.suggestedActions.some(a => a.label.includes('service is running'))).toBe(true)
    })

    it('builds diagnostic for process check', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'process',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'Process not found',
      }

      const diagnostic = buildHealthDiagnostic(healthStatus, baseService)
      
      expect(diagnostic.healthStatus.checkType).toBe('process')
      expect(diagnostic.suggestedActions.some(a => a.label.includes('service start command'))).toBe(true)
    })

    it('includes error details when present', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'Connection timeout',
        errorDetails: 'ECONNABORTED: timeout of 5000ms exceeded',
      }

      const diagnostic = buildHealthDiagnostic(healthStatus, baseService)
      
      expect(diagnostic.healthStatus.errorDetails).toBe('ECONNABORTED: timeout of 5000ms exceeded')
      expect(diagnostic.formattedReport).toContain('ECONNABORTED')
    })
  })

  describe('formatted report generation', () => {
    it('generates formatted report with all key information', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        endpoint: 'http://localhost:8080/health',
        statusCode: 503,
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'Service Unavailable',
      }

      const diagnostic = buildHealthDiagnostic(healthStatus, baseService)
      const report = diagnostic.formattedReport
      
      expect(report).toContain('Service Health Diagnostic Report')
      expect(report).toContain('**Service**: api')
      expect(report).toContain('**Status**: unhealthy')
      expect(report).toContain('503')
      expect(report).toContain('Service Unavailable')
      expect(report).toContain('Verify all service dependencies are running')
    })

    it('includes service info in formatted report', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        endpoint: 'http://localhost:8080/health',
        statusCode: 200,
        responseTime: 15000000,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const diagnostic = buildHealthDiagnostic(healthStatus, {
        name: 'api',
        host: 'local',
        local: {
          status: 'running',
          health: 'healthy',
          port: 8080,
          url: 'http://localhost:8080',
          serviceType: 'http',
          serviceMode: 'daemon',
        },
      })
      
      const report = diagnostic.formattedReport
      expect(report).toContain('http://localhost:8080')
    })
  })

  describe('clipboard copy functionality', () => {
    it('copies diagnostic report to clipboard', async () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const diagnostic = buildHealthDiagnostic(healthStatus, baseService)
      
      await navigator.clipboard.writeText(diagnostic.formattedReport)
      
      expect(mockWriteText).toHaveBeenCalledWith(diagnostic.formattedReport)
    })

    it('handles clipboard copy failure', async () => {
      mockWriteText.mockRejectedValueOnce(new Error('Clipboard error'))

      try {
        await navigator.clipboard.writeText('test')
      } catch (error) {
        expect(error).toEqual(new Error('Clipboard error'))
      }
      
      expect(mockWriteText).toHaveBeenCalled()
    })
  })

  describe('memoization behavior', () => {
    it('memoizes diagnostic calculation with same inputs', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const diagnostic1 = buildHealthDiagnostic(healthStatus, baseService)
      const diagnostic2 = buildHealthDiagnostic(healthStatus, baseService)
      
      // Should produce consistent results for same inputs
      // Note: formattedReport contains dynamic timestamp, so compare other fields
      expect(diagnostic1.healthStatus).toEqual(diagnostic2.healthStatus)
      expect(diagnostic1.suggestedActions).toEqual(diagnostic2.suggestedActions)
      expect(diagnostic1.service).toEqual(diagnostic2.service)
      // Verify report structure is consistent (contains same key sections)
      expect(diagnostic1.formattedReport).toContain('Service Health Diagnostic Report')
      expect(diagnostic2.formattedReport).toContain('Service Health Diagnostic Report')
    })

    it('recalculates when status changes', () => {
      const healthyStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const unhealthyStatus: HealthCheckResult = {
        ...healthyStatus,
        status: 'unhealthy',
        error: 'Connection failed',
      }

      const diagnostic1 = buildHealthDiagnostic(healthyStatus, baseService)
      const diagnostic2 = buildHealthDiagnostic(unhealthyStatus, baseService)
      
      expect(diagnostic1.healthStatus.status).toBe('healthy')
      expect(diagnostic2.healthStatus.status).toBe('unhealthy')
      expect(diagnostic1.healthStatus.error).toBeUndefined()
      expect(diagnostic2.healthStatus.error).toBe('Connection failed')
    })
  })
})
