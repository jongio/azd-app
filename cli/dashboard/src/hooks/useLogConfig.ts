/**
 * useLogConfig - Connect-RPC hooks for Log Analytics table selection
 * and per-service log configuration.
 *
 * Wire migration note: previously hit GET `/api/azure/tables`,
 * GET `/api/azure/logs/config`, and PUT `/api/azure/logs/config`. All
 * three now route through `AzureService` (`listAzureTables`,
 * `getAzureLogConfig`, `saveAzureLogConfig`). The hook return surface
 * is preserved verbatim - including the `TableInfo.category` field,
 * which the proto does not carry per-table. We synthesise it here by
 * inverting the `categories[].tables[]` relation so the
 * `TableSelector` component (which groups by `table.category`) keeps
 * working without changes.
 */
import { useState, useCallback, useEffect, useMemo } from 'react'
import { ConnectError, type Client, type Transport } from '@connectrpc/connect'
import { create } from '@bufbuild/protobuf'

import { useBackendConnection } from '@/hooks/useBackendConnection'
import { createAzureClient } from '@/lib/connectClient'
import type { AzureService } from '@/gen/proto/azdapp/v1/azure_pb.js'
import {
  AzureLogConfigMode as ProtoAzureLogConfigMode,
  AzureResourceType as ProtoAzureResourceType,
  GetAzureLogConfigRequestSchema,
  ListAzureTablesRequestSchema,
  type ListAzureTablesResponse,
  SaveAzureLogConfigRequestSchema,
} from '@/gen/proto/azdapp/v1/azure_pb.js'

// =============================================================================
// Types (preserved from the legacy hook surface)
// =============================================================================

export interface TableInfo {
  name: string
  category: string
  description: string
  columns?: string[]
  recommended?: boolean
}

export interface TableCategory {
  name: string
  displayName: string
  tables: string[]
}

export interface TablesResponse {
  tables: TableInfo[]
  recommended: string[]
  workspace: string
  categories: TableCategory[]
}

export interface LogConfig {
  service: string
  mode: 'tables' | 'custom'
  tables?: string[]
  query?: string
  resourceType?: string
}

export interface UseAvailableTablesOptions {
  /** Resource type to filter/recommend tables (e.g. 'containerapp', 'appservice', 'function') */
  resourceType?: string
  /** Auto-fetch on mount */
  autoFetch?: boolean
  /** Test transport injection - production omits */
  transport?: Transport
}

export interface UseAvailableTablesReturn {
  tables: TableInfo[]
  categories: TableCategory[]
  recommended: string[]
  workspace: string
  isLoading: boolean
  error: string | null
  fetchTables: () => Promise<void>
}

export interface UseLogConfigOptions {
  serviceName: string
  autoFetch?: boolean
  transport?: Transport
}

export interface UseLogConfigReturn {
  config: LogConfig | null
  isLoading: boolean
  isSaving: boolean
  error: string | null
  fetchConfig: () => Promise<void>
  saveConfig: (options: { tables?: string[]; query?: string }) => Promise<boolean>
}

// =============================================================================
// Mappers (proto <-> dashboard)
// =============================================================================

/**
 * Map the dashboard's lowercase resource type string to the proto enum.
 * Unknowns fall back to UNSPECIFIED, which the server defaults to
 * CONTAINER_APP - same behaviour as the legacy REST endpoint.
 */
function resourceTypeToProto(rt: string | undefined): ProtoAzureResourceType {
  switch (rt) {
    case 'containerapp':
      return ProtoAzureResourceType.CONTAINER_APP
    case 'appservice':
      return ProtoAzureResourceType.APP_SERVICE
    case 'function':
      return ProtoAzureResourceType.FUNCTION_APP
    default:
      return ProtoAzureResourceType.UNSPECIFIED
  }
}

function modeFromProto(mode: ProtoAzureLogConfigMode): 'tables' | 'custom' {
  switch (mode) {
    case ProtoAzureLogConfigMode.TABLES:
      return 'tables'
    case ProtoAzureLogConfigMode.CUSTOM:
      return 'custom'
    default:
      // Server returns UNSPECIFIED only for never-configured services.
      // Surfacing as 'tables' matches legacy default (empty list).
      return 'tables'
  }
}

function modeToProto(mode: 'tables' | 'custom'): ProtoAzureLogConfigMode {
  return mode === 'custom' ? ProtoAzureLogConfigMode.CUSTOM : ProtoAzureLogConfigMode.TABLES
}

/**
 * Build the (tableName -> categoryName) lookup the dashboard uses to
 * drive `TableInfo.category`. The proto puts category membership on
 * `AzureTableCategory.tables`, not on the table itself, so we invert
 * once per response. Tables not in any category default to 'other'
 * (matching `TableSelector`'s fallback).
 */
function buildCategoryIndex(resp: ListAzureTablesResponse): Map<string, string> {
  const index = new Map<string, string>()
  for (const cat of resp.categories) {
    for (const tableName of cat.tables) {
      index.set(tableName, cat.name)
    }
  }
  return index
}

function protoTablesToDashboard(resp: ListAzureTablesResponse): TablesResponse {
  const categoryIndex = buildCategoryIndex(resp)
  return {
    tables: resp.tables.map((t) => ({
      name: t.name,
      // Prefer explicit category mapping; fall back to 'other' so the
      // TableSelector groups orphans into a visible bucket instead of
      // dropping them.
      category: categoryIndex.get(t.name) ?? 'other',
      description: t.description,
      recommended: t.recommended,
    })),
    recommended: resp.recommended,
    workspace: resp.workspace,
    categories: resp.categories.map((c) => ({
      name: c.name,
      displayName: c.displayName,
      tables: c.tables,
    })),
  }
}

function connectErrMessage(err: unknown, fallback: string): string {
  if (err instanceof ConnectError) return err.rawMessage || err.message
  if (err instanceof Error) return err.message
  return fallback
}

// =============================================================================
// useAvailableTables
// =============================================================================

/**
 * Fetch the Log Analytics tables available for a given resource type,
 * along with category groupings and the "recommended" subset.
 */
export function useAvailableTables({
  resourceType = 'containerapp',
  autoFetch = true,
  transport,
}: UseAvailableTablesOptions = {}): UseAvailableTablesReturn {
  const { connected } = useBackendConnection()
  const client = useMemo<Client<typeof AzureService>>(
    () => createAzureClient(transport),
    [transport],
  )

  const [tables, setTables] = useState<TableInfo[]>([])
  const [categories, setCategories] = useState<TableCategory[]>([])
  const [recommended, setRecommended] = useState<string[]>([])
  const [workspace, setWorkspace] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchTables = useCallback(async () => {
    if (!connected) return

    setIsLoading(true)
    setError(null)

    try {
      const resp = await client.listAzureTables(
        create(ListAzureTablesRequestSchema, {
          resourceType: resourceTypeToProto(resourceType),
        }),
      )
      const data = protoTablesToDashboard(resp)
      setTables(data.tables)
      setCategories(data.categories)
      setRecommended(data.recommended)
      setWorkspace(data.workspace)
    } catch (err) {
      setError(connectErrMessage(err, 'Failed to fetch tables'))
      setTables([])
      setCategories([])
      setRecommended([])
    } finally {
      setIsLoading(false)
    }
  }, [client, connected, resourceType])

  /* eslint-disable react-hooks/set-state-in-effect -- async fetch; setState happens asynchronously */
  useEffect(() => {
    if (autoFetch) void fetchTables()
  }, [autoFetch, fetchTables])
  /* eslint-enable react-hooks/set-state-in-effect */

  return {
    tables,
    categories,
    recommended,
    workspace,
    isLoading,
    error,
    fetchTables,
  }
}

// =============================================================================
// useLogConfig
// =============================================================================

/**
 * Manage per-service log configuration: read the current mode/tables/
 * query from the server and save user edits back. The hook intentionally
 * accepts `{tables?, query?}` and infers `mode` so callers don't have
 * to thread the discriminator through their own state.
 */
export function useLogConfig({
  serviceName,
  autoFetch = true,
  transport,
}: UseLogConfigOptions): UseLogConfigReturn {
  const { connected } = useBackendConnection()
  const client = useMemo<Client<typeof AzureService>>(
    () => createAzureClient(transport),
    [transport],
  )

  const [config, setConfig] = useState<LogConfig | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchConfig = useCallback(async () => {
    if (!connected || !serviceName) return

    setIsLoading(true)
    setError(null)

    try {
      const resp = await client.getAzureLogConfig(
        create(GetAzureLogConfigRequestSchema, { service: serviceName }),
      )
      setConfig({
        service: resp.service,
        mode: modeFromProto(resp.mode),
        tables: resp.tables,
        query: resp.query,
        resourceType: resp.resourceType,
      })
    } catch (err) {
      setError(connectErrMessage(err, 'Failed to fetch config'))
      setConfig(null)
    } finally {
      setIsLoading(false)
    }
  }, [client, connected, serviceName])

  const saveConfig = useCallback(
    async (options: { tables?: string[]; query?: string }): Promise<boolean> => {
      if (!connected) {
        setError('Backend connection lost')
        return false
      }
      if (!serviceName) {
        setError('Service name is required')
        return false
      }

      const { tables, query } = options
      // Mirror legacy validation: mode is inferred and at least one
      // payload must be present. The server enforces this too but
      // catching client-side gives a clearer error and avoids a round
      // trip.
      if (!query && (!tables || tables.length === 0)) {
        setError('Either tables or query is required')
        return false
      }

      setIsSaving(true)
      setError(null)

      try {
        const mode: 'tables' | 'custom' = query ? 'custom' : 'tables'
        const resp = await client.saveAzureLogConfig(
          create(SaveAzureLogConfigRequestSchema, {
            service: serviceName,
            mode: modeToProto(mode),
            // Only include the field matching the inferred mode. The
            // server rejects empty payloads with InvalidArgument either
            // way; this just keeps the wire request minimal.
            tables: query ? [] : tables ?? [],
            query: query ?? '',
          }),
        )
        setConfig({
          service: resp.service,
          mode: modeFromProto(resp.mode),
          tables: resp.tables,
          query: resp.query,
        })
        return true
      } catch (err) {
        setError(connectErrMessage(err, 'Failed to save config'))
        return false
      } finally {
        setIsSaving(false)
      }
    },
    [client, connected, serviceName],
  )

  /* eslint-disable react-hooks/set-state-in-effect -- async fetch; setState happens asynchronously */
  useEffect(() => {
    if (autoFetch && serviceName) void fetchConfig()
  }, [autoFetch, serviceName, fetchConfig])
  /* eslint-enable react-hooks/set-state-in-effect */

  return {
    config,
    isLoading,
    isSaving,
    error,
    fetchConfig,
    saveConfig,
  }
}

export default useLogConfig
