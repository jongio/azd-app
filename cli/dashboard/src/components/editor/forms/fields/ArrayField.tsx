/**
 * Array Field Component - Repeatable field group for array schema properties
 * 
 * Features:
 * - Add/remove items
 * - Reorder items via drag-and-drop
 * - Nested field rendering for array items
 * - Min/max items validation
 */

import { useFormContext, useFieldArray } from 'react-hook-form'
import { Plus, X, GripVertical } from 'lucide-react'
import type { SchemaProperty } from '@/lib/schema'
import { Button } from '@/components/ui/button'
import { FieldLabel } from '../FieldLabel'
import { FieldError } from '../FieldError'
import { FieldRenderer } from '../FieldRenderer'
import { cn } from '@/lib/utils'

export interface ArrayFieldProps {
  name: string
  property: SchemaProperty
  autoSave?: boolean
  nested?: boolean
}

/**
 * Array Field Component
 */
export function ArrayField({
  name,
  property,
  autoSave = true,
  nested = false,
}: ArrayFieldProps) {
  const {
    control,
    formState: { errors },
  } = useFormContext()

  const { fields, append, remove, move } = useFieldArray({
    control,
    name,
  })

  const error = errors[name]
  const fieldId = `field-${name}`

  // Get label
  const label = property.title || name
  const itemSchema = property.items

  // Handle add item
  const handleAddItem = () => {
    const defaultValue = itemSchema?.defaultValue || (
      itemSchema?.type === 'string' ? '' :
      itemSchema?.type === 'number' ? 0 :
      itemSchema?.type === 'boolean' ? false :
      itemSchema?.type === 'object' ? {} :
      null
    )
    append(defaultValue)
  }

  // Handle drag start (for reordering)
  const handleDragStart = (e: React.DragEvent<HTMLDivElement>, index: number) => {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', String(index))
  }

  // Handle drag over
  const handleDragOver = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
  }

  // Handle drop (reorder)
  const handleDrop = (e: React.DragEvent<HTMLDivElement>, toIndex: number) => {
    e.preventDefault()
    const fromIndex = parseInt(e.dataTransfer.getData('text/plain'), 10)
    if (fromIndex !== toIndex) {
      move(fromIndex, toIndex)
    }
  }

  // Check min/max items
  const canRemove = !property.minItems || fields.length > property.minItems
  const canAdd = !property.maxItems || fields.length < property.maxItems

  return (
    <div className={cn('space-y-2', nested && 'ml-4')}>
      <div className="flex items-center justify-between">
        <FieldLabel
          htmlFor={fieldId}
          label={label}
          required={property.required}
          description={property.description}
          className="mb-0"
        />
        
        {canAdd && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleAddItem}
            className="gap-1"
          >
            <Plus className="w-3.5 h-3.5" />
            Add Item
          </Button>
        )}
      </div>

      <div className="space-y-2">
        {fields.length === 0 ? (
          <div className="text-sm text-muted-foreground italic py-4 text-center border border-dashed rounded-md">
            No items. Click "Add Item" to get started.
          </div>
        ) : (
          fields.map((field, index) => (
            <div
              key={field.id}
              draggable
              onDragStart={(e) => handleDragStart(e, index)}
              onDragOver={handleDragOver}
              onDrop={(e) => handleDrop(e, index)}
              className={cn(
                'relative flex gap-2 p-3 rounded-md border border-border bg-card',
                'hover:border-border-secondary transition-colors'
              )}
            >
              {/* Drag handle */}
              <div className="flex items-start pt-2 cursor-move text-muted-foreground hover:text-foreground">
                <GripVertical className="w-4 h-4" />
              </div>

              {/* Item field */}
              <div className="flex-1">
                {itemSchema && (
                  <FieldRenderer
                    name={`${name}.${index}`}
                    property={{
                      ...itemSchema,
                      title: `${label} #${index + 1}`,
                    }}
                    autoSave={autoSave}
                    nested
                  />
                )}
              </div>

              {/* Remove button */}
              {canRemove && (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  onClick={() => remove(index)}
                  className="flex-shrink-0 h-8 w-8 text-muted-foreground hover:text-destructive"
                  aria-label={`Remove item ${index + 1}`}
                >
                  <X className="w-4 h-4" />
                </Button>
              )}
            </div>
          ))
        )}
      </div>

      {error && (
        <FieldError
          message={error.message as string}
        />
      )}

      {property.minItems && fields.length < property.minItems && (
        <p className="text-sm text-muted-foreground">
          Minimum {property.minItems} item{property.minItems !== 1 ? 's' : ''} required
        </p>
      )}

      {property.maxItems && fields.length >= property.maxItems && (
        <p className="text-sm text-muted-foreground">
          Maximum {property.maxItems} item{property.maxItems !== 1 ? 's' : ''} reached
        </p>
      )}
    </div>
  )
}
