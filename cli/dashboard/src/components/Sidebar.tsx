import { Activity, Terminal, Settings2, BarChart3 } from 'lucide-react'
import type { HealthSummary, Service } from '@/types'
import { calculateStatusCounts, type StatusCounts } from '@/lib/service-utils'

interface SidebarProps {
  activeView: string
  onViewChange: (view: string) => void
  hasActiveErrors?: boolean
  healthSummary?: HealthSummary | null
  services?: Service[]
}

/** Determine the status indicator color based on status counts */
function getSidebarStatusIndicator(counts: StatusCounts, hasActiveErrors: boolean): { color: string; title: string } | null {
  // Priority: error > warn > running > stopped
  if (counts.error > 0) {
    return { color: 'bg-red-500', title: `${counts.error} unhealthy service(s)` }
  }
  if (counts.warn > 0) {
    return { color: 'bg-yellow-500', title: `${counts.warn} degraded/unknown service(s)` }
  }
  if (counts.running > 0) {
    return { color: 'bg-green-500', title: 'All services healthy' }
  }
  if (counts.stopped > 0) {
    return { color: 'bg-gray-400', title: `${counts.stopped} stopped service(s)` }
  }
  // Fallback to hasActiveErrors when no services/health data
  if (hasActiveErrors) {
    return { color: 'bg-red-500', title: 'Active errors detected' }
  }
  return null
}

export function Sidebar({ activeView, onViewChange, hasActiveErrors = false, healthSummary, services = [] }: SidebarProps) {
  // Use unified status count calculation
  const statusCounts = calculateStatusCounts(services, healthSummary, hasActiveErrors)

  const navItems = [
    { id: 'resources', label: 'Resources', icon: Activity },
    { id: 'console', label: 'Console', icon: Terminal },
    { id: 'environment', label: 'Environment', icon: Settings2 },
    { id: 'metrics', label: 'Metrics', icon: BarChart3 },
  ]

  return (
    <aside className="w-20 bg-background border-r border-border flex flex-col items-center py-4">
      {navItems.map((item) => {
        const Icon = item.icon
        const isActive = activeView === item.id
        const statusIndicator = item.id === 'console' ? getSidebarStatusIndicator(statusCounts, hasActiveErrors) : null
        
        return (
          <button
            key={item.id}
            onClick={() => onViewChange(item.id)}
            className={`
              w-16 py-3 mb-1 rounded-md flex flex-col items-center gap-1.5
              transition-all duration-200 relative
              ${isActive 
                ? 'bg-accent text-accent-foreground' 
                : 'text-foreground-tertiary hover:text-foreground hover:bg-secondary'
              }
            `}
          >
            <Icon className="w-5 h-5" />
            <span className="text-[10px] font-medium leading-tight text-center">{item.label}</span>
            {statusIndicator && (
              <span 
                className={`absolute top-1 right-1 w-2 h-2 ${statusIndicator.color} rounded-full ${statusIndicator.color === 'bg-red-500' ? 'animate-status-flash' : statusIndicator.color === 'bg-yellow-500' ? 'animate-caution-pulse' : statusIndicator.color === 'bg-green-500' ? 'animate-heartbeat' : ''}`}
                title={statusIndicator.title}
              />
            )}
          </button>
        )
      })}
    </aside>
  )
}
