/**
 * Number Field Component - Numeric input for number/integer schema properties
 * 
 * Features:
 * - Numeric input with spinner controls
 * - Min/max value validation
 * - Integer vs decimal support
 * - Auto-save on blur
 */

import { useFormContext } from 'react-hook-form'
import type { SchemaProperty } from '@/lib/schema'
import { Input } from '@/components/ui/input'
import { FieldLabel, HelpTooltip } from '../FieldLabel'
import { FieldError } from '../FieldError'
import { cn } from '@/lib/utils'

export interface NumberFieldProps {
  name: string
  property: SchemaProperty
  autoSave?: boolean
  nested?: boolean
}

/**
 * Number Field Component
 */
export function NumberField({
  name,
  property,
  nested = false,
}: NumberFieldProps) {
  const {
    register,
    formState: { errors },
  } = useFormContext()

  const error = errors[name]
  const fieldId = `field-${name}`

  // Build validation rules
  const validation: Record<string, unknown> = {}
  
  if (property.required) {
    validation.required = 'This field is required'
  }
  
  if (property.minimum !== undefined) {
    validation.min = {
      value: property.minimum,
      message: `Must be at least ${property.minimum}`,
    }
  }
  
  if (property.maximum !== undefined) {
    validation.max = {
      value: property.maximum,
      message: `Must be no more than ${property.maximum}`,
    }
  }

  // Determine if integer or decimal
  const step = property.type === 'number' ? 'any' : '1'

  // Get label and placeholder
  const label = property.title || name
  const placeholder = property.defaultValue !== undefined
    ? String(property.defaultValue)
    : ''

  const help = property.description ? <HelpTooltip description={property.description} /> : null

  return (
    <div className={cn('space-y-1', nested && 'ml-4')}>
      <FieldLabel
        htmlFor={fieldId}
        label={label}
        required={property.required}
        description={property.description}
        showHelpIcon={false}
      />
      <div className="flex items-start gap-2">
        <div className="flex-1">
          <Input
            id={fieldId}
            type="number"
            step={step}
            min={property.minimum}
            max={property.maximum}
            {...register(name, {
              ...validation,
              valueAsNumber: true, // Convert to number
            })}
            placeholder={placeholder}
            className={cn(error && 'border-destructive focus-visible:ring-destructive')}
            aria-invalid={!!error}
            aria-describedby={error ? `${fieldId}-error` : undefined}
          />
        </div>
        {help}
      </div>
      
      {error && (
        <FieldError
          message={error.message as string}
        />
      )}
    </div>
  )
}
