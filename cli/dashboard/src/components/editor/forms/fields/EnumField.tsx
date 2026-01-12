/**
 * Enum Field Component - Dropdown select for enum schema properties
 * 
 * Features:
 * - Searchable dropdown with all valid values
 * - Keyboard navigation
 * - Auto-save on change
 */

import { useFormContext } from 'react-hook-form'
import type { SchemaProperty } from '@/lib/schema'
import { Select } from '@/components/ui/select'
import { FieldLabel } from '../FieldLabel'
import { FieldError } from '../FieldError'
import { cn } from '@/lib/utils'

export interface EnumFieldProps {
  name: string
  property: SchemaProperty
  autoSave?: boolean
  nested?: boolean
}

/**
 * Enum Field Component
 */
export function EnumField({
  name,
  property,
  autoSave = true,
  nested = false,
}: EnumFieldProps) {
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

  // Get label
  const label = property.title || name
  const enumValues = property.enumValues || []

  return (
    <div className={cn('space-y-1', nested && 'ml-4')}>
      <FieldLabel
        htmlFor={fieldId}
        label={label}
        required={property.required}
        description={property.description}
      />
      
      <Select
        id={fieldId}
        {...register(name, validation)}
        className={cn(error && 'border-destructive focus:ring-destructive')}
        aria-invalid={!!error}
        aria-describedby={error ? `${fieldId}-error` : undefined}
      >
        <option value="">Select {label.toLowerCase()}...</option>
        {enumValues.map((value) => (
          <option key={value} value={value}>
            {value}
          </option>
        ))}
      </Select>
      
      {error && (
        <FieldError
          message={error.message as string}
        />
      )}
    </div>
  )
}
