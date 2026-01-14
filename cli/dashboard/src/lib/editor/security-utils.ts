/**
 * Security utilities for Azure YAML Editor
 * Provides input sanitization, XSS prevention, and secret masking
 */

/**
 * HTML entity encoding map for XSS prevention
 */
const HTML_ENTITIES: Record<string, string> = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#x27;',
  '/': '&#x2F;',
}

/**
 * Encode HTML entities to prevent XSS attacks
 * @param text - Text to encode
 * @returns Encoded text safe for HTML rendering
 */
export function encodeHtmlEntities(text: string): string {
  return text.replace(/[&<>"'/]/g, (char) => HTML_ENTITIES[char] || char)
}

/**
 * Sanitize user input by encoding HTML entities
 * @param input - User input to sanitize
 * @returns Sanitized input safe for display
 */
export function sanitizeInput(input: string): string {
  if (!input) return ''
  
  // Encode HTML entities
  let sanitized = encodeHtmlEntities(input)
  
  // Remove any potential script injection patterns
  sanitized = sanitized.replace(/<script[^>]*>.*?<\/script>/gi, '')
  sanitized = sanitized.replace(/javascript:/gi, '')
  sanitized = sanitized.replace(/on\w+\s*=/gi, '')
  
  return sanitized
}

/**
 * Secret patterns to detect in configuration
 */
const SECRET_PATTERNS = [
  // Common secret keys
  /apikey/i,
  /api_key/i,
  /password/i,
  /passwd/i,
  /secret/i,
  /token/i,
  /auth/i,
  /connectionstring/i,
  /connection_string/i,
  /credential/i,
  /private[_-]?key/i,
  /access[_-]?key/i,
  /client[_-]?secret/i,
  
  // Azure-specific
  /azure[_-]?storage[_-]?key/i,
  /cosmos[_-]?key/i,
  /service[_-]?bus[_-]?key/i,
  /eventhub[_-]?key/i,
  /redis[_-]?key/i,
]

/**
 * Value patterns that look like secrets
 */
const SECRET_VALUE_PATTERNS = [
  // Base64 encoded (likely secrets)
  /^[A-Za-z0-9+/]{40,}={0,2}$/,
  
  // Connection strings
  /^(?=.*AccountKey=)(?=.*AccountName=).*/i,
  /^(?=.*Endpoint=)(?=.*AccessKey=).*/i,
  
  // JWT tokens
  /^eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/,
  
  // Azure storage keys (88 chars base64)
  /^[A-Za-z0-9+/]{88}$/,
]

/**
 * Check if a key name suggests it contains a secret
 * @param key - The key name to check
 * @returns True if key name suggests a secret
 */
export function isSecretKey(key: string): boolean {
  return SECRET_PATTERNS.some(pattern => pattern.test(key))
}

/**
 * Check if a value looks like a secret
 * @param value - The value to check
 * @returns True if value looks like a secret
 */
export function isSecretValue(value: string): boolean {
  if (!value || typeof value !== 'string') return false
  
  // Check if value is long enough to be a secret (at least 20 chars)
  if (value.length < 20) return false
  
  return SECRET_VALUE_PATTERNS.some(pattern => pattern.test(value))
}

/**
 * Mask a secret value for display
 * @param value - The secret value to mask
 * @param visibleChars - Number of characters to show at start/end (default: 4)
 * @returns Masked value
 */
export function maskSecret(value: string, visibleChars: number = 4): string {
  if (!value || value.length <= visibleChars * 2) {
    return '****'
  }
  
  const start = value.substring(0, visibleChars)
  const end = value.substring(value.length - visibleChars)
  const maskLength = Math.min(value.length - visibleChars * 2, 20)
  const mask = '*'.repeat(maskLength)
  
  return `${start}${mask}${end}`
}

/**
 * Detect secrets in a YAML object
 * @param data - The data to scan for secrets
 * @param path - Current path in the object (for reporting)
 * @returns Array of paths that contain secrets
 */
export function detectSecrets(data: unknown, path: string = ''): string[] {
  const secretPaths: string[] = []
  
  if (typeof data !== 'object' || data === null) {
    return secretPaths
  }
  
  if (Array.isArray(data)) {
    data.forEach((item, index) => {
      const itemPath = path ? `${path}[${index}]` : `[${index}]`
      secretPaths.push(...detectSecrets(item, itemPath))
    })
  } else {
    Object.entries(data).forEach(([key, value]) => {
      const currentPath = path ? `${path}.${key}` : key
      
      // Check if key suggests a secret
      if (isSecretKey(key) && typeof value === 'string') {
        secretPaths.push(currentPath)
      }
      
      // Check if value looks like a secret
      if (typeof value === 'string' && isSecretValue(value)) {
        secretPaths.push(currentPath)
      }
      
      // Recurse into nested objects
      if (typeof value === 'object' && value !== null) {
        secretPaths.push(...detectSecrets(value, currentPath))
      }
    })
  }
  
  return secretPaths
}

/**
 * Mask secrets in a YAML object for safe display
 * @param data - The data to mask secrets in
 * @returns Deep copy with secrets masked
 */
export function maskSecretsInObject(data: unknown): unknown {
  if (typeof data !== 'object' || data === null) {
    return data
  }
  
  if (Array.isArray(data)) {
    return data.map(item => maskSecretsInObject(item))
  }
  
  const masked: Record<string, unknown> = {}
  
  Object.entries(data).forEach(([key, value]) => {
    // Mask if key or value suggests a secret
    if (typeof value === 'string') {
      if (isSecretKey(key) || isSecretValue(value)) {
        masked[key] = maskSecret(value)
      } else {
        masked[key] = value
      }
    } else if (typeof value === 'object' && value !== null) {
      // Recurse into nested objects
      masked[key] = maskSecretsInObject(value)
    } else {
      masked[key] = value
    }
  })
  
  return masked
}

/**
 * Validate that content is safe YAML without code execution risks
 * @param content - YAML content to validate
 * @returns Validation result
 */
export function validateSafeYaml(content: string): { safe: boolean; reason?: string } {
  // Check for potentially dangerous YAML features
  
  // Check for YAML tags that could execute code
  if (content.includes('!!python') || content.includes('!!perl') || content.includes('!!ruby')) {
    return { safe: false, reason: 'YAML contains potentially dangerous language tags' }
  }
  
  // Check for file inclusion attempts
  if (content.includes('!!include') || content.includes('!include')) {
    return { safe: false, reason: 'YAML contains file inclusion tags' }
  }
  
  // Check for binary data tags (could hide malicious content)
  if (content.includes('!!binary')) {
    return { safe: false, reason: 'YAML contains binary data tags' }
  }
  
  // Check for excessively nested structures (potential DoS)
  const maxNestingDepth = 20
  let currentDepth = 0
  let maxDepth = 0
  
  for (const char of content) {
    if (char === '{' || char === '[') {
      currentDepth++
      maxDepth = Math.max(maxDepth, currentDepth)
    } else if (char === '}' || char === ']') {
      currentDepth--
    }
  }
  
  if (maxDepth > maxNestingDepth) {
    return { safe: false, reason: `YAML nesting depth (${maxDepth}) exceeds maximum (${maxNestingDepth})` }
  }
  
  return { safe: true }
}

/**
 * File size formatting utilities
 */
export function formatBytes(bytes: number, decimals: number = 2): string {
  if (bytes === 0) return '0 Bytes'
  
  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['Bytes', 'KB', 'MB', 'GB']
  
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i]
}

/**
 * File size limits (in bytes)
 */
export const FILE_SIZE_LIMITS = {
  AZURE_YAML: 10 * 1024 * 1024, // 10MB
  BACKUP_TOTAL: 50 * 1024 * 1024, // 50MB
  REQUEST_BODY: 10 * 1024 * 1024, // 10MB
} as const

/**
 * Validate file size against limits
 * @param size - File size in bytes
 * @param type - Type of file ('azure_yaml' | 'backup_total' | 'request_body')
 * @returns Validation result
 */
export function validateFileSize(
  size: number,
  type: keyof typeof FILE_SIZE_LIMITS
): { valid: boolean; error?: string } {
  const limitKey = type.toUpperCase().replace(/_/g, '_') as keyof typeof FILE_SIZE_LIMITS
  const limit = FILE_SIZE_LIMITS[limitKey]
  
  if (size > limit) {
    return {
      valid: false,
      error: `File size (${formatBytes(size)}) exceeds maximum allowed (${formatBytes(limit)})`,
    }
  }
  
  return { valid: true }
}

/**
 * Export warning message for configurations containing secrets
 */
export function getExportWarningMessage(secretPaths: string[]): string {
  if (secretPaths.length === 0) {
    return ''
  }
  
  const pathsList = secretPaths.slice(0, 5).map(p => `  - ${p}`).join('\n')
  const remaining = secretPaths.length > 5 ? `\n  ... and ${secretPaths.length - 5} more` : ''
  
  return `⚠️ WARNING: This configuration contains ${secretPaths.length} potential secret(s):\n${pathsList}${remaining}\n\nExporting this file may expose sensitive information. Please review carefully before sharing.`
}
