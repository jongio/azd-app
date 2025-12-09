/**
 * KqlQueryInput - KQL query textarea with syntax highlighting placeholder
 * Provides a collapsible advanced query section with run and reset functionality.
 */
import * as React from 'react'
import { ChevronDown, ChevronRight, Play, RotateCcw } from 'lucide-react'
import { cn } from '@/lib/utils'

// =============================================================================
// Types
// =============================================================================

export interface KqlQueryInputProps {
  /** Current KQL query value */
  value: string
  /** Callback when query changes */
  onChange: (query: string) => void
  /** Callback when Run Query is clicked */
  onRunQuery: () => void
  /** Default query to restore on reset */
  defaultQuery?: string
  /** Placeholder text for the textarea */
  placeholder?: string
  /** Whether the section is collapsed */
  isCollapsed?: boolean
  /** Callback when collapsed state changes */
  onCollapsedChange?: (collapsed: boolean) => void
  /** Whether the input is disabled (e.g., during loading) */
  disabled?: boolean
  /** Additional class names */
  className?: string
}

// =============================================================================
// Constants
// =============================================================================

const DEFAULT_PLACEHOLDER = `ContainerAppConsoleLogs_CL
| where ContainerAppName_s == "{service}"
| where Log_s contains "error"
| project TimeGenerated, Log_s
| order by TimeGenerated desc`

// =============================================================================
// KqlQueryInput Component
// =============================================================================

export function KqlQueryInput({
  value,
  onChange,
  onRunQuery,
  defaultQuery = '',
  placeholder = DEFAULT_PLACEHOLDER,
  isCollapsed: controlledCollapsed,
  onCollapsedChange,
  disabled = false,
  className,
}: KqlQueryInputProps) {
  // Internal collapsed state when uncontrolled
  const [internalCollapsed, setInternalCollapsed] = React.useState(true)
  
  // Use controlled state if provided, otherwise internal
  const isCollapsed = controlledCollapsed ?? internalCollapsed
  const setIsCollapsed = (collapsed: boolean) => {
    if (onCollapsedChange) {
      onCollapsedChange(collapsed)
    } else {
      setInternalCollapsed(collapsed)
    }
  }

  const textareaRef = React.useRef<HTMLTextAreaElement>(null)

  const handleToggle = () => {
    setIsCollapsed(!isCollapsed)
  }

  const handleReset = () => {
    onChange(defaultQuery)
    // Focus textarea after reset
    if (textareaRef.current) {
      textareaRef.current.focus()
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Run query with Cmd/Ctrl + Enter
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault()
      if (!disabled) {
        onRunQuery()
      }
    }
  }

  const hasCustomQuery = value !== defaultQuery && value.trim() !== ''

  return (
    <div className={cn('space-y-2', className)}>
      {/* Collapsible Header */}
      <button
        type="button"
        onClick={handleToggle}
        className={cn(
          'flex items-center gap-2 w-full text-left',
          'text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider',
          'hover:text-slate-700 dark:hover:text-slate-300',
          'focus:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 rounded',
        )}
        aria-expanded={!isCollapsed}
        aria-controls="kql-query-section"
      >
        {isCollapsed ? (
          <ChevronRight className="w-4 h-4" />
        ) : (
          <ChevronDown className="w-4 h-4" />
        )}
        <span>Advanced Query</span>
        {hasCustomQuery && (
          <span className="ml-2 px-1.5 py-0.5 text-[10px] font-semibold rounded bg-cyan-100 dark:bg-cyan-900/50 text-cyan-700 dark:text-cyan-300">
            Modified
          </span>
        )}
      </button>

      {/* Collapsible Content */}
      {!isCollapsed && (
        <div 
          id="kql-query-section"
          className="p-3 rounded-lg bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 space-y-3"
        >
          {/* Textarea */}
          <textarea
            ref={textareaRef}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            disabled={disabled}
            rows={6}
            spellCheck={false}
            className={cn(
              'w-full px-3 py-2 rounded-md text-sm font-mono',
              'bg-white dark:bg-slate-800',
              'border border-slate-200 dark:border-slate-600',
              'text-slate-900 dark:text-slate-100',
              'placeholder:text-slate-400 dark:placeholder:text-slate-500',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-transparent',
              'resize-y min-h-[120px]',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
            aria-label="KQL query input"
          />

          {/* Hint */}
          <p className="text-xs text-slate-400 dark:text-slate-500">
            Press <kbd className="px-1 py-0.5 rounded bg-slate-200 dark:bg-slate-700 font-mono">Ctrl+Enter</kbd> to run
          </p>

          {/* Action Buttons */}
          <div className="flex items-center justify-between gap-3">
            <button
              type="button"
              onClick={onRunQuery}
              disabled={disabled}
              className={cn(
                'flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium',
                'bg-cyan-600 hover:bg-cyan-700 text-white',
                'transition-colors duration-200',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                'disabled:opacity-50 disabled:cursor-not-allowed',
              )}
            >
              <Play className="w-4 h-4" />
              Run Query
            </button>

            <button
              type="button"
              onClick={handleReset}
              disabled={disabled || !hasCustomQuery}
              className={cn(
                'flex items-center gap-2 px-3 py-2 rounded-md text-sm font-medium',
                'text-slate-600 dark:text-slate-300',
                'hover:bg-slate-100 dark:hover:bg-slate-700',
                'transition-colors duration-200',
                'focus:outline-none focus:ring-2 focus:ring-slate-500 focus:ring-offset-2',
                'disabled:opacity-50 disabled:cursor-not-allowed',
              )}
            >
              <RotateCcw className="w-4 h-4" />
              Reset
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

export default KqlQueryInput
