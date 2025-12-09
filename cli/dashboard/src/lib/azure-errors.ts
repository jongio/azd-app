/**
 * Azure error parsing utilities
 * Used by hooks and components to parse Azure API errors into specific types.
 */
import type { AzureErrorType, ParsedAzureError } from '@/types'

/**
 * Parses an error message to determine the specific Azure error type.
 */
export function parseAzureErrorType(message: string, statusCode?: number): AzureErrorType {
  const lower = message.toLowerCase()
  
  // Check status code first (most reliable)
  if (statusCode === 401) return 'auth'
  if (statusCode === 403) return 'permission'
  if (statusCode === 404) return 'not-found'
  if (statusCode === 429) return 'rate-limit'
  
  // Check message patterns for authentication
  if (lower.includes('azd auth login') || 
      lower.includes('authentication required') ||
      lower.includes('authentication not configured') ||
      lower.includes('unauthorized') ||
      lower.includes('no azure credentials')) {
    return 'auth'
  }
  
  // Permission errors
  if (lower.includes('authorization') ||
      lower.includes('authorizationfailed') ||
      lower.includes('permission') ||
      lower.includes('rbac') ||
      lower.includes('does not have authorization') ||
      lower.includes('forbidden')) {
    return 'permission'
  }
  
  // Resource not found
  if (lower.includes('resourcenotfound') ||
      lower.includes('not found') ||
      lower.includes('does not exist') ||
      lower.includes('could not find')) {
    return 'not-found'
  }
  
  // Rate limiting
  if (lower.includes('rate limit') ||
      lower.includes('throttl') ||
      lower.includes('too many requests') ||
      lower.includes('toomanyrequests')) {
    return 'rate-limit'
  }
  
  // Network errors
  if (lower.includes('network') ||
      lower.includes('timeout') ||
      lower.includes('timed out') ||
      lower.includes('connection refused') ||
      lower.includes('econnrefused') ||
      lower.includes('etimedout') ||
      lower.includes('dns') ||
      lower.includes('unreachable')) {
    return 'network'
  }
  
  // Workspace/Log Analytics configuration
  if (lower.includes('workspace') ||
      lower.includes('log analytics') ||
      lower.includes('diagnostic settings') ||
      lower.includes('workspacenotfound')) {
    return 'workspace'
  }
  
  // Query/syntax errors
  if (lower.includes('query') ||
      lower.includes('syntax') ||
      lower.includes('kql') ||
      lower.includes('badargumenterror') ||
      lower.includes('parse error')) {
    return 'query'
  }
  
  return 'generic'
}

/**
 * Extracts Retry-After value from response headers or message.
 * Returns undefined if no retry-after information is found.
 */
export function extractRetryAfter(response?: Response, message?: string): number | undefined {
  // Try from response headers
  if (response) {
    const retryAfter = response.headers.get('Retry-After')
    if (retryAfter) {
      const seconds = parseInt(retryAfter, 10)
      if (!isNaN(seconds)) return seconds
      
      // Could be a date string
      const date = new Date(retryAfter)
      if (!isNaN(date.getTime())) {
        return Math.ceil((date.getTime() - Date.now()) / 1000)
      }
    }
  }
  
  // Try to extract from message
  if (message) {
    const match = message.match(/retry.*?(\d+)\s*(?:second|sec|s)/i)
    if (match) {
      return parseInt(match[1], 10)
    }
  }
  
  // No retry-after information found - caller should apply default if needed
  return undefined
}

/**
 * Creates a ParsedAzureError from response and message.
 */
export function createParsedAzureError(
  message: string, 
  response?: Response
): ParsedAzureError {
  const statusCode = response?.status
  const errorType = parseAzureErrorType(message, statusCode)
  
  return {
    type: errorType,
    message,
    statusCode,
    retryAfter: errorType === 'rate-limit' ? extractRetryAfter(response, message) : undefined,
  }
}

/**
 * Convenience function to parse error from message string only.
 */
export function parseAzureError(message: string): AzureErrorType {
  return parseAzureErrorType(message)
}
