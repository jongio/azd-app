/**
 * Resource Template Selector
 * Visual grid for selecting resource templates
 */

import * as React from 'react'
import { Sparkles } from 'lucide-react'
import { cn } from '@/lib/utils'
import {
  getTemplatesForType,
  type ResourceType,
  type ResourceTemplate,
} from '@/lib/editor/resource-types'

export interface ResourceTemplateSelectorProps {
  /** Resource type to show templates for */
  resourceType: ResourceType
  
  /** Callback when template is selected */
  onSelect: (template: ResourceTemplate) => void
  
  /** Callback when user skips templates */
  onSkip: () => void
}

/**
 * Resource Template Selector Component
 */
export function ResourceTemplateSelector({
  resourceType,
  onSelect,
  onSkip,
}: ResourceTemplateSelectorProps) {
  const templates = React.useMemo(
    () => getTemplatesForType(resourceType.id),
    [resourceType.id]
  )

  if (templates.length === 0) {
    // No templates available, auto-skip
    React.useEffect(() => {
      onSkip()
    }, [onSkip])
    return null
  }

  return (
    <div className="space-y-3">
      {/* Header */}
      <div className="flex items-center gap-2 text-blue-600 dark:text-blue-400">
        <Sparkles className="w-4 h-4" />
        <span className="text-xs font-medium">
          Pre-configured templates for {resourceType.displayName}
        </span>
      </div>

      {/* Template Grid */}
      <div className="grid grid-cols-2 gap-3">
        {templates.map((template) => (
          <button
            key={template.id}
            type="button"
            onClick={() => onSelect(template)}
            className={cn(
              'flex items-start gap-3 p-4 rounded-lg border-2 transition-all text-left',
              'border-slate-200 dark:border-slate-700',
              'hover:border-cyan-500 hover:bg-cyan-50 dark:hover:bg-cyan-950/20',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2'
            )}
          >
            <span className="text-2xl shrink-0" aria-hidden="true">
              {template.icon || resourceType.icon || '📦'}
            </span>
            <div className="flex-1 min-w-0">
              <div className="font-semibold text-sm text-slate-900 dark:text-slate-100">
                {template.name}
              </div>
              <div className="text-xs text-slate-600 dark:text-slate-400 mt-0.5">
                {template.description}
              </div>
            </div>
          </button>
        ))}

        {/* Custom/Skip Option */}
        <button
          type="button"
          onClick={onSkip}
          className={cn(
            'flex items-start gap-3 p-4 rounded-lg border-2 transition-all text-left',
            'border-dashed border-slate-300 dark:border-slate-600',
            'hover:border-cyan-500 hover:bg-cyan-50 dark:hover:bg-cyan-950/20',
            'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2'
          )}
        >
          <span className="text-2xl shrink-0" aria-hidden="true">⚙️</span>
          <div className="flex-1 min-w-0">
            <div className="font-semibold text-sm text-slate-900 dark:text-slate-100">
              Custom Configuration
            </div>
            <div className="text-xs text-slate-600 dark:text-slate-400 mt-0.5">
              Configure manually without a template
            </div>
          </div>
        </button>
      </div>
    </div>
  )
}
