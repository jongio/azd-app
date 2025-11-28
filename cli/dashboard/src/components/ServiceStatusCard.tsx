import { XCircle, AlertTriangle, CheckCircle, Loader2, Activity } from 'lucide-react'
import type { Service, HealthSummary } from '@/types'

interface ServiceStatusCardProps {
  services: Service[]
  hasActiveErrors: boolean
  loading: boolean
  onClick: () => void
  healthSummary?: HealthSummary | null
  healthConnected?: boolean
}

export function ServiceStatusCard({ 
  services, 
  hasActiveErrors, 
  loading, 
  onClick,
  healthSummary,
  healthConnected 
}: ServiceStatusCardProps) {
  // Calculate service status counts - prefer health summary if available
  const statusCounts = healthSummary 
    ? {
        error: healthSummary.unhealthy,
        warn: healthSummary.degraded + healthSummary.unknown,
        running: healthSummary.healthy
      }
    : calculateStatusCounts(services, hasActiveErrors)

  return (
    <button
      onClick={onClick}
      className="flex items-center gap-3 px-3 py-1.5 rounded-md transition-all hover:bg-secondary/80 cursor-pointer group"
      title="Click to view console logs"
    >
      {loading ? (
        <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />
      ) : (
        <div className="flex items-center gap-3">
          {/* Health monitoring indicator */}
          {healthConnected !== undefined && (
            <div 
              className="flex items-center gap-1" 
              title={healthConnected ? "Health monitoring active" : "Health monitoring disconnected"}
            >
              <Activity className={`w-3 h-3 ${
                healthConnected 
                  ? 'text-green-400 animate-heartbeat' 
                  : 'text-muted-foreground/30'
              }`} />
            </div>
          )}

          {/* Error indicator */}
          <div className="flex items-center gap-1.5" title="Errors/Unhealthy">
            <div className={`w-5 h-5 rounded-full flex items-center justify-center transition-colors ${
              statusCounts.error > 0 
                ? 'bg-red-500/15 animate-status-flash' 
                : 'bg-transparent'
            }`}>
              <XCircle className={`w-4 h-4 ${
                statusCounts.error > 0 
                  ? 'text-red-500' 
                  : 'text-muted-foreground/30 group-hover:text-muted-foreground/50'
              }`} />
            </div>
            <span className={`text-sm tabular-nums ${
              statusCounts.error > 0 
                ? 'text-red-500 font-medium' 
                : 'text-muted-foreground/50'
            }`}>
              {statusCounts.error}
            </span>
          </div>

          {/* Warning/Degraded indicator */}
          <div className="flex items-center gap-1.5" title="Warnings/Degraded">
            <div className={`w-5 h-5 rounded-full flex items-center justify-center transition-colors ${
              statusCounts.warn > 0 
                ? 'bg-amber-500/15 animate-caution-pulse' 
                : 'bg-transparent'
            }`}>
              <AlertTriangle className={`w-4 h-4 ${
                statusCounts.warn > 0 
                  ? 'text-amber-500' 
                  : 'text-muted-foreground/30 group-hover:text-muted-foreground/50'
              }`} />
            </div>
            <span className={`text-sm tabular-nums ${
              statusCounts.warn > 0 
                ? 'text-amber-500 font-medium' 
                : 'text-muted-foreground/50'
            }`}>
              {statusCounts.warn}
            </span>
          </div>

          {/* Running/Healthy indicator */}
          <div className="flex items-center gap-1.5" title="Running/Healthy">
            <div className={`w-5 h-5 rounded-full flex items-center justify-center transition-colors ${
              statusCounts.running > 0 
                ? 'bg-green-500/15 animate-heartbeat' 
                : 'bg-transparent'
            }`}>
              <CheckCircle className={`w-4 h-4 ${
                statusCounts.running > 0 
                  ? 'text-green-500' 
                  : 'text-muted-foreground/30 group-hover:text-muted-foreground/50'
              }`} />
            </div>
            <span className={`text-sm tabular-nums ${
              statusCounts.running > 0 
                ? 'text-green-500 font-medium' 
                : 'text-muted-foreground/50'
            }`}>
              {statusCounts.running}
            </span>
          </div>
        </div>
      )}
    </button>
  )
}

/** Calculate status counts from services when health summary is not available */
function calculateStatusCounts(services: Service[], hasActiveErrors: boolean): { error: number; warn: number; running: number } {
  const statusCounts = {
    error: 0,
    warn: 0,
    running: 0
  }

  services.forEach(service => {
    const status = service.local?.status || service.status
    const health = service.local?.health || service.health
    
    if (status === 'stopped' || status === 'not-running' || status === 'error' || health === 'unhealthy') {
      statusCounts.error++
    } else if (health === 'degraded' || health === 'unknown' || status === 'starting' || status === 'stopping') {
      statusCounts.warn++
    } else {
      // healthy/running services
      statusCounts.running++
    }
  })

  // If there are active log errors but no service-level errors, show in warn
  if (hasActiveErrors && statusCounts.error === 0) {
    // Move running to warn to indicate log errors exist
    if (statusCounts.running > 0) {
      statusCounts.warn += statusCounts.running
      statusCounts.running = 0
    }
  }

  return statusCounts
}
