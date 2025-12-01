import * as React from 'react'
import { Copy, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

export interface InfoFieldProps {
  /** Label text describing the value */
  label: string

  /** Value to display - can be text or custom ReactNode */
  value: string | React.ReactNode

  /** Enable copy-to-clipboard functionality */
  copyable?: boolean

  /** Callback fired when copy is triggered */
  onCopy?: () => void

  /**
   * Value to copy (if different from displayed value)
   * Used when value is a ReactNode but you need to copy specific text
   */
  copyValue?: string

  /** Additional class names for the container */
  className?: string

  /** Data test ID for testing */
  'data-testid'?: string
}

export function InfoField({
  label,
  value,
  copyable = false,
  onCopy,
  copyValue,
  className,
  'data-testid': testId,
}: InfoFieldProps) {
  const [copied, setCopied] = React.useState(false)
  const id = React.useId()

  const handleCopy = () => {
    const textToCopy = copyValue ?? (typeof value === 'string' ? value : '')
    if (!textToCopy) return

    void navigator.clipboard.writeText(textToCopy).then(() => {
      setCopied(true)
      onCopy?.()
      setTimeout(() => setCopied(false), 2000)
    }).catch((error: unknown) => {
      console.error('Failed to copy:', error)
    })
  }

  const handleKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>) => {
    if (event.key === 'Escape') {
      event.currentTarget.blur()
    }
  }

  const canCopy = copyable && (typeof value === 'string' || copyValue)

  return (
    <div
      className={cn(
        'group py-2',
        'transition-colors duration-200',
        'hover:bg-secondary/30 rounded-md -mx-2 px-2',
        className
      )}
      data-testid={testId}
    >
      <span
        id={`${id}-label`}
        className="block text-xs font-medium text-muted-foreground mb-1"
      >
        {label}
      </span>

      <div
        className="flex items-center justify-between gap-2"
        aria-labelledby={`${id}-label`}
      >
        <span className="text-sm text-foreground break-all">
          {value}
        </span>

        {canCopy && (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={handleCopy}
            onKeyDown={handleKeyDown}
            aria-label={copied ? `${label} copied` : `Copy ${label}`}
            aria-live="polite"
            className={cn(
              'h-7 w-7 shrink-0',
              'opacity-0 group-hover:opacity-100 focus:opacity-100',
              'transition-all duration-200',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
              copied && 'text-green-600 dark:text-green-500'
            )}
          >
            {copied ? (
              <Check className="h-4 w-4" aria-hidden="true" />
            ) : (
              <Copy className="h-4 w-4" aria-hidden="true" />
            )}
          </Button>
        )}
      </div>
    </div>
  )
}
