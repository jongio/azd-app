/**
 * Validation Hook - Manages validation state and operations
 * 
 * Provides:
 * - Real-time validation with debouncing
 * - Validation state management
 * - Field-level error access
 * - Summary-level error access
 */

import { useState, useCallback, useEffect, useMemo } from 'react'
import { validateConfiguration } from '@/lib/editor/validation-engine'
import type { ValidationResult, ValidationError, ValidationOptions } from '@/lib/editor/validation-types'

const VALIDATION_DEBOUNCE_MS = 500

export interface UseValidationProps {
  /** Configuration to validate */
  config: Record<string, unknown> | null
  /** JSON Schema */
  schema: Record<string, unknown> | null
  /** Validation options */
  options?: ValidationOptions
  /** Whether to enable auto-validation */
  autoValidate?: boolean
}

export interface UseValidationResult {
  /** Validation result */
  result: ValidationResult | null
  /** Whether validation is in progress */
  isValidating: boolean
  /** Manually trigger validation */
  validate: () => void
  /** Get errors for a specific field path */
  getFieldErrors: (path: string) => ValidationError[]
  /** Whether configuration has any errors */
  hasErrors: boolean
  /** Whether configuration has any warnings */
  hasWarnings: boolean
  /** Whether configuration has any info messages */
  hasInfo: boolean
  /** Total issue count */
  totalIssues: number
}

/**
 * Hook for managing validation state
 */
export function useValidation({
  config,
  schema,
  options = {},
  autoValidate = true,
}: UseValidationProps): UseValidationResult {
  const [result, setResult] = useState<ValidationResult | null>(null)
  const [isValidating, setIsValidating] = useState(false)

  // Validate configuration
  const validate = useCallback(() => {
    if (!config || !schema) {
      setResult(null)
      return
    }

    setIsValidating(true)

    try {
      const validationResult = validateConfiguration(config, schema, options)
      setResult(validationResult)
    } catch (err) {
      setResult({
        valid: false,
        errors: [{
          level: 'error',
          message: err instanceof Error ? err.message : 'Validation failed',
          path: '',
          rule: 'validation-error',
        }],
        warnings: [],
        info: [],
      })
    } finally {
      setIsValidating(false)
    }
  }, [config, schema, options])

  // Auto-validate when config or schema changes (with debouncing)
  useEffect(() => {
    if (!autoValidate) return

    const timeoutId = setTimeout(() => {
      validate()
    }, VALIDATION_DEBOUNCE_MS)

    return () => clearTimeout(timeoutId)
  }, [autoValidate, validate])

  // Get errors for a specific field path
  const getFieldErrors = useCallback(
    (path: string): ValidationError[] => {
      if (!result) return []

      const allIssues = [...result.errors, ...result.warnings, ...result.info]
      
      // Match exact path or parent path
      return allIssues.filter(issue => {
        return issue.path === path || issue.path.startsWith(`${path}.`)
      })
    },
    [result]
  )

  // Computed properties
  const hasErrors = useMemo(() => (result?.errors.length || 0) > 0, [result])
  const hasWarnings = useMemo(() => (result?.warnings.length || 0) > 0, [result])
  const hasInfo = useMemo(() => (result?.info.length || 0) > 0, [result])
  const totalIssues = useMemo(
    () => (result?.errors.length || 0) + (result?.warnings.length || 0) + (result?.info.length || 0),
    [result]
  )

  return {
    result,
    isValidating,
    validate,
    getFieldErrors,
    hasErrors,
    hasWarnings,
    hasInfo,
    totalIssues,
  }
}
