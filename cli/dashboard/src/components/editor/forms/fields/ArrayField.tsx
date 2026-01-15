/**
 * Array Field Component - Repeatable field group for array schema properties
 * 
 * Features:
 * - Add/remove items
 * - Reorder items via drag-and-drop
 * - Nested field rendering for array items
 * - Min/max items validation
 */

import type React from 'react'
import { useFormContext, useWatch } from 'react-hook-form'
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
    setValue,
  } = useFormContext()

  const error = errors[name]
  const fieldId = `field-${name}`

  const parsedMax = property.maxItems !== undefined ? Number(property.maxItems) : undefined
  const parsedMin = property.minItems !== undefined ? Number(property.minItems) : undefined
  const validationMax = property.validation?.find((rule) => rule.type === 'maxItems')
  const validationMin = property.validation?.find((rule) => rule.type === 'minItems')

  const maxItems = Number.isFinite(parsedMax)
    ? parsedMax
    : validationMax && Number.isFinite(Number(validationMax.value))
      ? Number(validationMax.value)
      : undefined

  const minItems = Number.isFinite(parsedMin)
    ? parsedMin
    : validationMin && Number.isFinite(Number(validationMin.value))
      ? Number(validationMin.value)
      : undefined

  const watchedItems = useWatch({ control, name })
  const items = Array.isArray(watchedItems) ? watchedItems : []
  const itemCount = items.length

  const isAtMax = maxItems !== undefined && itemCount >= maxItems
  const canRemove = minItems === undefined || itemCount > minItems

  // Get label
  const label = property.title || name
  const itemSchema = property.items
  const itemLabel = itemSchema?.title || label.replace(/s$/i, '') || label

  // Handle add item
  const handleAddItem = () => {
    if (isAtMax) {
      return
    }
    const defaultValue = itemSchema?.defaultValue || (
      itemSchema?.type === 'string' ? '' :
      itemSchema?.type === 'number' ? 0 :
      itemSchema?.type === 'boolean' ? false :
      itemSchema?.type === 'object' ? {} :
      null
    )
    const nextItems = [...items, defaultValue]
    setValue(name, nextItems, { shouldDirty: true, shouldTouch: true })
  }

  const handleRemoveItem = (index: number) => {
    if (!canRemove) {
      return
    }

    const nextItems = items.filter((_, itemIndex) => itemIndex !== index)
    setValue(name, nextItems, { shouldDirty: true, shouldTouch: true })
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
    if (Number.isNaN(fromIndex) || fromIndex === toIndex) {
      return
    }

    const nextItems = [...items]
    const [moved] = nextItems.splice(fromIndex, 1)
    if (moved === undefined) {
      return
    }
    nextItems.splice(toIndex, 0, moved)
    setValue(name, nextItems, { shouldDirty: true, shouldTouch: true })
  }

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

        {!isAtMax && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleAddItem}
            className="gap-1"
            aria-disabled={isAtMax}
            title="Add array item"
          >
            <Plus className="w-3.5 h-3.5" />
            Add Item
          </Button>
        )}
      </div>

      <div className="space-y-2">
        {itemCount === 0 ? (
          <div className="text-sm text-muted-foreground italic py-4 text-center border border-dashed rounded-md">
            No items. Click "Add Item" to get started.
          </div>
        ) : (
          items.map((_, index) => (
            <div
              key={`${fieldId}-${index}`}
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
                      title: `${itemLabel} #${index + 1}`,
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
                  onClick={() => handleRemoveItem(index)}
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

      {minItems !== undefined && itemCount < minItems && (
        <p className="text-sm text-muted-foreground">
          Minimum {minItems} item{minItems !== 1 ? 's' : ''} required
        </p>
      )}

      {maxItems !== undefined && itemCount >= maxItems && (
        <p className="text-sm text-muted-foreground">
          Maximum {maxItems} item{maxItems !== 1 ? 's' : ''} reached
        </p>
      )}
    </div>
  )
}
