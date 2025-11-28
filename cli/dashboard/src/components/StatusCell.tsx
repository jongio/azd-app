import { getStatusDisplay } from '@/lib/service-utils'
import { Activity, Globe, Plug, Cpu } from 'lucide-react'
import type { HealthCheckResult, HealthStatus } from '@/types'
import { cn } from '@/lib/utils'

interface StatusCellProps {
  status: 'starting' | 'ready' | 'running' | 'stopping' | 'stopped' | 'error' | 'not-running'
  health: HealthStatus
  healthCheckResult?: HealthCheckResult
}

/** Format response time from nanoseconds to human-readable */
function formatResponseTime(ns?: number): string {
  if (!ns) return ''
  const ms = ns / 1_000_000
  if (ms < 1) return '<1ms'
  if (ms < 1000) return `${Math.round(ms)}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

/** Get icon for health check type */
function getCheckTypeIcon(checkType?: string) {
  switch (checkType) {
    case 'http':
      return Globe
    case 'port':
      return Plug
    case 'process':
      return Cpu
    default:
      return Activity
  }
}

export function StatusCell({ status, health, healthCheckResult }: StatusCellProps) {
  const statusDisplay = getStatusDisplay(status, health)
  const CheckIcon = getCheckTypeIcon(healthCheckResult?.checkType)
  
  // Determine which animation to show based on health status
  const isUnhealthy = health === 'unhealthy' || status === 'error'
  const isDegraded = health === 'degraded'
  const isHealthy = (status === 'ready' || status === 'running') && health === 'healthy'
  
  // Build tooltip content
  const tooltipLines: string[] = []
  if (healthCheckResult) {
    if (healthCheckResult.checkType) {
      tooltipLines.push(`Check: ${healthCheckResult.checkType.toUpperCase()}`)
    }
    if (healthCheckResult.endpoint) {
      tooltipLines.push(`Endpoint: ${healthCheckResult.endpoint}`)
    }
    if (healthCheckResult.responseTime) {
      tooltipLines.push(`Response: ${formatResponseTime(healthCheckResult.responseTime)}`)
    }
    if (healthCheckResult.statusCode) {
      tooltipLines.push(`Status: ${healthCheckResult.statusCode}`)
    }
    if (healthCheckResult.error) {
      tooltipLines.push(`Error: ${healthCheckResult.error}`)
    }
  }
  const tooltipText = tooltipLines.length > 0 ? tooltipLines.join('\n') : undefined

  return (
    <div 
      className="flex items-center gap-2 group transition-all duration-200"
      title={tooltipText}
    >
      <div className={cn(
        "w-2 h-2 rounded-full",
        statusDisplay.color,
        isUnhealthy && "animate-status-flash",
        isDegraded && "animate-caution-pulse",
        isHealthy && "animate-heartbeat",
        !isUnhealthy && !isDegraded && !isHealthy && "transition-all duration-200"
      )}></div>
      <span className={cn(
        "font-medium transition-colors duration-200",
        statusDisplay.textColor
      )}>
        {statusDisplay.text}
      </span>
      {/* Health check type indicator */}
      {healthCheckResult && (
        <span title={`${healthCheckResult.checkType} check`}>
          <CheckIcon 
            className="w-3 h-3 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" 
          />
        </span>
      )}
    </div>
  )
}
