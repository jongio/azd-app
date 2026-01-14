/**
 * Merge Strategy Selector Component
 * Allows selecting how to merge imported configuration
 */

import * as React from 'react'
import { cn } from '@/lib/utils'
import type { MergeStrategy } from '@/lib/editor/import-export-types'
import { Replace, Merge as MergeIcon, ListChecks } from 'lucide-react'

export interface MergeStrategySelectorProps {
  value: MergeStrategy
  onChange: (strategy: MergeStrategy) => void
}

const STRATEGIES: Array<{
  value: MergeStrategy
  label: string
  description: string
  icon: React.ComponentType<{ className?: string }>
}> = [
  {
    value: 'merge',
    label: 'Merge',
    description: 'Combine with existing configuration (services added, arrays appended)',
    icon: MergeIcon,
  },
  {
    value: 'replace',
    label: 'Replace',
    description: 'Replace entire configuration (backup created first)',
    icon: Replace,
  },
  {
    value: 'cherry-pick',
    label: 'Cherry-pick',
    description: 'Select specific sections to import',
    icon: ListChecks,
  },
]

/**
 * Merge Strategy Selector Component
 */
export function MergeStrategySelector({ value, onChange }: MergeStrategySelectorProps) {
  return (
    <div>
      <label className="block text-sm font-semibold text-slate-900 dark:text-slate-100 mb-3">
        Merge Strategy
      </label>
      <div className="space-y-2">
        {STRATEGIES.map((strategy) => {
          const Icon = strategy.icon
          const isSelected = value === strategy.value

          return (
            <button
              key={strategy.value}
              type="button"
              onClick={() => onChange(strategy.value)}
              className={cn(
                'w-full p-4 rounded-lg border-2 text-left transition-all duration-150',
                isSelected
                  ? 'border-cyan-500 bg-cyan-50 dark:bg-cyan-900/20'
                  : 'border-slate-200 dark:border-slate-700 hover:border-slate-300 dark:hover:border-slate-600'
              )}
            >
              <div className="flex items-start gap-3">
                <div className={cn(
                  'p-2 rounded-lg',
                  isSelected
                    ? 'bg-cyan-100 dark:bg-cyan-900/40'
                    : 'bg-slate-100 dark:bg-slate-800'
                )}>
                  <Icon className={cn(
                    'w-5 h-5',
                    isSelected
                      ? 'text-cyan-600 dark:text-cyan-400'
                      : 'text-slate-500'
                  )} />
                </div>
                <div className="flex-1">
                  <div className={cn(
                    'font-semibold text-sm mb-1',
                    isSelected
                      ? 'text-cyan-900 dark:text-cyan-100'
                      : 'text-slate-900 dark:text-slate-100'
                  )}>
                    {strategy.label}
                  </div>
                  <div className="text-xs text-slate-600 dark:text-slate-400">
                    {strategy.description}
                  </div>
                </div>
              </div>
            </button>
          )
        })}
      </div>
    </div>
  )
}
