/**
 * Schema Utilities - Helper functions for working with schemas in the editor
 * 
 * Responsibilities:
 * - Extract sub-schemas for specific paths
 * - Convert nested paths to form field names
 * - Map form values to YAML paths
 */

import type { ParsedSchema, SchemaProperty } from '@/lib/schema'
import { getPropertyByPath } from '@/lib/schema/schema-parser'

/**
 * Extract a sub-schema for a given path (e.g., "test", "services.api", "services.api.test")
 * Returns a ParsedSchema that can be used with SchemaForm
 */
export function extractSubSchema(
  schema: ParsedSchema,
  path: string
): ParsedSchema | null {
  const segments = path.split('.')
  
  // Get the property at this path
  const property = getPropertyByPath(schema, path)
  
  if (!property || property.type !== 'object') {
    return null
  }

  // Build a sub-schema from the object's properties
  const subSchema: ParsedSchema = {
    name: segments[segments.length - 1] || path,
    properties: property.properties || {},
    required: Object.values(property.properties || {})
      .filter(prop => prop.required)
      .map(prop => prop.name),
    definitions: schema.definitions, // Include all definitions for $ref resolution
  }

  return subSchema
}

/**
 * Convert a form field name (e.g., "name", "test.parallel") to a YAML path
 * If basePath is provided, prepend it (e.g., basePath="services.api", field="host" -> "services.api.host")
 */
export function fieldNameToPath(basePath: string | null, fieldName: string): string {
  if (!basePath) {
    return fieldName
  }
  return `${basePath}.${fieldName}`
}

/**
 * Convert form values (flat object with dot notation) to nested object structure
 * Example: { "test.parallel": true, "test.outputDir": "./results" } 
 * -> { test: { parallel: true, outputDir: "./results" } }
 */
export function flattenFormValues(values: Record<string, unknown>, basePath: string | null): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  
  for (const [key, value] of Object.entries(values)) {
    const fullPath = fieldNameToPath(basePath, key)
    const segments = fullPath.split('.')
    
    let current: Record<string, unknown> = result
    for (let i = 0; i < segments.length - 1; i++) {
      const segment = segments[i]
      if (!(segment in current) || typeof current[segment] !== 'object' || current[segment] === null || Array.isArray(current[segment])) {
        current[segment] = {}
      }
      current = current[segment] as Record<string, unknown>
    }
    
    const lastSegment = segments[segments.length - 1]
    current[lastSegment] = value
  }
  
  return result
}

/**
 * Convert nested object to flat form values (for defaultValues)
 * Example: { test: { parallel: true, outputDir: "./results" } }
 * -> { "parallel": true, "outputDir": "./results" } (if basePath is "test")
 */
export function unflattenFormValues(nested: Record<string, unknown>, basePath: string | null): Record<string, unknown> {
  if (!basePath) {
    return nested
  }
  
  const segments = basePath.split('.')
  let current: unknown = nested
  
  for (const segment of segments) {
    if (typeof current === 'object' && current !== null && !Array.isArray(current)) {
      current = (current as Record<string, unknown>)[segment]
    } else {
      return {}
    }
  }
  
  return typeof current === 'object' && current !== null && !Array.isArray(current)
    ? (current as Record<string, unknown>)
    : {}
}
