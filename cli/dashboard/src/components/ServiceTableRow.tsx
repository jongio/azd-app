import { Server, FileText, ExternalLink } from 'lucide-react'
import { TableRow, TableCell } from '@/components/ui/table'
import { StatusCell } from '@/components/StatusCell'
import { ServiceActions } from '@/components/ServiceActions'
import { useServiceOperations } from '@/hooks/useServiceOperations'
import { formatStartTime, getEffectiveStatus } from '@/lib/service-utils'
import type { Service, HealthCheckResult, HealthStatus } from '@/types'

interface ServiceTableRowProps {
  service: Service
  onViewLogs?: (serviceName: string) => void
  onClick?: () => void
  healthStatus?: HealthCheckResult
}

export function ServiceTableRow({ service, onViewLogs, onClick, healthStatus }: ServiceTableRowProps) {
  // Get operation state for optimistic UI updates
  const { getOperationState } = useServiceOperations()
  const operationState = getOperationState(service.name)
  
  // Get effective status - pass operation state for optimistic updates
  const { status: effectiveStatus, health: effectiveHealth } = getEffectiveStatus(service, operationState)
  // Use real-time health from stream if available (but only when not in operation)
  const status = effectiveStatus as 'starting' | 'ready' | 'running' | 'stopping' | 'stopped' | 'error' | 'not-running' | 'restarting'
  const health = (operationState === 'idle' 
    ? (healthStatus?.status ?? effectiveHealth) 
    : effectiveHealth) as HealthStatus

  const getStatusColor = (status: string, health: string) => {
    if ((status === 'ready' || status === 'running') && health === 'healthy') return 'text-success'
    if (status === 'starting') return 'text-warning'
    if (status === 'error' || health === 'unhealthy') return 'text-destructive'
    return 'text-muted-foreground'
  }

  const isHealthy = (status === 'ready' || status === 'running') && health === 'healthy'

  return (
    <TableRow 
      className={onClick ? 'cursor-pointer' : undefined}
      onClick={onClick}
    >
      {/* Name Column */}
      <TableCell className="font-medium">
        <div className="flex items-center gap-3">
          <div className={`p-2 rounded-lg transition-all ${
            isHealthy 
              ? 'bg-success/10' 
              : 'bg-muted/10'
          }`}>
            <Server className={`w-4 h-4 ${isHealthy ? 'text-success' : 'text-muted-foreground'}`} />
          </div>
          <span className={`font-semibold ${getStatusColor(status, health)}`}>
            {service.name}
          </span>
        </div>
      </TableCell>

      {/* State Column */}
      <TableCell>
        <StatusCell 
          status={status} 
          health={health}
          healthCheckResult={healthStatus}
        />
      </TableCell>

      {/* Start Time Column */}
      <TableCell className="text-muted-foreground">
        {formatStartTime(service.local?.startTime ?? service.startTime)}
      </TableCell>

      {/* Source Column */}
      <TableCell className="max-w-[250px]">
        <div className="truncate" title={service.project ?? service.framework ?? '-'}>
          <span className="text-sm text-foreground/90">
            {service.project ?? service.framework ?? '-'}
          </span>
        </div>
      </TableCell>

      {/* Local URL Column */}
      <TableCell className="max-w-[200px]">
        {service.local?.url && !service.local.url.match(/:0\/?$/) ? (
          <a
            href={service.local.url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary hover:underline flex items-center gap-1 transition-colors"
            title={service.local.url}
            onClick={(e) => e.stopPropagation()}
          >
            <span className="truncate">{service.local.url}</span>
            <ExternalLink className="w-3 h-3 shrink-0" />
          </a>
        ) : (
          <span className="text-muted-foreground">-</span>
        )}
      </TableCell>

      {/* Azure URL Column */}
      <TableCell className="max-w-[200px]">
        {service.azure?.url ? (
          <a
            href={service.azure.url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-blue-400 hover:text-blue-300 hover:underline flex items-center gap-1 transition-colors"
            title={service.azure.url}
            onClick={(e) => e.stopPropagation()}
          >
            <span className="truncate">{service.azure.url}</span>
            <ExternalLink className="w-3 h-3 shrink-0" />
          </a>
        ) : (
          <span className="text-muted-foreground">-</span>
        )}
      </TableCell>

      {/* Actions Column */}
      <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-end gap-2">
          <ServiceActions service={service} variant="compact" />
          <button
            onClick={() => onViewLogs?.(service.name)}
            className="p-2 rounded-lg hover:bg-secondary transition-colors group"
            title="View Logs"
          >
            <FileText className="w-4 h-4 text-muted-foreground group-hover:text-primary transition-colors" />
          </button>
        </div>
      </TableCell>
    </TableRow>
  )
}
