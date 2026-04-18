import { createContext, useContext, useState, useCallback, useMemo, type ReactNode } from 'react'
import type { Transport } from '@connectrpc/connect'

import { createServicesClient } from '@/lib/connectClient'
import type { Service } from '@/types'

/**
 * Operation type for service lifecycle management.
 */
export type ServiceOperation = 'start' | 'stop' | 'restart'

/**
 * Operation state for a service.
 */
export type OperationState = 'idle' | 'starting' | 'stopping' | 'restarting'

/**
 * Result of a single service operation.
 */
export interface ServiceOperationResult {
  name: string
  success: boolean
  error?: string
  duration?: string
}

/**
 * Result of a bulk operation.
 */
export interface BulkOperationResult {
  success: boolean
  message: string
  services: ServiceOperationResult[]
  successCount: number
  failureCount: number
  duration: string
}

/**
 * State for tracking operations on services.
 */
interface OperationTracker {
  // Map of service name to current operation state
  states: Map<string, OperationState>
  // Whether a bulk operation is in progress
  bulkInProgress: boolean
  // Current bulk operation type
  bulkOperation: ServiceOperation | null
}

/**
 * Context value type for service operations
 */
interface ServiceOperationsContextValue {
  // Single service operations
  startService: (serviceName: string) => Promise<boolean>
  stopService: (serviceName: string) => Promise<boolean>
  restartService: (serviceName: string) => Promise<boolean>
  executeOperation: (serviceName: string, operation: ServiceOperation) => Promise<boolean>
  
  // Bulk operations
  startAll: () => Promise<BulkOperationResult | null>
  stopAll: () => Promise<BulkOperationResult | null>
  restartAll: () => Promise<BulkOperationResult | null>
  executeBulkOperation: (operation: ServiceOperation) => Promise<BulkOperationResult | null>
  
  // State queries
  getOperationState: (serviceName: string) => OperationState
  getEffectiveOperationState: (serviceName: string) => OperationState
  isOperationInProgress: (serviceName: string) => boolean
  isBulkOperationInProgress: () => boolean
  getAvailableActions: (service: Service) => ServiceOperation[]
  canPerformAction: (service: Service, action: ServiceOperation) => boolean
  
  // State
  error: string | null
  lastResult: BulkOperationResult | null
  bulkOperation: ServiceOperation | null
  
  // Clear error
  clearError: () => void
}

const ServiceOperationsContext = createContext<ServiceOperationsContextValue | null>(null)

interface ServiceOperationsProviderProps {
  children: ReactNode
  /**
   * Optional Connect transport override for tests. Production code never
   * passes this.
   */
  transport?: Transport
}

/**
 * Provider for service operations context.
 * Wraps the application to share operation state across all components.
 */
export function ServiceOperationsProvider({ children, transport }: ServiceOperationsProviderProps) {
  const [tracker, setTracker] = useState<OperationTracker>({
    states: new Map(),
    bulkInProgress: false,
    bulkOperation: null,
  })
  const [error, setError] = useState<string | null>(null)
  const [lastResult, setLastResult] = useState<BulkOperationResult | null>(null)

  // Memoize the Connect client so callbacks below have a stable dep.
  const client = useMemo(() => createServicesClient(transport), [transport])

  /**
   * Synthesise a BulkOperationResult from the proto OperationResult.
   *
   * The new RPC returns a single OperationResult (success/message/op_id);
   * the legacy REST endpoint returned a richer per-service breakdown that
   * a few existing components type against. No component currently reads
   * `services[]` (verified via grep), so we keep the shape but fill it
   * with an empty array. Counts derive from the success flag so the
   * existing toast logic ("X succeeded, Y failed") keeps working.
   */
  const synthesizeBulkResult = useCallback((
    payload: { success: boolean; message: string },
  ): BulkOperationResult => ({
    success: payload.success,
    message: payload.message,
    services: [],
    successCount: payload.success ? 1 : 0,
    failureCount: payload.success ? 0 : 1,
    duration: '',
  }), [])

  /**
   * Get the operation state for a specific service.
   */
  const getOperationState = useCallback((serviceName: string): OperationState => {
    return tracker.states.get(serviceName) ?? 'idle'
  }, [tracker.states])

  /**
   * Get the effective operation state for a service, including bulk operation fallback.
   * This is the SINGLE SOURCE OF TRUTH for determining what operation state to display.
   * 
   * Priority:
   * 1. Individual operation state (if not idle)
   * 2. Bulk operation state (if bulk operation in progress)
   * 3. 'idle' (no operation)
   */
  const getEffectiveOperationState = useCallback((serviceName: string): OperationState => {
    const individualState = tracker.states.get(serviceName) ?? 'idle'
    
    // Individual operation takes priority
    if (individualState !== 'idle') {
      return individualState
    }
    
    // During bulk operations, derive operation state from bulk operation type
    if (tracker.bulkInProgress && tracker.bulkOperation) {
      switch (tracker.bulkOperation) {
        case 'stop':
          return 'stopping'
        case 'start':
          return 'starting'
        case 'restart':
          return 'restarting'
      }
    }
    
    return 'idle'
  }, [tracker.states, tracker.bulkInProgress, tracker.bulkOperation])

  /**
   * Check if any operation is in progress for a service.
   */
  const isOperationInProgress = useCallback((serviceName: string): boolean => {
    return getOperationState(serviceName) !== 'idle'
  }, [getOperationState])

  /**
   * Check if the bulk operation is in progress.
   */
  const isBulkOperationInProgress = useCallback((): boolean => {
    return tracker.bulkInProgress
  }, [tracker.bulkInProgress])

  /**
   * Set operation state for a service.
   */
  const setOperationState = useCallback((serviceName: string, state: OperationState) => {
    setTracker(prev => {
      const newStates = new Map(prev.states)
      if (state === 'idle') {
        newStates.delete(serviceName)
      } else {
        newStates.set(serviceName, state)
      }
      return { ...prev, states: newStates }
    })
  }, [])

  /**
   * Map operation to state.
   */
  const operationToState = (operation: ServiceOperation): OperationState => {
    switch (operation) {
      case 'start':
        return 'starting'
      case 'stop':
        return 'stopping'
      case 'restart':
        return 'restarting'
    }
  }

  /**
   * Execute a single service operation.
   */
  const executeOperation = useCallback(async (
    serviceName: string,
    operation: ServiceOperation
  ): Promise<boolean> => {
    // Check if operation already in progress
    if (isOperationInProgress(serviceName)) {
      setError(`Operation already in progress for ${serviceName}`)
      return false
    }

    setError(null)
    setOperationState(serviceName, operationToState(operation))

    try {
      // Route to the matching Connect RPC. The handler accepts an empty
      // service_name for bulk; here we always pass the explicit name and
      // let the handler fail with FAILED_PRECONDITION/NOT_FOUND/etc.
      switch (operation) {
        case 'start':
          await client.startService({ serviceName })
          break
        case 'stop':
          await client.stopService({ serviceName })
          break
        case 'restart':
          await client.restartService({ serviceName })
          break
      }
      return true
    } catch (err) {
      const message = err instanceof Error ? err.message : `Failed to ${operation} service`
      setError(message)
      console.error(`Error ${operation}ing service ${serviceName}:`, err)
      return false
    } finally {
      setOperationState(serviceName, 'idle')
    }
  }, [client, isOperationInProgress, setOperationState])

  /**
   * Start a service.
   */
  const startService = useCallback(async (serviceName: string): Promise<boolean> => {
    return executeOperation(serviceName, 'start')
  }, [executeOperation])

  /**
   * Stop a service.
   */
  const stopService = useCallback(async (serviceName: string): Promise<boolean> => {
    return executeOperation(serviceName, 'stop')
  }, [executeOperation])

  /**
   * Restart a service.
   */
  const restartService = useCallback(async (serviceName: string): Promise<boolean> => {
    return executeOperation(serviceName, 'restart')
  }, [executeOperation])

  /**
   * Execute a bulk operation on all applicable services.
   */
  const executeBulkOperation = useCallback(async (
    operation: ServiceOperation
  ): Promise<BulkOperationResult | null> => {
    if (tracker.bulkInProgress) {
      setError('Bulk operation already in progress')
      return null
    }

    setError(null)
    setLastResult(null)
    setTracker(prev => ({
      ...prev,
      bulkInProgress: true,
      bulkOperation: operation,
    }))

    try {
      // Bulk path: empty service_name routes through the same Connect
      // handler the single-name path uses. The Go adapter's bulk runner
      // returns a single OperationResult (e.g., "3 service(s) started, 0
      // failed"), which we lift into the legacy BulkOperationResult shape
      // for components still typed against it.
      let resp: { result?: { success: boolean; message: string } }
      switch (operation) {
        case 'start':
          resp = await client.startService({})
          break
        case 'stop':
          resp = await client.stopService({})
          break
        case 'restart':
          resp = await client.restartService({})
          break
      }
      const payload = resp.result ?? { success: false, message: '' }
      const result = synthesizeBulkResult({
        success: payload.success,
        message: payload.message,
      })
      setLastResult(result)
      return result
    } catch (err) {
      const message = err instanceof Error ? err.message : `Failed to ${operation} all services`
      setError(message)
      console.error(`Error ${operation}ing all services:`, err)
      return null
    } finally {
      setTracker(prev => ({
        ...prev,
        bulkInProgress: false,
        bulkOperation: null,
      }))
    }
  }, [client, synthesizeBulkResult, tracker.bulkInProgress])

  /**
   * Start all stopped services.
   */
  const startAll = useCallback(async (): Promise<BulkOperationResult | null> => {
    return executeBulkOperation('start')
  }, [executeBulkOperation])

  /**
   * Stop all running services.
   */
  const stopAll = useCallback(async (): Promise<BulkOperationResult | null> => {
    return executeBulkOperation('stop')
  }, [executeBulkOperation])

  /**
   * Restart all services.
   */
  const restartAll = useCallback(async (): Promise<BulkOperationResult | null> => {
    return executeBulkOperation('restart')
  }, [executeBulkOperation])

  /**
   * Get available actions for a service based on its current status.
   * 
   * IMPORTANT: This function uses PROCESS status (running/stopped/etc),
   * NOT health status (healthy/unhealthy/degraded). A running but unhealthy
   * service should show Stop/Restart because the process IS running.
   * 
   * For process services (type: 'process'), the status can be:
   * - 'watching': Process is watching for file changes (can stop/restart)
   * - 'building': Process is currently building (can stop)
   * - 'built': Process completed build (can start to rebuild)
   * - 'failed': Process failed (can start to retry)
   */
  const getAvailableActions = useCallback((service: Service): ServiceOperation[] => {
    const status = service.local?.status ?? 'not-running'
    const actions: ServiceOperation[] = []

    // First, check if process appears to be running based on PID or port
    // This is a fallback for when status might not reflect actual state
    const hasRunningProcess = !!(service.local?.pid || service.local?.port)

    switch (status) {
      case 'stopped':
      case 'not-running':
      case 'built':  // Process service completed build - can start to rebuild
      case 'completed': // Process service completed task - can start to re-run
      case 'failed': // Process service failed - can start to retry
        actions.push('start')
        break
      case 'running':
      case 'ready':
      case 'watching': // Process service actively watching - can stop/restart
        // Process is running - show stop/restart regardless of health status
        actions.push('restart', 'stop')
        break
      case 'starting':
      case 'building': // Process service building - can stop to cancel
        // Allow stopping a stuck startup or build
        actions.push('stop')
        break
      case 'stopping':
        // No actions during stopping
        break
      case 'error':
        // Error state needs special handling:
        // If process is alive (has PID), show stop/restart
        // If process is dead (no PID), show start
        if (service.local?.pid) {
          actions.push('restart', 'stop')
        } else {
          actions.push('start')
        }
        break
      default:
        // For any unknown status, infer from process indicators
        // If we have a PID or port, assume the process is running
        if (hasRunningProcess) {
          actions.push('restart', 'stop')
        } else {
          actions.push('start')
        }
        break
    }

    return actions
  }, [])

  /**
   * Check if a specific action is available for a service.
   */
  const canPerformAction = useCallback((service: Service, action: ServiceOperation): boolean => {
    if (isOperationInProgress(service.name)) {
      return false
    }
    return getAvailableActions(service).includes(action)
  }, [getAvailableActions, isOperationInProgress])

  const value: ServiceOperationsContextValue = {
    // Single service operations
    startService,
    stopService,
    restartService,
    executeOperation,
    
    // Bulk operations
    startAll,
    stopAll,
    restartAll,
    executeBulkOperation,
    
    // State queries
    getOperationState,
    getEffectiveOperationState,
    isOperationInProgress,
    isBulkOperationInProgress,
    getAvailableActions,
    canPerformAction,
    
    // State
    error,
    lastResult,
    bulkOperation: tracker.bulkOperation,
    
    // Clear error
    clearError: useCallback(() => setError(null), []),
  }

  return (
    <ServiceOperationsContext.Provider value={value}>
      {children}
    </ServiceOperationsContext.Provider>
  )
}

/**
 * Hook for accessing service operations from the context.
 * Must be used within a ServiceOperationsProvider.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function useServiceOperations(): ServiceOperationsContextValue {
  const context = useContext(ServiceOperationsContext)
  if (!context) {
    throw new Error('useServiceOperations must be used within a ServiceOperationsProvider')
  }
  return context
}

// Re-export types for convenience
export type { ServiceOperationsContextValue }
