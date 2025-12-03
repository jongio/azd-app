/**
 * ModernMetricsView - Performance metrics dashboard with modern styling
 * Displays aggregate metrics and service-level performance data
 */
import * as React from 'react'
import {
  Activity,
  Network,
  Clock,
  Heart,
  TrendingUp,
  TrendingDown,
  Minus,
  Server,
  CheckCircle2,
  AlertCircle,
  XCircle,
  Loader2,
  HelpCircle,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import {
  countActiveServices,
  countActivePorts,
  calculateAverageUptime,
  calculateHealthScore,
  formatDuration,
  formatResponseTime,
  getResponseTimeVariant,
  getHealthScoreVariant,
  getServiceUptime,
} from '@/lib/metrics-utils'
import { normalizeHealthStatus } from '@/lib/service-utils'
import type { Service, HealthReportEvent, HealthCheckResult, HealthStatus } from '@/types'

// =============================================================================
// Types
// =============================================================================

export interface ModernMetricsViewProps {
  /** Services data for computing metrics */
  services: Service[]
  /** Health report from health stream (optional) */
  healthReport?: HealthReportEvent | null
  /** Additional class names */
  className?: string
}

interface MetricCardProps {
  title: string
  value: string | number
  unit?: string
  icon: React.ComponentType<{ className?: string }>
  variant: 'primary' | 'info' | 'success' | 'warning' | 'error'
  trend?: 'up' | 'down' | 'stable' | null
  trendValue?: string
}

// Use HealthStatus from @/types - no local redefinition

// =============================================================================
// MetricCard Component
// =============================================================================

function MetricCard({ title, value, unit, icon: Icon, variant, trend, trendValue }: MetricCardProps) {
  const variantStyles = {
    primary: {
      bg: 'bg-cyan-50 dark:bg-cyan-500/10',
      border: 'border-cyan-200 dark:border-cyan-500/20',
      icon: 'bg-cyan-100 dark:bg-cyan-500/20 text-cyan-600 dark:text-cyan-400',
    },
    info: {
      bg: 'bg-blue-50 dark:bg-blue-500/10',
      border: 'border-blue-200 dark:border-blue-500/20',
      icon: 'bg-blue-100 dark:bg-blue-500/20 text-blue-600 dark:text-blue-400',
    },
    success: {
      bg: 'bg-emerald-50 dark:bg-emerald-500/10',
      border: 'border-emerald-200 dark:border-emerald-500/20',
      icon: 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-600 dark:text-emerald-400',
    },
    warning: {
      bg: 'bg-amber-50 dark:bg-amber-500/10',
      border: 'border-amber-200 dark:border-amber-500/20',
      icon: 'bg-amber-100 dark:bg-amber-500/20 text-amber-600 dark:text-amber-400',
    },
    error: {
      bg: 'bg-rose-50 dark:bg-rose-500/10',
      border: 'border-rose-200 dark:border-rose-500/20',
      icon: 'bg-rose-100 dark:bg-rose-500/20 text-rose-600 dark:text-rose-400',
    },
  }

  const TrendIcon = trend === 'up' ? TrendingUp : trend === 'down' ? TrendingDown : Minus
  const trendColor = trend === 'up' ? 'text-emerald-500' : trend === 'down' ? 'text-rose-500' : 'text-slate-400'
  const styles = variantStyles[variant]

  return (
    <div className={cn('rounded-xl border p-5', styles.bg, styles.border)}>
      <div className="flex items-center gap-3 mb-4">
        <div className={cn('p-2.5 rounded-lg', styles.icon)}>
          <Icon className="w-5 h-5" />
        </div>
        <span className="text-sm font-medium text-slate-500 dark:text-slate-400">{title}</span>
      </div>
      <div className="flex items-baseline gap-1.5">
        <span className="text-3xl font-bold text-slate-900 dark:text-slate-100">{value}</span>
        {unit && <span className="text-sm text-slate-500 dark:text-slate-400">{unit}</span>}
      </div>
      {trend && (
        <div className={cn('flex items-center gap-1 mt-2 text-sm', trendColor)}>
          <TrendIcon className="w-4 h-4" />
          {trendValue && <span>{trendValue}</span>}
        </div>
      )}
    </div>
  )
}

// =============================================================================
// Status Badge Components
// =============================================================================

function StatusBadge({ status }: { status?: string }) {
  const configs: Record<string, { icon: React.ReactNode; label: string; className: string }> = {
    running: {
      icon: <CheckCircle2 className="w-3.5 h-3.5" />,
      label: 'Running',
      className: 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 border-emerald-200 dark:border-emerald-500/30',
    },
    starting: {
      icon: <Loader2 className="w-3.5 h-3.5 animate-spin" />,
      label: 'Starting',
      className: 'bg-blue-100 dark:bg-blue-500/20 text-blue-700 dark:text-blue-300 border-blue-200 dark:border-blue-500/30',
    },
    stopped: {
      icon: <XCircle className="w-3.5 h-3.5" />,
      label: 'Stopped',
      className: 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300 border-slate-200 dark:border-slate-600',
    },
    error: {
      icon: <AlertCircle className="w-3.5 h-3.5" />,
      label: 'Error',
      className: 'bg-rose-100 dark:bg-rose-500/20 text-rose-700 dark:text-rose-300 border-rose-200 dark:border-rose-500/30',
    },
  }

  const config = configs[status ?? 'unknown'] ?? {
    icon: <HelpCircle className="w-3.5 h-3.5" />,
    label: 'Unknown',
    className: 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300 border-slate-200 dark:border-slate-600',
  }

  return (
    <span className={cn('inline-flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium border', config.className)}>
      {config.icon}
      {config.label}
    </span>
  )
}

function HealthBadge({ health }: { health?: string }) {
  const configs: Record<string, { label: string; className: string }> = {
    healthy: {
      label: 'Healthy',
      className: 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-300',
    },
    degraded: {
      label: 'Degraded',
      className: 'bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-300',
    },
    unhealthy: {
      label: 'Unhealthy',
      className: 'bg-rose-100 dark:bg-rose-500/20 text-rose-700 dark:text-rose-300',
    },
    // Note: 'starting' is a lifecycle state, not a health status
    // Health should be normalized to 'unknown' if backend sends 'starting'
  }

  const config = configs[health ?? 'unknown'] ?? {
    label: 'Unknown',
    className: 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300',
  }

  return (
    <span className={cn('inline-flex items-center px-2 py-1 rounded-md text-xs font-medium', config.className)}>
      {config.label}
    </span>
  )
}

// =============================================================================
// Response Time Display
// =============================================================================

function ResponseTimeCell({ ms }: { ms: number | null | undefined }) {
  const variant = getResponseTimeVariant(ms)
  const colorClass = {
    success: 'text-emerald-600 dark:text-emerald-400',
    warning: 'text-amber-600 dark:text-amber-400',
    error: 'text-rose-600 dark:text-rose-400',
    default: 'text-slate-500 dark:text-slate-400',
  }[variant]

  return <span className={cn('font-mono text-sm', colorClass)}>{formatResponseTime(ms)}</span>
}

// =============================================================================
// ModernMetricsView Component
// =============================================================================

export function ModernMetricsView({
  services,
  healthReport,
  className,
}: ModernMetricsViewProps) {
  // Compute aggregate metrics
  const activeCount = countActiveServices(services)
  const totalCount = services.length
  const activePorts = countActivePorts(services)
  const averageUptime = calculateAverageUptime(services)

  // Use health report for score if available
  const healthScore = healthReport
    ? Math.round((healthReport.summary.healthy / healthReport.summary.total) * 100) || 0
    : calculateHealthScore(services)

  const healthScoreVariant = getHealthScoreVariant(healthScore)

  // Get response times from health report
  const getServiceResponseTime = (serviceName: string): number | null => {
    if (!healthReport) return null
    const result = healthReport.services.find((s: HealthCheckResult) => s.serviceName === serviceName)
    return result?.responseTime ? result.responseTime / 1_000_000 : null
  }

  // Get health status from health report, normalized to valid HealthStatus values
  const getServiceHealthStatus = (service: Service): HealthStatus => {
    if (healthReport) {
      const result = healthReport.services.find((s: HealthCheckResult) => s.serviceName === service.name)
      if (result) {
        return normalizeHealthStatus(result.status)
      }
    }
    return normalizeHealthStatus(service.local?.health)
  }

  return (
    <div className={cn('space-y-8', className)}>
      {/* Overview Cards */}
      <section>
        <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-4">
          Overview
        </h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <MetricCard
            title="Active Services"
            value={`${activeCount}/${totalCount}`}
            icon={Activity}
            variant="primary"
          />
          <MetricCard
            title="Active Ports"
            value={activePorts}
            icon={Network}
            variant="info"
          />
          <MetricCard
            title="Average Uptime"
            value={formatDuration(averageUptime)}
            icon={Clock}
            variant="info"
          />
          <MetricCard
            title="Health Score"
            value={totalCount > 0 ? healthScore : '-'}
            unit={totalCount > 0 ? '%' : undefined}
            icon={Heart}
            variant={healthScoreVariant}
          />
        </div>
      </section>

      {/* Service Details Table */}
      <section>
        <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-4">
          Service Details
        </h2>
        {services.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-center bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700">
            <Server className="w-10 h-10 text-slate-300 dark:text-slate-600 mb-3" />
            <p className="text-sm text-slate-500 dark:text-slate-400">No services available</p>
          </div>
        ) : (
          <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
            {/* Table Header */}
            <div className="grid grid-cols-6 gap-4 px-4 py-3 bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-700 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              <div>Service</div>
              <div>Status</div>
              <div>Uptime</div>
              <div>Port</div>
              <div>Health</div>
              <div>Response</div>
            </div>

            {/* Table Body */}
            <div className="divide-y divide-slate-200 dark:divide-slate-700">
              {services.map((service) => {
                const uptime = getServiceUptime(service)
                const responseTime = getServiceResponseTime(service.name)
                const healthStatus = getServiceHealthStatus(service)

                return (
                  <div
                    key={service.name}
                    className="grid grid-cols-6 gap-4 px-4 py-3 items-center hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors"
                  >
                    <div className="font-medium text-slate-900 dark:text-slate-100 truncate">
                      {service.name}
                    </div>
                    <div>
                      <StatusBadge status={service.local?.status} />
                    </div>
                    <div className="text-sm text-slate-500 dark:text-slate-400">
                      {formatDuration(uptime ?? 0)}
                    </div>
                    <div className="text-sm text-slate-500 dark:text-slate-400 font-mono">
                      {service.local?.port ?? '-'}
                    </div>
                    <div>
                      <HealthBadge health={healthStatus} />
                    </div>
                    <div>
                      <ResponseTimeCell ms={responseTime} />
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        )}
      </section>
    </div>
  )
}
