/**
 * EnvironmentPanel - Environment variables panel with modern styling
 * Displays aggregated environment variables across all services
 */
import * as React from 'react'
import { Search, Eye, EyeOff, Lock, Copy, Check, Settings2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { 
  aggregateEnvironmentVariables, 
  filterEnvironmentVariables,
  type AggregatedEnvVar 
} from '@/lib/env-utils'
import { useClipboard } from '@/hooks/useClipboard'
import type { Service } from '@/types'

// =============================================================================
// Types
// =============================================================================

export interface EnvironmentPanelProps {
  /** Services data containing environment variables */
  services: Service[]
  /** Additional class names */
  className?: string
}

// =============================================================================
// ServiceBadge Component
// =============================================================================

interface ServiceBadgeProps {
  name: string
  isHighlighted?: boolean
}

function ServiceBadge({ name, isHighlighted }: ServiceBadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium',
        isHighlighted
          ? 'bg-cyan-100 dark:bg-cyan-500/20 text-cyan-700 dark:text-cyan-300'
          : 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300'
      )}
    >
      {name}
    </span>
  )
}

// =============================================================================
// EmptyState Component
// =============================================================================

interface EmptyStateProps {
  hasFilters: boolean
  onClearFilters: () => void
}

function EmptyState({ hasFilters, onClearFilters }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-8 text-center">
      <div className="w-14 h-14 mb-4 rounded-xl bg-slate-100 dark:bg-slate-800 flex items-center justify-center">
        <Settings2 className="w-7 h-7 text-slate-400 dark:text-slate-500" />
      </div>
      <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100 mb-1">
        {hasFilters ? 'No Results Found' : 'No Environment Variables'}
      </h3>
      <p className="text-sm text-slate-500 dark:text-slate-400 mb-4 max-w-sm">
        {hasFilters
          ? 'No variables match your current search or filter criteria.'
          : "Services haven't defined any environment variables."}
      </p>
      {hasFilters && (
        <button
          type="button"
          onClick={onClearFilters}
          className="px-3 py-1.5 text-sm font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-lg transition-colors"
        >
          Clear Filters
        </button>
      )}
    </div>
  )
}

// =============================================================================
// EnvironmentRow Component
// =============================================================================

interface EnvironmentRowProps {
  variable: AggregatedEnvVar
  showValue: boolean
  copied: boolean
  onCopy: () => void
  selectedService: string | null
}

function EnvironmentRow({
  variable,
  showValue,
  copied,
  onCopy,
  selectedService,
}: EnvironmentRowProps) {
  const displayValue = variable.isSensitive && !showValue ? '••••••••••••' : variable.value
  const visibleServices = variable.services.slice(0, 3)
  const overflowCount = variable.services.length - 3

  return (
    <div className="group flex items-center gap-4 px-4 py-3 bg-white dark:bg-slate-800/50 hover:bg-slate-50 dark:hover:bg-slate-800 border-b border-slate-200 dark:border-slate-700 last:border-b-0 transition-colors">
      {/* Variable Name */}
      <div className="w-[220px] shrink-0">
        <div className="flex items-center gap-2">
          {variable.isSensitive && (
            <Lock className="w-3.5 h-3.5 text-amber-500" aria-label="Sensitive value" />
          )}
          <span className="font-mono text-sm text-slate-900 dark:text-slate-100 truncate">
            {variable.name}
          </span>
        </div>
      </div>

      {/* Value */}
      <div className="flex-1 min-w-0 flex items-center gap-2">
        <input
          type="text"
          value={displayValue}
          readOnly
          className="flex-1 px-3 py-1.5 text-sm font-mono bg-slate-100 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-slate-600 dark:text-slate-300 focus:outline-none"
        />
        <button
          type="button"
          onClick={onCopy}
          className={cn(
            'p-1.5 rounded-lg opacity-0 group-hover:opacity-100 focus:opacity-100 transition-all',
            'hover:bg-slate-200 dark:hover:bg-slate-700',
            copied && 'text-emerald-500 opacity-100'
          )}
          aria-label={copied ? `${variable.name} copied` : `Copy ${variable.name} value`}
        >
          {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
        </button>
      </div>

      {/* Services */}
      <div className="w-[220px] shrink-0 flex flex-wrap gap-1">
        {visibleServices.map((service) => (
          <ServiceBadge
            key={service}
            name={service}
            isHighlighted={service === selectedService}
          />
        ))}
        {overflowCount > 0 && (
          <span className="px-2 py-0.5 text-xs text-slate-500 dark:text-slate-400">
            +{overflowCount} more
          </span>
        )}
      </div>
    </div>
  )
}

// =============================================================================
// EnvironmentPanel Component
// =============================================================================

export function EnvironmentPanel({
  services,
  className,
}: EnvironmentPanelProps) {
  const [showValues, setShowValues] = React.useState(false)
  const [searchQuery, setSearchQuery] = React.useState('')
  const [selectedService, setSelectedService] = React.useState<string | null>(null)
  const [debouncedSearch, setDebouncedSearch] = React.useState('')
  const { copiedField, copyToClipboard } = useClipboard()

  // Debounce search input
  React.useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(searchQuery), 300)
    return () => clearTimeout(timer)
  }, [searchQuery])

  // Aggregate environment variables
  const aggregatedVars = React.useMemo(
    () => aggregateEnvironmentVariables(services),
    [services]
  )

  // Get unique service names
  const availableServices = React.useMemo(
    () => [...new Set(services.map((s) => s.name))].sort(),
    [services]
  )

  // Apply filters
  const filteredVars = React.useMemo(
    () => filterEnvironmentVariables(aggregatedVars, debouncedSearch, selectedService),
    [aggregatedVars, debouncedSearch, selectedService]
  )

  const handleCopy = React.useCallback(
    async (name: string, value: string) => {
      await copyToClipboard(value, name)
    },
    [copyToClipboard]
  )

  const handleClearFilters = React.useCallback(() => {
    setSearchQuery('')
    setSelectedService(null)
  }, [])

  const hasFilters = Boolean(debouncedSearch || selectedService)

  return (
    <div className={cn('bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden', className)}>
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-4 px-4 py-3 bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-700">
        {/* Search */}
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
          <input
            type="search"
            placeholder="Search variables..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-9 pr-4 py-2 text-sm bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
          />
        </div>

        {/* Service Filter */}
        <select
          value={selectedService ?? 'all'}
          onChange={(e) => setSelectedService(e.target.value === 'all' ? null : e.target.value)}
          className="px-3 py-2 text-sm bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
        >
          <option value="all">All Services</option>
          {availableServices.map((service) => (
            <option key={service} value={service}>
              {service}
            </option>
          ))}
        </select>

        {/* Count */}
        <span className="text-xs text-slate-500 dark:text-slate-400 whitespace-nowrap">
          {filteredVars.length === aggregatedVars.length
            ? `${aggregatedVars.length} variables`
            : `${filteredVars.length} of ${aggregatedVars.length}`}
        </span>

        {/* Show/Hide Values */}
        <button
          type="button"
          onClick={() => setShowValues(!showValues)}
          className={cn(
            'flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-lg transition-colors',
            showValues
              ? 'bg-cyan-100 dark:bg-cyan-500/20 text-cyan-700 dark:text-cyan-300'
              : 'bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-700'
          )}
        >
          {showValues ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
          <span>{showValues ? 'Hide' : 'Show'}</span>
        </button>
      </div>

      {/* Header */}
      <div className="flex items-center gap-4 px-4 py-2 bg-slate-50 dark:bg-slate-800/30 border-b border-slate-200 dark:border-slate-700 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
        <div className="w-[220px] shrink-0">Variable</div>
        <div className="flex-1">Value</div>
        <div className="w-[220px] shrink-0">Services</div>
      </div>

      {/* Content */}
      {filteredVars.length === 0 ? (
        <EmptyState hasFilters={hasFilters} onClearFilters={handleClearFilters} />
      ) : (
        <div className="max-h-[calc(100vh-300px)] overflow-y-auto">
          {filteredVars.map((envVar) => (
            <EnvironmentRow
              key={envVar.name}
              variable={envVar}
              showValue={showValues}
              copied={copiedField === envVar.name}
              onCopy={() => void handleCopy(envVar.name, envVar.value)}
              selectedService={selectedService}
            />
          ))}
        </div>
      )}
    </div>
  )
}
