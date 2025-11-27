import { XCircle, AlertTriangle, CheckCircle, Loader2 } from 'lucide-react'
import type { Service } from '@/types'

interface ServiceStatusCardProps {
  services: Service[]
  hasActiveErrors: boolean
  loading: boolean
  onClick: () => void
}

export function ServiceStatusCard({ services, hasActiveErrors, loading, onClick }: ServiceStatusCardProps) {
  // Calculate service status counts
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
    } else if (health === 'unknown' || status === 'starting' || status === 'stopping') {
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

  // Determine overall status ring color
  let statusRing = ''
  if (statusCounts.error > 0) {
    statusRing = 'ring-2 ring-red-500/50'
  } else if (statusCounts.warn > 0 || hasActiveErrors) {
    statusRing = 'ring-2 ring-orange-500/50'
  }

  return (
    <button
      onClick={onClick}
      className={`
        flex flex-col gap-1 px-3 py-1.5 rounded-md transition-all
        bg-muted/50 hover:bg-muted/70 ${statusRing}
        cursor-pointer min-w-[140px]
      `}
      title="Click to view console logs"
    >
      <span className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">
        Service Status
      </span>
      {loading ? (
        <div className="flex items-center justify-center py-1">
          <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />
        </div>
      ) : (
        <div className="flex items-center gap-3 text-xs">
          <div className="flex items-center gap-1" title="Error">
            <XCircle className="w-3.5 h-3.5 text-red-500" />
            <span className={statusCounts.error > 0 ? 'text-red-600 dark:text-red-400 font-medium' : 'text-muted-foreground'}>
              {statusCounts.error}
            </span>
          </div>
          <div className="flex items-center gap-1" title="Warning">
            <AlertTriangle className="w-3.5 h-3.5 text-orange-500" />
            <span className={statusCounts.warn > 0 ? 'text-orange-600 dark:text-orange-400 font-medium' : 'text-muted-foreground'}>
              {statusCounts.warn}
            </span>
          </div>
          <div className="flex items-center gap-1" title="Running">
            <CheckCircle className="w-3.5 h-3.5 text-green-500" />
            <span className={statusCounts.running > 0 ? 'text-green-600 dark:text-green-400 font-medium' : 'text-muted-foreground'}>
              {statusCounts.running}
            </span>
          </div>
        </div>
      )}
    </button>
  )
}
