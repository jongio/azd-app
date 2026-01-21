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
import { cn } from '@/lib/utils'

// Import type only to avoid circular dependency
import type { FieldRendererProps } from '../FieldRenderer'

export interface ObjectFieldProps {
  name: string
  property: SchemaProperty
  autoSave?: boolean
  nested?: boolean
  /** FieldRenderer passed as a prop to break circular dependency */
  FieldRenderer: React.ComponentType<FieldRendererProps>
}

/**
 * Object Field Component
 */
export function ObjectField({
  name,
  property,
  autoSave = true,
  nested = false,
  FieldRenderer,
}: ObjectFieldProps) {
  const [isExpanded, setIsExpanded] = useState(true)

  // Get label
  const label = property.title || name
  const properties = property.properties || {}
  const propertyEntries = Object.entries(properties)

  return (
    <div className={cn('space-y-2', nested && 'ml-4')}>
      {/* Header with expand/collapse button */}
      {/* CRITICAL: Don't nest FieldLabel (which contains a button for tooltip) inside a button */}
      {/* Instead, use a div wrapper with click handler and separate button for expand/collapse */}
      <div
        className={cn(
          'flex items-center gap-2 w-full',
          'focus-within:outline-none focus-within:ring-2 focus-within:ring-ring rounded'
        )}
      >
        <button
          type="button"
          onClick={() => setIsExpanded(!isExpanded)}
          className="flex items-center justify-center p-1 text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
          aria-label={isExpanded ? 'Collapse section' : 'Expand section'}
          aria-expanded={isExpanded}
        >
          {isExpanded ? (
            <ChevronDown className="w-4 h-4" />
          ) : (
            <ChevronRight className="w-4 h-4" />
          )}
        </button>
        
        <div className="flex-1">
          <FieldLabel
            htmlFor={`field-${name}`}
            label={label}
            required={property.required}
            description={property.description}
            className="mb-0"
          />
        </div>
      </div>

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
