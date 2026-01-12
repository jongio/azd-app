/**
 * Schema Form - Dynamic form generator from JSON Schema
 * 
 * Responsibilities:
 * - Generate forms from ParsedSchema
 * - Manage form state with React Hook Form
 * - Route to appropriate field renderers
 * - Handle validation and submission
 */

import { useForm, FormProvider } from 'react-hook-form'
import { useEffect } from 'react'
import type { ParsedSchema } from '@/lib/schema'
import { FieldRenderer } from './FieldRenderer'
import { cn } from '@/lib/utils'

export interface SchemaFormProps {
  /** Parsed schema to generate form from */
  schema: ParsedSchema
  /** Initial form values */
  defaultValues?: Record<string, unknown>
  /** Callback when form changes (debounced) */
  onChange?: (values: Record<string, unknown>) => void
  /** Callback when form is submitted */
  onSubmit?: (values: Record<string, unknown>) => void
  /** Auto-save on blur */
  autoSave?: boolean
  /** Custom className */
  className?: string
  /** Show only specific fields (filter by property names) */
  fields?: string[]
}

/**
 * Schema Form Component
 * 
 * Dynamically generates a form from a parsed JSON Schema.
 * Uses React Hook Form for state management and validation.
 */
export function SchemaForm({
  schema,
  defaultValues = {},
  onChange,
  onSubmit,
  autoSave = true,
  className,
  fields,
}: SchemaFormProps) {
  const methods = useForm({
    defaultValues,
    mode: 'onBlur', // Validate on blur for better UX
  })

  const { handleSubmit, watch, formState: { errors } } = methods

  // Watch for changes and trigger onChange callback
  useEffect(() => {
    if (!onChange) return

    const subscription = watch((values) => {
      // Debounce onChange callback
      const timeoutId = setTimeout(() => {
        onChange(values as Record<string, unknown>)
      }, 500)

      return () => clearTimeout(timeoutId)
    })

    return () => subscription.unsubscribe()
  }, [watch, onChange])

  // Filter properties based on fields prop
  const propertiesToRender = fields
    ? Object.entries(schema.properties).filter(([name]) => fields.includes(name))
    : Object.entries(schema.properties)

  const handleFormSubmit = handleSubmit((data) => {
    onSubmit?.(data)
  })

  return (
    <FormProvider {...methods}>
      <form
        onSubmit={handleFormSubmit}
        className={cn('space-y-6', className)}
        noValidate
      >
        {propertiesToRender.map(([name, property]) => (
          <FieldRenderer
            key={name}
            name={name}
            property={property}
            autoSave={autoSave}
          />
        ))}
      </form>
    </FormProvider>
  )
}
