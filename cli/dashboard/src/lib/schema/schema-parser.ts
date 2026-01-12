/**
 * Schema Parser - Parses JSON Schema into internal model
 * 
 * Responsibilities:
 * - Parse schema definitions into usable TypeScript types
 * - Extract properties, types, validation rules, enums
 * - Build internal model for form generation
 */

export type FieldType = 'string' | 'number' | 'boolean' | 'object' | 'array' | 'enum'

export interface ValidationRule {
  type: 'required' | 'pattern' | 'min' | 'max' | 'minLength' | 'maxLength' | 'minItems' | 'maxItems' | 'enum'
  value: unknown
  message?: string
}

export interface SchemaProperty {
  name: string
  type: FieldType
  title?: string
  description?: string
  required: boolean
  defaultValue?: unknown
  validation: ValidationRule[]
  enumValues?: string[]
  properties?: Record<string, SchemaProperty> // For object types
  items?: SchemaProperty // For array types
  pattern?: string
  minimum?: number
  maximum?: number
  minLength?: number
  maxLength?: number
  minItems?: number
  maxItems?: number
}

export interface ParsedSchema {
  name: string
  properties: Record<string, SchemaProperty>
  required: string[]
  definitions: Record<string, SchemaProperty>
}

/**
 * Determine field type from JSON Schema type and other properties
 */
function determineFieldType(schemaProp: Record<string, unknown>): FieldType {
  if (schemaProp.enum && Array.isArray(schemaProp.enum)) {
    return 'enum'
  }

  const type = schemaProp.type as string | string[] | undefined
  
  if (Array.isArray(type)) {
    // Handle union types - pick first non-null type
    const nonNullType = type.find(t => t !== 'null')
    return (nonNullType as FieldType) || 'string'
  }

  switch (type) {
    case 'string':
      return 'string'
    case 'number':
    case 'integer':
      return 'number'
    case 'boolean':
      return 'boolean'
    case 'object':
      return 'object'
    case 'array':
      return 'array'
    default:
      return 'string'
  }
}

/**
 * Extract validation rules from schema property
 */
function extractValidationRules(
  schemaProp: Record<string, unknown>,
  isRequired: boolean
): ValidationRule[] {
  const rules: ValidationRule[] = []

  if (isRequired) {
    rules.push({ type: 'required', value: true })
  }

  if (typeof schemaProp.pattern === 'string') {
    rules.push({ 
      type: 'pattern', 
      value: schemaProp.pattern,
      message: `Must match pattern: ${schemaProp.pattern}`,
    })
  }

  if (typeof schemaProp.minimum === 'number') {
    rules.push({ type: 'min', value: schemaProp.minimum })
  }

  if (typeof schemaProp.maximum === 'number') {
    rules.push({ type: 'max', value: schemaProp.maximum })
  }

  if (typeof schemaProp.minLength === 'number') {
    rules.push({ type: 'minLength', value: schemaProp.minLength })
  }

  if (typeof schemaProp.maxLength === 'number') {
    rules.push({ type: 'maxLength', value: schemaProp.maxLength })
  }

  if (typeof schemaProp.minItems === 'number') {
    rules.push({ type: 'minItems', value: schemaProp.minItems })
  }

  if (typeof schemaProp.maxItems === 'number') {
    rules.push({ type: 'maxItems', value: schemaProp.maxItems })
  }

  if (Array.isArray(schemaProp.enum)) {
    rules.push({ type: 'enum', value: schemaProp.enum })
  }

  return rules
}

/**
 * Parse a single schema property
 */
function parseProperty(
  name: string,
  schemaProp: Record<string, unknown>,
  isRequired: boolean
): SchemaProperty {
  const fieldType = determineFieldType(schemaProp)
  
  const property: SchemaProperty = {
    name,
    type: fieldType,
    title: schemaProp.title as string | undefined,
    description: schemaProp.description as string | undefined,
    required: isRequired,
    defaultValue: schemaProp.default,
    validation: extractValidationRules(schemaProp, isRequired),
    pattern: schemaProp.pattern as string | undefined,
    minimum: schemaProp.minimum as number | undefined,
    maximum: schemaProp.maximum as number | undefined,
    minLength: schemaProp.minLength as number | undefined,
    maxLength: schemaProp.maxLength as number | undefined,
    minItems: schemaProp.minItems as number | undefined,
    maxItems: schemaProp.maxItems as number | undefined,
  }

  if (fieldType === 'enum' && Array.isArray(schemaProp.enum)) {
    property.enumValues = schemaProp.enum.map(String)
  }

  if (fieldType === 'object' && schemaProp.properties && typeof schemaProp.properties === 'object') {
    const props = schemaProp.properties as Record<string, Record<string, unknown>>
    const requiredProps = (schemaProp.required as string[]) || []
    
    property.properties = Object.fromEntries(
      Object.entries(props).map(([key, value]) => [
        key,
        parseProperty(key, value, requiredProps.includes(key)),
      ])
    )
  }

  if (fieldType === 'array' && schemaProp.items && typeof schemaProp.items === 'object') {
    property.items = parseProperty(
      `${name}[]`,
      schemaProp.items as Record<string, unknown>,
      false
    )
  }

  return property
}

/**
 * Resolve $ref references in schema
 */
function resolveRef(
  ref: string,
  schema: Record<string, unknown>
): Record<string, unknown> | null {
  if (!ref.startsWith('#/')) {
    return null
  }

  const path = ref.substring(2).split('/')
  let current: unknown = schema

  for (const segment of path) {
    if (current && typeof current === 'object' && segment in current) {
      current = (current as Record<string, unknown>)[segment]
    } else {
      return null
    }
  }

  return current as Record<string, unknown> | null
}

/**
 * Parse definitions section of schema
 */
function parseDefinitions(
  schema: Record<string, unknown>
): Record<string, SchemaProperty> {
  const definitions = schema.definitions as Record<string, Record<string, unknown>> | undefined
  
  if (!definitions) {
    return {}
  }

  return Object.fromEntries(
    Object.entries(definitions).map(([key, value]) => {
      // Handle $ref in definitions
      let actualValue = value
      if (value.$ref && typeof value.$ref === 'string') {
        const resolved = resolveRef(value.$ref, schema)
        if (resolved) {
          actualValue = resolved
        }
      }

      return [key, parseProperty(key, actualValue, false)]
    })
  )
}

/**
 * Parse JSON Schema into internal model
 */
export function parseSchema(schema: Record<string, unknown>): ParsedSchema {
  const properties = schema.properties as Record<string, Record<string, unknown>> | undefined
  const required = (schema.required as string[]) || []
  const name = (schema.title as string) || 'Azure YAML Configuration'

  const parsedProperties: Record<string, SchemaProperty> = {}

  if (properties) {
    for (const [key, value] of Object.entries(properties)) {
      // Handle $ref references
      let actualValue = value
      if (value.$ref && typeof value.$ref === 'string') {
        const resolved = resolveRef(value.$ref, schema)
        if (resolved) {
          actualValue = resolved
        }
      }

      parsedProperties[key] = parseProperty(key, actualValue, required.includes(key))
    }
  }

  return {
    name,
    properties: parsedProperties,
    required,
    definitions: parseDefinitions(schema),
  }
}

/**
 * Get property by path (supports nested paths like "services.api.host")
 */
export function getPropertyByPath(
  schema: ParsedSchema,
  path: string
): SchemaProperty | null {
  const segments = path.split('.')
  let current: SchemaProperty | undefined
  
  // Check top-level properties first
  current = schema.properties[segments[0]]
  
  if (!current) {
    // Check definitions
    current = schema.definitions[segments[0]]
  }

  if (!current) {
    return null
  }

  // Traverse nested properties
  for (let i = 1; i < segments.length; i++) {
    if (current.properties) {
      current = current.properties[segments[i]]
    } else if (current.items) {
      // For array items, continue with the item schema
      current = current.items
      i-- // Don't consume the segment, apply it to items
    } else {
      return null
    }

    if (!current) {
      return null
    }
  }

  return current
}
