/**
 * String Field Component - Text input for string schema properties
 * 
 * Features:
 * - Single-line text input
 * - Multi-line textarea for long strings (>100 chars or multiline pattern)
 * - Pattern validation (regex)
 * - Min/max length validation
 * - Placeholder with default value
 * - Auto-save on blur
 */

import { useFormContext } from 'react-hook-form'
import type { SchemaProperty } from '@/lib/schema'
import { Input } from '@/components/ui/input'
import { FieldLabel } from '../FieldLabel'
import { FieldError } from '../FieldError'
import { cn } from '@/lib/utils'

export interface StringFieldProps {
  name: string
  property: SchemaProperty
  autoSave?: boolean
  nested?: boolean
}

/**
 * String Field Component
 */
export function StringField({
  name,
  property,
  autoSave = true,
  nested = false,
}: StringFieldProps) {
  const {
    register,
    formState: { errors },
  } = useFormContext()

  const error = errors[name]
  const fieldId = `field-${name}`
  
  // Determine if we should use textarea (for long strings)
  const useTextarea = property.maxLength && property.maxLength > 100

  // Build validation rules
  const validation: Record<string, unknown> = {}
  
  if (property.required) {
    validation.required = 'This field is required'
  }
  
  if (property.pattern) {
    validation.pattern = {
      value: new RegExp(property.pattern),
      message: `Must match pattern: ${property.pattern}`,
    }
  }
  
  if (property.minLength) {
    validation.minLength = {
      value: property.minLength,
      message: `Must be at least ${property.minLength} characters`,
    }
  }
  
  if (property.maxLength) {
    validation.maxLength = {
      value: property.maxLength,
      message: `Must be no more than ${property.maxLength} characters`,
    }
  }

  // Get label and placeholder
  const label = property.title || name
  const placeholder = property.defaultValue 
    ? String(property.defaultValue)
    : property.description 
      ? `Enter ${label.toLowerCase()}...`
      : ''

  return (
    <div className={cn('space-y-1', nested && 'ml-4')}>
      <FieldLabel
        htmlFor={fieldId}
        label={label}
        required={property.required}
        description={property.description}
      />
      
      {useTextarea ? (
        <textarea
          id={fieldId}
          {...register(name, validation)}
          placeholder={placeholder}
          className={cn(
            'flex min-h-[120px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background',
            'placeholder:text-muted-foreground',
            'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
            'disabled:cursor-not-allowed disabled:opacity-50',
            'resize-y',
            error && 'border-destructive focus-visible:ring-destructive'
          )}
          aria-invalid={!!error}
          aria-describedby={error ? `${fieldId}-error` : undefined}
        />
      ) : (
        <Input
          id={fieldId}
          type="text"
          {...register(name, validation)}
          placeholder={placeholder}
          className={cn(error && 'border-destructive focus-visible:ring-destructive')}
          aria-invalid={!!error}
          aria-describedby={error ? `${fieldId}-error` : undefined}
        />
      )}
      
      {error && (
        <FieldError
          message={error.message as string}
        />
      )}
    </div>
  )
}
