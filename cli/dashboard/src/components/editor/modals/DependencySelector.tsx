/**
 * Dependency Selector
 * Multi-select component for managing resource/service dependencies
 */

import * as React from 'react'
import { X, Plus } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface DependencySelectorProps {
  /** Available items to select from */
  available: string[]
  
  /** Currently selected items */
  selected: string[]
  
  /** Callback when selection changes */
  onChange: (selected: string[]) => void
  
  /** Error message (e.g., circular dependency) */
  error?: string | null
}

/**
 * Dependency Selector Component
 */
export function DependencySelector({
  available,
  selected,
  onChange,
  error,
}: DependencySelectorProps) {
  const [isOpen, setIsOpen] = React.useState(false)
  const dropdownRef = React.useRef<HTMLDivElement>(null)

  // Close dropdown when clicking outside
  React.useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Add dependency
  const handleAdd = (item: string) => {
    if (!selected.includes(item)) {
      onChange([...selected, item])
    }
    setIsOpen(false)
  }

  // Remove dependency
  const handleRemove = (item: string) => {
    onChange(selected.filter(s => s !== item))
  }

  // Get available items (excluding already selected)
  const availableItems = React.useMemo(
    () => available.filter(item => !selected.includes(item)),
    [available, selected]
  )

  return (
    <div className="space-y-2">
      {/* Selected Dependencies */}
      {selected.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {selected.map((item) => (
            <div
              key={item}
              className={cn(
                'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg',
                'bg-cyan-100 dark:bg-cyan-900/30 text-cyan-900 dark:text-cyan-100',
                'text-sm font-medium',
                error && 'bg-red-100 dark:bg-red-900/30 text-red-900 dark:text-red-100'
              )}
            >
              <span>{item}</span>
              <button
                type="button"
                onClick={() => handleRemove(item)}
                className="hover:bg-cyan-200 dark:hover:bg-cyan-800 rounded p-0.5 transition-colors"
                aria-label={`Remove ${item}`}
              >
                <X className="w-3.5 h-3.5" />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Add Dependency Dropdown */}
      <div ref={dropdownRef} className="relative">
        <button
          type="button"
          onClick={() => setIsOpen(!isOpen)}
          disabled={availableItems.length === 0}
          className={cn(
            'inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium',
            'border border-slate-300 dark:border-slate-600',
            'text-slate-700 dark:text-slate-300',
            'hover:bg-slate-50 dark:hover:bg-slate-800',
            'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
            'disabled:opacity-50 disabled:cursor-not-allowed',
            'transition-colors duration-150'
          )}
        >
          <Plus className="w-4 h-4" />
          <span>Add Dependency</span>
        </button>

        {/* Dropdown Menu */}
        {isOpen && availableItems.length > 0 && (
          <div
            className={cn(
              'absolute top-full left-0 mt-1 w-64 z-10',
              'bg-white dark:bg-slate-800 rounded-lg shadow-lg',
              'border border-slate-200 dark:border-slate-700',
              'max-h-48 overflow-y-auto'
            )}
          >
            {availableItems.map((item) => (
              <button
                key={item}
                type="button"
                onClick={() => handleAdd(item)}
                className={cn(
                  'w-full text-left px-4 py-2.5 text-sm',
                  'text-slate-700 dark:text-slate-300',
                  'hover:bg-slate-100 dark:hover:bg-slate-700',
                  'focus:outline-none focus:bg-slate-100 dark:focus:bg-slate-700',
                  'transition-colors duration-100',
                  'first:rounded-t-lg last:rounded-b-lg'
                )}
              >
                {item}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Empty State */}
      {selected.length === 0 && (
        <p className="text-xs text-slate-500 dark:text-slate-400">
          No dependencies selected. Click "Add Dependency" to add services or resources.
        </p>
      )}
    </div>
  )
}
