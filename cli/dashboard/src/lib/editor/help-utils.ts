/**
 * Help Utilities - Extract and format help content from schema
 * 
 * Provides utilities to extract descriptions, examples, and help text
 * from JSON Schema for use in tooltips and help panels.
 */

import type { SchemaProperty } from '@/lib/schema'

/**
 * Extract tooltip text from schema property
 */
export function getTooltipText(property: SchemaProperty): string | undefined {
  return property.description || property.title
}

/**
 * Format validation rules as user-friendly text
 */
export function formatValidationRules(property: SchemaProperty): string[] {
  const rules: string[] = []

  if (property.required) {
    rules.push('Required field')
  }

  if (property.pattern) {
    rules.push(`Must match pattern: ${property.pattern}`)
  }

  if (property.minLength !== undefined) {
    rules.push(`Minimum length: ${property.minLength}`)
  }

  if (property.maxLength !== undefined) {
    rules.push(`Maximum length: ${property.maxLength}`)
  }

  if (property.minimum !== undefined) {
    rules.push(`Minimum value: ${property.minimum}`)
  }

  if (property.maximum !== undefined) {
    rules.push(`Maximum value: ${property.maximum}`)
  }

  if (property.minItems !== undefined) {
    rules.push(`Minimum items: ${property.minItems}`)
  }

  if (property.maxItems !== undefined) {
    rules.push(`Maximum items: ${property.maxItems}`)
  }

  if (property.enumValues && property.enumValues.length > 0) {
    rules.push(`Allowed values: ${property.enumValues.join(', ')}`)
  }

  return rules
}

/**
 * Get detailed help text for a property
 */
export function getDetailedHelp(property: SchemaProperty): {
  description?: string
  validation: string[]
  examples?: string[]
} {
  return {
    description: property.description,
    validation: formatValidationRules(property),
    examples: [], // Could be extended to include examples from schema
  }
}

/**
 * Detect section from field path
 * Examples:
 *   'services' -> 'services'
 *   'services.api.port' -> 'services'
 *   'hooks.postprovision' -> 'hooks'
 */
export function detectSection(fieldPath: string): string | undefined {
  if (!fieldPath) return undefined
  
  const segments = fieldPath.split('.')
  return segments[0]
}

/**
 * Get subsection from field path
 * Examples:
 *   'services.api.ports' -> 'ports'
 *   'services.api.healthcheck' -> 'healthcheck'
 *   'services.api.environment' -> 'environment'
 */
export function detectSubsection(fieldPath: string): string | undefined {
  if (!fieldPath) return undefined
  
  const segments = fieldPath.split('.')
  
  // For deeply nested paths, get the last meaningful segment
  if (segments.length >= 3) {
    const lastSegment = segments[segments.length - 1]
    
    // Check if it's a known subsection
    const knownSubsections = [
      'ports',
      'environment',
      'healthcheck',
      'test',
      'logs',
      'hooks',
      'docker',
      'k8s',
    ]
    
    if (knownSubsections.includes(lastSegment)) {
      return lastSegment
    }
    
    // Check second-to-last segment
    const secondLast = segments[segments.length - 2]
    if (knownSubsections.includes(secondLast)) {
      return secondLast
    }
  }
  
  return undefined
}

/**
 * Get the most specific help section for a field path
 */
export function getHelpSectionForField(fieldPath: string): string | undefined {
  const subsection = detectSubsection(fieldPath)
  if (subsection) return subsection
  
  return detectSection(fieldPath)
}

/**
 * Format example value for a property type
 */
export function formatExampleValue(property: SchemaProperty): string {
  if (property.defaultValue !== undefined) {
    return String(property.defaultValue)
  }

  // Check for enum values first (works for both 'enum' and 'string' types with enums)
  if (property.enumValues && property.enumValues.length > 0) {
    return property.enumValues[0]
  }

  switch (property.type) {
    case 'string':
      return 'example-value'

    case 'enum':
      return '' // Should have been handled above

    case 'number':
      if (property.minimum !== undefined) {
        return String(property.minimum)
      }
      return '0'

    case 'boolean':
      return 'true'

    case 'array':
      return '[]'

    case 'object':
      return '{}'

    default:
      return ''
  }
}

/**
 * Check if a field has complex validation that warrants a help icon
 */
export function hasComplexValidation(property: SchemaProperty): boolean {
  return (
    !!property.pattern ||
    (property.minLength !== undefined && property.maxLength !== undefined) ||
    (property.minimum !== undefined && property.maximum !== undefined) ||
    (property.enumValues && property.enumValues.length > 5) ||
    property.type === 'object' ||
    property.type === 'array'
  )
}
