/**
 * Schema-Driven Form - Wrapper that integrates SchemaForm with editor state
 * 
 * This component bridges React Hook Form (used by SchemaForm) with the editor's
 * updateField mechanism, allowing schema-driven forms to update the YAML configuration.
 * 
 * Best Practices:
 * - All hooks are called unconditionally in the same order
 * - defaultValues is memoized with stable dependencies
 * - Early returns happen AFTER all hooks are called
 */

import { useCallback, useMemo } from 'react'
import { SchemaForm } from './forms/SchemaForm'
import type { ParsedSchema } from '@/lib/schema'
import { extractSubSchema, unflattenFormValues } from '@/lib/editor/schema-utils'

export interface SchemaDrivenFormProps {
  /** Parsed schema */
  schema: ParsedSchema | null
  /** Path to the section being edited (e.g., "test", "services.api", "services.api.test") */
  path: string | null
  /** Current configuration values */
  config: Record<string, unknown> | null
  /** Callback to update a field at a specific path */
  onUpdateField: (path: string, value: unknown) => void
  /** Custom className */
  className?: string
  /** Fields to show (filter by property names) */
  fields?: string[]
}

/**
 * Schema-Driven Form Component
 * 
 * Dynamically generates a form from the schema for the given path,
 * and integrates it with the editor's update mechanism.
 * 
 * IMPORTANT: All hooks must be called unconditionally before any early returns.
 */
export function SchemaDrivenForm({
  schema,
  path,
  config,
  onUpdateField,
  className,
  fields,
}: SchemaDrivenFormProps) {
  // ALL HOOKS MUST BE CALLED FIRST - before any conditional logic or early returns
  
  // Extract sub-schema for the given path
  const subSchema = useMemo(() => {
    if (!schema) {
      return null
    }
    // If path is null, use the root schema
    if (!path) {
      return schema
    }
    return extractSubSchema(schema, path)
  }, [schema, path])

  // Normalize fields to always be an array (never undefined)
  // Use stable empty array reference
  const normalizedFields = useMemo(() => {
    return fields && Array.isArray(fields) && fields.length > 0 ? fields : []
  }, [fields])
  
  
  // Get default values for the form (unflatten from config)
  // Memoize with stable dependencies to prevent creating new objects on every render
  const defaultValues = useMemo(() => {
    if (!config) {
      return {}
    }
    
    // If path is null, use config directly (filtered by fields if provided)
    if (!path) {
      if (normalizedFields.length > 0) {
        const result: Record<string, unknown> = {}
        for (const field of normalizedFields) {
          result[field] = config[field] ?? undefined
        }
        return result
      }
      return {}
    }
    
    return unflattenFormValues(config, path)
  }, [config, path, normalizedFields])

  // Handle form changes - convert flat form values to YAML path updates
  const handleChange = useCallback((values: Record<string, unknown>) => {
    // Update each field individually to preserve YAML structure
    for (const [fieldName, value] of Object.entries(values)) {
      const fullPath = path ? `${path}.${fieldName}` : fieldName
      onUpdateField(fullPath, value)
    }
  }, [path, onUpdateField])

  // Create stable key for schema to force remount when schema structure changes
  // This ensures React resets all hooks cleanly when switching between different schema sections
  const schemaKey = useMemo(() => {
    if (!subSchema) return 'no-schema'
    // Create key from schema structure (properties names)
    const propertyKeys = Object.keys(subSchema.properties).sort().join(',')
    return `${path || 'root'}-${propertyKeys}`
  }, [subSchema, path])

  // NOW we can do early returns - all hooks have been called above
  if (!subSchema) {
    return (
      <div className="rounded-lg border border-border bg-card p-6">
        <p className="text-sm text-muted-foreground">
          {!schema ? 'Loading schema...' : !path ? 'No section selected' : `No schema found for path: ${path}`}
        </p>
      </div>
    )
  }

  return (
    <SchemaForm
      key={schemaKey} // Force remount when schema structure changes
      schema={subSchema}
      defaultValues={defaultValues}
      onChange={handleChange}
      className={className}
      fields={normalizedFields.length > 0 ? normalizedFields : undefined}
    />
  )
}
