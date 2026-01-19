/**
 * Navigation Item - Individual item in navigation tree
 * 
 * Features:
 * - Active state highlighting
 * - Error/warning badges
 * - Expandable indicator
 * - Keyboard accessible
 * - Icon support
 */

import { ChevronRight, ChevronDown, LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'

export interface NavigationItemProps {
  /** Item label */
  label: string
  /** Optional icon */
  icon?: LucideIcon
  /** Nesting depth (for indentation) */
  depth: number
  /** Whether item is currently active */
  isActive: boolean
  /** Whether item is expanded (for parent nodes) */
  isExpanded?: boolean
  /** Whether item has children */
  hasChildren?: boolean
  /** Number of errors in this section */
  errorCount?: number
  /** Number of warnings in this section */
  warningCount?: number
  /** Click handler */
  onClick: () => void
  /** Toggle expand/collapse handler */
  onToggle?: () => void
  /** Custom className */
  className?: string
}

/**
 * Navigation Item Component
 */
export function NavigationItem({
  label,
  icon: Icon,
  depth,
  isActive,
  isExpanded,
  hasChildren,
  errorCount = 0,
  warningCount = 0,
  onClick,
  className,
}: NavigationItemProps) {
  const hasIssues = errorCount > 0 || warningCount > 0

  return (
    <div
      role="treeitem"
      aria-expanded={hasChildren ? isExpanded : undefined}
      aria-current={isActive ? 'page' : undefined}
      aria-level={depth + 1}
      aria-label={`${label}${hasIssues ? ` (${errorCount} errors, ${warningCount} warnings)` : ''}`}
    >
      <button
        onClick={onClick}
        className={cn(
          'flex items-center gap-2 px-3 py-1.5 w-full text-sm transition-colors group',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset',
          isActive
            ? 'bg-accent text-accent-foreground font-medium'
            : 'text-foreground hover:bg-accent/50 hover:text-accent-foreground',
          className
        )}
        style={{ paddingLeft: `${depth * 12 + 12}px` }}
      >
      {/* Expand/collapse chevron (visual only; button handles toggle) */}
      {hasChildren ? (
        <span
          className="p-0.5 -ml-1 rounded transition-colors flex-shrink-0"
          aria-hidden="true"
          style={{ width: '14px' }} // Match chevron icon width (3.5 * 4px = 14px)
        >
          {isExpanded ? (
            <ChevronDown className="w-3.5 h-3.5" />
          ) : (
            <ChevronRight className="w-3.5 h-3.5" />
          )}
        </span>
      ) : (
        // Spacer for items without chevron to align icons with items that have chevrons
        // Matches chevron visual width (14px)
        <span className="w-[14px] flex-shrink-0" aria-hidden="true" />
      )}

      {/* Icon */}
      {Icon && (
        <Icon
          className="w-4 h-4 flex-shrink-0"
        />
      )}

      {/* Label */}
      <span className="flex-1 text-left truncate">{label}</span>

      {/* Validation badges */}
      {hasIssues && (
        <div className="flex items-center gap-1 ml-auto">
          {errorCount > 0 && (
            <Badge
              variant="destructive"
              className="h-4 min-w-4 px-1 text-xs font-medium rounded-full"
              aria-label={`${errorCount} error${errorCount !== 1 ? 's' : ''}`}
            >
              {errorCount}
            </Badge>
          )}
          {warningCount > 0 && (
            <Badge
              variant="warning"
              className="h-4 min-w-4 px-1 text-xs font-medium rounded-full"
              aria-label={`${warningCount} warning${warningCount !== 1 ? 's' : ''}`}
            >
              {warningCount}
            </Badge>
          )}
        </div>
      )}
      </button>
    </div>
  )
}
