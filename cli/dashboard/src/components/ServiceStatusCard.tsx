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
        flex flex-col items-center gap-2 px-4 py-2.5 rounded-lg transition-all
        bg-card border border-border/50 hover:border-border hover:bg-accent/50 
        shadow-sm hover:shadow ${statusRing}
        cursor-pointer min-w-[160px]
      `}
      title="Click to view console logs"
    >
      <span className="text-[11px] font-semibold text-foreground/80 uppercase tracking-wider text-center w-full">
        Service Status
      </span>
      {loading ? (
        <div className="flex items-center justify-center py-1.5">
          <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
        </div>
      ) : (
        <div className="flex items-center justify-center gap-4 text-sm w-full">
          <div className="flex flex-col items-center gap-0.5" title="Error">
            <XCircle className={`w-5 h-5 ${statusCounts.error > 0 ? 'text-red-500' : 'text-muted-foreground/40'}`} />
            <span className={`text-xs font-semibold ${statusCounts.error > 0 ? 'text-red-600 dark:text-red-400' : 'text-muted-foreground/60'}`}>
              {statusCounts.error}
            </span>
          </div>
          <div className="flex flex-col items-center gap-0.5" title="Warning">
            <AlertTriangle className={`w-5 h-5 ${statusCounts.warn > 0 ? 'text-orange-500' : 'text-muted-foreground/40'}`} />
            <span className={`text-xs font-semibold ${statusCounts.warn > 0 ? 'text-orange-600 dark:text-orange-400' : 'text-muted-foreground/60'}`}>
              {statusCounts.warn}
            </span>
          </div>
          <div className="flex flex-col items-center gap-0.5" title="Running">
            <CheckCircle className={`w-5 h-5 ${statusCounts.running > 0 ? 'text-green-500' : 'text-muted-foreground/40'}`} />
            <span className={`text-xs font-semibold ${statusCounts.running > 0 ? 'text-green-600 dark:text-green-400' : 'text-muted-foreground/60'}`}>
              {statusCounts.running}
            </span>
          </div>
        </div>
      )}
    </button>
  )
}
