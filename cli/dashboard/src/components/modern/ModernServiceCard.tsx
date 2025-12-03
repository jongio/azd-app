/**
 * ModernServiceCard - Grid view service card with modern visual treatments
 * Follows design spec: cli/dashboard/design/modern/components/service-card.md
 */
import { 
  Server, 
  ExternalLink, 
  Zap,
  Clock,
  Globe,
  XCircle,
  AlertTriangle,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { ModernStatusBadge, ModernStatusDot, type EffectiveStatus } from './ModernStatusIndicator'
import { ServiceActions } from '@/components/ServiceActions'
import { useServiceOperations } from '@/hooks/useServiceOperations'
import type { Service, HealthCheckResult } from '@/types'
import { 
  formatResponseTime, 
  formatUptime,
  getCheckTypeDisplay,
} from '@/lib/service-utils'

// =============================================================================
// Types
// =============================================================================

export interface ModernServiceCardProps {
  /** Service data */
  service: Service
  /** Real-time health status */
  healthStatus?: HealthCheckResult
  /** Click handler for card (opens detail panel) */
  onClick?: () => void
  /** Additional class names */
  className?: string
}

// =============================================================================
// Helper Functions
// =============================================================================

function getEffectiveStatusFromService(
  service: Service, 
  healthStatus?: HealthCheckResult,
  operationState?: string
): EffectiveStatus {
  // If operation is in progress, show that status
  if (operationState && operationState !== 'idle') {
    return operationState as EffectiveStatus
  }
  
  const processStatus = service.local?.status ?? service.status ?? 'not-running'
  
  // Process status takes priority
  if (processStatus === 'stopped') return 'stopped'
  if (processStatus === 'stopping') return 'stopping'
  if (processStatus === 'starting') return 'starting'
  if (processStatus === 'error') return 'error'
  if (processStatus === 'not-running') return 'not-running'
  
  // Then check health
  const health = healthStatus?.status ?? service.local?.health ?? service.health ?? 'unknown'
  if (health === 'healthy') return 'healthy'
  if (health === 'degraded') return 'degraded'
  if (health === 'unhealthy') return 'unhealthy'
  if (health === 'starting') return 'starting'
  
  return 'unknown'
}

// =============================================================================
// ModernServiceCard Component
// =============================================================================

export function ModernServiceCard({ 
  service, 
  healthStatus, 
  onClick,
  className,
}: ModernServiceCardProps) {
  // Get operation state for optimistic UI updates
  const { getOperationState } = useServiceOperations()
  const operationState = getOperationState(service.name)

  const effectiveStatus = getEffectiveStatusFromService(service, healthStatus, operationState)
  const isHealthy = effectiveStatus === 'healthy' || effectiveStatus === 'running'
  const hasError = !!service.error || healthStatus?.error

  // Health details from real-time data or service data
  const healthDetails = healthStatus ? {
    checkType: healthStatus.checkType,
    endpoint: healthStatus.endpoint,
    responseTime: healthStatus.responseTime ? healthStatus.responseTime / 1_000_000 : undefined,
    statusCode: healthStatus.statusCode,
    uptime: healthStatus.uptime ? healthStatus.uptime / 1_000_000_000 : undefined,
    lastError: healthStatus.error,
  } : service.local?.healthDetails

  // Build URLs
  const localUrl = service.local?.url && !service.local.url.match(/:0\/?$/) ? service.local.url : null
  const azureUrl = service.azure?.url

  return (
    <article
      className={cn(
        'group relative flex flex-col gap-4 p-5 rounded-2xl',
        'bg-white dark:bg-slate-800',
        'border border-slate-200 dark:border-slate-700',
        'shadow-sm hover:shadow-lg',
        'transition-all duration-200 ease-out',
        'hover:border-cyan-300 dark:hover:border-cyan-600',
        'hover:-translate-y-0.5',
        'cursor-pointer overflow-hidden',
        className
      )}
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={onClick ? (e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onClick()
        }
      } : undefined}
      aria-label={`${service.name} service - ${effectiveStatus}`}
    >
      {/* Gradient overlay on hover */}
      <div className="absolute inset-0 bg-gradient-to-br from-cyan-500/5 via-transparent to-cyan-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-300 pointer-events-none" />

      {/* Header */}
      <header className="relative flex items-start gap-3">
        {/* Service Icon */}
        <div className={cn(
          'w-11 h-11 rounded-xl flex items-center justify-center flex-shrink-0 transition-transform duration-200 group-hover:scale-105',
          isHealthy 
            ? 'bg-gradient-to-br from-emerald-100 to-emerald-50 dark:from-emerald-500/20 dark:to-emerald-500/10'
            : effectiveStatus === 'error' || effectiveStatus === 'unhealthy'
            ? 'bg-gradient-to-br from-rose-100 to-rose-50 dark:from-rose-500/20 dark:to-rose-500/10'
            : 'bg-slate-100 dark:bg-slate-700'
        )}>
          <Server className={cn(
            'w-5 h-5',
            isHealthy 
              ? 'text-emerald-600 dark:text-emerald-400'
              : effectiveStatus === 'error' || effectiveStatus === 'unhealthy'
              ? 'text-rose-600 dark:text-rose-400'
              : 'text-slate-500 dark:text-slate-400'
          )} />
        </div>

        {/* Service Info */}
        <div className="flex-1 min-w-0">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100 truncate group-hover:text-cyan-600 dark:group-hover:text-cyan-400 transition-colors">
            {service.name}
          </h3>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
            {service.language || 'Service'}
            {service.framework && (
              <>
                <span className="mx-1 text-slate-300 dark:text-slate-600">•</span>
                {service.framework}
              </>
            )}
          </p>
        </div>

        {/* Status Badge */}
        <ModernStatusBadge status={effectiveStatus} />
      </header>

      {/* Error Banner */}
      {hasError && (
        <div className="relative flex items-start gap-2.5 p-3 rounded-xl bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/30">
          <XCircle className="w-4 h-4 text-rose-600 dark:text-rose-400 flex-shrink-0 mt-0.5" />
          <div className="flex-1 min-w-0">
            <p className="text-xs font-medium text-rose-700 dark:text-rose-300">Error Detected</p>
            <p className="text-xs text-rose-600 dark:text-rose-400 mt-0.5 line-clamp-2">
              {service.error || healthDetails?.lastError}
            </p>
          </div>
        </div>
      )}

      {/* Degraded Warning Banner */}
      {!hasError && effectiveStatus === 'degraded' && (
        <div className="relative flex items-start gap-2.5 p-3 rounded-xl bg-amber-50 dark:bg-amber-500/10 border border-amber-200 dark:border-amber-500/30">
          <AlertTriangle className="w-4 h-4 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
          <div className="flex-1 min-w-0">
            <p className="text-xs font-medium text-amber-700 dark:text-amber-300">Performance Degraded</p>
            <p className="text-xs text-amber-600 dark:text-amber-400 mt-0.5">
              Service is responding slowly
            </p>
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="relative" onClick={(e) => e.stopPropagation()}>
        <ServiceActions service={service} variant="default" />
      </div>

      {/* URL Link */}
      {localUrl && (
        <a
          href={localUrl}
          target="_blank"
          rel="noopener noreferrer"
          onClick={(e) => e.stopPropagation()}
          className="relative flex items-center gap-2 p-2.5 rounded-xl bg-slate-50 dark:bg-slate-700/50 border border-slate-200 dark:border-slate-600 hover:border-cyan-300 dark:hover:border-cyan-600 transition-colors group/url"
        >
          <ExternalLink className="w-3.5 h-3.5 text-cyan-600 dark:text-cyan-400" />
          <span className="flex-1 text-xs font-mono text-slate-600 dark:text-slate-300 truncate group-hover/url:text-cyan-600 dark:group-hover/url:text-cyan-400 transition-colors">
            {localUrl}
          </span>
          <ExternalLink className="w-3 h-3 text-slate-400 dark:text-slate-500 opacity-0 group-hover/url:opacity-100 transform group-hover/url:translate-x-0.5 group-hover/url:-translate-y-0.5 transition-all" />
        </a>
      )}

      {/* Azure URL (if deployed) */}
      {azureUrl && (
        <a
          href={azureUrl}
          target="_blank"
          rel="noopener noreferrer"
          onClick={(e) => e.stopPropagation()}
          className="relative flex items-center gap-2 p-2.5 rounded-xl bg-cyan-50 dark:bg-cyan-500/10 border border-cyan-200 dark:border-cyan-500/30 hover:border-cyan-400 dark:hover:border-cyan-500 transition-colors group/azure"
        >
          <Globe className="w-3.5 h-3.5 text-cyan-600 dark:text-cyan-400" />
          <div className="flex-1 min-w-0">
            <span className="text-[10px] text-cyan-600/70 dark:text-cyan-400/70 block">Azure</span>
            <span className="text-xs font-mono text-cyan-700 dark:text-cyan-300 truncate block">
              {azureUrl}
            </span>
          </div>
          <ExternalLink className="w-3 h-3 text-cyan-500 dark:text-cyan-500 opacity-0 group-hover/azure:opacity-100 transform group-hover/azure:translate-x-0.5 group-hover/azure:-translate-y-0.5 transition-all" />
        </a>
      )}

      {/* Metrics Row */}
      <div className="relative flex items-center justify-between py-3 px-4 rounded-xl bg-gradient-to-r from-cyan-50 to-slate-50 dark:from-cyan-500/5 dark:to-slate-500/5 border border-slate-200 dark:border-slate-700">
        {/* Port */}
        {service.local?.port && service.local.port > 0 && (
          <div className="flex items-center gap-2">
            <ModernStatusDot status={effectiveStatus} size="sm" />
            <span className="text-xs text-slate-500 dark:text-slate-400">Port</span>
            <span className="font-mono font-semibold text-sm text-cyan-600 dark:text-cyan-400">
              {service.local.port}
            </span>
          </div>
        )}
        
        {/* Health Badge */}
        <div className="flex items-center gap-1.5">
          <ModernStatusDot status={effectiveStatus} size="md" />
          <span className={cn(
            'text-sm font-medium capitalize',
            effectiveStatus === 'healthy' && 'text-emerald-600 dark:text-emerald-400',
            effectiveStatus === 'degraded' && 'text-amber-600 dark:text-amber-400',
            (effectiveStatus === 'unhealthy' || effectiveStatus === 'error') && 'text-rose-600 dark:text-rose-400',
            !['healthy', 'degraded', 'unhealthy', 'error'].includes(effectiveStatus) && 'text-slate-500 dark:text-slate-400'
          )}>
            {effectiveStatus}
          </span>
        </div>
      </div>

      {/* Health Details */}
      {healthDetails && (
        <div className="relative grid grid-cols-3 gap-2">
          {/* Response Time */}
          <div className="bg-slate-50 dark:bg-slate-700/30 p-2 rounded-lg border border-slate-200 dark:border-slate-700">
            <div className="flex items-center gap-1 mb-0.5">
              <Zap className="w-3 h-3 text-amber-500" />
              <span className="text-[10px] text-slate-500 dark:text-slate-400">Response</span>
            </div>
            <p className="font-mono font-semibold text-xs text-slate-800 dark:text-slate-200">
              {formatResponseTime(healthDetails.responseTime ? healthDetails.responseTime * 1_000_000 : undefined)}
            </p>
          </div>
          {/* Check Type */}
          <div className="bg-slate-50 dark:bg-slate-700/30 p-2 rounded-lg border border-slate-200 dark:border-slate-700">
            <div className="flex items-center gap-1 mb-0.5">
              <Globe className="w-3 h-3 text-sky-500" />
              <span className="text-[10px] text-slate-500 dark:text-slate-400">Check</span>
            </div>
            <p className="font-semibold text-xs text-slate-800 dark:text-slate-200">
              {getCheckTypeDisplay(healthDetails.checkType)}
            </p>
          </div>
          {/* Uptime */}
          <div className="bg-slate-50 dark:bg-slate-700/30 p-2 rounded-lg border border-slate-200 dark:border-slate-700">
            <div className="flex items-center gap-1 mb-0.5">
              <Clock className="w-3 h-3 text-emerald-500" />
              <span className="text-[10px] text-slate-500 dark:text-slate-400">Uptime</span>
            </div>
            <p className="font-mono font-semibold text-xs text-slate-800 dark:text-slate-200">
              {formatUptime(healthDetails.uptime ? healthDetails.uptime * 1_000_000_000 : undefined)}
            </p>
          </div>
        </div>
      )}
    </article>
  )
}
