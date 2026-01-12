/**
 * Object Field Component - Nested fieldset for object schema properties
 * 
 * Features:
 * - Expandable/collapsible section
 * - Nested field rendering for object properties
 * - Recursive support for deeply nested objects
 */

import { useState } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import type { SchemaProperty } from '@/lib/schema'
import { FieldLabel } from '../FieldLabel'
import { FieldRenderer } from '../FieldRenderer'
import { cn } from '@/lib/utils'

export interface ObjectFieldProps {
  name: string
  property: SchemaProperty
  autoSave?: boolean
  nested?: boolean
}

/**
 * Object Field Component
 */
export function ObjectField({
  name,
  property,
  autoSave = true,
  nested = false,
}: ObjectFieldProps) {
  const [isExpanded, setIsExpanded] = useState(true)

  // Get label
  const label = property.title || name
  const properties = property.properties || {}
  const propertyEntries = Object.entries(properties)

  return (
    <div className={cn('space-y-2', nested && 'ml-4')}>
      {/* Header with expand/collapse button */}
      <button
        type="button"
        onClick={() => setIsExpanded(!isExpanded)}
        className={cn(
          'flex items-center gap-2 w-full text-left',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded'
        )}
      >
        {isExpanded ? (
          <ChevronDown className="w-4 h-4 text-muted-foreground" />
        ) : (
          <ChevronRight className="w-4 h-4 text-muted-foreground" />
        )}
        
        <FieldLabel
          htmlFor={`field-${name}`}
          label={label}
          required={property.required}
          description={property.description}
          className="mb-0"
        />
      </button>

      {/* Nested fields */}
      {isExpanded && (
        <fieldset
          className={cn(
            'border border-border rounded-md p-4 space-y-4',
            'bg-card/50'
          )}
        >
          <legend className="sr-only">{label}</legend>
          
          {propertyEntries.length > 0 ? (
            propertyEntries.map(([key, prop]) => (
              <FieldRenderer
                key={key}
                name={`${name}.${key}`}
                property={prop}
                autoSave={autoSave}
                nested
              />
            ))
          ) : (
            <p className="text-sm text-muted-foreground italic">
              No properties defined
            </p>
          )}
        </fieldset>
      )}
    </div>
  )
}
