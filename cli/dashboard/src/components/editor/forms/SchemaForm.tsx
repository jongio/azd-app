/**
 * Schema Form - Dynamic form generator from JSON Schema
 * 
 * Responsibilities:
 * - Generate forms from ParsedSchema
 * - Manage form state with React Hook Form
 * - Route to appropriate field renderers
 * - Handle validation and submission
 * 
 * Best Practices:
 * - useForm is initialized once with stable defaultValues
 * - Use reset() to update form values when defaultValues change
 * - All hooks are called unconditionally in the same order
 * - Memoize computed values to prevent unnecessary recalculations
 */

import { useForm, FormProvider } from 'react-hook-form'
import { useEffect, useRef, useMemo } from 'react'
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
 * 
 * IMPORTANT: All hooks must be called unconditionally in the same order every render.
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
  // Create stable initial defaultValues - only computed once on mount
  // This ensures useForm is initialized with a stable reference
  // We use a ref that's initialized once and never changes
  const initialDefaultValuesRef = useRef<Record<string, unknown>>(defaultValues)
  
  // Initialize form ONCE with stable initial defaultValues
  // React Hook Form uses defaultValues only for initial setup
  // We'll use reset() to update values when defaultValues prop changes
  // This hook must be called unconditionally and in the same order every render
  const methods = useForm({
    defaultValues: initialDefaultValuesRef.current, // Use stable ref - RHF only uses this on initial mount
    mode: 'onBlur', // Validate on blur for better UX
  })

  const { handleSubmit, watch, reset } = methods
  
  // Track previous defaultValues using a stable key (JSON string)
  // This prevents reset loops when defaultValues object reference changes but values are the same
  const prevDefaultValuesKeyRef = useRef<string>('')
  const isInitialMountRef = useRef(true)
  const isResettingRef = useRef(false)
  
  // Create stable key for defaultValues comparison
  // Only recalculate when defaultValues actually changes (deep comparison)
  const defaultValuesKey = useMemo(() => {
    try {
      return JSON.stringify(defaultValues)
    } catch {
      return ''
    }
  }, [defaultValues])
  
  // Initialize prevDefaultValuesKeyRef - use useEffect to avoid conditional during render
  useEffect(() => {
    if (prevDefaultValuesKeyRef.current === '') {
      prevDefaultValuesKeyRef.current = defaultValuesKey
    }
  }, []) // Only run once on mount
  
  // Reset form when defaultValues actually change (not on every render)
  // This effect must always run (no early returns) to maintain hook order
  useEffect(() => {
    // Skip on initial mount - form is already initialized with initialDefaultValues
    if (isInitialMountRef.current) {
      isInitialMountRef.current = false
      prevDefaultValuesKeyRef.current = defaultValuesKey
      return
    }
    
    // Only reset if values actually changed
    if (defaultValuesKey !== prevDefaultValuesKeyRef.current) {
      isResettingRef.current = true
      prevDefaultValuesKeyRef.current = defaultValuesKey
      reset(defaultValues, { keepDefaultValues: false })
      
      // Clear reset flag after a short delay
      const timeoutId = setTimeout(() => {
        isResettingRef.current = false
      }, 100)
      
      return () => {
        clearTimeout(timeoutId)
      }
    }
  }, [defaultValuesKey, defaultValues, reset])

  // Track previous form values to prevent unnecessary onChange calls
  const prevFormValuesKeyRef = useRef<string>('')
  const onChangeTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Watch for form changes and trigger onChange callback
  // This effect must always run (no early returns) to maintain hook order
  useEffect(() => {
    // If no onChange callback, don't set up watcher
    if (!onChange) {
      return
    }

    // Set up subscription to watch all form values
    const subscription = watch((values) => {
      // Don't trigger onChange during reset
      if (isResettingRef.current) {
        return
      }

      // Serialize current values for comparison
      const valuesKey = JSON.stringify(values)
      
      // Don't trigger onChange if values haven't actually changed
      if (valuesKey === prevFormValuesKeyRef.current) {
        return
      }

      // Clear any pending onChange call
      if (onChangeTimeoutRef.current) {
        clearTimeout(onChangeTimeoutRef.current)
      }

      // Debounce onChange calls
      onChangeTimeoutRef.current = setTimeout(() => {
        // Double-check we're not resetting and values actually changed
        if (!isResettingRef.current && valuesKey !== prevFormValuesKeyRef.current) {
          prevFormValuesKeyRef.current = valuesKey
          onChange(values as Record<string, unknown>)
        }
      }, 500)
    })

    // Cleanup function
    return () => {
      subscription.unsubscribe()
      if (onChangeTimeoutRef.current) {
        clearTimeout(onChangeTimeoutRef.current)
        onChangeTimeoutRef.current = null
      }
    }
  }, [watch, onChange])
  
  // Update prevFormValuesKeyRef when defaultValues change (initial sync)
  // This effect must always run to maintain hook order
  useEffect(() => {
    prevFormValuesKeyRef.current = defaultValuesKey
  }, [defaultValuesKey])

  // Memoize properties to render - prevents recalculation on every render
  // This ensures the number of FieldRenderer components is stable
  // Create stable key from schema properties to detect actual changes
  const schemaPropertiesKey = useMemo(() => {
    return Object.keys(schema.properties).sort().join(',')
  }, [schema.properties])
  
  // Memoize fields array to ensure stable reference
  const fieldsKey = useMemo(() => {
    return fields && Array.isArray(fields) && fields.length > 0 ? fields.sort().join(',') : 'all'
  }, [fields])
  
  const propertiesToRender = useMemo(() => {
    const allProperties = Object.entries(schema.properties)
    if (fields && Array.isArray(fields) && fields.length > 0) {
      return allProperties.filter(([name]) => fields.includes(name))
    }
    return allProperties
  }, [schemaPropertiesKey, fieldsKey, schema.properties])

  // Handle form submission
  const handleFormSubmit = handleSubmit((data) => {
    onSubmit?.(data)
  })

  // All hooks have been called unconditionally above
  // Now we can render conditionally if needed
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
        <button type="submit" aria-hidden="true" className="hidden" />
      </form>
    </FormProvider>
  )
}
