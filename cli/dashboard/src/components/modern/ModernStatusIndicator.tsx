/**
 * ModernStatusIndicator - Status indicator component with modern visual treatments
 * Follows design spec: cli/dashboard/design/modern/components/status-indicators.md
 */
import * as React from 'react'
import { 
  CheckCircle, 
  XCircle, 
  AlertTriangle, 
  Clock, 
  Circle, 
  HelpCircle,
  RefreshCw,
  Loader2,
  type LucideIcon 
} from 'lucide-react'
import { cn } from '@/lib/utils'
import type { HealthStatus } from '@/types'

// =============================================================================
// Types
// =============================================================================

export type ProcessStatus = 'running' | 'starting' | 'stopping' | 'stopped' | 'error' | 'restarting' | 'not-running'
export type EffectiveStatus = ProcessStatus | HealthStatus

export type StatusVariant = 'dot' | 'badge' | 'full'
export type AnimationType = 'heartbeat' | 'pulse' | 'breathe' | 'flash' | 'spin' | null

interface StatusConfig {
  color: string
  text: string
  icon: LucideIcon
  animation: AnimationType
  bgLight: string
  bgDark: string
  textLight: string
  textDark: string
}

// =============================================================================
// Status Configuration
// =============================================================================

const STATUS_CONFIG: Record<string, StatusConfig> = {
  running: {
    color: 'success',
    text: 'Running',
    icon: CheckCircle,
    animation: 'heartbeat',
    bgLight: 'bg-emerald-50',
    bgDark: 'dark:bg-emerald-500/15',
    textLight: 'text-emerald-600',
    textDark: 'dark:text-emerald-400',
  },
  healthy: {
    color: 'success',
    text: 'Healthy',
    icon: CheckCircle,
    animation: 'heartbeat',
    bgLight: 'bg-emerald-50',
    bgDark: 'dark:bg-emerald-500/15',
    textLight: 'text-emerald-600',
    textDark: 'dark:text-emerald-400',
  },
  starting: {
    color: 'info',
    text: 'Starting',
    icon: Loader2,
    animation: 'pulse',
    bgLight: 'bg-sky-50',
    bgDark: 'dark:bg-sky-500/15',
    textLight: 'text-sky-600',
    textDark: 'dark:text-sky-400',
  },
  stopping: {
    color: 'warning',
    text: 'Stopping',
    icon: Clock,
    animation: 'breathe',
    bgLight: 'bg-amber-50',
    bgDark: 'dark:bg-amber-500/15',
    textLight: 'text-amber-600',
    textDark: 'dark:text-amber-400',
  },
  stopped: {
    color: 'muted',
    text: 'Stopped',
    icon: Circle,
    animation: null,
    bgLight: 'bg-slate-100',
    bgDark: 'dark:bg-slate-500/15',
    textLight: 'text-slate-500',
    textDark: 'dark:text-slate-400',
  },
  degraded: {
    color: 'warning',
    text: 'Degraded',
    icon: AlertTriangle,
    animation: 'breathe',
    bgLight: 'bg-amber-50',
    bgDark: 'dark:bg-amber-500/15',
    textLight: 'text-amber-600',
    textDark: 'dark:text-amber-400',
  },
  error: {
    color: 'error',
    text: 'Error',
    icon: XCircle,
    animation: 'flash',
    bgLight: 'bg-rose-50',
    bgDark: 'dark:bg-rose-500/15',
    textLight: 'text-rose-600',
    textDark: 'dark:text-rose-400',
  },
  unhealthy: {
    color: 'error',
    text: 'Unhealthy',
    icon: XCircle,
    animation: 'flash',
    bgLight: 'bg-rose-50',
    bgDark: 'dark:bg-rose-500/15',
    textLight: 'text-rose-600',
    textDark: 'dark:text-rose-400',
  },
  unknown: {
    color: 'muted',
    text: 'Unknown',
    icon: HelpCircle,
    animation: null,
    bgLight: 'bg-slate-100',
    bgDark: 'dark:bg-slate-500/15',
    textLight: 'text-slate-500',
    textDark: 'dark:text-slate-400',
  },
  restarting: {
    color: 'info',
    text: 'Restarting',
    icon: RefreshCw,
    animation: 'spin',
    bgLight: 'bg-sky-50',
    bgDark: 'dark:bg-sky-500/15',
    textLight: 'text-sky-600',
    textDark: 'dark:text-sky-400',
  },
  'not-running': {
    color: 'muted',
    text: 'Not Running',
    icon: Circle,
    animation: null,
    bgLight: 'bg-slate-100',
    bgDark: 'dark:bg-slate-500/15',
    textLight: 'text-slate-500',
    textDark: 'dark:text-slate-400',
  },
}

// Animation class mapping
const ANIMATION_CLASSES: Record<AnimationType & string, string> = {
  heartbeat: 'animate-modern-heartbeat',
  pulse: 'animate-modern-pulse',
  breathe: 'animate-modern-breathe',
  flash: 'animate-modern-flash',
  spin: 'animate-spin',
}

// =============================================================================
// Helper Functions
// =============================================================================

function getStatusConfig(status: string): StatusConfig {
  return STATUS_CONFIG[status] || STATUS_CONFIG['unknown']
}

function getAnimationClass(animation: AnimationType, reduceMotion: boolean): string {
  if (!animation || reduceMotion) return ''
  return ANIMATION_CLASSES[animation] || ''
}

// =============================================================================
// ModernStatusDot Component
// =============================================================================

interface ModernStatusDotProps {
  status: EffectiveStatus
  animated?: boolean
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

export function ModernStatusDot({ 
  status, 
  animated = true, 
  size = 'md',
  className 
}: ModernStatusDotProps) {
  const config = getStatusConfig(status)
  const reduceMotion = React.useMemo(() => 
    typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches,
  [])
  
  const sizeClasses = {
    sm: 'w-1.5 h-1.5',
    md: 'w-2 h-2',
    lg: 'w-3 h-3',
  }
  
  const colorClasses = {
    success: 'bg-emerald-500 dark:bg-emerald-400',
    info: 'bg-sky-500 dark:bg-sky-400',
    warning: 'bg-amber-500 dark:bg-amber-400',
    error: 'bg-rose-500 dark:bg-rose-400',
    muted: 'bg-slate-400 dark:bg-slate-500',
  }

  return (
    <span
      className={cn(
        'inline-block rounded-full flex-shrink-0',
        sizeClasses[size],
        colorClasses[config.color as keyof typeof colorClasses],
        animated && getAnimationClass(config.animation, reduceMotion),
        className
      )}
      role="img"
      aria-label={config.text}
      title={config.text}
    />
  )
}

// =============================================================================
// ModernStatusBadge Component
// =============================================================================

interface ModernStatusBadgeProps {
  status: EffectiveStatus
  showDot?: boolean
  className?: string
}

export function ModernStatusBadge({ 
  status, 
  showDot = true,
  className 
}: ModernStatusBadgeProps) {
  const config = getStatusConfig(status)

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full',
        'text-xs font-semibold whitespace-nowrap',
        config.bgLight,
        config.bgDark,
        config.textLight,
        config.textDark,
        className
      )}
    >
      {showDot && <ModernStatusDot status={status} size="sm" />}
      <span>{config.text}</span>
    </span>
  )
}

// =============================================================================
// ModernStatusIndicator Component (Full)
// =============================================================================

interface ModernStatusIndicatorProps {
  status: EffectiveStatus
  variant?: StatusVariant
  animated?: boolean
  showLabel?: boolean
  className?: string
}

export function ModernStatusIndicator({
  status,
  variant = 'dot',
  animated = true,
  showLabel = false,
  className,
}: ModernStatusIndicatorProps) {
  const config = getStatusConfig(status)
  const Icon = config.icon
  const reduceMotion = React.useMemo(() => 
    typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches,
  [])

  if (variant === 'dot') {
    return (
      <div className={cn('inline-flex items-center gap-2', className)}>
        <ModernStatusDot status={status} animated={animated} />
        {showLabel && (
          <span className={cn('text-sm font-medium', config.textLight, config.textDark)}>
            {config.text}
          </span>
        )}
      </div>
    )
  }

  if (variant === 'badge') {
    return <ModernStatusBadge status={status} className={className} />
  }

  // Full variant with icon
  return (
    <div className={cn('inline-flex items-center gap-2', className)}>
      <Icon 
        className={cn(
          'w-4 h-4',
          config.textLight,
          config.textDark,
          animated && (config.animation === 'spin' && !reduceMotion) && 'animate-spin'
        )} 
        aria-hidden="true" 
      />
      <span className={cn('text-sm font-medium', config.textLight, config.textDark)}>
        {config.text}
      </span>
    </div>
  )
}

// =============================================================================
// ModernHealthPill Component
// =============================================================================

interface ModernHealthPillProps {
  total: number
  healthy: number
  degraded: number
  unhealthy: number
  starting: number
  onClick?: () => void
  expanded?: boolean
  className?: string
}

export function ModernHealthPill({
  total: _total,
  healthy,
  degraded,
  unhealthy,
  starting,
  onClick,
  expanded = false,
  className,
}: ModernHealthPillProps) {
  // Determine overall status
  let overallStatus: EffectiveStatus = 'healthy'
  let displayCount = healthy
  let displayLabel = 'Running'
  
  if (unhealthy > 0) {
    overallStatus = 'unhealthy'
    displayCount = unhealthy
    displayLabel = 'Unhealthy'
  } else if (degraded > 0) {
    overallStatus = 'degraded'
    displayCount = degraded
    displayLabel = 'Degraded'
  } else if (starting > 0) {
    overallStatus = 'starting'
    displayCount = starting
    displayLabel = 'Starting'
  }

  const config = getStatusConfig(overallStatus)

  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={`System status: ${displayCount} ${displayLabel.toLowerCase()} service${displayCount !== 1 ? 's' : ''}`}
      aria-haspopup={onClick ? 'true' : undefined}
      aria-expanded={onClick ? expanded : undefined}
      className={cn(
        'inline-flex items-center gap-2 px-3 py-1 rounded-full',
        'text-xs font-semibold cursor-pointer',
        'transition-all duration-150 ease-out',
        config.bgLight,
        config.bgDark,
        config.textLight,
        config.textDark,
        'hover:shadow-sm',
        className
      )}
    >
      <ModernStatusDot status={overallStatus} size="sm" />
      <span className="tabular-nums">{displayCount} {displayLabel}</span>
      {onClick && (
        <svg
          className={cn(
            'w-3.5 h-3.5 opacity-60 transition-transform duration-150',
            expanded && 'rotate-180'
          )}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      )}
    </button>
  )
}

// =============================================================================
// ModernConnectionStatus Component
// =============================================================================

interface ModernConnectionStatusProps {
  connected: boolean
  reconnecting?: boolean
  className?: string
}

export function ModernConnectionStatus({
  connected,
  reconnecting = false,
  className,
}: ModernConnectionStatusProps) {
  let status: EffectiveStatus = 'healthy'
  let label = 'Connected'
  
  if (!connected) {
    if (reconnecting) {
      status = 'starting'
      label = 'Reconnecting'
    } else {
      status = 'error'
      label = 'Disconnected'
    }
  }

  const config = getStatusConfig(status)

  return (
    <div
      className={cn(
        'inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs',
        config.textLight,
        config.textDark,
        className
      )}
    >
      <ModernStatusDot status={status} size="sm" />
      <span className="sr-only">{label}</span>
    </div>
  )
}

// =============================================================================
// Skeleton/Loading Components
// =============================================================================

export function ModernStatusSkeleton({ className }: { className?: string }) {
  return (
    <div 
      className={cn(
        'h-5 w-20 rounded-full bg-slate-200 dark:bg-slate-700 animate-pulse',
        className
      )} 
    />
  )
}

export function ModernSpinner({ size = 'md', className }: { size?: 'sm' | 'md' | 'lg'; className?: string }) {
  const sizeClasses = {
    sm: 'w-3.5 h-3.5 border-[1.5px]',
    md: 'w-5 h-5 border-2',
    lg: 'w-8 h-8 border-[3px]',
  }

  return (
    <div
      className={cn(
        'rounded-full border-slate-200 dark:border-slate-700 border-t-cyan-500 dark:border-t-cyan-400 animate-spin',
        sizeClasses[size],
        className
      )}
      role="status"
      aria-label="Loading"
    >
      <span className="sr-only">Loading...</span>
    </div>
  )
}
