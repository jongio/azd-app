/**
 * Health Check Types Tests
 */

import { describe, it, expect } from 'vitest'
import {
  formDataToHealthCheck,
  healthCheckToFormData,
  getDefaultHealthCheck,
  validateDuration,
  validateUrl,
  validatePort,
  type HealthCheckFormData,
  type HealthCheckConfig,
  type ServiceInfo,
} from './healthcheck-types'

describe('healthcheck-types', () => {
  describe('formDataToHealthCheck', () => {
    it('should convert http form data to health check config', () => {
      const formData: HealthCheckFormData = {
        type: 'http',
        url: 'http://localhost:8080/health',
        interval: '30s',
        timeout: '5s',
        retries: 3,
        startPeriod: '0s',
        startInterval: '5s',
      }

      const config = formDataToHealthCheck(formData)

      expect(config).toEqual({
        type: 'http',
        test: 'http://localhost:8080/health',
        path: '/health',
        interval: '30s',
        timeout: '5s',
        retries: 3,
        start_period: '0s',
        start_interval: '5s',
      })
    })

    it('should convert tcp form data to health check config', () => {
      const formData: HealthCheckFormData = {
        type: 'tcp',
        port: 5432,
        interval: '10s',
        timeout: '5s',
        retries: 3,
      }

      const config = formDataToHealthCheck(formData)

      expect(config).toEqual({
        type: 'tcp',
        test: 'tcp://localhost:5432',
        interval: '10s',
        timeout: '5s',
        retries: 3,
      })
    })

    it('should convert process form data to health check config', () => {
      const formData: HealthCheckFormData = {
        type: 'process',
        command: './health-check.sh',
        interval: '30s',
        timeout: '10s',
        retries: 2,
      }

      const config = formDataToHealthCheck(formData)

      expect(config).toEqual({
        type: 'process',
        test: './health-check.sh',
        interval: '30s',
        timeout: '10s',
        retries: 2,
      })
    })

    it('should convert output form data to health check config', () => {
      const formData: HealthCheckFormData = {
        type: 'output',
        pattern: 'Server started',
        interval: '15s',
        timeout: '5s',
        retries: 5,
      }

      const config = formDataToHealthCheck(formData)

      expect(config).toEqual({
        type: 'output',
        pattern: 'Server started',
        interval: '15s',
        timeout: '5s',
        retries: 5,
      })
    })

    it('should convert none type to disabled config', () => {
      const formData: HealthCheckFormData = {
        type: 'none',
      }

      const config = formDataToHealthCheck(formData)

      expect(config).toEqual({
        disable: true,
      })
    })

    it('should omit optional fields when not provided', () => {
      const formData: HealthCheckFormData = {
        type: 'http',
        url: 'http://localhost:3000/health',
      }

      const config = formDataToHealthCheck(formData)

      expect(config).toEqual({
        type: 'http',
        test: 'http://localhost:3000/health',
        path: '/health',
      })
    })
  })

  describe('healthCheckToFormData', () => {
    it('should convert http config to form data', () => {
      const config: HealthCheckConfig = {
        type: 'http',
        test: 'http://localhost:8080/healthz',
        interval: '30s',
        timeout: '5s',
        retries: 3,
      }

      const formData = healthCheckToFormData(config)

      expect(formData).toEqual({
        type: 'http',
        url: 'http://localhost:8080/healthz',
        interval: '30s',
        timeout: '5s',
        retries: 3,
        startPeriod: '0s',
        startInterval: '5s',
      })
    })

    it('should convert tcp config to form data', () => {
      const config: HealthCheckConfig = {
        type: 'tcp',
        test: 'tcp://localhost:5432',
        interval: '10s',
        timeout: '3s',
        retries: 2,
      }

      const formData = healthCheckToFormData(config)

      expect(formData).toEqual({
        type: 'tcp',
        port: 5432,
        interval: '10s',
        timeout: '3s',
        retries: 2,
        startPeriod: '0s',
        startInterval: '5s',
      })
    })

    it('should convert process config to form data', () => {
      const config: HealthCheckConfig = {
        type: 'process',
        test: 'node health.js',
      }

      const formData = healthCheckToFormData(config)

      expect(formData).toEqual({
        type: 'process',
        command: 'node health.js',
        interval: '30s',
        timeout: '5s',
        retries: 3,
        startPeriod: '0s',
        startInterval: '5s',
      })
    })

    it('should convert output config to form data', () => {
      const config: HealthCheckConfig = {
        type: 'output',
        pattern: 'Listening on port',
      }

      const formData = healthCheckToFormData(config)

      expect(formData).toEqual({
        type: 'output',
        pattern: 'Listening on port',
        interval: '30s',
        timeout: '5s',
        retries: 3,
        startPeriod: '0s',
        startInterval: '5s',
      })
    })

    it('should convert disabled config to none type', () => {
      const config: HealthCheckConfig = {
        disable: true,
      }

      const formData = healthCheckToFormData(config)

      expect(formData.type).toBe('none')
    })

    it('should return none type for undefined config', () => {
      const formData = healthCheckToFormData(undefined)

      expect(formData.type).toBe('none')
    })

    it('should use default values when config is missing fields', () => {
      const config: HealthCheckConfig = {
        type: 'http',
        test: 'http://localhost:3000/health',
      }

      const formData = healthCheckToFormData(config)

      expect(formData.interval).toBe('30s')
      expect(formData.timeout).toBe('5s')
      expect(formData.retries).toBe(3)
      expect(formData.startPeriod).toBe('0s')
      expect(formData.startInterval).toBe('5s')
    })
  })

  describe('getDefaultHealthCheck', () => {
    it('should return http health check for node service', () => {
      const service: ServiceInfo = {
        language: 'node',
        ports: ['3000:3000'],
      }

      const defaults = getDefaultHealthCheck(service)

      expect(defaults.type).toBe('http')
      expect(defaults.url).toBe('http://localhost:3000/health')
    })

    it('should return http health check with /healthz for dotnet service', () => {
      const service: ServiceInfo = {
        language: 'dotnet',
        ports: ['5000:5000'],
      }

      const defaults = getDefaultHealthCheck(service)

      expect(defaults.type).toBe('http')
      expect(defaults.url).toBe('http://localhost:5000/healthz')
    })

    it('should return http health check with /actuator/health for java service', () => {
      const service: ServiceInfo = {
        language: 'java',
        ports: ['8080:8080'],
      }

      const defaults = getDefaultHealthCheck(service)

      expect(defaults.type).toBe('http')
      expect(defaults.url).toBe('http://localhost:8080/actuator/health')
    })

    it('should return tcp health check for postgres', () => {
      const service: ServiceInfo = {
        image: 'postgres:16-alpine',
      }

      const defaults = getDefaultHealthCheck(service)

      expect(defaults.type).toBe('tcp')
      expect(defaults.port).toBe(5432)
    })

    it('should return tcp health check for redis', () => {
      const service: ServiceInfo = {
        image: 'redis:7-alpine',
      }

      const defaults = getDefaultHealthCheck(service)

      expect(defaults.type).toBe('tcp')
      expect(defaults.port).toBe(6379)
    })

    it('should return tcp health check for mongodb', () => {
      const service: ServiceInfo = {
        image: 'mongo:7',
      }

      const defaults = getDefaultHealthCheck(service)

      expect(defaults.type).toBe('tcp')
      expect(defaults.port).toBe(27017)
    })

    it('should use default port 8080 when no ports specified', () => {
      const service: ServiceInfo = {
        language: 'python',
      }

      const defaults = getDefaultHealthCheck(service)

      expect(defaults.url).toBe('http://localhost:8080/health')
    })

    it('should extract port from ports array', () => {
      const service: ServiceInfo = {
        language: 'node',
        ports: ['4000:4000', '4001:4001'],
      }

      const defaults = getDefaultHealthCheck(service)

      expect(defaults.url).toBe('http://localhost:4000/health')
    })
  })

  describe('validateDuration', () => {
    it('should validate correct duration formats', () => {
      expect(validateDuration('30s')).toBe(true)
      expect(validateDuration('1m')).toBe(true)
      expect(validateDuration('2h')).toBe(true)
      expect(validateDuration('60s')).toBe(true)
      expect(validateDuration('10m')).toBe(true)
    })

    it('should reject invalid duration formats', () => {
      expect(validateDuration('30')).toBe(false)
      expect(validateDuration('30seconds')).toBe(false)
      expect(validateDuration('s30')).toBe(false)
      expect(validateDuration('30 s')).toBe(false)
      expect(validateDuration('30ms')).toBe(false)
      expect(validateDuration('abc')).toBe(false)
      expect(validateDuration('')).toBe(false)
    })
  })

  describe('validateUrl', () => {
    it('should validate correct HTTP URLs', () => {
      expect(validateUrl('http://localhost:8080/health')).toBe(true)
      expect(validateUrl('https://example.com/healthz')).toBe(true)
      expect(validateUrl('http://127.0.0.1:3000/api/health')).toBe(true)
      expect(validateUrl('https://api.example.com/status')).toBe(true)
    })

    it('should reject invalid URLs', () => {
      expect(validateUrl('localhost:8080/health')).toBe(false)
      expect(validateUrl('tcp://localhost:8080')).toBe(false)
      expect(validateUrl('ftp://example.com')).toBe(false)
      expect(validateUrl('/health')).toBe(false)
      expect(validateUrl('not-a-url')).toBe(false)
      expect(validateUrl('')).toBe(false)
    })
  })

  describe('validatePort', () => {
    it('should validate correct port numbers', () => {
      expect(validatePort(1)).toBe(true)
      expect(validatePort(80)).toBe(true)
      expect(validatePort(8080)).toBe(true)
      expect(validatePort(65535)).toBe(true)
    })

    it('should reject invalid port numbers', () => {
      expect(validatePort(0)).toBe(false)
      expect(validatePort(-1)).toBe(false)
      expect(validatePort(65536)).toBe(false)
      expect(validatePort(100000)).toBe(false)
      expect(validatePort(3.14)).toBe(false)
      expect(validatePort(NaN)).toBe(false)
    })
  })

  describe('round-trip conversion', () => {
    it('should preserve data through form -> config -> form conversion', () => {
      const originalFormData: HealthCheckFormData = {
        type: 'http',
        url: 'http://localhost:8080/health',
        interval: '30s',
        timeout: '5s',
        retries: 3,
        startPeriod: '10s',
        startInterval: '5s',
      }

      const config = formDataToHealthCheck(originalFormData)
      const roundTripFormData = healthCheckToFormData(config!)

      expect(roundTripFormData).toEqual(originalFormData)
    })

    it('should preserve tcp configuration', () => {
      const originalFormData: HealthCheckFormData = {
        type: 'tcp',
        port: 5432,
        interval: '10s',
        timeout: '3s',
        retries: 2,
        startPeriod: '5s',
        startInterval: '2s',
      }

      const config = formDataToHealthCheck(originalFormData)
      const roundTripFormData = healthCheckToFormData(config!)

      expect(roundTripFormData).toEqual(originalFormData)
    })
  })
})
