/**
 * Boolean Field Component - Toggle switch for boolean schema properties
 * 
 * Features:
 * - Modern toggle switch UI
 * - Accessible with keyboard support
 * - Auto-save on change
 */

import { useFormContext } from 'react-hook-form'
import { useId } from 'react'
import type { SchemaProperty } from '@/lib/schema'
import { FieldLabel } from '../FieldLabel'
import { cn } from '@/lib/utils'

export interface BooleanFieldProps {
  name: string
  property: SchemaProperty
  autoSave?: boolean
  nested?: boolean
}

/**
 * Boolean Field Component
 */
export function BooleanField({
  name,
  property,
  nested = false,
}: BooleanFieldProps) {
  const {
    register,
    watch,
  } = useFormContext()

  const value = watch(name)
  const fieldId = useId()

  // Get label
  const label = property.title || name

  return (
    <div className={cn('flex items-center justify-between gap-4', nested && 'ml-4')}>
      <FieldLabel
        htmlFor={fieldId}
        label={label}
        required={property.required}
        description={property.description}
        className="mb-0"
      />
      
      <button
        id={fieldId}
        type="button"
        role="switch"
        aria-checked={!!value}
        {...register(name)}
        onClick={(e) => {
          const target = e.currentTarget
          const currentValue = target.getAttribute('aria-checked') === 'true'
          // Toggle value
          const newValue = !currentValue
          target.setAttribute('aria-checked', String(newValue))
          // Update form value
          const changeEvent = new Event('change', { bubbles: true })
          Object.defineProperty(changeEvent, 'target', {
            writable: false,
            value: { name, value: newValue, checked: newValue }
          })
          target.dispatchEvent(changeEvent)
        }}
        className={cn(
          'relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent',
          'transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
          'disabled:cursor-not-allowed disabled:opacity-50',
          value ? 'bg-primary' : 'bg-input'
        )}
      >
        <span
          className={cn(
            'pointer-events-none block h-5 w-5 rounded-full bg-background shadow-lg ring-0 transition-transform',
            value ? 'translate-x-5' : 'translate-x-0'
          )}
        />
      </button>
    </div>
  )
}
