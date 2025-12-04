import { describe, it, expect } from 'vitest'
import {
  getEffectiveStatus,
  getStatusDisplay,
  isServiceHealthy,
  formatRelativeTime,
  formatStartTime,
  formatLogTimestamp,
  formatResponseTime,
  formatUptime,
  getCheckTypeDisplay,
  mergeHealthIntoService,
  getLogPaneVisualStatus,
  getServiceTypeBadgeConfig,
  getServiceModeBadgeConfig,
  getServiceTypeLabel,
  getServiceModeLabel,
  isProcessService,
  isContinuousMode,
  isOneTimeMode,
  getStatusIndicator,
  getStatusBadgeConfig,
  normalizeHealthStatus,
  normalizeLifecycleState,
} from './service-utils'
import type { Service, HealthCheckResult } from '@/types'

describe('service-utils', () => {
  describe('normalizeHealthStatus', () => {
    it('should return unknown for undefined', () => {
      expect(normalizeHealthStatus(undefined)).toBe('unknown')
    })

    it('should return unknown for empty string', () => {
      expect(normalizeHealthStatus('')).toBe('unknown')
    })

    it('should normalize "starting" to "unknown"', () => {
      // Backend sends "starting" during grace period - this is a lifecycle state, not health
      expect(normalizeHealthStatus('starting')).toBe('unknown')
    })

    it('should pass through valid health statuses', () => {
      expect(normalizeHealthStatus('healthy')).toBe('healthy')
      expect(normalizeHealthStatus('degraded')).toBe('degraded')
      expect(normalizeHealthStatus('unhealthy')).toBe('unhealthy')
      expect(normalizeHealthStatus('unknown')).toBe('unknown')
    })

    it('should return unknown for invalid values', () => {
      expect(normalizeHealthStatus('invalid')).toBe('unknown')
      expect(normalizeHealthStatus('running')).toBe('unknown')
    })
  })

  describe('normalizeLifecycleState', () => {
    it('should return not-started for undefined', () => {
      expect(normalizeLifecycleState(undefined)).toBe('not-started')
    })

    it('should map legacy status values', () => {
      expect(normalizeLifecycleState('ready')).toBe('running')
      expect(normalizeLifecycleState('error')).toBe('failed')
      expect(normalizeLifecycleState('watching')).toBe('running')
      expect(normalizeLifecycleState('building')).toBe('running')
      expect(normalizeLifecycleState('built')).toBe('completed')
      expect(normalizeLifecycleState('not-running')).toBe('not-started')
    })

    it('should pass through valid lifecycle states', () => {
      expect(normalizeLifecycleState('not-started')).toBe('not-started')
      expect(normalizeLifecycleState('starting')).toBe('starting')
      expect(normalizeLifecycleState('running')).toBe('running')
      expect(normalizeLifecycleState('stopping')).toBe('stopping')
      expect(normalizeLifecycleState('stopped')).toBe('stopped')
      expect(normalizeLifecycleState('restarting')).toBe('restarting')
      expect(normalizeLifecycleState('completed')).toBe('completed')
      expect(normalizeLifecycleState('failed')).toBe('failed')
    })

    it('should return not-started for invalid values', () => {
      expect(normalizeLifecycleState('invalid')).toBe('not-started')
      expect(normalizeLifecycleState('healthy')).toBe('not-started')
    })
  })

  describe('getEffectiveStatus', () => {
    it('should prefer local status when available', () => {
      const service: Service = {
        name: 'api',
        status: 'stopped',
        health: 'unknown',
        local: {
          status: 'running',
          health: 'healthy',
          port: 5000,
          startTime: '2024-01-01T00:00:00Z',
        },
      }

      const result = getEffectiveStatus(service)
      expect(result.status).toBe('running')
      expect(result.health).toBe('healthy')
    })

    it('should use top-level status when local is not available', () => {
      const service: Service = {
        name: 'api',
        status: 'running',
        health: 'healthy',
      }

      const result = getEffectiveStatus(service)
      expect(result.status).toBe('running')
      expect(result.health).toBe('healthy')
    })

    it('should return not-running and unknown for empty service', () => {
      const service: Service = {
        name: 'api',
      }

      const result = getEffectiveStatus(service)
      expect(result.status).toBe('not-running')
      expect(result.health).toBe('unknown')
    })
  })

  describe('getStatusDisplay', () => {
    it('should return Running for ready/running and healthy', () => {
      const display = getStatusDisplay('running', 'healthy')
      expect(display.text).toBe('Running')
      expect(display.badgeVariant).toBe('success')
    })

    it('should return Running for ready status with healthy health', () => {
      const display = getStatusDisplay('ready', 'healthy')
      expect(display.text).toBe('Running')
    })

    it('should return Degraded for running and degraded', () => {
      const display = getStatusDisplay('running', 'degraded')
      expect(display.text).toBe('Degraded')
      expect(display.badgeVariant).toBe('warning')
    })

    it('should return Unhealthy for running and unhealthy', () => {
      const display = getStatusDisplay('running', 'unhealthy')
      expect(display.text).toBe('Unhealthy')
      expect(display.badgeVariant).toBe('destructive')
    })

    it('should return Starting for starting status', () => {
      const display = getStatusDisplay('starting', 'unknown')
      expect(display.text).toBe('Starting')
      expect(display.badgeVariant).toBe('warning')
    })

    it('should return Error for error status', () => {
      const display = getStatusDisplay('error', 'unknown')
      expect(display.text).toBe('Error')
      expect(display.badgeVariant).toBe('destructive')
    })

    it('should return Stopping for stopping status', () => {
      const display = getStatusDisplay('stopping', 'unknown')
      expect(display.text).toBe('Stopping')
      expect(display.badgeVariant).toBe('secondary')
    })

    it('should return Stopped for stopped status', () => {
      const display = getStatusDisplay('stopped', 'unknown')
      expect(display.text).toBe('Stopped')
      expect(display.badgeVariant).toBe('secondary')
    })

    it('should return Not Running for not-running status', () => {
      const display = getStatusDisplay('not-running', 'unknown')
      expect(display.text).toBe('Not Running')
    })

    it('should return Unknown for unrecognized status', () => {
      const display = getStatusDisplay('something-else', 'unknown')
      expect(display.text).toBe('Unknown')
      expect(display.badgeVariant).toBe('secondary')
    })
  })

  describe('isServiceHealthy', () => {
    it('should return true for running and healthy', () => {
      expect(isServiceHealthy('running', 'healthy')).toBe(true)
    })

    it('should return true for ready and healthy', () => {
      expect(isServiceHealthy('ready', 'healthy')).toBe(true)
    })

    it('should return false for running but unhealthy', () => {
      expect(isServiceHealthy('running', 'unhealthy')).toBe(false)
    })

    it('should return false for stopped', () => {
      expect(isServiceHealthy('stopped', 'unknown')).toBe(false)
    })
  })

  describe('formatRelativeTime', () => {
    it('should return N/A for undefined', () => {
      expect(formatRelativeTime(undefined)).toBe('N/A')
    })

    it('should return N/A for empty string', () => {
      expect(formatRelativeTime('')).toBe('N/A')
    })

    it('should format seconds ago', () => {
      const now = new Date()
      const thirtySecsAgo = new Date(now.getTime() - 30 * 1000)
      const result = formatRelativeTime(thirtySecsAgo.toISOString())
      expect(result).toMatch(/\d+s ago/)
    })

    it('should format minutes ago', () => {
      const now = new Date()
      const fiveMinAgo = new Date(now.getTime() - 5 * 60 * 1000)
      const result = formatRelativeTime(fiveMinAgo.toISOString())
      expect(result).toMatch(/\d+m ago/)
    })

    it('should format hours ago', () => {
      const now = new Date()
      const twoHoursAgo = new Date(now.getTime() - 2 * 60 * 60 * 1000)
      const result = formatRelativeTime(twoHoursAgo.toISOString())
      expect(result).toMatch(/\d+h ago/)
    })

    it('should format days ago', () => {
      const now = new Date()
      const twoDaysAgo = new Date(now.getTime() - 2 * 24 * 60 * 60 * 1000)
      const result = formatRelativeTime(twoDaysAgo.toISOString())
      expect(result).toMatch(/\d+d ago/)
    })

    it('should handle invalid date string gracefully', () => {
      // Invalid dates result in NaN calculations but don't throw
      const result = formatRelativeTime('invalid-date')
      expect(result).toContain('NaN')
    })
  })

  describe('formatStartTime', () => {
    it('should return - for undefined', () => {
      expect(formatStartTime(undefined)).toBe('-')
    })

    it('should format time as HH:MM:SS', () => {
      // Test with a known UTC time that will format consistently
      const result = formatStartTime('2024-01-15T10:30:45.000Z')
      // Should contain colons for time format
      expect(result).toMatch(/\d{1,2}:\d{2}:\d{2}/)
    })

    it('should handle invalid date string gracefully', () => {
      const result = formatStartTime('invalid-date')
      // Returns the original string on parse failure
      expect(result).toBe('invalid-date')
    })
  })

  describe('formatLogTimestamp', () => {
    it('should format timestamp as HH:MM:SS.mmm', () => {
      const result = formatLogTimestamp('2024-01-15T10:30:45.123Z')
      // The exact format may vary by locale, but should contain the milliseconds
      expect(result).toContain('123')
    })

    it('should handle invalid timestamp gracefully', () => {
      // Invalid timestamps result in Invalid Date but don't throw
      const result = formatLogTimestamp('invalid')
      expect(result).toContain('Invalid Date')
    })
  })

  describe('formatResponseTime', () => {
    it('should return - for undefined', () => {
      expect(formatResponseTime(undefined)).toBe('-')
    })

    it('should return - for zero', () => {
      expect(formatResponseTime(0)).toBe('-')
    })

    it('should return - for negative', () => {
      expect(formatResponseTime(-100)).toBe('-')
    })

    it('should return <1ms for sub-millisecond times', () => {
      expect(formatResponseTime(500_000)).toBe('<1ms') // 0.5ms
    })

    it('should format milliseconds', () => {
      expect(formatResponseTime(45_000_000)).toBe('45ms') // 45ms
    })

    it('should format seconds', () => {
      expect(formatResponseTime(2_500_000_000)).toBe('2.5s') // 2.5s
    })
  })

  describe('formatUptime', () => {
    it('should return - for undefined', () => {
      expect(formatUptime(undefined)).toBe('-')
    })

    it('should return - for zero', () => {
      expect(formatUptime(0)).toBe('-')
    })

    it('should return - for negative', () => {
      expect(formatUptime(-100)).toBe('-')
    })

    it('should format seconds', () => {
      expect(formatUptime(45_000_000_000)).toBe('45s')
    })

    it('should format minutes', () => {
      expect(formatUptime(5 * 60 * 1_000_000_000)).toBe('5m')
    })

    it('should format hours and minutes', () => {
      expect(formatUptime(2 * 60 * 60 * 1_000_000_000 + 30 * 60 * 1_000_000_000)).toBe('2h 30m')
    })

    it('should format days and hours', () => {
      const twoDaysThreeHours = (2 * 24 + 3) * 60 * 60 * 1_000_000_000
      expect(formatUptime(twoDaysThreeHours)).toBe('2d 3h')
    })
  })

  describe('getCheckTypeDisplay', () => {
    it('should return HTTP for http', () => {
      expect(getCheckTypeDisplay('http')).toBe('HTTP')
    })

    it('should return TCP for tcp', () => {
      expect(getCheckTypeDisplay('tcp')).toBe('TCP')
    })

    it('should return Process for process', () => {
      expect(getCheckTypeDisplay('process')).toBe('Process')
    })

    it('should return Unknown for undefined', () => {
      expect(getCheckTypeDisplay(undefined)).toBe('Unknown')
    })

    it('should return Unknown for unrecognized type', () => {
      expect(getCheckTypeDisplay('something-else')).toBe('Unknown')
    })
  })

  describe('mergeHealthIntoService', () => {
    const baseService: Service = {
      name: 'api',
      local: {
        status: 'running',
        health: 'unknown',
        port: 5000,
        startTime: '2024-01-01T00:00:00Z',
      },
    }

    it('should return service unchanged when healthResult is undefined', () => {
      const result = mergeHealthIntoService(baseService, undefined)
      expect(result).toEqual(baseService)
    })

    it('should merge healthy status', () => {
      const healthResult: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 45_000_000,
        timestamp: '2024-01-01T00:00:00Z',
      }

      const result = mergeHealthIntoService(baseService, healthResult)
      expect(result.local?.health).toBe('healthy')
    })

    it('should merge degraded status', () => {
      const healthResult: HealthCheckResult = {
        serviceName: 'api',
        status: 'degraded',
        checkType: 'http',
        responseTime: 2_000_000_000,
        timestamp: '2024-01-01T00:00:00Z',
      }

      const result = mergeHealthIntoService(baseService, healthResult)
      expect(result.local?.health).toBe('degraded')
    })

    it('should merge unhealthy status', () => {
      const healthResult: HealthCheckResult = {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'tcp',
        responseTime: 0,
        timestamp: '2024-01-01T00:00:00Z',
        error: 'connection refused',
      }

      const result = mergeHealthIntoService(baseService, healthResult)
      expect(result.local?.health).toBe('unhealthy')
      expect(result.local?.healthDetails?.lastError).toBe('connection refused')
    })

    it('should merge unknown status for unrecognized health result', () => {
      const healthResult: HealthCheckResult = {
        serviceName: 'api',
        status: 'pending' as 'healthy', // simulating unknown status
        checkType: 'process',
        responseTime: 0,
        timestamp: '2024-01-01T00:00:00Z',
      }

      const result = mergeHealthIntoService(baseService, healthResult)
      expect(result.local?.health).toBe('unknown')
    })

    it('should convert response time from nanoseconds to milliseconds', () => {
      const healthResult: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 45_000_000, // 45ms in nanos
        timestamp: '2024-01-01T00:00:00Z',
      }

      const result = mergeHealthIntoService(baseService, healthResult)
      expect(result.local?.healthDetails?.responseTime).toBe(45) // 45ms
    })

    it('should convert uptime from nanoseconds to seconds', () => {
      const healthResult: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 45_000_000,
        uptime: 3_600_000_000_000, // 1 hour in nanos
        timestamp: '2024-01-01T00:00:00Z',
      }

      const result = mergeHealthIntoService(baseService, healthResult)
      expect(result.local?.healthDetails?.uptime).toBe(3600) // 1 hour in seconds
    })

    it('should include health details', () => {
      const healthResult: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 45_000_000,
        endpoint: 'http://localhost:5000/health',
        statusCode: 200,
        timestamp: '2024-01-01T00:00:00Z',
        details: { version: '1.0.0' },
      }

      const result = mergeHealthIntoService(baseService, healthResult)
      expect(result.local?.healthDetails?.checkType).toBe('http')
      expect(result.local?.healthDetails?.endpoint).toBe('http://localhost:5000/health')
      expect(result.local?.healthDetails?.statusCode).toBe(200)
      expect(result.local?.healthDetails?.details).toEqual({ version: '1.0.0' })
    })

    it('should handle service without local property', () => {
      const serviceWithoutLocal: Service = {
        name: 'api',
      }

      const healthResult: HealthCheckResult = {
        serviceName: 'api',
        status: 'healthy',
        checkType: 'http',
        responseTime: 45_000_000,
        timestamp: '2024-01-01T00:00:00Z',
      }

      const result = mergeHealthIntoService(serviceWithoutLocal, healthResult)
      expect(result.local?.status).toBe('not-running')
      expect(result.local?.health).toBe('healthy')
    })
  })

  describe('getLogPaneVisualStatus', () => {
    it('should return error when serviceHealth is unhealthy', () => {
      expect(getLogPaneVisualStatus('unhealthy', 'info')).toBe('error')
    })

    it('should return warning when serviceHealth is degraded', () => {
      expect(getLogPaneVisualStatus('degraded', 'info')).toBe('warning')
    })

    it('should fall back to paneStatus for unknown health (including legacy "starting")', () => {
      // 'starting' from backend should be normalized to 'unknown' before calling this function
      // If 'starting' is passed directly, it's treated as invalid/unknown and falls back
      expect(getLogPaneVisualStatus('starting' as never, 'info')).toBe('info')
      expect(getLogPaneVisualStatus('starting' as never, 'warning')).toBe('warning')
    })

    it('should return healthy when serviceHealth is healthy', () => {
      expect(getLogPaneVisualStatus('healthy', 'error')).toBe('healthy')
    })

    it('should fall back to paneStatus when serviceHealth is unknown', () => {
      expect(getLogPaneVisualStatus('unknown', 'error')).toBe('error')
      expect(getLogPaneVisualStatus('unknown', 'warning')).toBe('warning')
      expect(getLogPaneVisualStatus('unknown', 'info')).toBe('info')
    })

    it('should fall back to paneStatus when serviceHealth is undefined', () => {
      expect(getLogPaneVisualStatus(undefined, 'error')).toBe('error')
      expect(getLogPaneVisualStatus(undefined, 'warning')).toBe('warning')
      expect(getLogPaneVisualStatus(undefined, 'info')).toBe('info')
    })

    it('should prioritize health status over log-based status', () => {
      // Even if logs show errors, healthy service should show as healthy
      expect(getLogPaneVisualStatus('healthy', 'error')).toBe('healthy')
      // But unhealthy service shows error even if logs show info
      expect(getLogPaneVisualStatus('unhealthy', 'info')).toBe('error')
    })

    it('should return stopped when processStatus is stopped', () => {
      // Stopped process status takes priority over health status
      expect(getLogPaneVisualStatus('healthy', 'info', 'stopped')).toBe('stopped')
      expect(getLogPaneVisualStatus('unhealthy', 'error', 'stopped')).toBe('stopped')
      expect(getLogPaneVisualStatus(undefined, 'info', 'stopped')).toBe('stopped')
    })

    it('should not affect status when processStatus is running', () => {
      // Running process status should not affect the result
      expect(getLogPaneVisualStatus('healthy', 'info', 'running')).toBe('healthy')
      expect(getLogPaneVisualStatus('unhealthy', 'error', 'running')).toBe('error')
    })

    it('should not affect status when processStatus is undefined', () => {
      // Undefined process status should fall back to health-based status
      expect(getLogPaneVisualStatus('healthy', 'info', undefined)).toBe('healthy')
      expect(getLogPaneVisualStatus('unhealthy', 'error', undefined)).toBe('error')
    })
  })

  describe('Process Service Status Display', () => {
    describe('getStatusDisplay for process services', () => {
      it('should return Watching for watching status', () => {
        const display = getStatusDisplay('watching', 'unknown')
        expect(display.text).toBe('Watching')
        expect(display.badgeVariant).toBe('success')
      })

      it('should return Building for building status', () => {
        const display = getStatusDisplay('building', 'unknown')
        expect(display.text).toBe('Building')
        expect(display.badgeVariant).toBe('warning')
      })

      it('should return Built for built status', () => {
        const display = getStatusDisplay('built', 'unknown')
        expect(display.text).toBe('Built')
        expect(display.badgeVariant).toBe('success')
      })

      it('should return Failed for failed status', () => {
        const display = getStatusDisplay('failed', 'unknown')
        expect(display.text).toBe('Failed')
        expect(display.badgeVariant).toBe('destructive')
      })

      it('should return Completed for completed status', () => {
        const display = getStatusDisplay('completed', 'unknown')
        expect(display.text).toBe('Completed')
        expect(display.badgeVariant).toBe('success')
      })
    })

    describe('getStatusIndicator for process services', () => {
      it('should return correct indicator for watching', () => {
        const indicator = getStatusIndicator('watching')
        expect(indicator.icon).toBe('👁')
        expect(indicator.color).toBe('text-green-500')
      })

      it('should return correct indicator for building', () => {
        const indicator = getStatusIndicator('building')
        expect(indicator.icon).toBe('🔨')
        expect(indicator.color).toBe('text-yellow-500')
        expect(indicator.animate).toBe('animate-pulse')
      })

      it('should return correct indicator for built', () => {
        const indicator = getStatusIndicator('built')
        expect(indicator.icon).toBe('✓')
        expect(indicator.color).toBe('text-green-500')
      })

      it('should return correct indicator for failed', () => {
        const indicator = getStatusIndicator('failed')
        expect(indicator.icon).toBe('✗')
        expect(indicator.color).toBe('text-red-500')
      })

      it('should return correct indicator for completed', () => {
        const indicator = getStatusIndicator('completed')
        expect(indicator.icon).toBe('✓')
        expect(indicator.color).toBe('text-green-500')
      })
    })

    describe('getStatusBadgeConfig for process services', () => {
      it('should return correct config for watching', () => {
        const config = getStatusBadgeConfig('watching')
        expect(config.label).toBe('Watching')
        expect(config.color).toContain('green')
      })

      it('should return correct config for building', () => {
        const config = getStatusBadgeConfig('building')
        expect(config.label).toBe('Building')
        expect(config.color).toContain('yellow')
      })

      it('should return correct config for built', () => {
        const config = getStatusBadgeConfig('built')
        expect(config.label).toBe('Built')
        expect(config.color).toContain('green')
      })

      it('should return correct config for failed', () => {
        const config = getStatusBadgeConfig('failed')
        expect(config.label).toBe('Failed')
        expect(config.color).toContain('red')
      })

      it('should return correct config for completed', () => {
        const config = getStatusBadgeConfig('completed')
        expect(config.label).toBe('Completed')
        expect(config.color).toContain('green')
      })
    })
  })

  describe('Service Type and Mode Utilities', () => {
    describe('getServiceTypeBadgeConfig', () => {
      it('should return HTTP config for http type', () => {
        const config = getServiceTypeBadgeConfig('http')
        expect(config.label).toBe('HTTP')
        expect(config.color).toContain('blue')
      })

      it('should return TCP config for tcp type', () => {
        const config = getServiceTypeBadgeConfig('tcp')
        expect(config.label).toBe('TCP')
        expect(config.color).toContain('purple')
      })

      it('should return Process config for process type', () => {
        const config = getServiceTypeBadgeConfig('process')
        expect(config.label).toBe('Process')
        expect(config.color).toContain('cyan')
      })

      it('should default to HTTP for undefined', () => {
        const config = getServiceTypeBadgeConfig(undefined)
        expect(config.label).toBe('HTTP')
      })
    })

    describe('getServiceModeBadgeConfig', () => {
      it('should return Watch config for watch mode', () => {
        const config = getServiceModeBadgeConfig('watch')
        expect(config.label).toBe('Watch')
        expect(config.description).toContain('watching')
      })

      it('should return Build config for build mode', () => {
        const config = getServiceModeBadgeConfig('build')
        expect(config.label).toBe('Build')
        expect(config.description.toLowerCase()).toContain('one-time')
      })

      it('should return Daemon config for daemon mode', () => {
        const config = getServiceModeBadgeConfig('daemon')
        expect(config.label).toBe('Daemon')
        expect(config.description.toLowerCase()).toContain('background')
      })

      it('should return Task config for task mode', () => {
        const config = getServiceModeBadgeConfig('task')
        expect(config.label).toBe('Task')
        expect(config.description.toLowerCase()).toContain('on-demand')
      })

      it('should default to Watch for undefined', () => {
        const config = getServiceModeBadgeConfig(undefined)
        expect(config.label).toBe('Watch')
      })
    })

    describe('getServiceTypeLabel', () => {
      it('should return HTTP Service for http', () => {
        expect(getServiceTypeLabel('http')).toBe('HTTP Service')
      })

      it('should return TCP Service for tcp', () => {
        expect(getServiceTypeLabel('tcp')).toBe('TCP Service')
      })

      it('should return Process Service for process', () => {
        expect(getServiceTypeLabel('process')).toBe('Process Service')
      })
    })

    describe('getServiceModeLabel', () => {
      it('should return Watch Mode for watch', () => {
        expect(getServiceModeLabel('watch')).toBe('Watch Mode')
      })

      it('should return Build Mode for build', () => {
        expect(getServiceModeLabel('build')).toBe('Build Mode')
      })

      it('should return Daemon Mode for daemon', () => {
        expect(getServiceModeLabel('daemon')).toBe('Daemon Mode')
      })

      it('should return Task Mode for task', () => {
        expect(getServiceModeLabel('task')).toBe('Task Mode')
      })
    })

    describe('isProcessService', () => {
      it('should return true for process type', () => {
        expect(isProcessService('process')).toBe(true)
      })

      it('should return false for http type', () => {
        expect(isProcessService('http')).toBe(false)
      })

      it('should return false for tcp type', () => {
        expect(isProcessService('tcp')).toBe(false)
      })

      it('should return false for undefined', () => {
        expect(isProcessService(undefined)).toBe(false)
      })
    })

    describe('isContinuousMode', () => {
      it('should return true for watch mode', () => {
        expect(isContinuousMode('watch')).toBe(true)
      })

      it('should return true for daemon mode', () => {
        expect(isContinuousMode('daemon')).toBe(true)
      })

      it('should return false for build mode', () => {
        expect(isContinuousMode('build')).toBe(false)
      })

      it('should return false for task mode', () => {
        expect(isContinuousMode('task')).toBe(false)
      })
    })

    describe('isOneTimeMode', () => {
      it('should return true for build mode', () => {
        expect(isOneTimeMode('build')).toBe(true)
      })

      it('should return true for task mode', () => {
        expect(isOneTimeMode('task')).toBe(true)
      })

      it('should return false for watch mode', () => {
        expect(isOneTimeMode('watch')).toBe(false)
      })

      it('should return false for daemon mode', () => {
        expect(isOneTimeMode('daemon')).toBe(false)
      })
    })
  })
})
