/**
 * Hook Execution Timeline
 * Visual timeline showing all lifecycle stages with configured hooks
 * - Shows which hooks are configured at each stage
 * - Highlights enabled vs disabled hooks
 * - Shows script summary on hover
 * - Click to edit hook
 */

import * as React from 'react'
import { Webhook, CheckCircle2, Circle, AlertTriangle, Code } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { HooksConfig, HookDisplayInfo, LifecycleEvent } from '@/lib/editor/hooks-types'
import { getHookDisplayInfo, LIFECYCLE_EVENTS } from '@/lib/editor/hooks-types'

export interface HookExecutionTimelineProps {
  /** Current hooks configuration */
  hooks: HooksConfig
  
  /** Callback when hook is clicked for editing */
  onEditHook: (event: LifecycleEvent) => void
  
  /** Callback to add new hook */
  onAddHook: (event: LifecycleEvent) => void
  
  /** Additional CSS classes */
  className?: string
}

/**
 * Hook Execution Timeline Component
 */
export function HookExecutionTimeline({
  hooks,
  onEditHook,
  onAddHook,
  className,
}: HookExecutionTimelineProps) {
  // Group events by category
  const eventsByCategory = React.useMemo(() => {
    const groups: Record<string, HookDisplayInfo[]> = {}
    
    LIFECYCLE_EVENTS.forEach((eventMeta) => {
      const hookConfig = hooks[eventMeta.event]
      const displayInfo = getHookDisplayInfo(eventMeta.event, hookConfig)
      
      if (!groups[displayInfo.category]) {
        groups[displayInfo.category] = []
      }
      groups[displayInfo.category].push(displayInfo)
    })
    
    return groups
  }, [hooks])

  const categories = Object.keys(eventsByCategory)

  return (
    <div className={cn('space-y-6', className)}>
      <div className="flex items-center gap-2 text-lg font-semibold text-slate-900 dark:text-slate-100">
        <Webhook className="w-5 h-5 text-cyan-600" />
        <span>Lifecycle Hooks Timeline</span>
      </div>

      {categories.map((category) => (
        <div key={category} className="space-y-3">
          {/* Category Header */}
          <div className="flex items-center gap-2 text-sm font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wide">
            <div className="h-px flex-1 bg-slate-200 dark:bg-slate-700" />
            <span>{category}</span>
            <div className="h-px flex-1 bg-slate-200 dark:bg-slate-700" />
          </div>

          {/* Events in Category */}
          <div className="space-y-2">
            {eventsByCategory[category].map((hookInfo) => (
              <HookTimelineItem
                key={hookInfo.event}
                hookInfo={hookInfo}
                onEdit={() => onEditHook(hookInfo.event)}
                onAdd={() => onAddHook(hookInfo.event)}
              />
            ))}
          </div>
        </div>
      ))}

      {/* Legend */}
      <div className="pt-4 border-t border-slate-200 dark:border-slate-700">
        <div className="flex flex-wrap items-center gap-4 text-xs text-slate-600 dark:text-slate-400">
          <div className="flex items-center gap-1.5">
            <CheckCircle2 className="w-3.5 h-3.5 text-green-600 dark:text-green-400" />
            <span>Configured & Enabled</span>
          </div>
          <div className="flex items-center gap-1.5">
            <Circle className="w-3.5 h-3.5 text-slate-400" />
            <span>Not Configured</span>
          </div>
          <div className="flex items-center gap-1.5">
            <AlertTriangle className="w-3.5 h-3.5 text-yellow-600 dark:text-yellow-400" />
            <span>Platform-Specific</span>
          </div>
        </div>
      </div>
    </div>
  )
}

interface HookTimelineItemProps {
  hookInfo: HookDisplayInfo
  onEdit: () => void
  onAdd: () => void
}

function HookTimelineItem({ hookInfo, onEdit, onAdd }: HookTimelineItemProps) {
  const [showTooltip, setShowTooltip] = React.useState(false)

  const handleClick = () => {
    if (hookInfo.configured) {
      onEdit()
    } else {
      onAdd()
    }
  }

  return (
    <div className="relative">
      <button
        type="button"
        onClick={handleClick}
        onMouseEnter={() => setShowTooltip(true)}
        onMouseLeave={() => setShowTooltip(false)}
        className={cn(
          'w-full flex items-center gap-3 p-3 rounded-lg border transition-all',
          'text-left hover:shadow-md',
          hookInfo.configured
            ? 'border-cyan-200 dark:border-cyan-900 bg-cyan-50 dark:bg-cyan-950/20 hover:border-cyan-300 dark:hover:border-cyan-800'
            : 'border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 hover:border-slate-300 dark:hover:border-slate-600'
        )}
      >
        {/* Status Icon */}
        <div className="shrink-0">
          {hookInfo.configured && hookInfo.enabled ? (
            <CheckCircle2 className="w-5 h-5 text-green-600 dark:text-green-400" />
          ) : (
            <Circle className="w-5 h-5 text-slate-400" />
          )}
        </div>

        {/* Event Info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-sm text-slate-900 dark:text-slate-100">
              {hookInfo.displayName}
            </span>
            {hookInfo.hasPlatformOverrides && (
              <AlertTriangle className="w-3.5 h-3.5 text-yellow-600 dark:text-yellow-400" />
            )}
          </div>
          <div className="text-xs text-slate-600 dark:text-slate-400 mt-0.5">
            {hookInfo.description}
          </div>
          {hookInfo.configured && hookInfo.scriptSummary && (
            <div className="flex items-center gap-1.5 mt-1.5 text-xs text-slate-500 dark:text-slate-500">
              <Code className="w-3 h-3" />
              <span className="font-mono truncate">{hookInfo.scriptSummary}</span>
            </div>
          )}
        </div>

        {/* Platform Indicators */}
        {hookInfo.configured && hookInfo.hasPlatformOverrides && (
          <div className="shrink-0 flex items-center gap-1.5">
            {hookInfo.hasWindows && (
              <span className="px-2 py-0.5 rounded text-xs font-medium bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300">
                Win
              </span>
            )}
            {hookInfo.hasPosix && (
              <span className="px-2 py-0.5 rounded text-xs font-medium bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300">
                POSIX
              </span>
            )}
          </div>
        )}
      </button>

      {/* Tooltip */}
      {showTooltip && hookInfo.configured && (
        <div className="absolute left-0 right-0 top-full mt-2 z-10 p-3 rounded-lg bg-slate-900 dark:bg-slate-800 border border-slate-700 shadow-lg">
          <div className="text-xs space-y-1">
            <div className="font-semibold text-white">{hookInfo.displayName}</div>
            {hookInfo.scriptSummary && (
              <div className="text-slate-300 font-mono">{hookInfo.scriptSummary}</div>
            )}
            {hookInfo.hasPlatformOverrides && (
              <div className="text-yellow-300 flex items-center gap-1 mt-1.5">
                <AlertTriangle className="w-3 h-3" />
                <span>Has platform-specific overrides</span>
              </div>
            )}
            <div className="text-slate-400 pt-1.5 border-t border-slate-700">
              Click to edit configuration
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
