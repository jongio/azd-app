/**
 * Unit tests for health-diagnostics.ts
 * Tests diagnostic helpers for service health monitoring
 */

import { describe, it, expect } from 'vitest'
import {
  buildHealthDiagnostic,
  getSuggestedActions,
  getHTTPSpecificActions,
  getTCPSpecificActions,
  getProcessSpecificActions,
  formatDiagnosticReport,
} from './health-diagnostics'
import type { HealthCheckResult, Service } from '@/types'

describe('health-diagnostics', () => {
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

  describe('buildHealthDiagnostic', () => {
    it('builds complete diagnostic for healthy service', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        endpoint: 'http://localhost:8080/health',
        statusCode: 200,
        responseTime: 12000000, // 12ms in nanoseconds
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 300000000000, // 5 minutes in nanoseconds
        port: 8080,
      }

      const diagnostic = buildHealthDiagnostic(healthStatus, baseService)

      expect(diagnostic.service).toBe(baseService)
      expect(diagnostic.healthStatus).toBe(healthStatus)
      expect(diagnostic.suggestedActions).toEqual([])
      expect(diagnostic.formattedReport).toContain('Service Health Diagnostic Report')
      expect(diagnostic.formattedReport).toContain('api')
      expect(diagnostic.formattedReport).toContain('healthy')
    })

    it('builds diagnostic for unhealthy service with actions', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        endpoint: 'http://localhost:8080/health',
        statusCode: 503,
        responseTime: 45000000, // 45ms
        timestamp: '2025-12-29T10:00:00Z',
        error: 'HTTP 503: Service Unavailable',
        errorDetails: 'Database connection pool exhausted',
        consecutiveFailures: 3,
        uptime: 900000000000, // 15 minutes
        port: 8080,
      }

      const diagnostic = buildHealthDiagnostic(healthStatus, baseService)

      expect(diagnostic.suggestedActions.length).toBeGreaterThan(0)
      expect(diagnostic.suggestedActions[0].label).toContain('Check service logs')
      expect(diagnostic.suggestedActions[0].command).toContain('azd app logs --service api')
      expect(diagnostic.formattedReport).toContain('HTTP 503')
      expect(diagnostic.formattedReport).toContain('**Consecutive Failures**: 3')
    })

    it('builds diagnostic for degraded service', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'degraded',
        checkType: 'http',
        endpoint: 'http://localhost:8080/health',
        statusCode: 200,
        responseTime: 1500000000, // 1500ms - slow
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 600000000000,
        port: 8080,
      }

      const diagnostic = buildHealthDiagnostic(healthStatus, baseService)

      expect(diagnostic.suggestedActions).toBeDefined()
      expect(diagnostic.formattedReport).toContain('degraded')
    })

    it('builds diagnostic for unknown status', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unknown',
        checkType: 'process',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'No health check configured',
      }

      const diagnostic = buildHealthDiagnostic(healthStatus, baseService)

      expect(diagnostic.suggestedActions).toBeDefined()
      expect(diagnostic.formattedReport).toContain('unknown')
    })
  })

  describe('getSuggestedActions', () => {
    it('suggests viewing logs for unhealthy services', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'Connection failed',
      }

      const actions = getSuggestedActions(healthStatus, baseService)

      expect(actions.length).toBeGreaterThan(0)
      expect(actions[0].label).toBe('Check service logs')
      expect(actions[0].icon).toBe('terminal')
      expect(actions[0].command).toBe('azd app logs --service api')
    })

    it('includes HTTP-specific actions for HTTP checks', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        statusCode: 503,
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const actions = getSuggestedActions(healthStatus, baseService)

      expect(actions.length).toBeGreaterThan(1)
      expect(actions.some(a => a.label.includes('dependencies'))).toBe(true)
    })

    it('includes TCP-specific actions for TCP checks', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'tcp',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'connection refused',
      }

      const actions = getSuggestedActions(healthStatus, baseService)

      expect(actions.some(a => a.label.includes('running'))).toBe(true)
    })

    it('includes process-specific actions for process checks', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'process',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'process not running',
        pid: 12345,
      }

      const actions = getSuggestedActions(healthStatus, baseService)

      expect(actions.some(a => a.label.includes('start'))).toBe(true)
    })

    it('includes backend suggestion if provided', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        details: {
          suggestion: 'Custom backend suggestion',
        },
      }

      const actions = getSuggestedActions(healthStatus, baseService)

      expect(actions.some(a => a.label === 'Custom backend suggestion')).toBe(true)
    })

    it('does not duplicate backend suggestions', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        details: {
          suggestion: 'Check service logs',
        },
      }

      const actions = getSuggestedActions(healthStatus, baseService)

      const logActions = actions.filter(a => a.label === 'Check service logs')
      expect(logActions.length).toBe(1)
    })

    it('adds performance checks for degraded services', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'degraded',
        checkType: 'http',
        responseTime: 1500000000,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const actions = getSuggestedActions(healthStatus, baseService)

      expect(actions.some(a => a.label.includes('CPU'))).toBe(true)
      expect(actions.some(a => a.label.includes('performance'))).toBe(true)
    })

    it('adds consecutive failure warning', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        consecutiveFailures: 5,
      }

      const actions = getSuggestedActions(healthStatus, baseService)

      expect(actions.some(a => a.label.includes('5 times consecutively'))).toBe(true)
      expect(actions.some(a => a.label.includes('restarting'))).toBe(true)
    })

    it('returns empty array for healthy services', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const actions = getSuggestedActions(healthStatus, baseService)

      expect(actions).toEqual([])
    })
  })

  describe('getHTTPSpecificActions', () => {
    it('provides actions for 503 Service Unavailable', () => {
      const actions = getHTTPSpecificActions(503, baseService)

      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Verify all service dependencies are running' })
      )
      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Check database connection status' })
      )
      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Review connection pool settings' })
      )
    })

    it('provides actions for 500 Internal Server Error', () => {
      const actions = getHTTPSpecificActions(500, baseService)

      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Check application logs for errors' })
      )
      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Review error stack traces' })
      )
    })

    it('provides actions for 404 Not Found', () => {
      const actions = getHTTPSpecificActions(404, baseService)

      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Verify health endpoint configuration' })
      )
      expect(actions.some(a => a.docsUrl?.includes('health-checks'))).toBe(true)
    })

    it('provides actions for 401 Unauthorized', () => {
      const actions = getHTTPSpecificActions(401, baseService)

      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Check authentication credentials' })
      )
      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Verify API keys and tokens' })
      )
    })

    it('provides actions for 403 Forbidden', () => {
      const actions = getHTTPSpecificActions(403, baseService)

      expect(actions.some(a => a.label.includes('authentication'))).toBe(true)
    })

    it('provides actions for 429 Too Many Requests', () => {
      const actions = getHTTPSpecificActions(429, baseService)

      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Reduce request rate' })
      )
      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Check rate limiting quotas' })
      )
    })

    it('provides actions for 408 Request Timeout', () => {
      const actions = getHTTPSpecificActions(408, baseService)

      expect(actions.some(a => a.label.includes('network'))).toBe(true)
    })

    it('provides actions for 504 Gateway Timeout', () => {
      const actions = getHTTPSpecificActions(504, baseService)

      // 504 matches the >= 500 && < 600 case (server error)
      expect(actions.length).toBeGreaterThan(0)
      expect(actions.some(a => a.label.includes('logs'))).toBe(true)
    })

    it('provides generic server error actions for other 5xx codes', () => {
      const actions = getHTTPSpecificActions(502, baseService)

      expect(actions.some(a => a.label.includes('logs'))).toBe(true)
    })

    it('returns empty array for success codes', () => {
      const actions = getHTTPSpecificActions(200, baseService)

      expect(actions).toEqual([])
    })
  })

  describe('getTCPSpecificActions', () => {
    it('provides actions for connection refused', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'tcp',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'connection refused',
      }

      const actions = getTCPSpecificActions(healthStatus, baseService)

      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Verify service is running' })
      )
      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Check if port is correct' })
      )
      expect(actions.some(a => a.label.includes('restart'))).toBe(true)
    })

    it('provides actions for timeout', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'tcp',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'connection timeout',
      }

      const actions = getTCPSpecificActions(healthStatus, baseService)

      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Check network connectivity' })
      )
      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Verify firewall rules' })
      )
    })

    it('provides actions for port binding issues', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'tcp',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'port already in use',
      }

      const actions = getTCPSpecificActions(healthStatus, baseService)

      expect(actions.some(a => a.label.includes('port'))).toBe(true)
    })

    it('returns empty array for unknown TCP errors', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'tcp',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'unknown tcp error',
      }

      const actions = getTCPSpecificActions(healthStatus, baseService)

      expect(actions).toEqual([])
    })
  })

  describe('getProcessSpecificActions', () => {
    it('provides actions for process not running', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'process',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'process not running',
        pid: 12345,
      }

      const actions = getProcessSpecificActions(healthStatus, baseService)

      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Verify service start command' })
      )
      expect(actions.some(a => a.label.includes('Start service'))).toBe(true)
    })

    it('provides actions for crashed process', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'process',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'process crashed with exit code 1',
      }

      const actions = getProcessSpecificActions(healthStatus, baseService)

      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Review crash logs for exit code' })
      )
      expect(actions).toContainEqual(
        expect.objectContaining({ label: 'Check for runtime errors' })
      )
    })

    it('provides actions for pattern not matched', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'process',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'output pattern not matched',
      }

      const actions = getProcessSpecificActions(healthStatus, baseService)

      // Should have at least one action
      expect(actions.length).toBeGreaterThan(0)
      // Should include pattern-related actions
      expect(actions.some(a => a.label.includes('pattern'))).toBe(true)
    })

    it('returns empty array for unknown process errors', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'process',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'unknown error',
      }

      const actions = getProcessSpecificActions(healthStatus, baseService)

      expect(actions).toEqual([])
    })
  })

  describe('formatDiagnosticReport', () => {
    it('formats complete report with all sections', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        endpoint: 'http://localhost:8080/health',
        statusCode: 503,
        responseTime: 45000000,
        timestamp: '2025-12-29T10:00:00Z',
        error: 'HTTP 503: Service Unavailable',
        errorDetails: 'Database connection pool exhausted',
        consecutiveFailures: 3,
        lastSuccessTime: '2025-12-29T09:50:00Z',
        uptime: 900000000000,
        port: 8080,
        pid: 12345,
      }

      const actions = getSuggestedActions(healthStatus, baseService)
      const report = formatDiagnosticReport(healthStatus, baseService, actions)

      // Header
      expect(report).toContain('# Service Health Diagnostic Report')
      expect(report).toContain('**Service**: api')
      expect(report).toContain('**Status**: unhealthy')
      expect(report).toContain('**Timestamp**:')

      // Health Check section
      expect(report).toContain('## Health Check')
      expect(report).toContain('**Type**: HTTP')
      expect(report).toContain('**Endpoint**: http://localhost:8080/health')
      expect(report).toContain('**Status Code**: 503')
      expect(report).toContain('**Consecutive Failures**: 3')
      expect(report).toContain('**Last Success**:')

      // Error section
      expect(report).toContain('## Error')
      expect(report).toContain('HTTP 503: Service Unavailable')
      expect(report).toContain('Database connection pool exhausted')

      // Service Info section
      expect(report).toContain('## Service Info')
      expect(report).toContain('**Port**: 8080')
      expect(report).toContain('**PID**: 12345')

      // Suggested Actions section
      expect(report).toContain('## Suggested Actions')

      // Footer
      expect(report).toContain('Generated by azd app health')
    })

    it('formats report without optional fields', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'process',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
      }

      const report = formatDiagnosticReport(healthStatus, baseService, [])

      expect(report).toContain('**Service**: api')
      expect(report).toContain('**Status**: healthy')
      expect(report).not.toContain('## Error')
      expect(report).not.toContain('**Endpoint**:')
      expect(report).not.toContain('**Status Code**:')
      expect(report).not.toContain('## Suggested Actions')
    })

    it('includes service type and mode if available', () => {
      const service: Service = {
        ...baseService,
        local: {
          ...baseService.local!,
          serviceType: 'http',
          serviceMode: 'daemon',
        },
      }

      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
      }

      const report = formatDiagnosticReport(healthStatus, service, [])

      expect(report).toContain('**Type**: http')
      expect(report).toContain('**Mode**: daemon')
    })

    it('formats actions with commands and docs URLs', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const actions = [
        { label: 'Check logs', command: 'azd app logs --service api' },
        { label: 'Read docs', docsUrl: 'https://example.com/docs' },
      ]

      const report = formatDiagnosticReport(healthStatus, baseService, actions)

      expect(report).toContain('1. Check logs → `azd app logs --service api`')
      expect(report).toContain('2. Read docs → [Documentation](https://example.com/docs)')
    })

    it('handles zero port gracefully', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'process',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        port: 0,
        uptime: 60000000000,
      }

      const report = formatDiagnosticReport(healthStatus, baseService, [])

      expect(report).not.toContain('**Port**: 0')
    })

    it('handles missing uptime gracefully', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const report = formatDiagnosticReport(healthStatus, baseService, [])

      expect(report).toContain('**Uptime**:')
    })
  })

  describe('edge cases', () => {
    it('handles null/undefined values in health status', () => {
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unknown',
        checkType: 'http',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
      }

      const diagnostic = buildHealthDiagnostic(healthStatus, baseService)

      expect(diagnostic).toBeDefined()
      expect(diagnostic.formattedReport).toContain('api')
    })

    it('handles service without local config', () => {
      const service: Service = {
        name: 'api',
        host: 'local',
      }

      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 10000000,
        timestamp: '2025-12-29T10:00:00Z',
        uptime: 60000000000,
      }

      const diagnostic = buildHealthDiagnostic(healthStatus, service)

      expect(diagnostic.formattedReport).toBeDefined()
    })

    it('handles very long error messages', () => {
      const longError = 'x'.repeat(500)
      const healthStatus: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        responseTime: 0,
        timestamp: '2025-12-29T10:00:00Z',
        error: longError,
      }

      const report = formatDiagnosticReport(healthStatus, baseService, [])

      expect(report).toContain(longError)
    })
  })
})
