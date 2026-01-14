/**
 * Validation Summary Panel - Displays all validation errors, warnings, and info messages
 * 
 * Features:
 * - Grouped by severity (error/warning/info)
 * - Shows counts in header
 * - Clickable items that navigate to problem field
 * - Expandable/collapsible sections
 * - Color-coded by severity
 */

import { useState } from 'react'
import { AlertCircle, AlertTriangle, Info, ChevronDown, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import { VirtualList } from '@/lib/performance'
import type { ValidationError } from '@/lib/editor/validation-types'

export interface ValidationSummaryPanelProps {
  /** Validation errors */
  errors: ValidationError[]
  /** Validation warnings */
  warnings: ValidationError[]
  /** Validation info messages */
  info: ValidationError[]
  /** Callback when user clicks on a validation item (for navigation) */
  onItemClick?: (path: string) => void
  /** Custom className */
  className?: string
}

/**
 * Validation Summary Panel Component
 */
export function ValidationSummaryPanel({
  errors,
  warnings,
  info,
  onItemClick,
  className,
}: ValidationSummaryPanelProps) {
  const [errorsExpanded, setErrorsExpanded] = useState(true)
  const [warningsExpanded, setWarningsExpanded] = useState(true)
  const [infoExpanded, setInfoExpanded] = useState(false)

  const hasIssues = errors.length > 0 || warnings.length > 0 || info.length > 0

  if (!hasIssues) {
    return (
      <div
        className={cn(
          'border-t border-border bg-card p-4 text-center text-sm text-muted-foreground',
          className
        )}
        role="status"
        aria-live="polite"
      >
        ✓ No validation issues
      </div>
    )
  }

  return (
    <div
      className={cn('border-t border-border bg-card', className)}
      role="region"
      aria-label="Validation Summary"
    >
      {/* Header */}
      <div className="border-b border-border bg-muted/50 px-4 py-2 font-medium text-sm">
        Validation:{' '}
        {errors.length > 0 && (
          <span className="text-destructive">
            {errors.length} error{errors.length !== 1 ? 's' : ''}
          </span>
        )}
        {errors.length > 0 && warnings.length > 0 && ', '}
        {warnings.length > 0 && (
          <span className="text-warning">
            {warnings.length} warning{warnings.length !== 1 ? 's' : ''}
          </span>
        )}
        {(errors.length > 0 || warnings.length > 0) && info.length > 0 && ', '}
        {info.length > 0 && (
          <span className="text-info">
            {info.length} suggestion{info.length !== 1 ? 's' : ''}
          </span>
        )}
      </div>

      {/* Content */}
      <div className="max-h-64 overflow-y-auto">
        {/* Errors */}
        {errors.length > 0 && (
          <ValidationSection
            title="Errors"
            icon={AlertCircle}
            count={errors.length}
            items={errors}
            expanded={errorsExpanded}
            onToggle={() => setErrorsExpanded(!errorsExpanded)}
            onItemClick={onItemClick}
            variant="error"
          />
        )}

        {/* Warnings */}
        {warnings.length > 0 && (
          <ValidationSection
            title="Warnings"
            icon={AlertTriangle}
            count={warnings.length}
            items={warnings}
            expanded={warningsExpanded}
            onToggle={() => setWarningsExpanded(!warningsExpanded)}
            onItemClick={onItemClick}
            variant="warning"
          />
        )}

        {/* Info */}
        {info.length > 0 && (
          <ValidationSection
            title="Suggestions"
            icon={Info}
            count={info.length}
            items={info}
            expanded={infoExpanded}
            onToggle={() => setInfoExpanded(!infoExpanded)}
            onItemClick={onItemClick}
            variant="info"
          />
        )}
      </div>
    </div>
  )
}

interface ValidationSectionProps {
  title: string
  icon: React.ComponentType<{ className?: string }>
  count: number
  items: ValidationError[]
  expanded: boolean
  onToggle: () => void
  onItemClick?: (path: string) => void
  variant: 'error' | 'warning' | 'info'
}

function ValidationSection({
  title,
  icon: Icon,
  count,
  items,
  expanded,
  onToggle,
  onItemClick,
  variant,
}: ValidationSectionProps) {
  const variantStyles = {
    error: 'text-destructive',
    warning: 'text-warning',
    info: 'text-info',
  }
  
  // Use virtual scrolling for large lists
  const useVirtualScrolling = items.length > 20

  return (
    <div className="border-b border-border last:border-b-0">
      {/* Section Header */}
      <button
        onClick={onToggle}
        className={cn(
          'flex items-center gap-2 w-full px-4 py-2 text-sm font-medium',
          'hover:bg-accent/50 transition-colors',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset',
          variantStyles[variant]
        )}
        aria-expanded={expanded}
        aria-label={`${expanded ? 'Collapse' : 'Expand'} ${title} (${count})`}
      >
        {expanded ? (
          <ChevronDown className="w-4 h-4" />
        ) : (
          <ChevronRight className="w-4 h-4" />
        )}
        <Icon className="w-4 h-4" />
        <span>
          {title} ({count})
        </span>
      </button>

      {/* Section Items */}
      {expanded && (
        useVirtualScrolling ? (
          <VirtualList
            items={items}
            itemHeight={60}
            height={Math.min(items.length * 60, 300)}
            width="100%"
            overscanCount={2}
            renderItem={(item) => (
              <ValidationItem item={item} onClick={onItemClick} variant={variant} />
            )}
            getItemKey={(index, items) => `${items[index].path}-${index}`}
          />
        ) : (
          <div className="divide-y divide-border" role="list">
            {items.map((item, idx) => (
              <ValidationItem
                key={idx}
                item={item}
                onClick={onItemClick}
                variant={variant}
              />
            ))}
          </div>
        )
      )}
    </div>
  )
}

interface ValidationItemProps {
  item: ValidationError
  onClick?: (path: string) => void
  variant: 'error' | 'warning' | 'info'
}

function ValidationItem({ item, onClick, variant }: ValidationItemProps) {
  const variantStyles = {
    error: 'border-l-destructive hover:bg-destructive/5',
    warning: 'border-l-warning hover:bg-warning/5',
    info: 'border-l-info hover:bg-info/5',
  }

  return (
    <button
      onClick={() => onClick?.(item.path)}
      className={cn(
        'flex flex-col gap-1 w-full px-4 py-2 text-left text-sm border-l-4',
        'transition-colors',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset',
        variantStyles[variant],
        onClick && 'cursor-pointer'
      )}
      role="listitem"
      aria-label={`${variant}: ${item.message} at ${item.path || 'root'}`}
      disabled={!onClick}
    >
      {/* Message */}
      <div className="font-medium">{item.message}</div>

      {/* Path */}
      {item.path && (
        <div className="text-xs text-muted-foreground">
          Path: {item.path}
        </div>
      )}

      {/* Context */}
      {item.context && (
        <div className="text-xs text-muted-foreground italic">
          {item.context}
        </div>
      )}
    </button>
  )
}

