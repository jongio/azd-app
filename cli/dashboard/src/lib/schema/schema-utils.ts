/**
 * Schema Utilities - Helper functions for working with schemas
 */

import type { ParsedSchema, SchemaProperty } from './schema-parser'

/**
 * Extract schema for a specific path in the configuration
 * 
 * @param schema - The full parsed schema
 * @param path - Dot-separated path (e.g., "services.api", "resources.storage")
 * @returns Schema for the specific path, or null if not found
 * 
 * @example
 * ```ts
 * const serviceSchema = getSchemaForPath(schema, "services.api")
 * // Returns the schema for a service definition
 * ```
 */
export function getSchemaForPath(
  schema: ParsedSchema,
  path: string
): ParsedSchema | null {
  const segments = path.split('.')
  
  if (segments.length === 0) {
    return schema
  }

  // First segment should be a top-level property (e.g., "services", "resources")
  const rootProperty = schema.properties[segments[0]]
  
  if (!rootProperty) {
    return null
  }

  // If only one segment, return the root property as a schema
  if (segments.length === 1) {
    return propertyToSchema(rootProperty)
  }

  // For paths like "services.api", we need to get the schema for individual items
  // This typically means getting the items schema or properties schema
  if (rootProperty.type === 'object' && rootProperty.properties) {
    // Object with named properties (less common for services/resources)
    const childProperty = rootProperty.properties[segments[1]]
    if (childProperty) {
      return propertyToSchema(childProperty)
    }
  }

  // More commonly, services/resources are defined as objects with arbitrary keys
  // In this case, we look for patternProperties or additionalProperties in the raw schema
  // For now, we'll use a heuristic: if it's an object type, extract the value schema
  
  // Look in definitions for common patterns
  const definitions = schema.definitions
  
  // Try to find "Service" or "Resource" definition based on the root property name
  const definitionName = getDefinitionNameForProperty(segments[0])
  const definition = definitions[definitionName]
  
  if (definition) {
    return propertyToSchema(definition)
  }

  // Fallback: return the root property schema
  return propertyToSchema(rootProperty)
}

/**
 * Convert a SchemaProperty to a ParsedSchema
 */
function propertyToSchema(property: SchemaProperty): ParsedSchema {
  return {
    name: property.title || property.name,
    properties: property.properties || {},
    required: property.validation
      .filter(rule => rule.type === 'required')
      .map(() => property.name),
    definitions: {},
  }
}

/**
 * Get the definition name for a property
 * Follows common naming conventions
 */
function getDefinitionNameForProperty(propertyName: string): string {
  const mappings: Record<string, string> = {
    'services': 'Service',
    'resources': 'Resource',
    'hooks': 'Hooks',
    'reqs': 'Requirement',
  }

  return mappings[propertyName] || propertyName
}

/**
 * Get service schema from the full schema
 */
export function getServiceSchema(schema: ParsedSchema): ParsedSchema | null {
  return schema.definitions['Service'] 
    ? propertyToSchema(schema.definitions['Service'])
    : null
}

/**
 * Get resource schema from the full schema
 */
export function getResourceSchema(schema: ParsedSchema): ParsedSchema | null {
  return schema.definitions['Resource']
    ? propertyToSchema(schema.definitions['Resource'])
    : null
}
