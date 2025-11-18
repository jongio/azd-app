import { RefreshCw, Trash2, Download, Terminal, Zap, FileText } from 'lucide-react'
import type { Service } from '@/types'

interface QuickActionsProps {
  services: Service[]
  onAction?: (action: string, serviceName?: string) => void
}

export function QuickActions({ services, onAction }: QuickActionsProps) {
  const handleAction = (action: string, serviceName?: string) => {
    console.log(`Quick action: ${action}`, serviceName)
    onAction?.(action, serviceName)
  }

  const runningServices = services.filter(s => 
    s.local?.status === 'ready' || s.local?.status === 'running' || s.status === 'ready' || s.status === 'running'
  )
  const healthyServices = services.filter(s => 
    (s.local?.health || s.health) === 'healthy'
  )
  const errorServices = services.filter(s => 
    s.local?.status === 'error' || s.status === 'error' || (s.local?.health || s.health) === 'unhealthy'
  )

  const actions = [
    {
      id: 'refresh-all',
      icon: RefreshCw,
      label: 'Refresh All',
      description: 'Refresh all service status',
      variant: 'primary' as const,
      onClick: () => handleAction('refresh-all')
    },
    {
      id: 'clear-logs',
      icon: Trash2,
      label: 'Clear Logs',
      description: 'Clear all log buffers',
      variant: 'secondary' as const,
      onClick: () => handleAction('clear-logs')
    },
    {
      id: 'export-logs',
      icon: Download,
      label: 'Export Logs',
      description: 'Download logs as file',
      variant: 'secondary' as const,
      onClick: () => handleAction('export-logs')
    },
    {
      id: 'view-terminal',
      icon: Terminal,
      label: 'Open Terminal',
      description: 'Open system terminal',
      variant: 'secondary' as const,
      onClick: () => handleAction('view-terminal')
    }
  ]

  return (
    <div className="space-y-6">
      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="glass p-4 rounded-xl border border-white/10">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs text-muted-foreground mb-1">Running Services</p>
              <p className="text-2xl font-bold text-foreground">{runningServices.length}</p>
            </div>
            <div className="p-3 rounded-xl bg-primary/10">
              <Zap className="w-6 h-6 text-primary" />
            </div>
          </div>
        </div>

        <div className="glass p-4 rounded-xl border border-white/10">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs text-muted-foreground mb-1">Healthy</p>
              <p className="text-2xl font-bold text-success">{healthyServices.length}</p>
            </div>
            <div className="p-3 rounded-xl bg-success/10">
              <Zap className="w-6 h-6 text-success" />
            </div>
          </div>
        </div>

        <div className="glass p-4 rounded-xl border border-white/10">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs text-muted-foreground mb-1">Errors</p>
              <p className="text-2xl font-bold text-destructive">{errorServices.length}</p>
            </div>
            <div className="p-3 rounded-xl bg-destructive/10">
              <Zap className="w-6 h-6 text-destructive" />
            </div>
          </div>
        </div>
      </div>

      {/* Quick Actions Grid */}
      <div>
        <h3 className="text-sm font-semibold text-foreground mb-3">Quick Actions</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          {actions.map((action) => {
            const Icon = action.icon
            return (
              <button
                key={action.id}
                onClick={action.onClick}
                className={`p-4 rounded-xl border transition-all hover:scale-105 group ${
                  action.variant === 'primary'
                    ? 'bg-primary/10 border-primary/30 hover:bg-primary/20 hover:border-primary/50'
                    : 'glass border-white/10 hover:bg-white/5 hover:border-white/20'
                }`}
              >
                <Icon className={`w-5 h-5 mb-2 transition-transform group-hover:scale-110 ${
                  action.variant === 'primary' ? 'text-primary' : 'text-muted-foreground'
                }`} />
                <div className="text-left">
                  <p className="text-sm font-semibold text-foreground mb-0.5">
                    {action.label}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {action.description}
                  </p>
                </div>
              </button>
            )
          })}
        </div>
      </div>

      {/* Service-Specific Actions */}
      {services.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-foreground mb-3">Service Actions</h3>
          <div className="glass rounded-xl border border-white/10 overflow-hidden">
            <div className="max-h-[300px] overflow-y-auto">
              {services.map((service) => {
                const isRunning = service.local?.status === 'ready' || service.local?.status === 'running' || 
                                service.status === 'ready' || service.status === 'running'
                const hasError = service.local?.status === 'error' || service.status === 'error'
                
                return (
                  <div
                    key={service.name}
                    className="p-3 border-b border-white/5 last:border-b-0 hover:bg-white/5 transition-colors"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className={`w-2 h-2 rounded-full ${
                          isRunning ? 'bg-success animate-pulse' : 
                          hasError ? 'bg-destructive' : 
                          'bg-muted-foreground'
                        }`}></div>
                        <div>
                          <p className="text-sm font-medium text-foreground">{service.name}</p>
                          <p className="text-xs text-muted-foreground">
                            {service.framework} • {service.language}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => handleAction('view-logs', service.name)}
                          className="p-2 hover:bg-white/10 rounded-md transition-colors"
                          title="View logs"
                        >
                          <FileText className="w-4 h-4 text-muted-foreground hover:text-foreground" />
                        </button>
                        <button
                          onClick={() => handleAction('refresh', service.name)}
                          className="p-2 hover:bg-white/10 rounded-md transition-colors"
                          title="Refresh status"
                        >
                          <RefreshCw className="w-4 h-4 text-muted-foreground hover:text-foreground" />
                        </button>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
