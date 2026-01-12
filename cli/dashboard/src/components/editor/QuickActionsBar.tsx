/**
 * Quick Actions Bar
 * Provides one-click buttons for common service additions
 */

import * as React from 'react'
import { Plus } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { WellKnownService } from '@/lib/editor/wellknown-types'

export interface QuickActionsBarProps {
  /** Callback when a quick action button is clicked */
  onAddService: (service: WellKnownService) => void
  
  /** Quick action service names (defaults to: azurite, cosmos, redis, postgres) */
  quickServices?: string[]
  
  /** All available well-known services */
  services: WellKnownService[]
  
  /** Additional CSS classes */
  className?: string
}

/**
 * Quick Actions Bar Component
 * Displays buttons for quickly adding common well-known services
 */
export function QuickActionsBar({ 
  onAddService, 
  quickServices = ['azurite', 'cosmos', 'redis', 'postgres'],
  services,
  className 
}: QuickActionsBarProps) {
  // Get services for quick actions
  const actionServices = React.useMemo(() => {
    return quickServices
      .map(name => services.find(s => s.name === name))
      .filter((s): s is WellKnownService => s !== undefined)
  }, [quickServices, services])

  if (actionServices.length === 0) {
    return null
  }

  return (
    <div 
      className={cn(
        'fixed bottom-0 left-0 right-0 z-50',
        'bg-white dark:bg-slate-900 border-t border-slate-200 dark:border-slate-700',
        'shadow-lg',
        className
      )}
    >
      <div className="container mx-auto px-4 py-3">
        <div className="flex items-center justify-between flex-wrap gap-3">
          {/* Label */}
          <div className="flex items-center gap-2">
            <div className="text-2xl" aria-hidden="true">📋</div>
            <span className="text-sm font-semibold text-slate-700 dark:text-slate-300">
              Quick Actions
            </span>
          </div>

          {/* Action Buttons */}
          <div className="flex items-center gap-2 flex-wrap">
            {actionServices.map((service) => (
              <button
                key={service.name}
                onClick={() => onAddService(service)}
                className={cn(
                  'inline-flex items-center gap-2 px-4 py-2 rounded-lg',
                  'bg-cyan-500 hover:bg-cyan-600 active:bg-cyan-700',
                  'text-white font-medium text-sm',
                  'transition-colors duration-150',
                  'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                  'shadow-sm hover:shadow-md'
                )}
                aria-label={`Add ${service.displayName}`}
              >
                <Plus className="w-4 h-4" aria-hidden="true" />
                <span className="hidden sm:inline">Add {service.displayName.split('(')[0].trim()}</span>
                <span className="sm:hidden">{service.icon || '📦'} {service.name}</span>
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
