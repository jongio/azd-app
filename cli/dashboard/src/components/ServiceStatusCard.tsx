import { CheckCircle2, AlertCircle, XCircle, Loader2 } from 'lucide-react'
import type { Service } from '@/types'

interface ServiceStatusCardProps {
  services: Service[]
  hasActiveErrors: boolean
  loading: boolean
  onClick: () => void
}

export function ServiceStatusCard({ services, hasActiveErrors, loading, onClick }: ServiceStatusCardProps) {
  // Calculate health status
  const healthyCounts = {
    healthy: 0,
    unhealthy: 0,
    unknown: 0,
    stopped: 0
  }

  services.forEach(service => {
    const status = service.local?.status || service.status
    const health = service.local?.health || service.health
    
    if (status === 'stopped' || status === 'not-running' || status === 'error') {
      healthyCounts.stopped++
    } else if (health === 'unhealthy') {
      healthyCounts.unhealthy++
    } else if (health === 'healthy') {
      healthyCounts.healthy++
    } else {
      healthyCounts.unknown++
    }
  })

  const totalServices = services.length
  const allHealthy = healthyCounts.healthy === totalServices && totalServices > 0
  const issueCount = healthyCounts.unhealthy + healthyCounts.stopped
  const hasServiceIssues = issueCount > 0

  // Determine status based on service health
  let statusColor = 'text-muted-foreground'
  let statusBg = 'bg-muted/50'
  let statusIcon = <Loader2 className="w-4 h-4 animate-spin" />
  let statusText = 'Loading...'
  let statusRing = ''

  if (loading) {
    statusColor = 'text-muted-foreground'
    statusBg = 'bg-muted/50'
    statusIcon = <Loader2 className="w-4 h-4 animate-spin" />
    statusText = 'Loading...'
  } else if (totalServices === 0) {
    statusColor = 'text-muted-foreground'
    statusBg = 'bg-muted/50'
    statusIcon = <AlertCircle className="w-4 h-4" />
    statusText = 'No services'
  } else if (hasServiceIssues) {
    // Red: Service-level issues (unhealthy/stopped)
    statusColor = 'text-red-600 dark:text-red-400'
    statusBg = 'bg-red-50 dark:bg-red-950/30'
    statusIcon = <XCircle className="w-4 h-4" />
    statusText = `${issueCount} issue${issueCount !== 1 ? 's' : ''}`
    statusRing = 'ring-2 ring-red-500/50'
  } else if (hasActiveErrors) {
    // Yellow/Orange: Log errors detected but services are healthy
    statusColor = 'text-orange-600 dark:text-orange-400'
    statusBg = 'bg-orange-50 dark:bg-orange-950/30'
    statusIcon = <AlertCircle className="w-4 h-4" />
    statusText = 'Log errors'
    statusRing = 'ring-2 ring-orange-500/50'
  } else if (allHealthy) {
    statusColor = 'text-green-600 dark:text-green-400'
    statusBg = 'bg-green-50 dark:bg-green-950/30'
    statusIcon = <CheckCircle2 className="w-4 h-4" />
    statusText = 'All healthy'
  } else {
    statusColor = 'text-yellow-600 dark:text-yellow-400'
    statusBg = 'bg-yellow-50 dark:bg-yellow-950/30'
    statusIcon = <AlertCircle className="w-4 h-4" />
    statusText = `${healthyCounts.healthy}/${totalServices} healthy`
  }

  return (
    <button
      onClick={onClick}
      className={`
        flex items-center gap-2 px-3 py-1.5 rounded-md transition-all
        ${statusBg} ${statusColor} hover:opacity-80 ${statusRing}
        cursor-pointer
      `}
      title="Click to view console logs"
    >
      {statusIcon}
      <div className="flex flex-col items-start">
        <span className="text-xs font-medium leading-tight">{statusText}</span>
        {totalServices > 0 && !loading && (
          <span className="text-[10px] opacity-70 leading-tight">
            {totalServices} service{totalServices !== 1 ? 's' : ''}
          </span>
        )}
      </div>
    </button>
  )
}
