/**
 * ModernServiceStatusCard - Compact status summary with counts for errors, warnings, running, stopped
 * Follows modern design system with status indicators and counts
 */
import { XCircle, AlertTriangle, CheckCircle, Loader2, Activity, Circle } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Service, HealthSummary } from '@/types'
import { calculateStatusCounts } from '@/lib/service-utils'

// =============================================================================
// Types
// =============================================================================

export interface ModernServiceStatusCardProps {
  /** List of services to calculate status counts from */
  services: Service[]
  /** Whether there are active log errors */
  hasActiveErrors: boolean
  /** Whether the dashboard is in loading state */
  loading: boolean
  /** Click handler - typically navigates to console */
  onClick: () => void
  /** Real-time health summary from health stream */
  healthSummary?: HealthSummary | null
  /** Whether connected to health monitoring stream */
  healthConnected?: boolean
  /** Additional class names */
  className?: string
}

// =============================================================================
// StatusCount Component (internal)
// =============================================================================

interface StatusCountProps {
  icon: React.ReactNode
  count: number
  label: string
  activeColor: string
  inactiveColor?: string
  animate?: boolean
}

function StatusCount({ 
  icon, 
  count, 
  label, 
  activeColor, 
  inactiveColor = 'text-slate-400 dark:text-slate-500',
  animate = false 
}: StatusCountProps) {
  const isActive = count > 0
  
  return (
    <div 
      className="flex items-center gap-1.5" 
      title={`${count} ${label}`}
      role="status"
      aria-label={`${count} ${label}`}
    >
      <div className={cn(
        'w-5 h-5 rounded-full flex items-center justify-center transition-colors',
        isActive && animate && 'animate-modern-pulse'
      )}>
        <span className={isActive ? activeColor : inactiveColor}>
          {icon}
        </span>
      </div>
      <span className={cn(
        'text-sm tabular-nums font-medium',
        isActive ? activeColor : 'text-slate-400 dark:text-slate-500'
      )}>
        {count}
      </span>
    </div>
  )
}

// =============================================================================
// ModernServiceStatusCard Component
// =============================================================================

export function ModernServiceStatusCard({ 
  services, 
  hasActiveErrors, 
  loading, 
  onClick,
  healthSummary,
  healthConnected,
  className,
}: ModernServiceStatusCardProps) {
  // Use unified status count calculation
  const statusCounts = calculateStatusCounts(services, healthSummary, hasActiveErrors)

  if (loading) {
    return (
      <div className={cn(
        'flex items-center gap-2 px-3 py-1.5 rounded-lg',
        'bg-slate-100 dark:bg-slate-800/50',
        className
      )}>
        <Loader2 className="w-4 h-4 animate-spin text-slate-400" />
        <span className="text-xs text-slate-400">Loading...</span>
      </div>
    )
  }

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'flex items-center gap-4 px-3 py-1.5 rounded-lg',
        'transition-all duration-150 ease-out',
        'hover:bg-slate-100 dark:hover:bg-slate-800/50',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 focus-visible:ring-offset-2',
        'cursor-pointer group',
        className
      )}
      title="Click to view console logs"
      aria-label="Service status summary. Click to view console."
    >
      {/* Health monitoring indicator */}
      {healthConnected !== undefined && (
        <div 
          className="flex items-center gap-1" 
          title={healthConnected ? "Health monitoring active" : "Health monitoring disconnected"}
        >
          <Activity className={cn(
            'w-3.5 h-3.5 transition-colors',
            healthConnected 
              ? 'text-emerald-500 dark:text-emerald-400 animate-modern-heartbeat' 
              : 'text-slate-300 dark:text-slate-600'
          )} />
        </div>
      )}

      {/* Divider */}
      {healthConnected !== undefined && (
        <div className="w-px h-4 bg-slate-200 dark:bg-slate-700" />
      )}

      {/* Error count */}
      <StatusCount
        icon={<XCircle className="w-4 h-4" />}
        count={statusCounts.error}
        label="errors"
        activeColor="text-rose-500 dark:text-rose-400"
        animate={statusCounts.error > 0}
      />

      {/* Warning count */}
      <StatusCount
        icon={<AlertTriangle className="w-4 h-4" />}
        count={statusCounts.warn}
        label="warnings"
        activeColor="text-amber-500 dark:text-amber-400"
        animate={statusCounts.warn > 0}
      />

      {/* Running count */}
      <StatusCount
        icon={<CheckCircle className="w-4 h-4" />}
        count={statusCounts.running}
        label="running"
        activeColor="text-emerald-500 dark:text-emerald-400"
        animate={statusCounts.running > 0}
      />

      {/* Stopped count */}
      <StatusCount
        icon={<Circle className="w-4 h-4" />}
        count={statusCounts.stopped}
        label="stopped"
        activeColor="text-slate-500 dark:text-slate-400"
      />
    </button>
  )
}
