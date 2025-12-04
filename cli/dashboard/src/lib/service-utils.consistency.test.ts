/**
 * Integration tests to verify all status display functions return consistent
 * values for the same service status across the entire UI.
 */
import { describe, it, expect } from 'vitest'
import {
  getStatusIndicator,
  getStatusDisplay,
  getStatusBadgeConfig,
  getHealthBadgeConfig,
  calculateStatusCounts,
} from './service-utils'
import type { Service, HealthSummary } from '@/types'

describe('Status Display Consistency', () => {
  describe('getStatusIndicator should be consistent with getStatusBadgeConfig', () => {
    const statuses = ['running', 'ready', 'starting', 'stopping', 'stopped', 'error', 'not-running']

    it.each(statuses)('should have matching icons for status: %s', (status) => {
      const indicator = getStatusIndicator(status)
      const badge = getStatusBadgeConfig(status)
      
      expect(indicator.icon).toBe(badge.icon)
    })

    it('should return not-running indicator for undefined status', () => {
      const indicator = getStatusIndicator(undefined)
      const badge = getStatusBadgeConfig(undefined)
      
      expect(indicator.icon).toBe('○')
      expect(badge.icon).toBe('○')
      expect(badge.label).toBe('Not Running')
    })
  })

  describe('getStatusDisplay should be consistent with getStatusIndicator', () => {
    it('running + healthy should show green/running', () => {
      const display = getStatusDisplay('running', 'healthy')
      const indicator = getStatusIndicator('running')
      
      expect(display.text).toBe('Running')
      expect(indicator.color).toContain('green')
    })

    it('stopped should show gray/stopped', () => {
      const display = getStatusDisplay('stopped', 'unknown')
      const indicator = getStatusIndicator('stopped')
      
      expect(display.text).toBe('Stopped')
      expect(indicator.icon).toBe('◉')
      expect(indicator.color).toContain('gray')
    })

    it('not-running should show gray/not-running', () => {
      const display = getStatusDisplay('not-running', 'unknown')
      const indicator = getStatusIndicator('not-running')
      
      expect(display.text).toBe('Not Running')
      expect(indicator.icon).toBe('○')
      expect(indicator.color).toContain('gray')
    })
  })

  describe('getHealthBadgeConfig should cover all health statuses', () => {
    // Note: 'starting' is NOT a health status - it's a lifecycle state
    // Backend may send 'starting' but normalizeHealthStatus converts it to 'unknown'
    const healthStatuses = ['healthy', 'degraded', 'unhealthy', 'unknown']

    it.each(healthStatuses)('should return valid config for health: %s', (health) => {
      const badge = getHealthBadgeConfig(health)
      
      expect(badge.color).toBeTruthy()
      expect(badge.label).toBeTruthy()
    })

    it('should return unknown for undefined health', () => {
      const badge = getHealthBadgeConfig(undefined)
      expect(badge.label).toBe('Unknown')
    })
  })

  describe('calculateStatusCounts', () => {
    it('should count stopped services from process status', () => {
      const services: Service[] = [
        { name: 'api', local: { status: 'stopped', health: 'unknown' } },
        { name: 'web', local: { status: 'running', health: 'healthy' } },
      ]
      
      const counts = calculateStatusCounts(services)
      
      expect(counts.stopped).toBe(1)
      // Without healthSummary, running services count from service status
      expect(counts.running).toBe(1)
    })

    it('should adjust unhealthy count when stopped services are present', () => {
      const services: Service[] = [
        { name: 'api', local: { status: 'stopped', health: 'unknown' } },
        { name: 'web', local: { status: 'stopped', health: 'unknown' } },
      ]
      // Health summary shows unhealthy because stopped services fail health checks
      const healthSummary: HealthSummary = {
        total: 2,
        healthy: 0,
        degraded: 0,
        unhealthy: 2,
        starting: 0,
        stopped: 0,
        unknown: 0,
        overall: 'unhealthy'
      }
      
      const counts = calculateStatusCounts(services, healthSummary)
      
      // Unhealthy should be adjusted to 0 because both services are stopped
      expect(counts.error).toBe(0)
      expect(counts.stopped).toBe(2)
    })

    it('should prioritize healthSummary over hasActiveErrors', () => {
      const services: Service[] = []
      const healthSummary: HealthSummary = {
        total: 4,
        healthy: 4,
        degraded: 0,
        unhealthy: 0,
        starting: 0,
        stopped: 0,
        unknown: 0,
        overall: 'healthy'
      }
      
      const counts = calculateStatusCounts(services, healthSummary, true)
      
      // Should show healthy services, not affected by hasActiveErrors
      expect(counts.running).toBe(4)
      expect(counts.error).toBe(0)
      expect(counts.warn).toBe(0)
    })

    it('should use hasActiveErrors when no healthSummary', () => {
      const services: Service[] = [
        { name: 'api', local: { status: 'running', health: 'healthy' } },
      ]
      
      const counts = calculateStatusCounts(services, null, true)
      
      // Without healthSummary, hasActiveErrors should move running to warn
      expect(counts.running).toBe(0)
      expect(counts.warn).toBe(1)
    })
  })

  describe('Icon consistency across status types', () => {
    it('should use ● for running/ready states', () => {
      expect(getStatusIndicator('running').icon).toBe('●')
      expect(getStatusIndicator('ready').icon).toBe('●')
      expect(getStatusBadgeConfig('running').icon).toBe('●')
      expect(getStatusBadgeConfig('ready').icon).toBe('●')
    })

    it('should use ◐/◑ for starting/stopping states', () => {
      expect(getStatusIndicator('starting').icon).toBe('◐')
      expect(getStatusIndicator('stopping').icon).toBe('◑')
    })

    it('should use ◉ for stopped state', () => {
      expect(getStatusIndicator('stopped').icon).toBe('◉')
      expect(getStatusBadgeConfig('stopped').icon).toBe('◉')
    })

    it('should use ○ for not-running state', () => {
      expect(getStatusIndicator('not-running').icon).toBe('○')
      expect(getStatusBadgeConfig('not-running').icon).toBe('○')
    })

    it('should use ⚠ for error state', () => {
      expect(getStatusIndicator('error').icon).toBe('⚠')
      expect(getStatusBadgeConfig('error').icon).toBe('⚠')
    })
  })

  describe('Color consistency', () => {
    it('should use green for running/healthy states', () => {
      expect(getStatusIndicator('running').color).toContain('green')
      expect(getStatusIndicator('ready').color).toContain('green')
      expect(getStatusBadgeConfig('running').color).toContain('green')
      expect(getHealthBadgeConfig('healthy').color).toContain('green')
    })

    it('should use yellow for starting/stopping/degraded states', () => {
      expect(getStatusIndicator('starting').color).toContain('yellow')
      expect(getStatusIndicator('stopping').color).toContain('yellow')
      expect(getStatusBadgeConfig('starting').color).toContain('yellow')
      expect(getHealthBadgeConfig('degraded').color).toContain('yellow')
    })

    it('should use gray for stopped/not-running states', () => {
      expect(getStatusIndicator('stopped').color).toContain('gray')
      expect(getStatusIndicator('not-running').color).toContain('gray')
    })

    it('should use red for error/unhealthy states', () => {
      expect(getStatusIndicator('error').color).toContain('red')
      expect(getStatusBadgeConfig('error').color).toContain('red')
      expect(getHealthBadgeConfig('unhealthy').color).toContain('red')
    })
  })
})
