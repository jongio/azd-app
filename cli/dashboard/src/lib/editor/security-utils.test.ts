/**
 * Tests for security utilities
 */

import { describe, it, expect } from 'vitest'
import {
  encodeHtmlEntities,
  sanitizeInput,
  isSecretKey,
  isSecretValue,
  maskSecret,
  detectSecrets,
  maskSecretsInObject,
  validateSafeYaml,
  validateFileSize,
  formatBytes,
  FILE_SIZE_LIMITS,
  getExportWarningMessage,
} from './security-utils'

describe('HTML Entity Encoding', () => {
  it('should encode HTML entities', () => {
    expect(encodeHtmlEntities('<script>alert("xss")</script>')).toBe(
      '&lt;script&gt;alert(&quot;xss&quot;)&lt;&#x2F;script&gt;'
    )
  })

  it('should encode all dangerous characters', () => {
    expect(encodeHtmlEntities('& < > " \' /')).toBe('&amp; &lt; &gt; &quot; &#x27; &#x2F;')
  })

  it('should handle empty string', () => {
    expect(encodeHtmlEntities('')).toBe('')
  })
})

describe('Input Sanitization', () => {
  it('should remove script tags', () => {
    const input = 'Hello <script>alert("xss")</script> World'
    expect(sanitizeInput(input)).not.toContain('<script>')
  })

  it('should remove javascript: protocol', () => {
    const input = '<a href="javascript:alert(1)">Click</a>'
    expect(sanitizeInput(input)).not.toContain('javascript:')
  })

  it('should remove event handlers', () => {
    const input = '<div onclick="alert(1)">Click</div>'
    const sanitized = sanitizeInput(input)
    expect(sanitized).not.toContain('onclick=')
  })

  it('should handle empty input', () => {
    expect(sanitizeInput('')).toBe('')
  })

  it('should preserve safe text', () => {
    const input = 'This is safe text with numbers 123'
    expect(sanitizeInput(input)).toContain('This is safe text')
  })
})

describe('Secret Detection', () => {
  it('should detect secret key patterns', () => {
    expect(isSecretKey('apiKey')).toBe(true)
    expect(isSecretKey('API_KEY')).toBe(true)
    expect(isSecretKey('password')).toBe(true)
    expect(isSecretKey('secret')).toBe(true)
    expect(isSecretKey('token')).toBe(true)
    expect(isSecretKey('connectionString')).toBe(true)
    expect(isSecretKey('client_secret')).toBe(true)
  })

  it('should not flag non-secret keys', () => {
    expect(isSecretKey('name')).toBe(false)
    expect(isSecretKey('port')).toBe(false)
    expect(isSecretKey('version')).toBe(false)
  })

  it('should detect secret value patterns', () => {
    // Base64 encoded string (40+ chars)
    expect(isSecretValue('dGhpcyBpcyBhIHZlcnkgbG9uZyBiYXNlNjQgZW5jb2RlZCBzdHJpbmc=')).toBe(true)
    
    // Azure storage key format (88 chars)
    expect(isSecretValue('ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/ABCDEFGHIJKLMNOPQRSTUVWXYZabcd')).toBe(true)
    
    // Connection string
    expect(isSecretValue('AccountName=myaccount;AccountKey=secretkey123456789012345678901234567890')).toBe(true)
  })

  it('should not flag short or simple values', () => {
    expect(isSecretValue('short')).toBe(false)
    expect(isSecretValue('localhost')).toBe(false)
    expect(isSecretValue('123')).toBe(false)
  })

  it('should mask secrets correctly', () => {
    const secret = 'this_is_a_very_long_secret_key_1234567890'
    const masked = maskSecret(secret)
    
    expect(masked).toContain('this')
    expect(masked).toContain('7890')
    expect(masked).toContain('****')
    expect(masked.length).toBeLessThan(secret.length)
  })

  it('should mask short secrets completely', () => {
    expect(maskSecret('short')).toBe('****')
  })

  it('should detect secrets in nested objects', () => {
    const data = {
      name: 'MyApp',
      config: {
        apiKey: 'sk-1234567890abcdefghijklmnopqrstuvwxyz',
        endpoint: 'https://api.example.com',
        nested: {
          password: 'super_secret_password_12345',
        },
      },
    }

    const secretPaths = detectSecrets(data)
    
    expect(secretPaths).toContain('config.apiKey')
    expect(secretPaths).toContain('config.nested.password')
    expect(secretPaths).not.toContain('name')
    expect(secretPaths).not.toContain('config.endpoint')
  })

  it('should handle arrays in secret detection', () => {
    const data = {
      secrets: [
        { apiKey: 'sk-1234567890abcdefghijklmnopqrstuvwxyz' },
        { token: 'tk-1234567890abcdefghijklmnopqrstuvwxyz' },
      ],
    }

    const secretPaths = detectSecrets(data)
    
    expect(secretPaths).toContain('secrets[0].apiKey')
    expect(secretPaths).toContain('secrets[1].token')
  })

  it('should mask secrets in objects', () => {
    const data = {
      name: 'MyApp',
      apiKey: 'sk-1234567890abcdefghijklmnopqrstuvwxyz',
      port: 8080,
    }

    const masked = maskSecretsInObject(data) as typeof data
    
    expect(masked.name).toBe('MyApp')
    expect(masked.port).toBe(8080)
    expect(masked.apiKey).not.toBe(data.apiKey)
    expect(masked.apiKey).toContain('****')
  })
})

describe('Safe YAML Validation', () => {
  it('should reject dangerous YAML tags', () => {
    expect(validateSafeYaml('!!python/object/apply:os.system')).toEqual({
      safe: false,
      reason: 'YAML contains potentially dangerous language tags',
    })
  })

  it('should reject file inclusion', () => {
    expect(validateSafeYaml('config: !include config.yaml')).toEqual({
      safe: false,
      reason: 'YAML contains file inclusion tags',
    })
  })

  it('should reject binary data', () => {
    expect(validateSafeYaml('data: !!binary SGVsbG8=')).toEqual({
      safe: false,
      reason: 'YAML contains binary data tags',
    })
  })

  it('should reject excessively nested structures', () => {
    const nested = '{'.repeat(25) + '}}'.repeat(25)
    const result = validateSafeYaml(nested)
    expect(result.safe).toBe(false)
    expect(result.reason).toContain('nesting depth')
  })

  it('should accept safe YAML', () => {
    const safeYaml = `
name: my-app
services:
  api:
    port: 8080
    `
    expect(validateSafeYaml(safeYaml)).toEqual({ safe: true })
  })
})

describe('File Size Validation', () => {
  it('should format bytes correctly', () => {
    expect(formatBytes(0)).toBe('0 Bytes')
    expect(formatBytes(1024)).toBe('1 KB')
    expect(formatBytes(1024 * 1024)).toBe('1 MB')
    expect(formatBytes(1024 * 1024 * 10)).toBe('10 MB')
  })

  it('should validate file sizes', () => {
    expect(validateFileSize(1024, 'AZURE_YAML')).toEqual({ valid: true })
    
    expect(validateFileSize(FILE_SIZE_LIMITS.AZURE_YAML + 1, 'AZURE_YAML')).toEqual({
      valid: false,
      error: expect.stringContaining('exceeds maximum'),
    })
  })

  it('should check all size limits', () => {
    expect(validateFileSize(5 * 1024 * 1024, 'AZURE_YAML').valid).toBe(true)
    expect(validateFileSize(11 * 1024 * 1024, 'AZURE_YAML').valid).toBe(false)
    
    expect(validateFileSize(30 * 1024 * 1024, 'BACKUP_TOTAL').valid).toBe(true)
    expect(validateFileSize(51 * 1024 * 1024, 'BACKUP_TOTAL').valid).toBe(false)
  })
})

describe('Export Warning', () => {
  it('should generate warning message for secrets', () => {
    const secretPaths = ['config.apiKey', 'config.password', 'secrets.token']
    const message = getExportWarningMessage(secretPaths)
    
    expect(message).toContain('WARNING')
    expect(message).toContain('3 potential secret(s)')
    expect(message).toContain('config.apiKey')
    expect(message).toContain('config.password')
  })

  it('should truncate long lists', () => {
    const secretPaths = Array.from({ length: 10 }, (_, i) => `secret${i}`)
    const message = getExportWarningMessage(secretPaths)
    
    expect(message).toContain('and 5 more')
  })

  it('should return empty string for no secrets', () => {
    expect(getExportWarningMessage([])).toBe('')
  })
})
