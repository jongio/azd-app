import { Server, MoreVertical, FileText, ExternalLink } from 'lucide-react'
import { TableRow, TableCell } from '@/components/ui/table'
import { StatusCell } from '@/components/StatusCell'
import type { Service } from '@/types'

interface ServiceTableRowProps {
  service: Service
  onViewLogs?: (serviceName: string) => void
}

export function ServiceTableRow({ service, onViewLogs }: ServiceTableRowProps) {
  // Get status and health from local (with fallbacks)
  const status = service.local?.status || service.status || 'not-running'
  const health = service.local?.health || service.health || 'unknown'
  
  const formatStartTime = (timeStr?: string) => {
    if (!timeStr) return '-'
    const date = new Date(timeStr)
    return date.toLocaleTimeString('en-US', { 
      hour: '2-digit', 
      minute: '2-digit', 
      second: '2-digit' 
    })
  }

  const getStatusColor = (status: string, health: string) => {
    if ((status === 'ready' || status === 'running') && health === 'healthy') return 'text-success'
    if (status === 'starting') return 'text-warning'
    if (status === 'error' || health === 'unhealthy') return 'text-destructive'
    return 'text-muted-foreground'
  }

  const isHealthy = (status === 'ready' || status === 'running') && health === 'healthy'

  return (
    <TableRow>
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
        <StatusCell status={status} health={health} />
      </TableCell>

      {/* Start Time Column */}
      <TableCell className="text-muted-foreground">
        {formatStartTime(service.local?.startTime || service.startTime)}
      </TableCell>

      {/* Source Column */}
      <TableCell className="max-w-[250px]">
        <div className="truncate" title={service.project || service.framework || '-'}>
          <span className="text-sm text-foreground/90">
            {service.project || service.framework || '-'}
          </span>
        </div>
      </TableCell>

      {/* Local URL Column */}
      <TableCell className="max-w-[200px]">
        {service.local?.url ? (
          <a
            href={service.local.url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary hover:underline flex items-center gap-1 transition-colors"
            title={service.local.url}
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
          >
            <span className="truncate">{service.azure.url}</span>
            <ExternalLink className="w-3 h-3 shrink-0" />
          </a>
        ) : (
          <span className="text-muted-foreground">-</span>
        )}
      </TableCell>

      {/* Actions Column */}
      <TableCell className="text-right">
        <div className="flex items-center justify-end gap-2">
          <button
            onClick={() => onViewLogs?.(service.name)}
            className="p-2 rounded-lg hover:bg-white/10 transition-colors group"
            title="View Logs"
          >
            <FileText className="w-4 h-4 text-muted-foreground group-hover:text-primary transition-colors" />
          </button>
          <button
            className="p-2 rounded-lg hover:bg-white/10 transition-colors group"
            title="More Options"
          >
            <MoreVertical className="w-4 h-4 text-muted-foreground group-hover:text-primary transition-colors" />
          </button>
        </div>
      </TableCell>
    </TableRow>
  )
}
