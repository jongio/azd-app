import { useState, useCallback } from 'react'
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
 * Hook for managing service lifecycle operations.
 * Provides functions for starting, stopping, and restarting services,
 * both individually and in bulk. Tracks operation state for UI feedback.
 */
export function useServiceOperations() {
  const [tracker, setTracker] = useState<OperationTracker>({
    states: new Map(),
    bulkInProgress: false,
    bulkOperation: null,
  })
  const [error, setError] = useState<string | null>(null)
  const [lastResult, setLastResult] = useState<BulkOperationResult | null>(null)

  /**
   * Get the operation state for a specific service.
   */
  const getOperationState = useCallback((serviceName: string): OperationState => {
    return tracker.states.get(serviceName) ?? 'idle'
  }, [tracker.states])

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
      const response = await fetch(
        `/api/services/${operation}?service=${encodeURIComponent(serviceName)}`,
        { method: 'POST' }
      )

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Unknown error' })) as { error?: string }
        throw new Error(errorData.error ?? `Failed to ${operation} service`)
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
  }, [isOperationInProgress, setOperationState])

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
      const response = await fetch(`/api/services/${operation}`, {
        method: 'POST',
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Unknown error' })) as { error?: string }
        throw new Error(errorData.error ?? `Failed to ${operation} all services`)
      }

      const result = await response.json() as BulkOperationResult
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
  }, [tracker.bulkInProgress])

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
   */
  const getAvailableActions = useCallback((service: Service): ServiceOperation[] => {
    const status = service.local?.status ?? 'not-running'
    const actions: ServiceOperation[] = []

    switch (status) {
      case 'stopped':
      case 'not-running':
      case 'error':
        actions.push('start')
        break
      case 'running':
      case 'ready':
        actions.push('restart', 'stop')
        break
      case 'starting':
        // Limited actions during transition
        actions.push('stop')
        break
      case 'stopping':
        // No actions during stopping
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

  return {
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
}
