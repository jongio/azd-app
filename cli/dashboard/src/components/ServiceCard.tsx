import { Activity, Server, CheckCircle, XCircle, ExternalLink, Code, Layers, AlertCircle } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import type { Service } from '@/types'
import { getEffectiveStatus, getStatusDisplay, isServiceHealthy, formatRelativeTime } from '@/lib/service-utils'

interface ServiceCardProps {
  service: Service
}

export function ServiceCard({ service }: ServiceCardProps) {
  const { status, health } = getEffectiveStatus(service)
  const statusDisplay = getStatusDisplay(status, health)
  const healthy = isServiceHealthy(status, health)
  const Icon = statusDisplay.icon

  return (
    <div className="group glass rounded-2xl p-6 transition-all-smooth hover:scale-[1.02] hover:border-primary/50 relative overflow-hidden">
      {/* Animated gradient background on hover */}
      <div className="absolute inset-0 bg-linear-to-br from-primary/5 via-transparent to-accent/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
      
      {/* Content */}
      <div className="relative z-10">
        {/* Header */}
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className={`p-2.5 rounded-xl transition-all-smooth ${
              healthy 
                ? 'bg-linear-to-br from-success/20 to-success/10 group-hover:scale-110' 
                : 'bg-linear-to-br from-muted/20 to-muted/10'
            }`}>
              <Server className={`w-5 h-5 ${healthy ? 'text-success' : 'text-muted-foreground'}`} />
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
                <Icon className={status === 'starting' ? 'w-4 h-4 animate-spin' : status === 'stopping' ? 'w-4 h-4 animate-pulse' : 'w-4 h-4'} />
                {healthy && (
                  <>
                    <span className="absolute -top-0.5 -right-0.5 w-1.5 h-1.5 bg-success rounded-full animate-ping"></span>
                    <span className="absolute -top-0.5 -right-0.5 w-1.5 h-1.5 bg-success rounded-full"></span>
                  </>
                )}
              </div>
              {statusDisplay.text}
            </span>
          </Badge>
        </div>

        {/* Local URL Link (if available) */}
        {service.local?.url && (
          <a 
            href={service.local.url} 
            target="_blank" 
            rel="noopener noreferrer"
            className="flex items-center gap-2 mb-4 p-3 rounded-xl glass border border-white/5 hover:border-primary/50 transition-all-smooth group/link"
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
            className="flex items-center gap-2 mb-4 p-3 rounded-xl glass border border-blue-500/20 hover:border-blue-500/50 transition-all-smooth group/link bg-blue-500/5"
          >
            <Activity className="w-4 h-4 text-blue-400" />
            <div className="flex-1 truncate">
              <div className="text-xs text-blue-300/70 mb-0.5">Azure URL</div>
              <span className="text-sm text-blue-100 group-hover/link:text-blue-300 transition-colors truncate block">
                {service.azure.url}
              </span>
            </div>
            <ExternalLink className="w-4 h-4 text-blue-400 group-hover/link:text-blue-300 transition-colors" />
          </a>
        )}

        {/* Tech Stack */}
        <div className="grid grid-cols-2 gap-3 mb-4">
          <div className="glass p-3 rounded-xl border border-white/5">
            <div className="flex items-center gap-2 mb-1">
              <Code className="w-3.5 h-3.5 text-accent" />
              <span className="text-xs text-muted-foreground">Framework</span>
            </div>
            <p className="font-semibold text-sm text-foreground">{service.framework}</p>
          </div>
          <div className="glass p-3 rounded-xl border border-white/5">
            <div className="flex items-center gap-2 mb-1">
              <Layers className="w-3.5 h-3.5 text-secondary" />
              <span className="text-xs text-muted-foreground">Language</span>
            </div>
            <p className="font-semibold text-sm text-foreground">{service.language}</p>
          </div>
        </div>

        {/* Metrics Row */}
        <div className="flex items-center justify-between py-3 px-4 rounded-xl bg-linear-to-r from-primary/5 to-accent/5 border border-white/5 mb-4">
          {service.local?.port && (
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-primary animate-pulse"></div>
              <span className="text-xs text-muted-foreground">Port</span>
              <span className="font-mono font-semibold text-sm text-primary">{service.local.port}</span>
            </div>
          )}
          <div className="flex items-center gap-2">
            {health === 'healthy' ? (
              <CheckCircle className="w-4 h-4 text-success" />
            ) : (
              <XCircle className="w-4 h-4 text-destructive" />
            )}
            <span className={`text-sm font-medium ${
              health === 'healthy' ? 'text-success' : 'text-destructive'
            }`}>
              {health}
            </span>
          </div>
        </div>

        {/* Footer */}
        {(service.local?.startTime || service.local?.lastChecked || service.startTime || service.lastChecked) && (
          <div className="pt-4 border-t border-white/5 space-y-1.5 text-xs text-muted-foreground">
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

        {/* Error State */}
        {service.error && (
          <div className="mt-3 p-3 rounded-xl bg-destructive/10 border border-destructive/30">
            <div className="flex items-start gap-2">
              <AlertCircle className="w-4 h-4 text-destructive shrink-0 mt-0.5" />
              <div>
                <p className="text-xs font-medium text-destructive mb-1">Error Detected</p>
                <p className="text-xs text-destructive/80">{service.error}</p>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
