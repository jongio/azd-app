import * as React from 'react'
import { Search, Eye, EyeOff, Lock, Copy, Check, Settings2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Select } from '@/components/ui/select'
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from '@/components/ui/table'
import { useClipboard } from '@/hooks/useClipboard'
import type { Service } from '@/types'

/** Aggregated environment variable with service associations */
interface AggregatedEnvVar {
  /** Environment variable name */
  name: string
  /** Environment variable value */
  value: string
  /** List of services that use this variable */
  services: string[]
  /** Whether the variable is considered sensitive */
  isSensitive: boolean
}

/** Props for the main EnvironmentPanel component */
interface EnvironmentPanelProps {
  /** Services data containing environment variables */
  services: Service[]
  /** Additional class names */
  className?: string
  /** Data test ID for testing */
  'data-testid'?: string
}

/** Props for ServiceBadgeGroup */
interface ServiceBadgeGroupProps {
  /** List of service names */
  services: string[]
  /** Maximum badges to show before "+N more" */
  maxVisible?: number
  /** Service to highlight */
  highlightedService?: string | null
}

/** Props for EmptyState */
interface EmptyStateProps {
  /** Whether filters are currently active */
  hasFilters: boolean
  /** Callback to clear all filters */
  onClearFilters: () => void
}

/** Props for EnvironmentRow */
interface EnvironmentRowProps {
  /** Environment variable data */
  variable: AggregatedEnvVar
  /** Whether to show the value or mask it */
  showValue: boolean
  /** Whether this row's value was just copied */
  copied: boolean
  /** Callback when value is copied */
  onCopy: () => void
  /** Currently selected service filter */
  selectedService: string | null
}

/** Sensitive patterns for detecting sensitive environment variables */
const SENSITIVE_PATTERNS = [
  'key',
  'secret',
  'password',
  'token',
  'credential',
  'auth',
  'api_key',
  'apikey',
  'private',
  'cert',
  'connection_string',
  'connectionstring',
]

/**
 * Determines if an environment variable name indicates a sensitive value
 */
function isSensitiveVariable(name: string): boolean {
  const lowerName = name.toLowerCase()
  return SENSITIVE_PATTERNS.some(pattern => lowerName.includes(pattern))
}

/**
 * Aggregates environment variables from all services
 * Groups by variable name, collects all services that use each variable
 */
function aggregateEnvironmentVariables(services: Service[]): AggregatedEnvVar[] {
  const envMap = new Map<string, AggregatedEnvVar>()

  for (const service of services) {
    const envVars = service.environmentVariables ?? {}

    for (const [name, value] of Object.entries(envVars)) {
      const existing = envMap.get(name)

      if (existing) {
        // Variable exists - add service to list
        if (!existing.services.includes(service.name)) {
          existing.services.push(service.name)
        }
      } else {
        // New variable
        envMap.set(name, {
          name,
          value,
          services: [service.name],
          isSensitive: isSensitiveVariable(name),
        })
      }
    }
  }

  // Sort alphabetically by variable name
  return Array.from(envMap.values()).sort((a, b) =>
    a.name.localeCompare(b.name)
  )
}

/**
 * Filters environment variables based on search query and service selection
 */
function filterEnvironmentVariables(
  variables: AggregatedEnvVar[],
  searchQuery: string,
  selectedService: string | null
): AggregatedEnvVar[] {
  return variables.filter(envVar => {
    // Service filter
    if (selectedService && !envVar.services.includes(selectedService)) {
      return false
    }

    // Search filter (name OR value)
    if (searchQuery) {
      const query = searchQuery.toLowerCase()
      const matchesName = envVar.name.toLowerCase().includes(query)
      const matchesValue = envVar.value.toLowerCase().includes(query)
      if (!matchesName && !matchesValue) {
        return false
      }
    }

    return true
  })
}

/**
 * ServiceBadgeGroup - Displays a group of service badges with overflow handling
 */
function ServiceBadgeGroup({
  services,
  maxVisible = 3,
  highlightedService,
}: ServiceBadgeGroupProps) {
  const visible = services.slice(0, maxVisible)
  const overflow = services.length - maxVisible

  return (
    <div className="flex flex-wrap gap-1.5">
      {visible.map(service => (
        <Badge
          key={service}
          variant={service === highlightedService ? 'default' : 'secondary'}
          className="text-xs"
        >
          {service}
        </Badge>
      ))}
      {overflow > 0 && (
        <Badge variant="outline" className="text-xs">
          +{overflow} more
        </Badge>
      )}
    </div>
  )
}

/**
 * EmptyState - Displayed when no environment variables match current filters
 */
function EmptyState({ hasFilters, onClearFilters }: EmptyStateProps) {
  return (
    <div
      className="flex flex-col items-center justify-center py-12 px-4 text-center"
      role="status"
      aria-live="polite"
    >
      <Settings2 className="h-12 w-12 text-muted-foreground/50 mb-4" aria-hidden="true" />
      <h3 className="text-lg font-medium text-foreground mb-2">
        {hasFilters ? 'No Results Found' : 'No Environment Variables'}
      </h3>
      <p className="text-sm text-muted-foreground mb-4">
        {hasFilters
          ? 'No variables match your current search or filter criteria.'
          : "Services haven't defined any environment variables."}
      </p>
      {hasFilters && (
        <Button variant="outline" size="sm" onClick={onClearFilters}>
          Clear Filters
        </Button>
      )}
    </div>
  )
}

/**
 * EnvironmentRow - A single row in the environment variables table
 */
function EnvironmentRow({
  variable,
  showValue,
  copied,
  onCopy,
  selectedService,
}: EnvironmentRowProps) {
  const displayValue =
    variable.isSensitive && !showValue ? '••••••••••••' : variable.value

  return (
    <TableRow className="group">
      <TableCell className="font-medium">
        <div className="flex items-center gap-2">
          {variable.isSensitive && (
            <Lock
              className="h-3.5 w-3.5 text-amber-500 shrink-0"
              aria-label="Sensitive value"
              role="img"
            />
          )}
          <span className="font-mono text-sm">{variable.name}</span>
        </div>
      </TableCell>

      <TableCell>
        <div className="flex items-center gap-2">
          <code className="text-sm font-mono text-muted-foreground truncate max-w-md">
            {displayValue}
          </code>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={onCopy}
            aria-label={
              copied ? `${variable.name} copied` : `Copy ${variable.name} value`
            }
            aria-live="polite"
            className={cn(
              'h-7 w-7 shrink-0',
              'opacity-0 group-hover:opacity-100 focus:opacity-100',
              'transition-opacity',
              copied && 'text-green-600 dark:text-green-500'
            )}
          >
            {copied ? (
              <Check className="h-4 w-4" aria-hidden="true" />
            ) : (
              <Copy className="h-4 w-4" aria-hidden="true" />
            )}
          </Button>
        </div>
      </TableCell>

      <TableCell>
        <ServiceBadgeGroup
          services={variable.services}
          highlightedService={selectedService}
        />
      </TableCell>
    </TableRow>
  )
}

/**
 * EnvironmentPanel - Main component for displaying aggregated environment variables
 *
 * Features:
 * - Aggregated view of environment variables across all services
 * - Search/filter by variable name or value
 * - Service filter dropdown
 * - Show/Hide values toggle (masks sensitive values by default)
 * - Copy to clipboard with visual feedback
 * - Sensitive value detection
 * - Full keyboard accessibility
 * - WCAG 2.1 AA compliant
 */
export function EnvironmentPanel({
  services,
  className,
  'data-testid': testId,
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
    () => [...new Set(services.map(s => s.name))].sort(),
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

  const handleSearchChange = React.useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setSearchQuery(e.target.value)
    },
    []
  )

  const handleServiceChange = React.useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      setSelectedService(e.target.value === 'all' ? null : e.target.value)
    },
    []
  )

  const handleToggleShowValues = React.useCallback(() => {
    setShowValues(prev => !prev)
  }, [])

  const handleClearFilters = React.useCallback(() => {
    setSearchQuery('')
    setSelectedService(null)
  }, [])

  const handleSearchKeyDown = React.useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Escape') {
        setSearchQuery('')
      }
    },
    []
  )

  const hasFilters = Boolean(debouncedSearch || selectedService)

  return (
    <section
      aria-labelledby="env-panel-title"
      className={cn('bg-card rounded-lg border border-border', className)}
      data-testid={testId}
    >
      {/* Toolbar */}
      <div className="flex items-center justify-between p-4 border-b border-border">
        <h2
          id="env-panel-title"
          className="text-sm font-semibold text-foreground"
        >
          Environment Variables
        </h2>
        <Button
          variant="ghost"
          size="sm"
          onClick={handleToggleShowValues}
          aria-pressed={showValues}
          aria-label={showValues ? 'Hide sensitive values' : 'Show sensitive values'}
          className="gap-2"
        >
          {showValues ? (
            <>
              <EyeOff className="h-4 w-4" aria-hidden="true" />
              Hide Values
            </>
          ) : (
            <>
              <Eye className="h-4 w-4" aria-hidden="true" />
              Show Values
            </>
          )}
        </Button>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3 p-4 border-b border-border">
        <div className="relative flex-1">
          <Search
            className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground pointer-events-none"
            aria-hidden="true"
          />
          <Input
            type="search"
            placeholder="Search variables..."
            value={searchQuery}
            onChange={handleSearchChange}
            onKeyDown={handleSearchKeyDown}
            className="pl-9"
            aria-label="Search environment variables"
            aria-describedby="env-search-hint"
          />
          <span id="env-search-hint" className="sr-only">
            Search by variable name or value
          </span>
        </div>

        <Select
          value={selectedService ?? 'all'}
          onChange={handleServiceChange}
          className="w-[180px]"
          aria-label="Filter by service"
        >
          <option value="all">All Services</option>
          {availableServices.map(service => (
            <option key={service} value={service}>
              {service}
            </option>
          ))}
        </Select>

        <div
          className="text-xs text-muted-foreground whitespace-nowrap"
          aria-live="polite"
          aria-atomic="true"
        >
          {filteredVars.length === aggregatedVars.length
            ? `${aggregatedVars.length} variables`
            : `${filteredVars.length} of ${aggregatedVars.length} variables`}
        </div>
      </div>

      {/* Table or Empty State */}
      {filteredVars.length === 0 ? (
        <EmptyState hasFilters={hasFilters} onClearFilters={handleClearFilters} />
      ) : (
        <Table aria-label="Environment variables">
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead scope="col" className="w-[250px]">
                Variable
              </TableHead>
              <TableHead scope="col">Value</TableHead>
              <TableHead scope="col" className="w-[250px]">
                Services
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filteredVars.map(envVar => (
              <EnvironmentRow
                key={envVar.name}
                variable={envVar}
                showValue={showValues}
                copied={copiedField === envVar.name}
                onCopy={() => void handleCopy(envVar.name, envVar.value)}
                selectedService={selectedService}
              />
            ))}
          </TableBody>
        </Table>
      )}
    </section>
  )
}

// Export helper functions for testing
export { isSensitiveVariable, aggregateEnvironmentVariables, filterEnvironmentVariables }
export type { AggregatedEnvVar, EnvironmentPanelProps }
