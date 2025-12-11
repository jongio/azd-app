/**
 * LogConfigPanel - Modal panel for configuring log sources per service
 * Provides table selection mode and custom KQL query mode.
 */
import * as React from 'react'
import { X, Settings2, Table2, Code, Loader2, AlertCircle, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useEscapeKey } from '@/hooks/useEscapeKey'
import { useAvailableTables, useLogConfig } from '@/hooks/useLogConfig'
import { TableSelector } from './TableSelector'
import { KqlQueryInput } from './KqlQueryInput'

// =============================================================================
// Types
// =============================================================================

export interface LogConfigPanelProps {
  /** Service name to configure */
  serviceName: string
  /** Resource type for table recommendations */
  resourceType?: string
  /** Whether panel is visible */
  isOpen: boolean
  /** Close panel callback */
  onClose: () => void
  /** Callback after successful save */
  onSave?: () => void
  /** Additional class names */
  className?: string
}

type ConfigMode = 'tables' | 'custom'

// =============================================================================
// Constants
// =============================================================================

const PANEL_WIDTH = 520

// =============================================================================
// LogConfigPanel Component
// =============================================================================

export function LogConfigPanel({
  serviceName,
  resourceType = 'containerapp',
  isOpen,
  onClose,
  onSave,
  className,
}: LogConfigPanelProps) {
  const panelRef = React.useRef<HTMLDivElement>(null)

  // Hooks for data fetching
  const {
    tables,
    categories,
    recommended,
    isLoading: isLoadingTables,
    error: tablesError,
    fetchTables,
  } = useAvailableTables({ resourceType, autoFetch: false })

  const {
    config,
    isLoading: isLoadingConfig,
    isSaving,
    error: configError,
    fetchConfig,
    saveConfig,
  } = useLogConfig({ serviceName, autoFetch: false })

  // Local state
  const [mode, setMode] = React.useState<ConfigMode>('tables')
  const [selectedTables, setSelectedTables] = React.useState<string[]>([])
  const [customQuery, setCustomQuery] = React.useState('')
  const [saveSuccess, setSaveSuccess] = React.useState(false)

  // Close on Escape
  useEscapeKey(onClose, isOpen)

  // Fetch data when panel opens
  React.useEffect(() => {
    if (isOpen) {
      void fetchTables()
      void fetchConfig()
      setSaveSuccess(false)
    }
  }, [isOpen, fetchTables, fetchConfig])

  // Sync local state with fetched config
  React.useEffect(() => {
    if (config) {
      setMode(config.mode || 'tables')
      setSelectedTables(config.tables || [])
      setCustomQuery(config.query || '')
    }
  }, [config])

  // Focus management
  React.useEffect(() => {
    if (isOpen && panelRef.current) {
      const closeButton = panelRef.current.querySelector<HTMLButtonElement>('[data-close-button]')
      closeButton?.focus()
    }
  }, [isOpen])

  // Handle save
  const handleSave = async () => {
    // Save with either tables or query - mode is inferred by backend
    const success = await saveConfig({
      tables: mode === 'tables' ? selectedTables : undefined,
      query: mode === 'custom' ? customQuery : undefined,
    })

    if (success) {
      setSaveSuccess(true)
      setTimeout(() => setSaveSuccess(false), 2000)
      onSave?.()
    }
  }

  // Handle test query (for custom mode)
  const handleTestQuery = () => {
    // TODO: Implement query testing
    console.log('Test query:', customQuery)
  }

  // Check if form is valid
  const isValid = mode === 'tables' 
    ? selectedTables.length > 0 
    : customQuery.trim().length > 0

  // Check if form has changes
  const hasChanges = React.useMemo(() => {
    if (!config) return true
    if (mode !== config.mode) return true
    if (mode === 'tables') {
      const configTables = config.tables || []
      if (selectedTables.length !== configTables.length) return true
      return !selectedTables.every(t => configTables.includes(t))
    }
    return customQuery !== (config.query || '')
  }, [config, mode, selectedTables, customQuery])

  const isLoading = isLoadingTables || isLoadingConfig
  const error = tablesError || configError

  if (!isOpen) {
    return null
  }

  return (
    <>
      {/* Backdrop - using solid opacity instead of blur for performance */}
      <div
        className="fixed inset-0 z-40 bg-black/50 dark:bg-black/70 animate-fade-in"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Panel */}
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="log-config-panel-title"
        style={{ width: PANEL_WIDTH, willChange: 'transform, opacity' }}
        className={cn(
          'fixed right-0 top-0 z-50 h-screen',
          'bg-white dark:bg-slate-900',
          'border-l border-slate-200 dark:border-slate-700',
          'shadow-2xl',
          'flex flex-col',
          'animate-slide-in-right',
          className
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between gap-3 p-4 border-b border-slate-200 dark:border-slate-700 shrink-0">
          <div className="flex items-center gap-3 min-w-0">
            <div className="w-9 h-9 rounded-lg flex items-center justify-center bg-cyan-100 dark:bg-cyan-500/20 shrink-0">
              <Settings2 className="w-5 h-5 text-cyan-600 dark:text-cyan-400" />
            </div>
            <div className="min-w-0">
              <h2
                id="log-config-panel-title"
                className="text-lg font-semibold text-slate-900 dark:text-slate-100 truncate"
              >
                Configure Logs
              </h2>
              <p className="text-sm text-slate-500 dark:text-slate-400 truncate">
                {serviceName}
              </p>
            </div>
          </div>
          <button
            type="button"
            data-close-button
            onClick={onClose}
            className="p-2 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors shrink-0"
            aria-label="Close panel"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Mode Toggle */}
        <div className="p-4 border-b border-slate-200 dark:border-slate-700 shrink-0">
          <div className="flex items-center gap-2 p-1 bg-slate-100 dark:bg-slate-800 rounded-lg">
            <button
              type="button"
              onClick={() => setMode('tables')}
              className={cn(
                'flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-md text-sm font-medium',
                'transition-colors duration-200',
                mode === 'tables'
                  ? 'bg-white dark:bg-slate-700 text-cyan-600 dark:text-cyan-400 shadow-sm'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200',
              )}
            >
              <Table2 className="w-4 h-4" />
              Tables
            </button>
            <button
              type="button"
              onClick={() => setMode('custom')}
              className={cn(
                'flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-md text-sm font-medium',
                'transition-colors duration-200',
                mode === 'custom'
                  ? 'bg-white dark:bg-slate-700 text-cyan-600 dark:text-cyan-400 shadow-sm'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200',
              )}
            >
              <Code className="w-4 h-4" />
              Custom KQL
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4">
          {/* Loading State */}
          {isLoading && (
            <div className="flex flex-col items-center justify-center py-12">
              <Loader2 className="w-8 h-8 animate-spin text-cyan-500 mb-4" />
              <p className="text-slate-500 dark:text-slate-400">
                Loading configuration...
              </p>
            </div>
          )}

          {/* Error State */}
          {!isLoading && error && (
            <div className="flex items-start gap-3 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
              <AlertCircle className="w-5 h-5 text-red-500 shrink-0 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-red-800 dark:text-red-200">
                  Failed to load configuration
                </p>
                <p className="text-sm text-red-600 dark:text-red-300 mt-1">
                  {error}
                </p>
              </div>
            </div>
          )}

          {/* Tables Mode */}
          {!isLoading && !error && mode === 'tables' && (
            <div className="space-y-4">
              <div>
                <h3 className="text-sm font-medium text-slate-700 dark:text-slate-200 mb-1">
                  Select Tables
                </h3>
                <p className="text-xs text-slate-500 dark:text-slate-400 mb-3">
                  Choose which Log Analytics tables to query for this service.
                </p>
              </div>

              <TableSelector
                tables={tables}
                categories={categories}
                selectedTables={selectedTables}
                onSelectionChange={setSelectedTables}
                recommendedTables={recommended}
                isLoading={isLoadingTables}
              />
            </div>
          )}

          {/* Custom KQL Mode */}
          {!isLoading && !error && mode === 'custom' && (
            <div className="space-y-4">
              <div>
                <h3 className="text-sm font-medium text-slate-700 dark:text-slate-200 mb-1">
                  Custom KQL Query
                </h3>
                <p className="text-xs text-slate-500 dark:text-slate-400 mb-3">
                  Write a custom KQL query for advanced filtering.
                  Use <code className="px-1 py-0.5 bg-slate-100 dark:bg-slate-800 rounded">{'{serviceName}'}</code> and{' '}
                  <code className="px-1 py-0.5 bg-slate-100 dark:bg-slate-800 rounded">{'{timespan}'}</code> placeholders.
                </p>
              </div>

              <KqlQueryInput
                value={customQuery}
                onChange={setCustomQuery}
                onRunQuery={handleTestQuery}
                isCollapsed={false}
                placeholder={`ContainerAppConsoleLogs_CL
| where ContainerAppName_s =~ "{serviceName}"
| where TimeGenerated > ago({timespan})
| project TimeGenerated, Log_s, Stream_s
| order by TimeGenerated desc
| take 1000`}
              />

              <div className="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg">
                <p className="text-xs text-amber-700 dark:text-amber-300">
                  <strong>Tip:</strong> Make sure your query includes{' '}
                  <code className="px-1 py-0.5 bg-amber-100 dark:bg-amber-900/50 rounded">TimeGenerated</code>{' '}
                  in the projection for proper timestamp display.
                </p>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between gap-3 p-4 border-t border-slate-200 dark:border-slate-700 shrink-0 bg-slate-50 dark:bg-slate-800/50">
          <div className="text-sm text-slate-500 dark:text-slate-400">
            {mode === 'tables' && selectedTables.length > 0 && (
              <span>{selectedTables.length} table{selectedTables.length !== 1 ? 's' : ''} selected</span>
            )}
            {mode === 'custom' && customQuery.trim() && (
              <span>Custom query configured</span>
            )}
          </div>

          <div className="flex items-center gap-3">
            {saveSuccess && (
              <span className="flex items-center gap-1 text-sm text-green-600 dark:text-green-400">
                <Check className="w-4 h-4" />
                Saved
              </span>
            )}
            <button
              type="button"
              onClick={onClose}
              className={cn(
                'px-4 py-2 rounded-md text-sm font-medium',
                'text-slate-600 dark:text-slate-300',
                'hover:bg-slate-100 dark:hover:bg-slate-700',
                'transition-colors duration-200',
              )}
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleSave}
              disabled={!isValid || isSaving || !hasChanges}
              className={cn(
                'flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium',
                'bg-cyan-600 hover:bg-cyan-700 text-white',
                'transition-colors duration-200',
                'disabled:opacity-50 disabled:cursor-not-allowed',
              )}
            >
              {isSaving && <Loader2 className="w-4 h-4 animate-spin" />}
              Save Configuration
            </button>
          </div>
        </div>
      </div>
    </>
  )
}

export default LogConfigPanel
