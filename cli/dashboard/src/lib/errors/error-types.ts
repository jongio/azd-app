/**
 * Error Types and Categories for Editor Error Handling
 */

export type ErrorCategory =
  | 'schema'
  | 'file-system'
  | 'validation'
  | 'network'
  | 'parse'
  | 'user-input'

export type ErrorSeverity = 'error' | 'warning' | 'info'

/**
 * Comprehensive error metadata
 */
export interface EditorError {
  /** Error type/category */
  type: ErrorCategory
  /** Severity level */
  severity: ErrorSeverity
  /** User-friendly message */
  message: string
  /** Technical details for debugging */
  technicalDetails?: string
  /** Field path (for field-level errors) */
  path?: string
  /** Timestamp */
  timestamp: Date
  /** Whether this error is retry-able */
  retryable: boolean
  /** Original error object */
  originalError?: Error
  /** Stack trace (development only) */
  stack?: string
}

/**
 * Error display strategy
 */
export type ErrorDisplayStrategy = 'inline' | 'summary' | 'modal' | 'toast'

/**
 * Toast notification type
 */
export interface ToastNotification {
  id: string
  type: 'success' | 'error' | 'warning' | 'info'
  message: string
  description?: string
  duration?: number // ms, 0 = no auto-dismiss
  action?: {
    label: string
    onClick: () => void
  }
}

/**
 * Modal error options
 */
export interface ModalErrorOptions {
  title: string
  message: string
  technicalDetails?: string
  actions?: Array<{
    label: string
    onClick: () => void
    variant?: 'primary' | 'secondary' | 'danger'
  }>
  dismissible?: boolean
}

/**
 * Recovery options
 */
export interface RecoveryOptions {
  /** Auto-save to localStorage */
  autoSave?: boolean
  /** Retry failed operation */
  retry?: boolean
  /** Fallback value */
  fallback?: any
  /** Custom recovery function */
  onRecover?: () => void
}
