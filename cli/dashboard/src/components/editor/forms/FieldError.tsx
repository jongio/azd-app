/**
 * Field Error Component - Displays validation error message
 */

import { AlertCircle } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface FieldErrorProps {
  /** Error message to display */
  message: string
  /** Custom className */
  className?: string
}

/**
 * Field Error Component
 * 
 * Displays validation error message with icon.
 */
export function FieldError({ message, className }: FieldErrorProps) {
  return (
    <div
      className={cn('flex items-center gap-1.5 mt-1.5 text-sm text-destructive', className)}
      role="alert"
      aria-live="polite"
    >
      <AlertCircle className="w-3.5 h-3.5 flex-shrink-0" />
      <span>{message}</span>
    </div>
  )
}
