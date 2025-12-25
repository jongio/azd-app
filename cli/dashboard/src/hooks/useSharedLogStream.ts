import { useEffect, useMemo, useRef, useState } from 'react'
import type { LogEntry } from '@/components/LogsPane'

type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'error'
type StateChangeCallback = (state: ConnectionState) => void

interface ManagerConfig {
  maxReconnectAttempts: number
  heartbeatInterval: number
  heartbeatTimeout: number
}

const DEFAULT_CONFIG: ManagerConfig = {
  maxReconnectAttempts: 10,
  heartbeatInterval: 30000, // 30s
  heartbeatTimeout: 5000,   // 5s
}

// Shared WebSocket connection manager for all services
// This prevents creating multiple connections and hitting resource limits

interface WebSocketHandlers {
  onopen: () => void
  onmessage: (event: MessageEvent) => void
  onerror: (event: Event) => void
  onclose: (event: CloseEvent) => void
}

class SharedLogStreamManager {
  protected ws: WebSocket | null = null
  private readonly wsHandlers = new WeakMap<WebSocket, WebSocketHandlers>()
  private readonly subscribers = new Map<string, Set<(entry: LogEntry) => void>>()
  private readonly stateSubscribers = new Set<StateChangeCallback>()
  private stateSubscriberTimeouts = new WeakMap<StateChangeCallback, ReturnType<typeof setTimeout>>()
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private disconnectTimer: ReturnType<typeof setTimeout> | null = null
  private heartbeatTimer: ReturnType<typeof setTimeout> | null = null
  private heartbeatTimeoutTimer: ReturnType<typeof setTimeout> | null = null
  private backoffDelay = 1000
  private readonly maxBackoff = 30000
  private readonly minBackoff = 1000
  private reconnectAttempts = 0
  private isConnecting = false
  private isDestroyed = false
  private currentState: ConnectionState = 'disconnected'
  private readonly config: ManagerConfig
  
  // Message buffer for late subscribers (last 100 messages)
  private messageBuffer: LogEntry[] = []
  private readonly maxBufferSize = 100
  
  // Sequence tracking for gap detection
  private readonly lastSeenSequence = new Map<string, number>()
  private readonly gapCallbacks = new Map<string, (gap: { start: number; end: number }) => void>()
  
  // Init message tracking for configuration
  protected initSent = false
  protected pendingInitConfigs = new Map<string, { since?: string }>()

  constructor(config: Partial<ManagerConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config }
  }

  subscribeToState(callback: StateChangeCallback): () => void {
    if (this.isDestroyed) return () => {}
    
    this.stateSubscribers.add(callback)
    
    // Immediately call with current state (use setTimeout to avoid sync issues)
    const timeoutId = setTimeout(() => {
      // Double-check still subscribed before calling
      if (this.stateSubscribers.has(callback)) {
        this.stateSubscriberTimeouts.delete(callback)
        try {
          callback(this.currentState)
        } catch (err) {
          console.error('[SharedLogStream] State subscriber initial callback error:', err)
          this.stateSubscribers.delete(callback)
        }
      }
    }, 0)
    
    // Track timeout for cleanup
    this.stateSubscriberTimeouts.set(callback, timeoutId)
    
    return () => {
      // Cancel pending initial callback if unsubscribed before it fires
      const pendingTimeout = this.stateSubscriberTimeouts.get(callback)
      if (pendingTimeout) {
        clearTimeout(pendingTimeout)
        this.stateSubscriberTimeouts.delete(callback)
      }
      this.stateSubscribers.delete(callback)
    }
  }

  private setState(newState: ConnectionState): void {
    if (this.currentState === newState) return
    
    this.currentState = newState
    
    // Call state subscribers safely, removing any that throw
    const toRemove: StateChangeCallback[] = []
    this.stateSubscribers.forEach(callback => {
      try {
        callback(newState)
      } catch (err) {
        console.error('[SharedLogStream] State subscriber error:', err)
        toRemove.push(callback)
      }
    })
    toRemove.forEach(cb => this.stateSubscribers.delete(cb))
  }

  getState(): ConnectionState {
    return this.currentState
  }

  subscribe(serviceName: string, callback: (entry: LogEntry) => void, config?: { onGapDetected?: (gap: { start: number; end: number }) => void; since?: string }): () => void {
    if (this.isDestroyed) return () => {}
    
    // Check if we need to start connection BEFORE adding subscriber
    const shouldConnect = this.subscribers.size === 0 && !this.ws && !this.isConnecting
    
    // Add subscriber (deduplicate by checking if already exists)
    if (!this.subscribers.has(serviceName)) {
      this.subscribers.set(serviceName, new Set())
    }
    const subs = this.subscribers.get(serviceName)
    if (!subs) return () => {} // Should never happen, but guard anyway
    
    // If already subscribed with this exact callback, don't add again
    if (subs.has(callback)) {
      console.warn(`[SharedLogStream] Duplicate subscription for service: ${serviceName}`)
    } else {
      subs.add(callback)
    }
    
    // Store init config for this service (if provided)
    if (config?.since) {
      this.pendingInitConfigs.set(serviceName, { since: config.since })
    }
    
    // Register gap detection callback
    if (config?.onGapDetected) {
      this.gapCallbacks.set(serviceName, config.onGapDetected)
    }
    
    // Replay buffered messages for this subscriber
    this.replayBufferedMessages(serviceName, callback)

    // Start connection after adding subscriber
    if (shouldConnect) {
      this.connect()
    }

    // Return unsubscribe function
    return () => {
      const serviceSubs = this.subscribers.get(serviceName)
      if (serviceSubs) {
        serviceSubs.delete(callback)
        if (serviceSubs.size === 0) {
          this.subscribers.delete(serviceName)
          this.gapCallbacks.delete(serviceName)
          this.lastSeenSequence.delete(serviceName)
        }
      }

      // Close connection if no more subscribers (debounced to avoid flapping)
      if (this.subscribers.size === 0) {
        // Clear any existing disconnect timer
        if (this.disconnectTimer) {
          clearTimeout(this.disconnectTimer)
        }
        // Small delay to handle rapid re-subscriptions
        this.disconnectTimer = setTimeout(() => {
          this.disconnectTimer = null
          if (this.subscribers.size === 0) {
            this.disconnect()
          }
        }, 100)
      }
    }
  }

  private replayBufferedMessages(serviceName: string, callback: (entry: LogEntry) => void): void {
    // Replay messages for this service or "all"
    this.messageBuffer
      .filter(entry => entry.service === serviceName || serviceName === 'all')
      .forEach(entry => {
        try {
          callback(entry)
        } catch (err) {
          console.error('[SharedLogStream] Error replaying message:', err)
        }
      })
  }

  protected getStreamUrl(): string {
    const protocol = globalThis.location.protocol === 'https:' ? 'wss:' : 'ws:'
    // Don't filter by service - get all logs and multiplex client-side
    return `${protocol}//${globalThis.location.host}/api/logs/stream`
  }

  private connect(): void {
    if (this.isDestroyed || this.ws || this.isConnecting) return
    
    // Check max reconnect attempts
    if (this.reconnectAttempts >= this.config.maxReconnectAttempts) {
      // Only log on first time hitting max attempts
      if (this.currentState !== 'error') {
        console.warn(`[SharedLogStream] Max reconnection attempts (${this.config.maxReconnectAttempts}) reached`)
      }
      this.setState('error')
      return
    }

    this.isConnecting = true
    this.setState('connecting')
    this.reconnectAttempts++
    
    const url = this.getStreamUrl()

    try {
      const ws = new WebSocket(url)
      
      // Store bound handlers so we can remove them later
      const handlers: WebSocketHandlers = {
        onopen: this.handleOpen.bind(this),
        onmessage: this.handleMessage.bind(this),
        onerror: this.handleError.bind(this),
        onclose: this.handleClose.bind(this),
      }

      ws.addEventListener('open', handlers.onopen)
      ws.addEventListener('message', handlers.onmessage)
      ws.addEventListener('error', handlers.onerror)
      ws.addEventListener('close', handlers.onclose)
      
      // Store handlers for cleanup using WeakMap
      this.wsHandlers.set(ws, handlers)

      this.ws = ws
    } catch (err) {
      this.isConnecting = false
      this.setState('error')
      // Only log on first attempt to avoid spam
      if (this.reconnectAttempts === 1) {
        console.warn('[SharedLogStream] Failed to create WebSocket:', err instanceof Error ? err.message : 'Unknown error')
      }
      if (this.subscribers.size > 0 && !this.isDestroyed) {
        this.scheduleReconnect()
      }
    }
  }

  private handleOpen(): void {
    if (this.isDestroyed) return
    
    const wasReconnecting = this.reconnectAttempts > 1
    this.isConnecting = false
    this.backoffDelay = this.minBackoff
    this.reconnectAttempts = 0
    this.setState('connected')
    
    // Send init message if we have pending configs and haven't sent yet
    this.sendInitMessage()
    
    this.startHeartbeat()
    
    // Only log initial connection and successful reconnections
    if (wasReconnecting) {
      console.warn('[SharedLogStream] Reconnected successfully')
    }
  }

  protected sendInitMessage(): void {
    // Base implementation does nothing - subclasses can override
    // This is called after WebSocket opens
  }

  private handleMessage(event: MessageEvent): void {
    // Reset heartbeat timeout on any message
    if (this.heartbeatTimeoutTimer) {
      clearTimeout(this.heartbeatTimeoutTimer)
      this.heartbeatTimeoutTimer = null
    }
    
    try {
      const message = JSON.parse(event.data as string) as unknown
      
      // Handle status messages (connection health updates from backend)
      if (typeof message === 'object' && message !== null && 'type' in message && message.type === 'status') {
        // Status messages don't get dispatched as log entries
        // They could be used for connection health indicators in the future
        return
      }
      
      // Handle batched log entries (array) or single entry
      const entries = Array.isArray(message) ? message : [message as LogEntry]
      
      entries.forEach((entry: LogEntry) => {
        // Validate log entry structure
        if (!entry || typeof entry !== 'object' || !entry.service) {
          console.warn('[SharedLogStream] Invalid log entry received:', entry)
          return
        }
        
        // Check for sequence gaps (only for Azure logs with sequence numbers)
        if (typeof entry.sequence === 'number') {
          const serviceId = String(entry.service)
          const lastSeq = this.lastSeenSequence.get(serviceId)
          if (lastSeq !== undefined && entry.sequence > lastSeq + 1) {
            // Gap detected!
            const gap = { start: lastSeq + 1, end: entry.sequence - 1 }
            console.warn(`[SharedLogStream] Gap detected for ${serviceId}: missing sequences ${gap.start}-${gap.end}`)
            
            // Call gap callback if registered
            const gapCallback = this.gapCallbacks.get(serviceId)
            if (gapCallback) {
              try {
                gapCallback(gap)
              } catch (err) {
                console.error('[SharedLogStream] Gap callback error:', err)
              }
            }
          }
          this.lastSeenSequence.set(serviceId, entry.sequence)
        }
        
        // Add to buffer (maintain max size)
        this.messageBuffer.push(entry)
        if (this.messageBuffer.length > this.maxBufferSize) {
          this.messageBuffer.shift()
        }
        
        const serviceName = String(entry.service)

        // Dispatch to subscribers of this specific service
        const serviceSubs = this.subscribers.get(serviceName)
        if (serviceSubs && serviceSubs.size > 0) {
          // Direct iteration without Array.from for better performance
          const toRemove: Array<(entry: LogEntry) => void> = []
          serviceSubs.forEach(callback => {
            try {
              callback(entry)
            } catch (err) {
              console.error('[SharedLogStream] Subscriber callback error:', err)
              toRemove.push(callback)
            }
          })
          // Remove callbacks that threw errors
          toRemove.forEach(cb => serviceSubs.delete(cb))
        }

        // Also dispatch to "all" subscribers
        const allSubs = this.subscribers.get('all')
        if (allSubs && allSubs.size > 0) {
          const toRemove: Array<(entry: LogEntry) => void> = []
          allSubs.forEach(callback => {
            try {
              callback(entry)
            } catch (err) {
              console.error('[SharedLogStream] Subscriber callback error:', err)
              toRemove.push(callback)
            }
          })
          toRemove.forEach(cb => allSubs.delete(cb))
        }
      })
    } catch (err) {
      console.error('[SharedLogStream] Failed to parse message:', err)
    }
  }

  private handleError(event: Event): void {
    // Don't set error state here - onclose will handle it
    // onerror is always followed by onclose
    // Don't log here to avoid spam - handleClose will log if needed
    void event // Silence unused parameter warning
  }

  private handleClose(event: CloseEvent): void {
    if (this.isDestroyed) return
    
    this.isConnecting = false
    const wasConnected = this.ws !== null
    
    // Clean up event listeners before nullifying
    if (this.ws) {
      const handlers = this.wsHandlers.get(this.ws)
      if (handlers) {
        this.ws.removeEventListener('open', handlers.onopen)
        this.ws.removeEventListener('message', handlers.onmessage)
        this.ws.removeEventListener('error', handlers.onerror)
        this.ws.removeEventListener('close', handlers.onclose)
        this.wsHandlers.delete(this.ws)
      }
    }
    
    this.ws = null
    this.stopHeartbeat()

    // Clean close or no subscribers - disconnect
    if (event.code === 1000 || this.subscribers.size === 0 || this.isDestroyed) {
      if (wasConnected && !this.isDestroyed) {
        this.setState('disconnected')
      }
      return
    }
    
    // Connection failed - try to reconnect
    if (!this.isDestroyed) {
      this.setState('error')
      if (this.subscribers.size > 0) {
        this.scheduleReconnect()
      }
    }
  }

  private startHeartbeat(): void {
    this.stopHeartbeat()
    
    this.heartbeatTimer = setInterval(() => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        this.stopHeartbeat()
        return
      }
      
      // Clear any existing timeout before creating new one
      if (this.heartbeatTimeoutTimer) {
        clearTimeout(this.heartbeatTimeoutTimer)
      }
      
      // Set timeout for this heartbeat interval
      // Will be cleared when we receive any message in handleMessage
      this.heartbeatTimeoutTimer = setTimeout(() => {
        console.warn('[SharedLogStream] Heartbeat timeout - no messages received')
        // Force reconnect if still connected
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
          this.ws.close(1000, 'Heartbeat timeout')
        }
      }, this.config.heartbeatTimeout)
    }, this.config.heartbeatInterval)
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
    if (this.heartbeatTimeoutTimer) {
      clearTimeout(this.heartbeatTimeoutTimer)
      this.heartbeatTimeoutTimer = null
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer || this.isDestroyed) return
    
    if (this.reconnectAttempts >= this.config.maxReconnectAttempts) {
      // Already logged in connect(), don't log again
      return
    }

    const delay = this.backoffDelay
    // Only log first 3 reconnection attempts to avoid spam
    if (this.reconnectAttempts <= 2) {
      console.warn(`[SharedLogStream] Reconnecting in ${Math.round(delay / 1000)}s (attempt ${this.reconnectAttempts + 1}/${this.config.maxReconnectAttempts})`)
    }
    
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      if (this.subscribers.size > 0 && !this.isDestroyed) {
        this.connect()
      }
    }, delay)

    // Exponential backoff with jitter
    this.backoffDelay = Math.min(
      this.backoffDelay * 2 + Math.random() * 1000,
      this.maxBackoff
    )
  }

  private disconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    
    if (this.disconnectTimer) {
      clearTimeout(this.disconnectTimer)
      this.disconnectTimer = null
    }
    
    this.stopHeartbeat()

    if (this.ws) {
      const currentWs = this.ws
      
      // Clean up event listeners using WeakMap
      const handlers = this.wsHandlers.get(currentWs)
      if (handlers) {
        currentWs.removeEventListener('open', handlers.onopen)
        currentWs.removeEventListener('message', handlers.onmessage)
        currentWs.removeEventListener('error', handlers.onerror)
        currentWs.removeEventListener('close', handlers.onclose)
        this.wsHandlers.delete(currentWs)
      }
      
      this.ws = null
      
      // Only close OPEN connections to avoid warnings
      if (currentWs.readyState === WebSocket.OPEN) {
        currentWs.close(1000, 'No more subscribers')
      }
    }

    this.isConnecting = false
    this.backoffDelay = this.minBackoff
    this.reconnectAttempts = 0
    // Clear message buffer and sequence tracking on disconnect
    this.messageBuffer = []
    this.lastSeenSequence.clear()
    this.gapCallbacks.clear()
    this.initSent = false
    this.pendingInitConfigs.clear()
    
    if (!this.isDestroyed) {
      this.setState('disconnected')
    }
  }

  // Reset reconnection attempts to allow manual reconnect
  resetReconnectionState(): void {
    this.reconnectAttempts = 0
    this.backoffDelay = this.minBackoff
  }

  // For testing: reset state
  destroy(): void {
    this.isDestroyed = true
    
    // Clear all pending state subscriber initial callbacks
    this.stateSubscribers.forEach(callback => {
      const pendingTimeout = this.stateSubscriberTimeouts.get(callback)
      if (pendingTimeout) {
        clearTimeout(pendingTimeout)
      }
    })
    
    this.disconnect()
    this.subscribers.clear()
    this.stateSubscribers.clear()
    this.stateSubscriberTimeouts = new WeakMap()
    this.messageBuffer = []
  }

  // For testing: check if connected
  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN
  }
}

// Singleton instances - one for local logs, one for Azure logs
// Created lazily to avoid issues with HMR and testing
let localLogManager: SharedLogStreamManager | null = null
let azureLogManager: SharedLogStreamManager | null = null

function getLocalLogManager(): SharedLogStreamManager {
  localLogManager ??= new SharedLogStreamManager()
  return localLogManager
}

function getAzureLogManager(): SharedLogStreamManager {
  if (!azureLogManager) {
    // Azure log manager (for realtime streaming)
    class AzureLogStreamManager extends SharedLogStreamManager {
      protected getStreamUrl(): string {
        const protocol = globalThis.location.protocol === 'https:' ? 'wss:' : 'ws:'
        // Azure realtime endpoint - doesn't filter by service, gets all Azure logs
        return `${protocol}//${globalThis.location.host}/api/azure/logs/stream?realtime=true`
      }

      protected sendInitMessage(): void {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return
        if (this.initSent) return

        // Get first subscriber's config (they all share same connection)
        const firstConfig = Array.from(this.pendingInitConfigs.values())[0]
        const firstService = Array.from(this.pendingInitConfigs.keys())[0]

        if (firstConfig || firstService) {
          const initMsg = {
            type: 'init',
            service: firstService || 'all',
            since: firstConfig?.since || '1h', // Default to 1 hour
          }

          try {
            this.ws.send(JSON.stringify(initMsg))
            this.initSent = true
          } catch (err) {
            console.error('[AzureLogStream] Failed to send init message:', err)
          }
        }
      }
    }
    azureLogManager = new AzureLogStreamManager()
  }
  return azureLogManager
}

// For testing: reset managers
export function resetManagers(): void {
  localLogManager?.destroy()
  azureLogManager?.destroy()
  localLogManager = null
  azureLogManager = null
}

interface UseSharedLogStreamOptions {
  serviceName: string
  enabled: boolean
  mode: 'local' | 'azure'
  onLogEntry: (entry: LogEntry) => void
  since?: string  // Time range for initial fetch (e.g., '1h', '30m')
}

interface UseSharedLogStreamReturn {
  connectionState: ConnectionState
}

/**
 * Hook that uses a shared WebSocket connection for all services
 * to avoid hitting resource limits with multiple connections
 */
export function useSharedLogStream({
  serviceName,
  enabled,
  mode,
  onLogEntry,
  since,
}: UseSharedLogStreamOptions): UseSharedLogStreamReturn {
  const callbackRef = useRef(onLogEntry)
  const [connectionState, setConnectionState] = useState<ConnectionState>('disconnected')
  const isMountedRef = useRef(true)
  
  // Get manager reference (stable per mode) - memoized to avoid recalculation
  const manager = useMemo(
    () => (mode === 'azure' ? getAzureLogManager() : getLocalLogManager()),
    [mode]
  )

  // Keep callback ref up to date
  useEffect(() => {
    callbackRef.current = onLogEntry
  }, [onLogEntry])

  // Track mounted state to prevent setState after unmount
  useEffect(() => {
    isMountedRef.current = true
    return () => {
      isMountedRef.current = false
    }
  }, [])

  // Track enabled state in a ref to avoid recreating subscriptions
  const enabledRef = useRef(enabled)
  useEffect(() => {
    enabledRef.current = enabled
  }, [enabled])

  // Subscribe to connection state changes
  useEffect(() => {
    const unsubscribeState = manager.subscribeToState((state) => {
      // Guard against setState after unmount
      if (isMountedRef.current) {
        setConnectionState(enabledRef.current ? state : 'disconnected')
      }
    })

    return unsubscribeState
  }, [manager])

  // Subscribe to log entries
  useEffect(() => {
    if (!enabled) return

    const unsubscribe = manager.subscribe(serviceName, (entry) => {
      // Use callback ref to avoid stale closures
      // Only call if still mounted
      if (isMountedRef.current) {
        callbackRef.current(entry)
      }
    }, { since })

    return unsubscribe
  }, [serviceName, enabled, manager, since])

  return { connectionState }
}
