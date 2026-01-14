/**
 * Validation Types - Type definitions for validation system
 */

export type ValidationLevel = 'error' | 'warning' | 'info'

export interface ValidationError {
  /** Severity level */
  level: ValidationLevel
  /** Error message */
  message: string
  /** Field path (dot notation, e.g., 'services.api.port') */
  path: string
  /** Validation rule that failed */
  rule?: string
  /** Additional context or suggestions */
  context?: string
}

export interface ValidationResult {
  /** Whether validation passed (no errors) */
  valid: boolean
  /** All validation errors */
  errors: ValidationError[]
  /** All validation warnings */
  warnings: ValidationError[]
  /** All validation info messages */
  info: ValidationError[]
}

export interface ValidationOptions {
  /** Whether to validate entire configuration */
  full?: boolean
  /** Whether to include warnings */
  includeWarnings?: boolean
  /** Whether to include info messages */
  includeInfo?: boolean
}
