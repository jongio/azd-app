/**
 * EnvironmentBadge - Displays Azure environment name in the header
 * Shows a subtle badge with the environment name (e.g., "· dev")
 * Hidden on mobile devices (<640px)
 */
import { cn } from '@/lib/utils'

// =============================================================================
// Types
// =============================================================================

export interface EnvironmentBadgeProps {
  /** Azure environment name to display */
  environmentName?: string
  /** Additional class names */
  className?: string
}

// =============================================================================
// EnvironmentBadge Component
// =============================================================================

export function EnvironmentBadge({ 
  environmentName, 
  className 
}: EnvironmentBadgeProps) {
  // Don't render if no environment name
  if (!environmentName) {
    return null
  }

  return (
    <span
      className={cn(
        'hidden sm:inline-flex items-center gap-1.5',
        'px-2 py-[3px] rounded',
        'text-xs font-medium leading-none',
        'bg-slate-100 dark:bg-slate-800',
        'text-slate-500 dark:text-slate-400',
        className
      )}
      aria-label={`Environment: ${environmentName}`}
      role="status"
    >
      <span className="text-cyan-500 dark:text-cyan-400 leading-none" aria-hidden="true">·</span>
      <span className="leading-none">{environmentName}</span>
    </span>
  )
}
