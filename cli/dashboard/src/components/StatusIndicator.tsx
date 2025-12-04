/**
 * StatusIndicator - Status indicator component with modern visual treatments
 * Follows design spec: cli/dashboard/design/components/status-indicators.md
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
  Eye,
  Hammer,
  CircleOff,
  CircleX,
  Heart,
  HeartPulse,
  HeartCrack,
  type LucideIcon 
} from 'lucide-react'
import { cn } from '@/lib/utils'
import type { HealthStatus } from '@/types'

// =============================================================================
// Types
// =============================================================================

export type ProcessStatus = 'running' | 'starting' | 'stopping' | 'stopped' | 'error' | 'restarting' | 'not-running' | 'watching' | 'building' | 'built' | 'completed' | 'failed'
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
  // Process service specific statuses
  watching: {
    color: 'success',
    text: 'Watching',
    icon: Eye,
    animation: 'heartbeat',
    bgLight: 'bg-emerald-50',
    bgDark: 'dark:bg-emerald-500/15',
    textLight: 'text-emerald-600',
    textDark: 'dark:text-emerald-400',
  },
  building: {
    color: 'warning',
    text: 'Building',
    icon: Hammer,
    animation: 'pulse',
    bgLight: 'bg-amber-50',
    bgDark: 'dark:bg-amber-500/15',
    textLight: 'text-amber-600',
    textDark: 'dark:text-amber-400',
  },
  built: {
    color: 'success',
    text: 'Built',
    icon: CheckCircle,
    animation: null,
    bgLight: 'bg-emerald-50',
    bgDark: 'dark:bg-emerald-500/15',
    textLight: 'text-emerald-600',
    textDark: 'dark:text-emerald-400',
  },
  completed: {
    color: 'success',
    text: 'Completed',
    icon: CheckCircle,
    animation: null,
    bgLight: 'bg-emerald-50',
    bgDark: 'dark:bg-emerald-500/15',
    textLight: 'text-emerald-600',
    textDark: 'dark:text-emerald-400',
  },
  failed: {
    color: 'error',
    text: 'Failed',
    icon: XCircle,
    animation: 'flash',
    bgLight: 'bg-rose-50',
    bgDark: 'dark:bg-rose-500/15',
    textLight: 'text-rose-600',
    textDark: 'dark:text-rose-400',
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
// StatusDot Component
// =============================================================================

interface StatusDotProps {
  status: EffectiveStatus
  animated?: boolean
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

export function StatusDot({ 
  status, 
  animated = true, 
  size = 'md',
  className 
}: StatusDotProps) {
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
        'inline-block rounded-full shrink-0',
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
// DualStatusBadge Component - Shows running state + health as dual icons
// =============================================================================

interface DualStatusBadgeProps {
  status: EffectiveStatus
  health?: HealthStatus
  className?: string
}

export function DualStatusBadge({ 
  status, 
  health,
  className 
}: DualStatusBadgeProps) {
  const reduceMotion = React.useMemo(() => 
    typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches,
  [])
  
  // Determine running state
  const isRunning = status === 'running' || status === 'healthy' || status === 'watching' || status === 'built' || status === 'completed'
  const isStopped = status === 'stopped' || status === 'not-running' || status === 'unknown'
  const isStarting = status === 'starting' || status === 'building'
  const isStopping = status === 'stopping'
  const isRestarting = status === 'restarting'
  const isError = status === 'error' || status === 'unhealthy' || status === 'failed'
  
  // Determine health status - use provided health or infer from status
  const effectiveHealth: HealthStatus = health ?? (
    (status === 'healthy' || status === 'running' || status === 'watching' || status === 'built' || status === 'completed') ? 'healthy' :
    status === 'degraded' ? 'degraded' :
    (status === 'unhealthy' || status === 'error' || status === 'failed') ? 'unhealthy' :
    'unknown'
  )

  return (
    <div className={cn('flex items-center gap-1.5', className)}>
      {/* Running State Icon - shows lifecycle state (running/stopped/starting/etc) */}
      <span 
        className={cn(
          "inline-flex items-center justify-center w-6 h-6 rounded-full transition-all duration-200",
          isRunning && "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30",
          isStopped && "bg-slate-500/10 text-slate-600 dark:text-slate-400 border border-slate-500/30",
          isStarting && "bg-sky-500/10 text-sky-600 dark:text-sky-400 border border-sky-500/30",
          isStopping && "bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/30",
          isRestarting && "bg-sky-500/10 text-sky-600 dark:text-sky-400 border border-sky-500/30",
          isError && "bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/30"
        )}
        title={`Process state: ${status}`}
      >
        {isRunning && <CheckCircle className="w-3 h-3 shrink-0" />}
        {isStopped && <CircleOff className="w-3 h-3 shrink-0" />}
        {isStarting && <Loader2 className={cn("w-3 h-3 shrink-0", !reduceMotion && "animate-spin")} />}
        {isStopping && <Loader2 className={cn("w-3 h-3 shrink-0", !reduceMotion && "animate-spin")} />}
        {isRestarting && <RefreshCw className={cn("w-3 h-3 shrink-0", !reduceMotion && "animate-spin")} />}
        {isError && <CircleX className="w-3 h-3 shrink-0" />}
      </span>
      
      {/* Health Status Icon - from real-time health checks */}
      <span 
        className={cn(
          "inline-flex items-center justify-center w-6 h-6 rounded-full transition-all duration-200",
          effectiveHealth === 'healthy' && "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30",
          effectiveHealth === 'degraded' && "bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/30",
          effectiveHealth === 'unhealthy' && "bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/30",
          effectiveHealth === 'unknown' && "bg-slate-500/10 text-slate-500 dark:text-slate-400 border border-slate-500/30"
        )}
        title={`Service health: ${effectiveHealth} (from health checks)`}
      >
        {effectiveHealth === 'healthy' ? (
          <Heart className={cn("w-3 h-3 shrink-0", !reduceMotion && "animate-modern-heartbeat")} />
        ) : effectiveHealth === 'degraded' ? (
          <HeartPulse className={cn("w-3 h-3 shrink-0", !reduceMotion && "animate-modern-breathe")} />
        ) : effectiveHealth === 'unhealthy' ? (
          <HeartCrack className={cn("w-3 h-3 shrink-0", !reduceMotion && "animate-modern-flash")} />
        ) : (
          <HelpCircle className="w-3 h-3 shrink-0" />
        )}
      </span>
    </div>
  )
}

// =============================================================================
// StatusBadge Component
// =============================================================================

interface StatusBadgeProps {
  status: EffectiveStatus
  showDot?: boolean
  className?: string
}

export function StatusBadge({ 
  status, 
  showDot = true,
  className 
}: StatusBadgeProps) {
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
      {showDot && <StatusDot status={status} size="sm" />}
      <span>{config.text}</span>
    </span>
  )
}

// =============================================================================
// StatusIndicator Component (Full)
// =============================================================================

interface StatusIndicatorProps {
  status: EffectiveStatus
  variant?: StatusVariant
  animated?: boolean
  showLabel?: boolean
  className?: string
}

export function StatusIndicator({
  status,
  variant = 'dot',
  animated = true,
  showLabel = false,
  className,
}: StatusIndicatorProps) {
  const config = getStatusConfig(status)
  const Icon = config.icon
  const reduceMotion = React.useMemo(() => 
    typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches,
  [])

  if (variant === 'dot') {
    return (
      <div className={cn('inline-flex items-center gap-2', className)}>
        <StatusDot status={status} animated={animated} />
        {showLabel && (
          <span className={cn('text-sm font-medium', config.textLight, config.textDark)}>
            {config.text}
          </span>
        )}
      </div>
    )
  }

  if (variant === 'badge') {
    return <StatusBadge status={status} className={className} />
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
// HealthPill Component
// =============================================================================

interface HealthPillProps {
  total: number
  healthy: number
  degraded: number
  unhealthy: number
  starting: number
  onClick?: () => void
  expanded?: boolean
  className?: string
}

export function HealthPill({
  total: _total,
  healthy,
  degraded,
  unhealthy,
  starting,
  onClick,
  expanded = false,
  className,
}: HealthPillProps) {
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
      <StatusDot status={overallStatus} size="sm" />
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
// ConnectionStatus Component
// =============================================================================

interface ConnectionStatusProps {
  connected: boolean
  reconnecting?: boolean
  className?: string
}

export function ConnectionStatus({
  connected,
  reconnecting = false,
  className,
}: ConnectionStatusProps) {
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
      <StatusDot status={status} size="sm" />
      <span className="sr-only">{label}</span>
    </div>
  )
}

// =============================================================================
// Skeleton/Loading Components
// =============================================================================

export function StatusSkeleton({ className }: { className?: string }) {
  return (
    <div 
      className={cn(
        'h-5 w-20 rounded-full bg-slate-200 dark:bg-slate-700 animate-pulse',
        className
      )} 
    />
  )
}

export function Spinner({ size = 'md', className }: { size?: 'sm' | 'md' | 'lg'; className?: string }) {
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
