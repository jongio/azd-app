import { Activity, Server, CheckCircle, XCircle, ExternalLink, Code, Layers, AlertTriangle, Clock, Zap, Globe } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { ServiceActions } from '@/components/ServiceActions'
import { useServiceOperations } from '@/hooks/useServiceOperations'
import type { Service, HealthCheckResult } from '@/types'
import { getEffectiveStatus, getStatusDisplay, isServiceHealthy, formatRelativeTime, formatResponseTime, formatUptime, getCheckTypeDisplay } from '@/lib/service-utils'

interface ServiceCardProps {
  service: Service
  healthStatus?: HealthCheckResult
  onClick?: () => void
}

export function ServiceCard({ service, healthStatus, onClick }: ServiceCardProps) {
  // Get operation state for optimistic UI updates
  const { getOperationState } = useServiceOperations()
  const operationState = getOperationState(service.name)
  
  // Use real-time health from health stream if available
  // Pass operation state to getEffectiveStatus for optimistic updates
  const { status, health: baseHealth } = getEffectiveStatus(service, operationState)
  const health = healthStatus?.status || baseHealth
  const statusDisplay = getStatusDisplay(status, health)
  const healthy = isServiceHealthy(status, health)
  const Icon = statusDisplay.icon
  // Prefer health details from healthStatus (real-time) over service.local.healthDetails
  const healthDetails = healthStatus ? {
    checkType: healthStatus.checkType,
    endpoint: healthStatus.endpoint,
    responseTime: healthStatus.responseTime ? healthStatus.responseTime / 1_000_000 : undefined, // Convert ns to ms
    statusCode: healthStatus.statusCode,
    uptime: healthStatus.uptime ? healthStatus.uptime / 1_000_000_000 : undefined, // Convert ns to s
    lastError: healthStatus.error,
  } : service.local?.healthDetails

  return (
    <div 
      className="group rounded-2xl p-6 transition-all-smooth hover:scale-[1.02] hover:border-primary/50 relative overflow-hidden bg-card border border-border shadow-sm hover:shadow-md cursor-pointer"
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={onClick ? (e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onClick(); } } : undefined}
    >
      {/* Animated gradient background on hover */}
      <div className="absolute inset-0 bg-linear-to-br from-primary/5 via-transparent to-accent/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
      
      {/* Content */}
      <div className="relative z-10">
        {/* Header */}
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className={`p-2.5 rounded-xl transition-all-smooth ${
              healthy 
                ? 'bg-linear-to-br from-green-500/20 to-green-500/10 group-hover:scale-110' 
                : 'bg-linear-to-br from-muted/20 to-muted/10'
            }`}>
              <Server className={`w-5 h-5 ${healthy ? 'text-green-500' : 'text-muted-foreground'}`} />
            </div>
            <div>
              <h3 className="font-semibold text-xl text-foreground group-hover:text-primary transition-colors">
                {service.name}
              </h3>
              <p className="text-xs text-muted-foreground mt-0.5">Service Instance</p>
            </div>
          </div>
          
          <Badge 
            variant={statusDisplay.badgeVariant}
            className="transition-all-smooth group-hover:scale-105"
          >
            <span className="flex items-center gap-1.5">
              <div className="relative">
                <Icon className={status === 'starting' || status === 'restarting' ? 'w-4 h-4 animate-spin' : status === 'stopping' ? 'w-4 h-4 animate-pulse' : 'w-4 h-4'} />
                {healthy && (
                  <>
                    <span className="absolute -top-0.5 -right-0.5 w-1.5 h-1.5 bg-green-500 rounded-full animate-ping"></span>
                    <span className="absolute -top-0.5 -right-0.5 w-1.5 h-1.5 bg-green-500 rounded-full"></span>
                  </>
                )}
              </div>
              {statusDisplay.text}
            </span>
          </Badge>
        </div>

        {/* Error/Warning Banner - Shown prominently after header */}
        {(service.error || healthDetails?.lastError) && (
          <div className="mb-4 p-3 rounded-xl bg-destructive/15 border border-destructive/50">
            <div className="flex items-start gap-3">
              <XCircle className="w-5 h-5 text-destructive shrink-0 mt-0.5" />
              <div className="flex-1 min-w-0">
                <p className="font-semibold text-sm text-destructive">Error Detected</p>
                <p className="text-xs text-destructive/80 mt-1">
                  {service.error || healthDetails?.lastError}
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Degraded Warning Banner */}
        {!service.error && !healthDetails?.lastError && health === 'degraded' && (
          <div className="mb-4 p-3 rounded-xl bg-amber-500/15 border border-amber-500/50">
            <div className="flex items-start gap-3">
              <AlertTriangle className="w-5 h-5 text-amber-500 shrink-0 mt-0.5" />
              <div className="flex-1 min-w-0">
                <p className="font-semibold text-sm text-amber-600 dark:text-amber-400">Performance Degraded</p>
                <p className="text-xs text-amber-600/80 dark:text-amber-400/80 mt-1">
                  Service is responding slowly or intermittently
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Action Buttons Row */}
        <div className="mb-4" onClick={(e) => e.stopPropagation()}>
          <ServiceActions service={service} variant="default" />
        </div>

        {/* Local URL Link (if available and not port 0) */}
        {service.local?.url && !service.local.url.match(/:0\/?$/) && (
          <a 
            href={service.local.url} 
            target="_blank" 
            rel="noopener noreferrer"
            className="flex items-center gap-2 mb-4 p-3 rounded-xl bg-muted/50 border border-border hover:border-primary/50 transition-all-smooth group/link"
            onClick={(e) => e.stopPropagation()}
          >
            <Activity className="w-4 h-4 text-primary" />
            <span className="text-sm text-foreground/90 group-hover/link:text-primary transition-colors flex-1 truncate">
              {service.local.url}
            </span>
            <ExternalLink className="w-4 h-4 text-muted-foreground group-hover/link:text-primary transition-colors" />
          </a>
        )}

        {/* Azure URL Link (if available) */}
        {service.azure?.url && (
          <a 
            href={service.azure.url} 
            target="_blank" 
            rel="noopener noreferrer"
            className="flex items-center gap-2 mb-4 p-3 rounded-xl border border-primary/30 hover:border-primary/50 transition-all-smooth group/link bg-primary/5 hover:bg-primary/10"
            onClick={(e) => e.stopPropagation()}
          >
            <Activity className="w-4 h-4 text-primary" />
            <div className="flex-1 truncate">
              <div className="text-xs text-muted-foreground mb-0.5">Azure URL</div>
              <span className="text-sm text-foreground group-hover/link:text-primary transition-colors truncate block">
                {service.azure.url}
              </span>
            </div>
            <ExternalLink className="w-4 h-4 text-primary group-hover/link:text-primary/80 transition-colors" />
          </a>
        )}

        {/* Tech Stack */}
        <div className="grid grid-cols-2 gap-3 mb-4">
          <div className="bg-muted/50 p-3 rounded-xl border border-border">
            <div className="flex items-center gap-2 mb-1">
              <Code className="w-3.5 h-3.5 text-accent" />
              <span className="text-xs text-muted-foreground">Framework</span>
            </div>
            <p className="font-semibold text-sm text-foreground">{service.framework}</p>
          </div>
          <div className="bg-muted/50 p-3 rounded-xl border border-border">
            <div className="flex items-center gap-2 mb-1">
              <Layers className="w-3.5 h-3.5 text-secondary" />
              <span className="text-xs text-muted-foreground">Language</span>
            </div>
            <p className="font-semibold text-sm text-foreground">{service.language}</p>
          </div>
        </div>

        {/* Metrics Row */}
        <div className="flex items-center justify-between py-3 px-4 rounded-xl bg-linear-to-r from-primary/5 to-accent/5 border border-border mb-4">
          {service.local?.port && (
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-primary animate-pulse"></div>
              <span className="text-xs text-muted-foreground">Port</span>
              <span className="font-mono font-semibold text-sm text-primary">{service.local.port}</span>
            </div>
          )}
          <div className="flex items-center gap-2">
            {health === 'healthy' ? (
              <CheckCircle className="w-4 h-4 text-green-500" />
            ) : health === 'degraded' ? (
              <AlertTriangle className="w-4 h-4 text-amber-500" />
            ) : health === 'unhealthy' ? (
              <XCircle className="w-4 h-4 text-red-500" />
            ) : (
              <XCircle className="w-4 h-4 text-gray-400" />
            )}
            <span className={`text-sm font-medium ${
              health === 'healthy' ? 'text-green-500' 
              : health === 'degraded' ? 'text-amber-500' 
              : health === 'unhealthy' ? 'text-red-500'
              : 'text-gray-400'
            }`}>
              {health}
            </span>
          </div>
        </div>

        {/* Health Details (when available) */}
        {healthDetails && (
          <div className="grid grid-cols-3 gap-2 mb-4">
            {/* Response Time */}
            <div className="bg-muted/50 p-2 rounded-lg border border-border">
              <div className="flex items-center gap-1 mb-0.5">
                <Zap className="w-3 h-3 text-yellow-400" />
                <span className="text-[10px] text-muted-foreground">Response</span>
              </div>
              <p className="font-mono font-semibold text-xs text-foreground">
                {formatResponseTime(healthDetails.responseTime ? healthDetails.responseTime * 1_000_000 : undefined)}
              </p>
            </div>
            {/* Check Type */}
            <div className="bg-muted/50 p-2 rounded-lg border border-border">
              <div className="flex items-center gap-1 mb-0.5">
                <Globe className="w-3 h-3 text-blue-400" />
                <span className="text-[10px] text-muted-foreground">Check</span>
              </div>
              <p className="font-semibold text-xs text-foreground">
                {getCheckTypeDisplay(healthDetails.checkType)}
              </p>
            </div>
            {/* Uptime */}
            <div className="bg-muted/50 p-2 rounded-lg border border-border">
              <div className="flex items-center gap-1 mb-0.5">
                <Clock className="w-3 h-3 text-green-400" />
                <span className="text-[10px] text-muted-foreground">Uptime</span>
              </div>
              <p className="font-mono font-semibold text-xs text-foreground">
                {formatUptime(healthDetails.uptime ? healthDetails.uptime * 1_000_000_000 : undefined)}
              </p>
            </div>
          </div>
        )}

        {/* Health Endpoint (when available) */}
        {healthDetails?.endpoint && (
          <div className="mb-4 px-3 py-2 rounded-lg bg-muted/50 border border-border">
            <div className="flex items-center gap-2">
              <Activity className="w-3 h-3 text-muted-foreground" />
              <span className="text-[10px] text-muted-foreground">Health Endpoint:</span>
              <span className="text-xs font-mono text-foreground/80 truncate flex-1">
                {healthDetails.endpoint}
              </span>
              {healthDetails.statusCode && (
                <Badge variant={healthDetails.statusCode < 400 ? 'success' : 'destructive'} className="text-[10px] px-1.5 py-0">
                  {healthDetails.statusCode}
                </Badge>
              )}
            </div>
          </div>
        )}

        {/* Footer */}
        {(service.local?.startTime || service.local?.lastChecked || service.startTime || service.lastChecked) && (
          <div className="pt-4 border-t border-border space-y-1.5 text-xs text-muted-foreground">
            {(service.local?.startTime || service.startTime) && (
              <div className="flex items-center justify-between">
                <span>Started</span>
                <span className="font-medium">{formatRelativeTime(service.local?.startTime || service.startTime)}</span>
              </div>
            )}
            {(service.local?.lastChecked || service.lastChecked) && (
              <div className="flex items-center justify-between">
                <span>Last checked</span>
                <span className="font-medium">{formatRelativeTime(service.local?.lastChecked || service.lastChecked)}</span>
              </div>
            )}
          </div>
        )}

      </div>
    </div>
  )
}
