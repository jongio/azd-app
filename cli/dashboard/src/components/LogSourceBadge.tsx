/**
 * LogSourceBadge - Badge indicating the source of a log entry
 * Shows whether a log came from local services or Azure
 */
import { Monitor, Cloud } from 'lucide-react'
import { cn } from '@/lib/utils'

// =============================================================================
// Types
// =============================================================================

export type LogSource = 'local' | 'azure'

export interface LogSourceBadgeProps {
  /** Source of the log entry */
  source: LogSource
  /** Size variant */
  size?: 'sm' | 'md'
  /** Whether to show the label text */
  showLabel?: boolean
  /** Additional class names */
  className?: string
}

// =============================================================================
// LogSourceBadge Component
// =============================================================================

export function LogSourceBadge({
  source,
  size = 'sm',
  showLabel = true,
  className,
}: LogSourceBadgeProps) {
  const isLocal = source === 'local'
  const Icon = isLocal ? Monitor : Cloud
  const label = isLocal ? 'Local' : 'Azure'

  const sizeClasses = {
    sm: 'text-[10px] px-1.5 py-0.5 gap-1',
    md: 'text-xs px-2 py-1 gap-1.5',
  }

  const iconSizes = {
    sm: 'w-3 h-3',
    md: 'w-3.5 h-3.5',
  }

  return (
    <span
      className={cn(
        'inline-flex items-center rounded font-medium',
        sizeClasses[size],
        isLocal ? [
          'bg-slate-100 dark:bg-slate-700',
          'text-slate-600 dark:text-slate-300',
        ] : [
          'bg-cyan-100 dark:bg-cyan-900/30',
          'text-cyan-700 dark:text-cyan-300',
        ],
        className
      )}
      title={isLocal ? 'Log from local service' : 'Log from Azure'}
    >
      <Icon className={iconSizes[size]} aria-hidden="true" />
      {showLabel && <span>{label}</span>}
    </span>
  )
}

// =============================================================================
// Inline variant for log entries
// =============================================================================

export interface InlineLogSourceProps {
  source: LogSource
  className?: string
}

/**
 * Minimal inline indicator for use within log entry rows
 */
export function InlineLogSource({ source, className }: InlineLogSourceProps) {
  const isLocal = source === 'local'
  const Icon = isLocal ? Monitor : Cloud

  return (
    <span
      className={cn(
        'inline-flex items-center justify-center rounded',
        isLocal 
          ? 'w-4 h-4 text-slate-400 dark:text-slate-500'
          : 'w-5 h-5 -mt-0.5 text-emerald-500 dark:text-emerald-400',
        className
      )}
      title={isLocal ? 'Local' : 'Azure'}
    >
      <Icon className={isLocal ? 'w-3 h-3' : 'w-3.5 h-3.5'} aria-hidden="true" />
    </span>
  )
}

export default LogSourceBadge
