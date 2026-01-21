/**
 * Field Renderer - Routes schema properties to appropriate field components
 * 
 * Responsibilities:
 * - Detect field type from schema property
 * - Route to appropriate field component
 * - Apply common field behaviors (validation, tooltips, labels)
 */

import type { SchemaProperty } from '@/lib/schema'
import { StringField } from './fields/StringField'
import { NumberField } from './fields/NumberField'
import { BooleanField } from './fields/BooleanField'
import { EnumField } from './fields/EnumField'
import { ArrayField } from './fields/ArrayField'
import { ObjectField } from './fields/ObjectField'

export interface FieldRendererProps {
  /** Field name (becomes form field key) */
  name: string
  /** Schema property definition */
  property: SchemaProperty
  /** Auto-save on blur */
  autoSave?: boolean
  /** Field is nested (affects styling) */
  nested?: boolean
}

/**
 * Field Renderer Component
 * 
 * Determines the appropriate field component based on schema property type
 * and renders it with common field behaviors.
 */
export function FieldRenderer({
  name,
  property,
  autoSave = true,
  nested = false,
}: FieldRendererProps) {
  // Route to appropriate field component based on type
  switch (property.type) {
    case 'string':
      return (
        <StringField
          name={name}
          property={property}
          autoSave={autoSave}
          nested={nested}
        />
      )

    case 'number':
      return (
        <NumberField
          name={name}
          property={property}
          autoSave={autoSave}
          nested={nested}
        />
      )

    case 'boolean':
      return (
        <BooleanField
          name={name}
          property={property}
          autoSave={autoSave}
          nested={nested}
        />
      )

    case 'enum':
      return (
        <EnumField
          name={name}
          property={property}
          autoSave={autoSave}
          nested={nested}
        />
      )

    case 'array':
      return (
        <ArrayField
          name={name}
          property={property}
          autoSave={autoSave}
          nested={nested}
          FieldRenderer={FieldRenderer}
        />
      )

    case 'object':
      return (
        <ObjectField
          name={name}
          property={property}
          autoSave={autoSave}
          nested={nested}
          FieldRenderer={FieldRenderer}
        />
      )

    default:
      // Fallback to string field for unknown types
      return (
        <StringField
          name={name}
          property={property}
          autoSave={autoSave}
          nested={nested}
        />
      )
  }
}
