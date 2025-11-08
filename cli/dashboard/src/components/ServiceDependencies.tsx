import { Network, ArrowRight, Circle } from 'lucide-react'
import type { Service } from '@/types'

interface ServiceDependenciesProps {
  services: Service[]
}

export function ServiceDependencies({ services }: ServiceDependenciesProps) {
  // Group services by language/framework for visualization
  const groupedServices = services.reduce((acc, service) => {
    const key = service.language || 'Unknown'
    if (!acc[key]) acc[key] = []
    acc[key].push(service)
    return acc
  }, {} as Record<string, Service[]>)

  const getStatusColor = (service: Service) => {
    const status = service.local?.status || service.status || 'not-running'
    const health = service.local?.health || service.health || 'unknown'
    
    if ((status === 'ready' || status === 'running') && health === 'healthy') return 'bg-success'
    if (status === 'starting') return 'bg-warning'
    if (status === 'error' || health === 'unhealthy') return 'bg-destructive'
    return 'bg-muted-foreground'
  }

  return (
    <div className="space-y-6">
      {/* Info Banner */}
      <div className="glass p-4 rounded-xl border border-blue-500/20 bg-blue-500/5">
        <div className="flex items-start gap-3">
          <Network className="w-5 h-5 text-blue-400 shrink-0 mt-0.5" />
          <div>
            <p className="text-sm font-medium text-blue-300 mb-1">Service Dependencies</p>
            <p className="text-xs text-blue-200/70">
              Visual representation of your service architecture. Services are grouped by language/technology.
            </p>
          </div>
        </div>
      </div>

      {/* Language Groups */}
      <div className="space-y-4">
        {Object.entries(groupedServices).map(([language, langServices]) => (
          <div key={language} className="glass p-6 rounded-xl border border-white/10">
            <div className="flex items-center gap-2 mb-4">
              <div className="p-2 rounded-lg bg-primary/10">
                <Network className="w-4 h-4 text-primary" />
              </div>
              <h3 className="text-lg font-semibold text-foreground">{language}</h3>
              <span className="text-sm text-muted-foreground">
                ({langServices.length} service{langServices.length !== 1 ? 's' : ''})
              </span>
            </div>

            {/* Service Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {langServices.map((service) => {
                const isRunning = service.local?.status === 'ready' || service.local?.status === 'running' ||
                                service.status === 'ready' || service.status === 'running'
                
                return (
                  <div
                    key={service.name}
                    className="p-4 rounded-lg bg-[#0d0d0d] border border-white/5 hover:border-white/20 transition-all group"
                  >
                    <div className="flex items-center gap-3 mb-3">
                      <div className={`w-3 h-3 rounded-full ${getStatusColor(service)} ${
                        isRunning ? 'animate-pulse' : ''
                      }`}></div>
                      <h4 className="font-semibold text-foreground group-hover:text-primary transition-colors">
                        {service.name}
                      </h4>
                    </div>

                    <div className="space-y-2 text-sm">
                      <div className="flex items-center justify-between">
                        <span className="text-muted-foreground">Framework:</span>
                        <span className="text-foreground font-medium">{service.framework}</span>
                      </div>
                      
                      {service.local?.port && (
                        <div className="flex items-center justify-between">
                          <span className="text-muted-foreground">Port:</span>
                          <span className="text-foreground font-mono">{service.local.port}</span>
                        </div>
                      )}

                      {service.local?.url && (
                        <div className="mt-2">
                          <a
                            href={service.local.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-xs text-primary hover:underline truncate block"
                          >
                            {service.local.url}
                          </a>
                        </div>
                      )}
                    </div>

                    {/* Show environment variables count if available */}
                    {service.environmentVariables && Object.keys(service.environmentVariables).length > 0 && (
                      <div className="mt-3 pt-3 border-t border-white/5">
                        <span className="text-xs text-muted-foreground">
                          {Object.keys(service.environmentVariables).length} env var{Object.keys(service.environmentVariables).length !== 1 ? 's' : ''}
                        </span>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        ))}
      </div>

      {/* Connection Flow Visualization */}
      {services.length > 1 && (
        <div className="glass p-6 rounded-xl border border-white/10">
          <h3 className="text-lg font-semibold text-foreground mb-4 flex items-center gap-2">
            <ArrowRight className="w-5 h-5 text-primary" />
            Typical Flow
          </h3>
          <div className="flex items-center justify-center gap-4 flex-wrap">
            {services.map((service, index) => (
              <div key={service.name} className="flex items-center gap-4">
                <div className="flex flex-col items-center gap-2">
                  <div className="relative">
                    <Circle className={`w-16 h-16 ${getStatusColor(service)}`} fill="currentColor" />
                    <div className="absolute inset-0 flex items-center justify-center">
                      <span className="text-xs font-semibold text-white">
                        {service.name.substring(0, 3).toUpperCase()}
                      </span>
                    </div>
                  </div>
                  <span className="text-xs text-muted-foreground">{service.name}</span>
                </div>
                {index < services.length - 1 && (
                  <ArrowRight className="w-6 h-6 text-primary" />
                )}
              </div>
            ))}
          </div>
          <p className="text-xs text-muted-foreground text-center mt-4">
            Simplified service communication flow
          </p>
        </div>
      )}

      {/* Empty State */}
      {services.length === 0 && (
        <div className="glass p-12 rounded-xl border border-white/10 text-center">
          <Network className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
          <h3 className="text-lg font-semibold text-foreground mb-2">No Services Available</h3>
          <p className="text-sm text-muted-foreground">
            Start your services to see their dependencies and relationships
          </p>
        </div>
      )}
    </div>
  )
}
