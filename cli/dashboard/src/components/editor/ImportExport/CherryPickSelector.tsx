/**
 * Cherry-Pick Selector Component
 * Allows selecting specific sections to import
 */

import * as React from 'react'
import { cn } from '@/lib/utils'
import type { CherryPickSection } from '@/lib/editor/import-export-types'
import { Server, Database, Webhook, Settings, Info } from 'lucide-react'

export interface CherryPickSelectorProps {
  sections: CherryPickSection[]
  onToggle: (id: string) => void
}

const SECTION_ICONS: Record<string, React.ComponentType<{ className?: string }>> = {
  service: Server,
  resource: Database,
  hooks: Webhook,
  pipeline: Settings,
  metadata: Info,
}

/**
 * Cherry-Pick Selector Component
 */
export function CherryPickSelector({ sections, onToggle }: CherryPickSelectorProps) {
  const selectedCount = sections.filter(s => s.selected).length

  const handleToggleAll = () => {
    const shouldSelectAll = selectedCount < sections.length
    sections.forEach(section => {
      if (section.selected !== shouldSelectAll) {
        onToggle(section.id)
      }
    })
  }

  return (
    <div className="border border-slate-200 dark:border-slate-700 rounded-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <label className="text-sm font-semibold text-slate-900 dark:text-slate-100">
          Select Sections to Import
        </label>
        <button
          type="button"
          onClick={handleToggleAll}
          className="text-xs text-cyan-600 dark:text-cyan-400 hover:underline font-medium"
        >
          {selectedCount < sections.length ? 'Select All' : 'Deselect All'}
        </button>
      </div>

      <div className="space-y-2">
        {sections.map((section) => {
          const Icon = SECTION_ICONS[section.type] || Info

          return (
            <label
              key={section.id}
              className={cn(
                'flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-colors',
                section.selected
                  ? 'border-cyan-500 bg-cyan-50 dark:bg-cyan-900/20'
                  : 'border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800/50'
              )}
            >
              <input
                type="checkbox"
                checked={section.selected}
                onChange={() => onToggle(section.id)}
                className="mt-0.5 w-4 h-4 text-cyan-600 border-slate-300 rounded focus:ring-cyan-500"
              />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <Icon className={cn(
                    'w-4 h-4 flex-shrink-0',
                    section.selected
                      ? 'text-cyan-600 dark:text-cyan-400'
                      : 'text-slate-500'
                  )} />
                  <span className={cn(
                    'text-sm font-medium',
                    section.selected
                      ? 'text-cyan-900 dark:text-cyan-100'
                      : 'text-slate-900 dark:text-slate-100'
                  )}>
                    {section.name}
                  </span>
                </div>
                <p className="text-xs text-slate-600 dark:text-slate-400">
                  {section.description}
                </p>
              </div>
            </label>
          )
        })}
      </div>

      <div className="mt-3 text-xs text-slate-600 dark:text-slate-400">
        {selectedCount} of {sections.length} sections selected
      </div>
    </div>
  )
}
