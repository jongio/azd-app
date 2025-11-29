import { Activity, Terminal, FileText, GitBranch, BarChart3 } from 'lucide-react'
import type { HealthSummary } from '@/types'

interface SidebarProps {
  activeView: string
  onViewChange: (view: string) => void
  hasActiveErrors?: boolean
  healthSummary?: HealthSummary | null
}

/** Determine the status indicator color based on health summary */
function getStatusIndicator(healthSummary?: HealthSummary | null, hasActiveErrors?: boolean): { color: string; title: string } | null {
  if (healthSummary) {
    if (healthSummary.unhealthy > 0) {
      return { color: 'bg-red-500', title: `${healthSummary.unhealthy} unhealthy service(s)` }
    }
    if (healthSummary.degraded > 0 || healthSummary.unknown > 0) {
      return { color: 'bg-yellow-500', title: `${healthSummary.degraded + healthSummary.unknown} degraded/unknown service(s)` }
    }
    if (healthSummary.healthy > 0) {
      return { color: 'bg-green-500', title: 'All services healthy' }
    }
  }
  // Fallback to hasActiveErrors if no health summary
  if (hasActiveErrors) {
    return { color: 'bg-red-500', title: 'Active errors detected' }
  }
  return null
}

export function Sidebar({ activeView, onViewChange, hasActiveErrors = false, healthSummary }: SidebarProps) {
  const navItems = [
    { id: 'resources', label: 'Resources', icon: Activity },
    { id: 'console', label: 'Console', icon: Terminal },
    { id: 'structured', label: 'Structured', icon: FileText },
    { id: 'traces', label: 'Traces', icon: GitBranch },
    { id: 'metrics', label: 'Metrics', icon: BarChart3 },
  ]

  return (
    <aside className="w-20 bg-background border-r border-border flex flex-col items-center py-4">
      {navItems.map((item) => {
        const Icon = item.icon
        const isActive = activeView === item.id
        const statusIndicator = item.id === 'console' ? getStatusIndicator(healthSummary, hasActiveErrors) : null
        
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
