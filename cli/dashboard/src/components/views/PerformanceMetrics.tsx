import * as React from 'react'
import {
  Activity,
  Network,
  Clock,
  Heart,
  TrendingUp,
  TrendingDown,
  Minus
} from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import {
  countActiveServices,
  countActivePorts,
  calculateAverageUptime,
  calculateHealthScore,
  formatDuration,
  formatResponseTime,
  getResponseTimeVariant,
  getHealthScoreVariant,
  getServiceUptime
} from '@/lib/metrics-utils'
import { getStatusBadgeConfig, getHealthBadgeConfig } from '@/lib/service-utils'
import type { Service, HealthReportEvent, HealthCheckResult } from '@/types'

// ============================================================================
// Types
// ============================================================================

export interface PerformanceMetricsProps {
  /** Services data for computing metrics */
  services: Service[]
  /** Health report from health stream (optional) */
  healthReport?: HealthReportEvent | null
  /** Additional class names */
  className?: string
  /** Data test ID for testing */
  'data-testid'?: string
}

export interface MetricCardProps {
  /** Card title */
  title: string
  /** Primary display value */
  value: string | number
  /** Unit or secondary text */
  unit?: string
  /** Card icon */
  icon: React.ComponentType<{ className?: string; 'aria-hidden'?: boolean | 'true' | 'false' }>
  /** Card color variant */
  variant: 'primary' | 'info' | 'success' | 'warning' | 'error'
  /** Trend direction */
  trend?: 'up' | 'down' | 'stable' | null
  /** Trend value (e.g., "+5%") */
  trendValue?: string
}

// ============================================================================
// MetricCard Component
// ============================================================================

function MetricCard({ 
  title, 
  value, 
  unit, 
  icon: Icon, 
  variant, 
  trend, 
  trendValue 
}: MetricCardProps) {
  const variantStyles = {
    primary: {
      card: 'border-primary/20 bg-primary/5',
      icon: 'text-primary bg-primary/10',
    },
    info: {
      card: 'border-blue-500/20 bg-blue-500/5',
      icon: 'text-blue-500 bg-blue-500/10',
    },
    success: {
      card: 'border-green-500/20 bg-green-500/5',
      icon: 'text-green-500 bg-green-500/10',
    },
    warning: {
      card: 'border-amber-500/20 bg-amber-500/5',
      icon: 'text-amber-500 bg-amber-500/10',
    },
    error: {
      card: 'border-destructive/20 bg-destructive/5',
      icon: 'text-destructive bg-destructive/10',
    },
  }

  const trendIcons = {
    up: TrendingUp,
    down: TrendingDown,
    stable: Minus,
  }

  const trendColors = {
    up: 'text-green-500',
    down: 'text-red-500',
    stable: 'text-muted-foreground',
  }

  const styles = variantStyles[variant]
  const TrendIcon = trend ? trendIcons[trend] : null

  return (
    <div
      className={`rounded-lg border p-6 ${styles.card}`}
      role="status"
      aria-label={`${title}: ${value}${unit ? ` ${unit}` : ''}`}
      data-testid={`metric-card-${title.toLowerCase().replace(/\s+/g, '-')}`}
    >
      <div className="flex items-center gap-3 mb-4">
        <div className={`p-2 rounded-lg ${styles.icon}`}>
          <Icon className="h-6 w-6" aria-hidden="true" />
        </div>
        <span className="text-sm font-medium text-muted-foreground">{title}</span>
      </div>
      <div className="flex items-baseline gap-1">
        <span className="text-3xl font-bold text-foreground">{value}</span>
        {unit && <span className="text-sm text-muted-foreground">{unit}</span>}
      </div>
      {trend && TrendIcon && (
        <div className={`flex items-center gap-1 mt-2 text-sm ${trendColors[trend]}`}>
          <TrendIcon className="h-4 w-4" aria-hidden="true" />
          {trendValue && <span>{trendValue}</span>}
        </div>
      )}
    </div>
  )
}

// ============================================================================
// Status Badge Component
// ============================================================================

function StatusBadge({ status }: { status?: string }) {
  const config = getStatusBadgeConfig(status)

  return (
    <Badge variant="outline" className={`${config.color} border`}>
      <span className="mr-1">{config.icon}</span>
      {config.label}
    </Badge>
  )
}

// ============================================================================
// Health Badge Component
// ============================================================================

function HealthBadge({ health }: { health?: string }) {
  const config = getHealthBadgeConfig(health)

  return (
    <Badge variant="outline" className={`${config.color} border`}>
      {config.label}
    </Badge>
  )
}

// ============================================================================
// Response Time Cell
// ============================================================================

function ResponseTimeCell({ ms }: { ms: number | null | undefined }) {
  const variant = getResponseTimeVariant(ms)
  const colorClass = {
    success: 'text-green-500',
    warning: 'text-yellow-500',
    error: 'text-red-500',
    default: 'text-muted-foreground',
  }[variant]

  return (
    <span className={colorClass}>
      {formatResponseTime(ms)}
    </span>
  )
}

// ============================================================================
// PerformanceMetrics Component
// ============================================================================

export function PerformanceMetrics({
  services,
  healthReport,
  className = '',
  'data-testid': testId = 'performance-metrics',
}: PerformanceMetricsProps) {
  // Compute aggregate metrics
  const activeCount = countActiveServices(services)
  const totalCount = services.length
  const activePorts = countActivePorts(services)
  const averageUptime = calculateAverageUptime(services)
  
  // Use health report for score if available, otherwise calculate from services
  const healthScore = healthReport
    ? Math.round((healthReport.summary.healthy / healthReport.summary.total) * 100) || 0
    : calculateHealthScore(services)

  const healthScoreVariant = getHealthScoreVariant(healthScore)

  // Get response times from health report
  const getServiceResponseTime = (serviceName: string): number | null => {
    if (!healthReport) return null
    const result = healthReport.services.find((s: HealthCheckResult) => s.serviceName === serviceName)
    // Convert nanoseconds to milliseconds
    return result?.responseTime ? result.responseTime / 1_000_000 : null
  }

  // Get health status from health report (preferred) or service data (fallback)
  const getServiceHealthStatus = (service: Service): Service['local'] extends infer T ? T extends { health: infer H } ? H : never : never => {
    if (healthReport) {
      const result = healthReport.services.find((s: HealthCheckResult) => s.serviceName === service.name)
      if (result) {
        return result.status as Service['local'] extends infer T ? T extends { health: infer H } ? H : never : never
      }
    }
    // Fallback to service's local health
    return service.local?.health ?? 'unknown'
  }

  return (
    <section
      aria-labelledby="performance-metrics-title"
      className={`space-y-6 ${className}`}
      data-testid={testId}
    >
      <h2 id="performance-metrics-title" className="sr-only">
        Performance Metrics Dashboard
      </h2>

      {/* Aggregate Metrics Section */}
      <section aria-labelledby="aggregate-metrics-title">
        <h3 id="aggregate-metrics-title" className="text-lg font-semibold text-foreground mb-4">
          Overview
        </h3>
        <div
          className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4"
          data-testid="metrics-grid"
        >
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
            title="Avg Uptime"
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

      {/* Service Metrics Table */}
      <section aria-labelledby="service-metrics-title">
        <h3 id="service-metrics-title" className="text-lg font-semibold text-foreground mb-4">
          Service Details
        </h3>
        {services.length === 0 ? (
          <div 
            className="text-center py-8 text-muted-foreground border border-border rounded-lg bg-card"
            data-testid="empty-state"
          >
            No services available
          </div>
        ) : (
          <div className="border border-border rounded-lg overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[200px]">Service</TableHead>
                  <TableHead className="w-[120px]">Status</TableHead>
                  <TableHead className="w-[120px]">Uptime</TableHead>
                  <TableHead className="w-[100px]">Port</TableHead>
                  <TableHead className="w-[120px]">Health</TableHead>
                  <TableHead className="w-[120px]">Response</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {services.map((service) => {
                  const uptime = getServiceUptime(service)
                  const responseTime = getServiceResponseTime(service.name)
                  const healthStatus = getServiceHealthStatus(service)
                  
                  return (
                    <TableRow key={service.name} data-testid={`service-row-${service.name}`}>
                      <TableCell className="font-medium">{service.name}</TableCell>
                      <TableCell>
                        <StatusBadge status={service.local?.status} />
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatDuration(uptime ?? 0)}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {service.local?.port ?? '-'}
                      </TableCell>
                      <TableCell>
                        <HealthBadge health={healthStatus} />
                      </TableCell>
                      <TableCell>
                        <ResponseTimeCell ms={responseTime} />
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        )}
      </section>
    </section>
  )
}
