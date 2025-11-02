import { Activity, Server, CheckCircle, XCircle, Clock, AlertCircle, StopCircle } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import type { Service } from '@/types'

interface ServiceCardProps {
  service: Service
}

export function ServiceCard({ service }: ServiceCardProps) {
  const getStatusColor = (status: Service['status'], health: Service['health']) => {
    if ((status === 'ready' || status === 'running') && health === 'healthy') return 'success'
    if (status === 'starting') return 'warning'
    if (status === 'error' || health === 'unhealthy') return 'destructive'
    if (status === 'stopped' || status === 'stopping') return 'secondary'
    return 'secondary'
  }

  const getStatusIcon = (status: Service['status'], health: Service['health']) => {
    if ((status === 'ready' || status === 'running') && health === 'healthy') return <CheckCircle className="w-4 h-4" />
    if (status === 'starting') return <Clock className="w-4 h-4 animate-spin" />
    if (status === 'error' || health === 'unhealthy') return <XCircle className="w-4 h-4" />
    if (status === 'stopped') return <StopCircle className="w-4 h-4" />
    if (status === 'stopping') return <StopCircle className="w-4 h-4 animate-pulse" />
    return <AlertCircle className="w-4 h-4" />
  }

  const formatTime = (timeStr: string) => {
    const date = new Date(timeStr)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const seconds = Math.floor(diff / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)

    if (seconds < 60) return `${seconds}s ago`
    if (minutes < 60) return `${minutes}m ago`
    if (hours < 24) return `${hours}h ago`
    return `${days}d ago`
  }

  return (
    <div className="border rounded-lg p-4 bg-card text-card-foreground shadow-sm hover:shadow-md transition-shadow">
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2">
          <Server className="w-5 h-5 text-muted-foreground" />
          <h3 className="font-semibold text-lg">{service.name}</h3>
        </div>
        <Badge variant={getStatusColor(service.status, service.health)}>
          <span className="flex items-center gap-1.5">
            <div className="relative">
              {getStatusIcon(service.status, service.health)}
              {((service.status === 'ready' || service.status === 'running') && service.health === 'healthy') && (
                <>
                  <span className="absolute -top-0.5 -right-0.5 w-1.5 h-1.5 bg-green-400 rounded-full animate-ping"></span>
                  <span className="absolute -top-0.5 -right-0.5 w-1.5 h-1.5 bg-green-400 rounded-full"></span>
                </>
              )}
            </div>
            {service.status}
          </span>
        </Badge>
      </div>

      <div className="space-y-2 text-sm">
        <div className="flex items-center gap-2">
          <Activity className="w-4 h-4 text-muted-foreground" />
          <a 
            href={service.url} 
            target="_blank" 
            rel="noopener noreferrer"
            className="text-blue-500 hover:underline"
          >
            {service.url}
          </a>
        </div>

        <div className="grid grid-cols-2 gap-2 pt-2 border-t">
          <div>
            <span className="text-muted-foreground">Framework:</span>
            <p className="font-medium">{service.framework}</p>
          </div>
          <div>
            <span className="text-muted-foreground">Language:</span>
            <p className="font-medium">{service.language}</p>
          </div>
          <div>
            <span className="text-muted-foreground">Port:</span>
            <p className="font-medium">{service.port}</p>
          </div>
          <div>
            <span className="text-muted-foreground">Health:</span>
            <p className="font-medium capitalize flex items-center gap-1">
              {service.health === 'healthy' ? (
                <CheckCircle className="w-4 h-4 text-green-500" />
              ) : (
                <XCircle className="w-4 h-4 text-red-500" />
              )}
              {service.health}
            </p>
          </div>
        </div>

        <div className="pt-2 border-t text-xs text-muted-foreground">
          <div>Started: {formatTime(service.startTime)}</div>
          <div>Last checked: {formatTime(service.lastChecked)}</div>
          {service.error && (
            <div className="mt-2 text-red-500 font-medium">
              Error: {service.error}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
