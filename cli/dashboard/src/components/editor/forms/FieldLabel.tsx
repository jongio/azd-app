/**
 * Field Label Component - Reusable label with required indicator and help tooltip
 */

import { Info } from 'lucide-react'
import * as Tooltip from '@radix-ui/react-tooltip'
import { cn } from '@/lib/utils'

export interface FieldLabelProps {
  /** HTML for attribute (connects to input) */
  htmlFor: string
  /** Label text */
  label: string
  /** Whether field is required */
  required?: boolean
  /** Help text to show in tooltip */
  description?: string
  /** Custom className */
  className?: string
  /** Render the help icon inline with the label */
  showHelpIcon?: boolean
}

/**
 * Field Label Component
 * 
 * Displays field label with optional required indicator (*) and help tooltip (ⓘ).
 */
export function FieldLabel({
  htmlFor,
  label,
  required,
  description,
  className,
  showHelpIcon = true,
}: FieldLabelProps) {
  return (
    <div className={cn('flex items-center gap-2 mb-1.5', className)}>
      <label
        htmlFor={htmlFor}
        className="text-sm font-medium text-foreground"
      >
        {label}
        {required && <span className="text-destructive ml-1" aria-label="required">*</span>}
      </label>
      
      {description && showHelpIcon && (
        <HelpTooltip description={description} />
      )}
    </div>
  )
}

export function HelpTooltip({ description }: { description: string }) {
  return (
    <Tooltip.Provider delayDuration={200}>
      <Tooltip.Root>
        <Tooltip.Trigger asChild>
          <button
            type="button"
            className="inline-flex items-center justify-center text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
            aria-label="Help"
          >
            <Info className="w-3.5 h-3.5" />
          </button>
        </Tooltip.Trigger>
        <Tooltip.Portal>
          <Tooltip.Content
            className="max-w-xs rounded-md bg-popover px-3 py-2 text-sm text-popover-foreground shadow-md border border-border z-50"
            sideOffset={4}
          >
            {description}
            <Tooltip.Arrow className="fill-border" />
          </Tooltip.Content>
        </Tooltip.Portal>
      </Tooltip.Root>
    </Tooltip.Provider>
  )
}
