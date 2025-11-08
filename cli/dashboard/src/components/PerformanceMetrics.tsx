import { Activity, Cpu, Network, Clock } from 'lucide-react'
import type { Service } from '@/types'

interface PerformanceMetricsProps {
  services: Service[]
}

interface MetricCardProps {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: string | number
  unit?: string
  trend?: 'up' | 'down' | 'stable'
  color?: 'primary' | 'success' | 'warning' | 'destructive'
}

function MetricCard({ icon: Icon, label, value, unit, trend, color = 'primary' }: MetricCardProps) {
  const colorClasses = {
    primary: 'text-primary bg-primary/10',
    success: 'text-success bg-success/10',
    warning: 'text-warning bg-warning/10',
    destructive: 'text-destructive bg-destructive/10'
  }

  return (
    <div className="glass p-4 rounded-xl border border-white/10 hover:border-white/20 transition-all group">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-2">
            <div className={`p-2 rounded-lg ${colorClasses[color]}`}>
              <Icon className={`w-4 h-4 ${colorClasses[color].split(' ')[0]}`} />
            </div>
            <span className="text-xs text-muted-foreground">{label}</span>
          </div>
          <div className="flex items-baseline gap-1">
            <span className="text-2xl font-bold text-foreground">{value}</span>
            {unit && <span className="text-sm text-muted-foreground">{unit}</span>}
          </div>
        </div>
        {trend && (
          <div className={`text-xs px-2 py-1 rounded ${
            trend === 'up' ? 'bg-success/10 text-success' :
            trend === 'down' ? 'bg-destructive/10 text-destructive' :
            'bg-muted/10 text-muted-foreground'
          }`}>
            {trend === 'up' ? '↑' : trend === 'down' ? '↓' : '→'}
          </div>
        )}
      </div>
    </div>
  )
}

export function PerformanceMetrics({ services }: PerformanceMetricsProps) {
  // Calculate aggregate metrics
  const totalServices = services.length
  const runningServices = services.filter(s => 
    s.local?.status === 'ready' || s.local?.status === 'running' || 
    s.status === 'ready' || s.status === 'running'
  ).length
  
  // Count active ports
  const activePorts = new Set(
    services
      .filter(s => s.local?.port)
      .map(s => s.local!.port)
  ).size

  // Calculate average uptime
  const uptimes = services
    .filter(s => s.local?.startTime || s.startTime)
    .map(s => {
      const startTime = new Date(s.local?.startTime || s.startTime!)
      const now = new Date()
      return Math.floor((now.getTime() - startTime.getTime()) / 1000)
    })
  
  const avgUptimeSeconds = uptimes.length > 0 
    ? Math.floor(uptimes.reduce((a, b) => a + b, 0) / uptimes.length)
    : 0

  const formatUptime = (seconds: number): string => {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    if (hours > 0) return `${hours}h ${minutes}m`
    if (minutes > 0) return `${minutes}m`
    return `${seconds}s`
  }

  // Simulated metrics (in real implementation, these would come from actual monitoring)
  const metrics: MetricCardProps[] = [
    {
      icon: Cpu,
      label: 'Active Services',
      value: runningServices,
      unit: `/ ${totalServices}`,
      trend: runningServices === totalServices ? 'up' : runningServices > totalServices / 2 ? 'stable' : 'down',
      color: runningServices === totalServices ? 'success' : 'warning'
    },
    {
      icon: Network,
      label: 'Active Ports',
      value: activePorts,
      unit: 'ports',
      trend: 'stable',
      color: 'primary'
    },
    {
      icon: Clock,
      label: 'Avg Uptime',
      value: formatUptime(avgUptimeSeconds),
      trend: avgUptimeSeconds > 3600 ? 'up' : 'stable',
      color: 'success'
    },
    {
      icon: Activity,
      label: 'Health Score',
      value: totalServices > 0 ? Math.round((runningServices / totalServices) * 100) : 0,
      unit: '%',
      trend: runningServices === totalServices ? 'up' : 'stable',
      color: runningServices === totalServices ? 'success' : 'warning'
    }
  ]

  return (
    <div className="space-y-6">
      {/* Metrics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {metrics.map((metric) => (
          <MetricCard
            key={metric.label}
            icon={metric.icon}
            label={metric.label}
            value={metric.value}
            unit={metric.unit}
            trend={metric.trend}
            color={metric.color}
          />
        ))}
      </div>

      {/* Service-Level Metrics */}
      {services.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-foreground mb-3">Service Metrics</h3>
          <div className="glass rounded-xl border border-white/10 overflow-hidden">
            <div className="max-h-[400px] overflow-y-auto">
              <table className="w-full">
                <thead className="sticky top-0 bg-[#0d0d0d] border-b border-white/10 z-10">
                  <tr>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                      Service
                    </th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                      Status
                    </th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                      Uptime
                    </th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                      Port
                    </th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                      Health
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/5">
                  {services.map((service) => {
                    const status = service.local?.status || service.status || 'not-running'
                    const health = service.local?.health || service.health || 'unknown'
                    const startTime = service.local?.startTime || service.startTime
                    const uptime = startTime 
                      ? formatUptime(Math.floor((new Date().getTime() - new Date(startTime).getTime()) / 1000))
                      : 'N/A'

                    return (
                      <tr key={service.name} className="hover:bg-white/5 transition-colors">
                        <td className="px-4 py-3">
                          <div>
                            <p className="text-sm font-medium text-foreground">{service.name}</p>
                            <p className="text-xs text-muted-foreground">
                              {service.framework}
                            </p>
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <span className={`inline-flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium ${
                            status === 'ready' || status === 'running' 
                              ? 'bg-success/10 text-success'
                              : status === 'starting'
                              ? 'bg-warning/10 text-warning'
                              : status === 'error'
                              ? 'bg-destructive/10 text-destructive'
                              : 'bg-muted/10 text-muted-foreground'
                          }`}>
                            <span className={`w-1.5 h-1.5 rounded-full ${
                              status === 'ready' || status === 'running'
                                ? 'bg-success animate-pulse'
                                : status === 'starting'
                                ? 'bg-warning animate-pulse'
                                : status === 'error'
                                ? 'bg-destructive'
                                : 'bg-muted-foreground'
                            }`}></span>
                            {status}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          <span className="text-sm font-mono text-muted-foreground">
                            {uptime}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          <span className="text-sm font-mono text-foreground">
                            {service.local?.port || 'N/A'}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          <span className={`inline-flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium ${
                            health === 'healthy'
                              ? 'bg-success/10 text-success'
                              : health === 'unhealthy'
                              ? 'bg-destructive/10 text-destructive'
                              : 'bg-muted/10 text-muted-foreground'
                          }`}>
                            {health}
                          </span>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* Info Note */}
      <div className="glass p-4 rounded-xl border border-blue-500/20 bg-blue-500/5">
        <div className="flex items-start gap-3">
          <Activity className="w-5 h-5 text-blue-400 shrink-0 mt-0.5" />
          <div>
            <p className="text-sm font-medium text-blue-300 mb-1">Performance Monitoring</p>
            <p className="text-xs text-blue-200/70">
              These metrics provide a real-time overview of your local development services. 
              For detailed performance metrics (CPU, memory, network), consider integrating with monitoring tools.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
