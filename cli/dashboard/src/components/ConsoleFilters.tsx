/**
 * ConsoleFilters - Filter bar for services, log levels, state, and health
 */
import * as React from 'react'
import {
  Info,
  AlertTriangle,
  XCircle,
  CheckCircle,
  Circle,
  Loader2,
  Heart,
  HeartPulse,
  HeartCrack,
  HelpCircle,
  Globe,
  Server,
  Database,
  Box,
  Cpu,
  Zap,
  Package,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { normalizeHealthStatus } from '@/lib/service-utils'
import type { Service, HealthReportEvent, HealthStatus } from '@/types'
import type { LogLevel, FilterableLifecycleState } from '@/hooks/useConsoleFilters'

// =============================================================================
// Helper Functions  
// =============================================================================

interface ServiceIconColor {
  icon: typeof Globe
  colorScheme: {
    selected: string
    unselected: string
  }
}

function getServiceIconAndColor(serviceName: string, health: HealthStatus = 'unknown'): ServiceIconColor {
  const lowerName = serviceName.toLowerCase()
  
  // Determine icon based on service name patterns
  let icon = Package // default
  if (lowerName.includes('web') || lowerName.includes('frontend') || lowerName.includes('ui') || lowerName.includes('app')) {
    icon = Globe
  } else if (lowerName.includes('api') || lowerName.includes('backend') || lowerName.includes('server')) {
    icon = Server
  } else if (lowerName.includes('worker') || lowerName.includes('queue') || lowerName.includes('background')) {
    icon = Cpu
  } else if (lowerName.includes('function') || lowerName.includes('func')) {
    icon = Zap
  } else if (lowerName.includes('container')) {
    icon = Box
  } else if (lowerName.includes('db') || lowerName.includes('database') || lowerName.includes('postgres') || lowerName.includes('redis') || lowerName.includes('mongo') || lowerName.includes('mysql')) {
    icon = Database
  }
  
  // Health-based colors matching log pane indicators
  // Red (unhealthy), Yellow (degraded/unknown), Green (healthy)
  const healthColorSchemes: Record<HealthStatus, { selected: string; unselected: string }> = {
    healthy: {
      selected: 'bg-green-100 dark:bg-green-500/20 text-green-700 dark:text-green-300 ring-1 ring-green-500',
      unselected: 'text-green-600 dark:text-green-400 hover:bg-green-50 dark:hover:bg-green-500/10'
    },
    degraded: {
      selected: 'bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-300 ring-1 ring-amber-500',
      unselected: 'text-amber-600 dark:text-amber-400 hover:bg-amber-50 dark:hover:bg-amber-500/10'
    },
    unhealthy: {
      selected: 'bg-red-100 dark:bg-red-500/20 text-red-700 dark:text-red-300 ring-1 ring-red-500',
      unselected: 'text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-500/10'
    },
    unknown: {
      selected: 'bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-300 ring-1 ring-amber-500',
      unselected: 'text-amber-600 dark:text-amber-400 hover:bg-amber-50 dark:hover:bg-amber-500/10'
    }
  }
  
  return {
    icon,
    colorScheme: healthColorSchemes[health]
  }
}

// =============================================================================
// Component
// =============================================================================

export interface ConsoleFiltersProps {
  services: Service[]
  selectedServices: Set<string>
  onToggleService: (name: string) => void
  levelFilter: Set<LogLevel>
  onToggleLevel: (level: LogLevel) => void
  stateFilter: Set<FilterableLifecycleState>
  onToggleState: (state: FilterableLifecycleState) => void
  healthFilter: Set<HealthStatus>
  onToggleHealth: (status: HealthStatus) => void
  healthReport?: HealthReportEvent | null
}

export function ConsoleFilters({
  services,
  selectedServices,
  onToggleService,
  levelFilter,
  onToggleLevel,
  stateFilter,
  onToggleState,
  healthFilter,
  onToggleHealth,
  healthReport,
}: Readonly<ConsoleFiltersProps>) {
  const sortedServices = React.useMemo(() => {
    return [...services].sort((a, b) => a.name.localeCompare(b.name))
  }, [services])

  return (
    <div className="flex flex-col md:flex-row md:flex-wrap gap-6 p-4 bg-slate-100 dark:bg-slate-800 border-b border-slate-300 dark:border-slate-700 shrink-0">
      {/* Services */}
      <div className="flex flex-col gap-2">
        <span className="text-xs font-medium text-slate-500">Services</span>
        <div className="flex flex-wrap gap-2">
          {sortedServices.map((service) => {
            // Get health status from health report
            const serviceHealth = healthReport?.services.find(
              (s) => s.serviceName === service.name
            )?.status ?? 'unknown'
            const normalizedHealth = normalizeHealthStatus(serviceHealth)
            
            const { icon: IconComponent, colorScheme } = getServiceIconAndColor(service.name, normalizedHealth)
            const isSelected = selectedServices.has(service.name)
            
            return (
              <button
                key={service.name}
                type="button"
                onClick={() => onToggleService(service.name)}
                className={cn(
                  'flex items-center gap-1.5 px-2.5 py-1.5 rounded-md transition-all max-w-[150px]',
                  isSelected
                    ? colorScheme.selected
                    : cn('bg-transparent', colorScheme.unselected)
                )}
                aria-label={`Toggle ${service.name}`}
                title={`${service.name} - ${normalizedHealth}`}
              >
                <IconComponent className="w-3.5 h-3.5 shrink-0" />
                <span className="text-xs font-medium truncate">{service.name}</span>
              </button>
            )
          })}
        </div>
      </div>

      <div className="hidden md:block w-px bg-slate-300 dark:bg-slate-700 self-stretch" />

      {/* Log Levels */}
      <div className="flex flex-col gap-2">
        <span className="text-xs font-medium text-slate-500">Log Levels</span>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => onToggleLevel('info')}
            aria-label="Info"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              levelFilter.has('info')
                ? 'bg-sky-100 dark:bg-sky-500/20 text-sky-700 dark:text-sky-300 ring-1 ring-sky-300 dark:ring-sky-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Info logs"
          >
            <Info className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-sky-700/95 text-white text-xs px-2 py-1">Info</span>
          </button>
          <button
            type="button"
            onClick={() => onToggleLevel('warning')}
            aria-label="Warning"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              levelFilter.has('warning')
                ? 'bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-300 ring-1 ring-amber-300 dark:ring-amber-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Warning logs"
          >
            <AlertTriangle className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-amber-700/95 text-white text-xs px-2 py-1">Warning</span>
          </button>
          <button
            type="button"
            onClick={() => onToggleLevel('error')}
            aria-label="Error"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              levelFilter.has('error')
                ? 'bg-rose-100 dark:bg-rose-500/20 text-rose-700 dark:text-rose-300 ring-1 ring-rose-300 dark:ring-rose-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Error logs"
          >
            <XCircle className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-rose-700/95 text-white text-xs px-2 py-1">Error</span>
          </button>
        </div>
      </div>

      <div className="hidden md:block w-px bg-slate-300 dark:bg-slate-700 self-stretch" />

      {/* State Filter */}
      <div className="flex flex-col gap-2">
        <span className="text-xs font-medium text-slate-500">State</span>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => onToggleState('running')}
            aria-label="Running"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              stateFilter.has('running')
                ? 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 ring-1 ring-emerald-300 dark:ring-emerald-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Running services"
          >
            <CheckCircle className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-emerald-700/95 text-white text-xs px-2 py-1">Running</span>
          </button>
          <button
            type="button"
            onClick={() => onToggleState('stopped')}
            aria-label="Stopped"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              stateFilter.has('stopped')
                ? 'bg-slate-200 dark:bg-slate-600/40 text-slate-700 dark:text-slate-300 ring-1 ring-slate-300 dark:ring-slate-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Stopped services"
          >
            <Circle className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-slate-700/95 text-white text-xs px-2 py-1">Stopped</span>
          </button>
          <button
            type="button"
            onClick={() => onToggleState('starting')}
            aria-label="Starting"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              stateFilter.has('starting')
                ? 'bg-sky-100 dark:bg-sky-500/20 text-sky-700 dark:text-sky-300 ring-1 ring-sky-300 dark:ring-sky-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Starting services"
          >
            <Loader2 className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-sky-700/95 text-white text-xs px-2 py-1">Starting</span>
          </button>
        </div>
      </div>

      <div className="hidden md:block w-px bg-slate-300 dark:bg-slate-700 self-stretch" />

      {/* Health Status */}
      <div className="flex flex-col gap-2">
        <span className="text-xs font-medium text-slate-500">Health Status</span>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => onToggleHealth('healthy')}
            aria-label="Healthy"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              healthFilter.has('healthy')
                ? 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 ring-1 ring-emerald-300 dark:ring-emerald-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Healthy services"
          >
            <Heart className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-emerald-700/95 text-white text-xs px-2 py-1">Healthy</span>
          </button>
          <button
            type="button"
            onClick={() => onToggleHealth('degraded')}
            aria-label="Degraded"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              healthFilter.has('degraded')
                ? 'bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-300 ring-1 ring-amber-300 dark:ring-amber-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Degraded services"
          >
            <HeartPulse className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-amber-700/95 text-white text-xs px-2 py-1">Degraded</span>
          </button>
          <button
            type="button"
            onClick={() => onToggleHealth('unhealthy')}
            aria-label="Unhealthy"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              healthFilter.has('unhealthy')
                ? 'bg-rose-100 dark:bg-rose-500/20 text-rose-700 dark:text-rose-300 ring-1 ring-rose-300 dark:ring-rose-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Unhealthy services"
          >
            <HeartCrack className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-rose-700/95 text-white text-xs px-2 py-1">Unhealthy</span>
          </button>
          <button
            type="button"
            onClick={() => onToggleHealth('unknown')}
            aria-label="Unknown"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              healthFilter.has('unknown')
                ? 'bg-slate-200 dark:bg-slate-600/40 text-slate-700 dark:text-slate-300 ring-1 ring-slate-300 dark:ring-slate-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Unknown services"
          >
            <HelpCircle className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-slate-700/95 text-white text-xs px-2 py-1">Unknown</span>
          </button>
        </div>
      </div>
    </div>
  )
}
