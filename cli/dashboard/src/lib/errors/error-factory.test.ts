/**
 * Error Factory Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  createError,
  createSchemaError,
  createFileSystemError,
  createValidationError,
  createNetworkError,
  createParseError,
  createUserInputError,
  formatErrorMessage,
  getUserFriendlyMessage,
} from './error-factory'

describe('error-factory', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('createError', () => {
    it('should create basic error', () => {
      const error = createError('schema', 'Test error')

      expect(error).toMatchObject({
        type: 'schema',
        severity: 'error',
        message: 'Test error',
        retryable: false,
      })
      expect(error.timestamp).toBeInstanceOf(Date)
    })

    it('should create error with all options', () => {
      const originalError = new Error('Original')
      const error = createError('network', 'Test error', {
        severity: 'warning',
        technicalDetails: 'Technical info',
        path: 'services.api',
        retryable: true,
        originalError,
      })

      expect(error).toMatchObject({
        type: 'network',
        severity: 'warning',
        message: 'Test error',
        technicalDetails: 'Technical info',
        path: 'services.api',
        retryable: true,
        originalError,
      })
    })

    it('should default severity to error', () => {
      const error = createError('validation', 'Test')
      expect(error.severity).toBe('error')
    })

    it('should default retryable to false', () => {
      const error = createError('schema', 'Test')
      expect(error.retryable).toBe(false)
    })
  })

  describe('createSchemaError', () => {
    it('should create schema error', () => {
      const error = createSchemaError('Invalid schema', 'services.api', 'Type mismatch')

      expect(error).toMatchObject({
        type: 'schema',
        severity: 'error',
        message: 'Invalid schema',
        path: 'services.api',
        technicalDetails: 'Type mismatch',
        retryable: false,
      })
    })
  })

  describe('createFileSystemError', () => {
    it('should create file system error', () => {
      const originalError = new Error('ENOENT')
      const error = createFileSystemError('File not found', originalError, false)

      expect(error).toMatchObject({
        type: 'file-system',
        severity: 'error',
        message: 'File not found',
        technicalDetails: 'ENOENT',
        retryable: false,
        originalError,
      })
    })

    it('should create retryable file system error', () => {
      const error = createFileSystemError('Temporary error', undefined, true)
      expect(error.retryable).toBe(true)
    })
  })

  describe('createValidationError', () => {
    it('should create validation error', () => {
      const error = createValidationError('Invalid value', 'services.api.port')

      expect(error).toMatchObject({
        type: 'validation',
        severity: 'error',
        message: 'Invalid value',
        path: 'services.api.port',
        retryable: false,
      })
    })

    it('should create validation warning', () => {
      const error = createValidationError('Recommended', 'services.api', 'warning')
      expect(error.severity).toBe('warning')
    })
  })

  describe('createNetworkError', () => {
    it('should create network error', () => {
      const originalError = new Error('Network timeout')
      const error = createNetworkError('Request failed', originalError)

      expect(error).toMatchObject({
        type: 'network',
        severity: 'error',
        message: 'Request failed',
        technicalDetails: 'Network timeout',
        retryable: true,
        originalError,
      })
    })

    it('should create non-retryable network error', () => {
      const error = createNetworkError('Forbidden', undefined, false)
      expect(error.retryable).toBe(false)
    })
  })

  describe('createParseError', () => {
    it('should create parse error', () => {
      const error = createParseError('Invalid YAML', 'Unexpected token', 'line 5')

      expect(error).toMatchObject({
        type: 'parse',
        severity: 'error',
        message: 'Invalid YAML',
        technicalDetails: 'Unexpected token',
        path: 'line 5',
        retryable: false,
      })
    })
  })

  describe('createUserInputError', () => {
    it('should create user input error', () => {
      const error = createUserInputError('Invalid port', 'services.api.port')

      expect(error).toMatchObject({
        type: 'user-input',
        severity: 'error',
        message: 'Invalid port',
        path: 'services.api.port',
        retryable: false,
      })
    })

    it('should create user input info', () => {
      const error = createUserInputError('Helpful tip', undefined, 'info')
      expect(error.severity).toBe('info')
    })
  })

  describe('formatErrorMessage', () => {
    it('should format error with path', () => {
      const error = createError('schema', 'Test error', { path: 'services.api' })
      expect(formatErrorMessage(error)).toBe('services.api: Test error')
    })

    it('should format error without path', () => {
      const error = createError('schema', 'Test error')
      expect(formatErrorMessage(error)).toBe('Test error')
    })
  })

  describe('getUserFriendlyMessage', () => {
    it('should handle string errors', () => {
      expect(getUserFriendlyMessage('Simple error')).toBe('Simple error')
    })

    it('should handle ENOENT error', () => {
      const error = new Error('ENOENT: file not found')
      expect(getUserFriendlyMessage(error)).toBe('File not found. Please check the file path.')
    })

    it('should handle EACCES error', () => {
      const error = new Error('EACCES: permission denied')
      expect(getUserFriendlyMessage(error)).toBe(
        'Permission denied. Please check file permissions.'
      )
    })

    it('should handle EPERM error', () => {
      const error = new Error('EPERM: operation not permitted')
      expect(getUserFriendlyMessage(error)).toBe(
        'Permission denied. Please check file permissions.'
      )
    })

    it('should handle ENOSPC error', () => {
      const error = new Error('ENOSPC: no space left')
      expect(getUserFriendlyMessage(error)).toBe('Disk full. Please free up disk space.')
    })

    it('should handle network error', () => {
      const error = new Error('Network request failed')
      expect(getUserFriendlyMessage(error)).toBe(
        'Network error. Please check your connection.'
      )
    })

    it('should handle fetch error', () => {
      const error = new Error('fetch failed')
      expect(getUserFriendlyMessage(error)).toBe(
        'Network error. Please check your connection.'
      )
    })

    it('should handle timeout error', () => {
      const error = new Error('Request timeout')
      expect(getUserFriendlyMessage(error)).toBe('Request timeout. Please try again.')
    })

    it('should handle generic Error', () => {
      const error = new Error('Some error')
      expect(getUserFriendlyMessage(error)).toBe('Some error')
    })

    it('should handle object with message', () => {
      expect(getUserFriendlyMessage({ message: 'Object error' })).toBe('Object error')
    })

    it('should handle unknown error', () => {
      expect(getUserFriendlyMessage(null)).toBe('An unknown error occurred')
      expect(getUserFriendlyMessage(undefined)).toBe('An unknown error occurred')
      expect(getUserFriendlyMessage(123)).toBe('An unknown error occurred')
    })
  })
})
