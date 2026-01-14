/**
 * Error Factory - Create structured errors with metadata
 */

import type { EditorError, ErrorCategory, ErrorSeverity } from './error-types'

/**
 * Create a structured EditorError
 */
export function createError(
  type: ErrorCategory,
  message: string,
  options?: {
    severity?: ErrorSeverity
    technicalDetails?: string
    path?: string
    retryable?: boolean
    originalError?: Error
  }
): EditorError {
  const error: EditorError = {
    type,
    severity: options?.severity ?? 'error',
    message,
    technicalDetails: options?.technicalDetails,
    path: options?.path,
    timestamp: new Date(),
    retryable: options?.retryable ?? false,
    originalError: options?.originalError,
  }

  // Include stack trace in development
  if (((import.meta as any).env?.DEV || process.env.NODE_ENV === 'development') && options?.originalError) {
    error.stack = options.originalError.stack
  }

  return error
}

/**
 * Create a schema validation error
 */
export function createSchemaError(
  message: string,
  path?: string,
  technicalDetails?: string
): EditorError {
  return createError('schema', message, {
    severity: 'error',
    path,
    technicalDetails,
    retryable: false,
  })
}

/**
 * Create a file system error
 */
export function createFileSystemError(
  message: string,
  originalError?: Error,
  retryable = false
): EditorError {
  return createError('file-system', message, {
    severity: 'error',
    technicalDetails: originalError?.message,
    retryable,
    originalError,
  })
}

/**
 * Create a validation error
 */
export function createValidationError(
  message: string,
  path?: string,
  severity: ErrorSeverity = 'error'
): EditorError {
  return createError('validation', message, {
    severity,
    path,
    retryable: false,
  })
}

/**
 * Create a network error
 */
export function createNetworkError(
  message: string,
  originalError?: Error,
  retryable = true
): EditorError {
  return createError('network', message, {
    severity: 'error',
    technicalDetails: originalError?.message,
    retryable,
    originalError,
  })
}

/**
 * Create a parse error
 */
export function createParseError(
  message: string,
  technicalDetails?: string,
  path?: string
): EditorError {
  return createError('parse', message, {
    severity: 'error',
    path,
    technicalDetails,
    retryable: false,
  })
}

/**
 * Create a user input error
 */
export function createUserInputError(
  message: string,
  path?: string,
  severity: ErrorSeverity = 'error'
): EditorError {
  return createError('user-input', message, {
    severity,
    path,
    retryable: false,
  })
}

/**
 * Format error for display
 */
export function formatErrorMessage(error: EditorError): string {
  if (error.path) {
    return `${error.path}: ${error.message}`
  }
  return error.message
}

/**
 * Get user-friendly error message from various error types
 */
export function getUserFriendlyMessage(error: unknown): string {
  if (typeof error === 'string') {
    return error
  }

  if (error instanceof Error) {
    // Map common error messages to user-friendly versions
    if (error.message.includes('ENOENT')) {
      return 'File not found. Please check the file path.'
    }
    if (error.message.includes('EACCES') || error.message.includes('EPERM')) {
      return 'Permission denied. Please check file permissions.'
    }
    if (error.message.includes('ENOSPC')) {
      return 'Disk full. Please free up disk space.'
    }
    if (error.message.includes('Network') || error.message.includes('fetch')) {
      return 'Network error. Please check your connection.'
    }
    if (error.message.includes('timeout')) {
      return 'Request timeout. Please try again.'
    }

    return error.message
  }

  if (error && typeof error === 'object' && 'message' in error) {
    return String(error.message)
  }

  return 'An unknown error occurred'
}
