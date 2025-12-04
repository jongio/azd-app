/**
 * ServiceStatusCard - Compact health status summary
 * Shows ONLY health status (healthy/degraded/unhealthy), NOT lifecycle states (running/stopped)
 * This follows the principle that health and lifecycle are orthogonal concepts
 */
import { Loader2, Activity, HelpCircle, Heart, HeartPulse, HeartCrack } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Service, HealthSummary } from '@/types'

// =============================================================================
// Types
// =============================================================================

export interface ServiceStatusCardProps {
  /** List of services (used for total count fallback) */
  services: Service[]
  /** Whether there are active log errors (shown as additional warning) */
  hasActiveErrors: boolean
  /** Whether the dashboard is in loading state */
  loading: boolean
  /** Click handler - typically navigates to console */
  onClick: () => void
  /** Real-time health summary from health stream - primary source of truth */
  healthSummary?: HealthSummary | null
  /** Whether connected to health monitoring stream */
  healthConnected?: boolean
  /** Additional class names */
  className?: string
}

// =============================================================================
// HealthCount Component (internal)
// =============================================================================

interface HealthCountProps {
  icon: React.ReactNode
  count: number
  label: string
  activeColor: string
  inactiveColor?: string
  animate?: boolean
}

function HealthCount({ 
  icon, 
  count, 
  label, 
  activeColor, 
  inactiveColor = 'text-slate-400 dark:text-slate-500',
  animate = false 
}: HealthCountProps) {
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
// ServiceStatusCard Component
// =============================================================================

export function ServiceStatusCard({ 
  services, 
  hasActiveErrors, 
  loading, 
  onClick,
  healthSummary,
  healthConnected,
  className,
}: ServiceStatusCardProps) {
  // Extract health-only counts from healthSummary
  // We show: unhealthy, degraded, healthy (and unknown if > 0)
  // We do NOT show lifecycle states like running/stopped
  const healthCounts = {
    unhealthy: healthSummary?.unhealthy ?? 0,
    degraded: healthSummary?.degraded ?? 0,
    healthy: healthSummary?.healthy ?? 0,
    unknown: (healthSummary?.unknown ?? 0) + (healthSummary?.starting ?? 0),
    total: healthSummary?.total ?? services.length,
  }
  
  // Add active log errors to degraded count if we have them
  // (log errors are a form of degradation even if health checks pass)
  const effectiveDegraded = healthCounts.degraded + (hasActiveErrors && healthCounts.unhealthy === 0 ? 1 : 0)

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
      aria-label="Service health summary. Click to view console."
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

      {/* Unhealthy count */}
      <HealthCount
        icon={<HeartCrack className="w-4 h-4" />}
        count={healthCounts.unhealthy}
        label="unhealthy"
        activeColor="text-rose-500 dark:text-rose-400"
        animate={healthCounts.unhealthy > 0}
      />

      {/* Degraded count (includes log errors) */}
      <HealthCount
        icon={<HeartPulse className="w-4 h-4" />}
        count={effectiveDegraded}
        label="degraded"
        activeColor="text-amber-500 dark:text-amber-400"
        animate={effectiveDegraded > 0}
      />

      {/* Healthy count */}
      <HealthCount
        icon={<Heart className="w-4 h-4" />}
        count={healthCounts.healthy}
        label="healthy"
        activeColor="text-emerald-500 dark:text-emerald-400"
        animate={healthCounts.healthy > 0}
      />

      {/* Unknown count - only show if there are unknowns */}
      {healthCounts.unknown > 0 && (
        <HealthCount
          icon={<HelpCircle className="w-4 h-4" />}
          count={healthCounts.unknown}
          label="unknown"
          activeColor="text-slate-500 dark:text-slate-400"
        />
      )}
    </button>
  )
}
