/**
 * useConsoleFilters - Manages console filter state and persistence
 */
import * as React from 'react'
import type { Service, HealthStatus } from '@/types'

// =============================================================================
// Types
// =============================================================================

export type LogLevel = 'info' | 'warning' | 'error'
export type FilterableLifecycleState = 'running' | 'stopped' | 'starting'
export type ConsoleServiceSelectionMode = 'all' | 'custom'

export interface SavedConsoleFiltersV1 {
  version: 1
  serviceSelectionMode: ConsoleServiceSelectionMode
  selectedServices?: string[]
  levelFilter: LogLevel[]
  stateFilter: FilterableLifecycleState[]
  healthFilter: HealthStatus[]
}

// =============================================================================
// Storage Helpers
// =============================================================================

const CONSOLE_FILTERS_STORAGE_KEY = 'console-filters-v1'

function isLogLevel(value: unknown): value is LogLevel {
  return value === 'info' || value === 'warning' || value === 'error'
}

function isFilterableLifecycleState(value: unknown): value is FilterableLifecycleState {
  return value === 'running' || value === 'stopped' || value === 'starting'
}

function isHealthStatus(value: unknown): value is HealthStatus {
  return value === 'healthy' || value === 'degraded' || value === 'unhealthy' || value === 'unknown'
}

function loadSavedConsoleFilters(): SavedConsoleFiltersV1 | null {
  if (globalThis.localStorage === undefined) {
    return null
  }

  try {
    const raw = globalThis.localStorage.getItem(CONSOLE_FILTERS_STORAGE_KEY)
    if (!raw) {
      return null
    }

    const parsed: unknown = JSON.parse(raw)
    if (typeof parsed !== 'object' || parsed === null) {
      return null
    }

    const record = parsed as Record<string, unknown>
    if (record.version !== 1) {
      return null
    }

    const serviceSelectionMode = record.serviceSelectionMode === 'custom' ? 'custom' : 'all'
    const selectedServices = Array.isArray(record.selectedServices)
      ? record.selectedServices.filter((v): v is string => typeof v === 'string')
      : undefined

    const levelFilter = Array.isArray(record.levelFilter)
      ? record.levelFilter.filter(isLogLevel)
      : []
    const stateFilter = Array.isArray(record.stateFilter)
      ? record.stateFilter.filter(isFilterableLifecycleState)
      : []
    const healthFilter = Array.isArray(record.healthFilter)
      ? record.healthFilter.filter(isHealthStatus)
      : []

    return {
      version: 1,
      serviceSelectionMode,
      selectedServices,
      levelFilter,
      stateFilter,
      healthFilter,
    }
  } catch {
    return null
  }
}

function saveConsoleFilters(value: SavedConsoleFiltersV1): void {
  if (globalThis.localStorage === undefined) {
    return
  }

  try {
    globalThis.localStorage.setItem(CONSOLE_FILTERS_STORAGE_KEY, JSON.stringify(value))
  } catch {
    // Ignore persistence failures
  }
}

// =============================================================================
// Hook
// =============================================================================

export interface UseConsoleFiltersResult {
  // Service selection
  serviceSelectionMode: ConsoleServiceSelectionMode
  selectedServices: Set<string>
  toggleService: (serviceName: string) => void
  
  // Level filter
  levelFilter: Set<LogLevel>
  toggleLevel: (level: LogLevel) => void
  
  // State filter
  stateFilter: Set<FilterableLifecycleState>
  toggleState: (state: FilterableLifecycleState) => void
  
  // Health filter
  healthFilter: Set<HealthStatus>
  toggleHealth: (status: HealthStatus) => void
}

export function useConsoleFilters(services: Service[]): UseConsoleFiltersResult {
  const savedFilters = React.useMemo(() => loadSavedConsoleFilters(), [])
  
  const [serviceSelectionMode, setServiceSelectionMode] = React.useState<ConsoleServiceSelectionMode>(
    () => savedFilters?.serviceSelectionMode ?? 'all'
  )
  
  const [selectedServices, setSelectedServices] = React.useState<Set<string>>(() => {
    // If mode is 'all' and services are available on first render, initialize with all services
    if ((savedFilters?.serviceSelectionMode ?? 'all') === 'all' && services.length > 0) {
      return new Set(services.map((s) => s.name))
    }
    // Otherwise use saved custom selection or empty set
    return new Set(savedFilters?.serviceSelectionMode === 'custom' ? (savedFilters?.selectedServices ?? []) : [])
  })
  
  const [levelFilter, setLevelFilter] = React.useState<Set<LogLevel>>(
    () => new Set(savedFilters?.levelFilter?.length ? savedFilters.levelFilter : ['info', 'warning', 'error'])
  )
  
  const [stateFilter, setStateFilter] = React.useState<Set<FilterableLifecycleState>>(
    () => new Set(savedFilters?.stateFilter?.length ? savedFilters.stateFilter : ['running', 'stopped', 'starting'])
  )
  
  const [healthFilter, setHealthFilter] = React.useState<Set<HealthStatus>>(
    () => new Set(savedFilters?.healthFilter?.length ? savedFilters.healthFilter : ['healthy', 'degraded', 'unhealthy', 'unknown'])
  )

  // Sync selected services with available services
  React.useEffect(() => {
    if (services.length > 0) {
      const currentServiceNames = new Set(services.map((s) => s.name))

      if (serviceSelectionMode === 'all') {
        setSelectedServices(currentServiceNames)
        return
      }

      // Custom selection: preserve user's choices, but drop services that no longer exist.
      setSelectedServices((prev) => {
        const next = new Set(Array.from(prev).filter((name) => currentServiceNames.has(name)))
        return next.size > 0 ? next : currentServiceNames
      })
    }
  }, [services, serviceSelectionMode])

  // Persist filter state to localStorage
  React.useEffect(() => {
    const currentServiceNames = new Set(services.map((s) => s.name))

    const isAllSelected =
      currentServiceNames.size > 0 &&
      selectedServices.size === currentServiceNames.size &&
      Array.from(selectedServices).every((name) => currentServiceNames.has(name))

    saveConsoleFilters({
      version: 1,
      serviceSelectionMode: isAllSelected ? 'all' : 'custom',
      selectedServices: isAllSelected ? undefined : Array.from(selectedServices).sort((a, b) => a.localeCompare(b)),
      levelFilter: Array.from(levelFilter),
      stateFilter: Array.from(stateFilter),
      healthFilter: Array.from(healthFilter),
    })
  }, [services, selectedServices, levelFilter, stateFilter, healthFilter])

  const toggleService = React.useCallback((serviceName: string) => {
    setServiceSelectionMode('custom')
    setSelectedServices((prev) => {
      const next = new Set(prev)
      if (next.has(serviceName)) {
        next.delete(serviceName)
      } else {
        next.add(serviceName)
      }
      return next
    })
  }, [])

  const toggleLevel = React.useCallback((level: LogLevel) => {
    setLevelFilter((prev) => {
      const next = new Set(prev)
      if (next.has(level)) {
        next.delete(level)
      } else {
        next.add(level)
      }
      return next
    })
  }, [])

  const toggleState = React.useCallback((state: FilterableLifecycleState) => {
    setStateFilter((prev) => {
      const next = new Set(prev)
      if (next.has(state)) {
        next.delete(state)
      } else {
        next.add(state)
      }
      return next
    })
  }, [])

  const toggleHealth = React.useCallback((status: HealthStatus) => {
    setHealthFilter((prev) => {
      const next = new Set(prev)
      if (next.has(status)) {
        next.delete(status)
      } else {
        next.add(status)
      }
      return next
    })
  }, [])

  return {
    serviceSelectionMode,
    selectedServices,
    toggleService,
    levelFilter,
    toggleLevel,
    stateFilter,
    toggleState,
    healthFilter,
    toggleHealth,
  }
}
