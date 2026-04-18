/**
 * useSharedLogStream — singleton multiplexer over the live local-log
 * stream so a dashboard with N panes opens one upstream connection
 * instead of N. Each pane subscribes for one service (or "all"); the
 * manager fans incoming entries out to every matching subscriber.
 *
 * Wire migration note (April 2026):
 *   - Local mode now drives `LogsService.StreamLocalLogs` (Connect
 *     server-streaming) instead of the legacy `/api/logs/stream`
 *     WebSocket. The drop-OLDEST 1024-entry ring lives server-side
 *     (cli/src/internal/rpc/logs.go) so a stalled tab catches up to
 *     the latest state instead of replaying ancient history; when the
 *     ring overflows the next on-wire event is a `DroppedNotice` and
 *     we surface the running total via `droppedCount` on the hook
 *     return so the UI can show a "lost N lines" banner.
 *   - Azure mode (`/api/azure/logs/stream`) still runs on the legacy
 *     WebSocket. AzureService.StreamAzureLogs is the next migration
 *     target; flipping that path is intentionally a separate review
 *     unit because the back-pressure policy (block-with-backoff)
 *     differs from local (drop-oldest) and warrants its own commit.
 *
 * The class-based manager pattern from the WebSocket era is preserved
 * because it carries useful state independent of the wire protocol:
 * subscribe/unsubscribe ref-counting, late-subscriber message replay,
 * connection-state callbacks, and the "no subscribers? close after a
 * debounce" flap-prevention. Each manager (Connect for local, WS for
 * Azure) owns its own lifecycle but exposes the same surface so
 * `useSharedLogStream` can pick one without branching.
 */
import { useEffect, useMemo, useRef, useState } from 'react'
import { Code, ConnectError, type PromiseClient, type Transport } from '@connectrpc/connect'

import type { LogEntry } from '@/components/LogsPane'
import { createLogsClient } from '@/lib/connectClient'
import type { LogsService } from '@/gen/proto/azdapp/v1/logs_connect.js'
import {
  StreamLocalLogsRequest,
  type StreamLocalLogsResponse,
} from '@/gen/proto/azdapp/v1/logs_pb.js'
import { LogLevel } from '@/gen/proto/azdapp/v1/common_pb.js'
import type { LogEntry as ProtoLogEntry } from '@/gen/proto/azdapp/v1/common_pb.js'
import type { Timestamp } from '@bufbuild/protobuf'

// =============================================================================
// Shared types
// =============================================================================

export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'error'

type StateChangeCallback = (state: ConnectionState) => void
type DroppedCallback = (count: number) => void
type LogCallback = (entry: LogEntry) => void

interface ManagerConfig {
  /** Maximum reconnect attempts before surfacing 'error' state. */
  maxReconnectAttempts: number
}

const DEFAULT_CONFIG: ManagerConfig = {
  maxReconnectAttempts: 10,
}

const MIN_BACKOFF_MS = 1000
const MAX_BACKOFF_MS = 30_000
const MAX_BUFFER_SIZE = 100
const DISCONNECT_DEBOUNCE_MS = 100

/**
 * Common surface every log-stream manager exposes. Defining it as an
 * interface lets `useSharedLogStream` select between Connect-backed and
 * WS-backed implementations without branching at call sites.
 */
interface LogStreamManager {
  subscribe(
    serviceName: string,
    callback: LogCallback,
    config?: { since?: string; onGapDetected?: (gap: { start: number; end: number }) => void },
  ): () => void
  subscribeToState(callback: StateChangeCallback): () => void
  /**
   * Subscribe to running drop-counter updates. Production hook surfaces
   * this so consumers can render a "lost N lines" banner. Connect
   * (local) increments the counter from server `DroppedNotice` events;
   * the WS (Azure) manager never increments because the legacy WS wire
   * has no equivalent signal.
   */
  subscribeToDroppedCount(callback: DroppedCallback): () => void
  getState(): ConnectionState
  getDroppedCount(): number
  destroy(): void
  resetReconnectionState(): void
}

// =============================================================================
// Subscriber ref-counting helpers (shared between managers)
// =============================================================================

/**
 * Bag of subscribers keyed by service name. Encapsulates the add /
 * remove / dispatch routines both managers share so the wire-specific
 * code stays focused on connection lifecycle.
 *
 * "all" is a special key: dispatching to a service ALSO dispatches to
 * any "all" subscribers. This mirrors the legacy WS contract that
 * LogsView.tsx and useLogsStream depend on.
 */
class SubscriberRegistry {
  private readonly subscribers = new Map<string, Set<LogCallback>>()
  private readonly stateSubscribers = new Set<StateChangeCallback>()
  private readonly droppedSubscribers = new Set<DroppedCallback>()
  private currentState: ConnectionState = 'disconnected'
  private droppedCount = 0
  // Keep the last MAX_BUFFER_SIZE entries so a late-mounting subscriber
  // immediately sees recent activity instead of staring at an empty
  // pane until the next live message arrives.
  private readonly messageBuffer: LogEntry[] = []

  size(): number {
    return this.subscribers.size
  }

  hasSubscribers(): boolean {
    return this.subscribers.size > 0
  }

  addLogSubscriber(serviceName: string, callback: LogCallback): boolean {
    let bucket = this.subscribers.get(serviceName)
    const isFirstForService = !bucket
    if (!bucket) {
      bucket = new Set()
      this.subscribers.set(serviceName, bucket)
    }
    bucket.add(callback)
    // Replay buffered entries so the new subscriber sees the same tail
    // an early subscriber would have. Order matters: callers depend on
    // chronological dispatch.
    this.messageBuffer
      .filter((entry) => entry.service === serviceName || serviceName === 'all')
      .forEach((entry) => {
        try {
          callback(entry)
        } catch (err) {
          console.error('[useSharedLogStream] Replay subscriber error:', err)
        }
      })
    return isFirstForService
  }

  removeLogSubscriber(serviceName: string, callback: LogCallback): boolean {
    const bucket = this.subscribers.get(serviceName)
    if (!bucket) return false
    bucket.delete(callback)
    if (bucket.size === 0) {
      this.subscribers.delete(serviceName)
    }
    return this.subscribers.size === 0
  }

  addStateSubscriber(callback: StateChangeCallback): void {
    this.stateSubscribers.add(callback)
  }

  removeStateSubscriber(callback: StateChangeCallback): void {
    this.stateSubscribers.delete(callback)
  }

  addDroppedSubscriber(callback: DroppedCallback): void {
    this.droppedSubscribers.add(callback)
  }

  removeDroppedSubscriber(callback: DroppedCallback): void {
    this.droppedSubscribers.delete(callback)
  }

  setState(newState: ConnectionState): void {
    if (this.currentState === newState) return
    this.currentState = newState
    // Iterate via a snapshot so a subscriber that synchronously
    // unsubscribes from inside its own callback can't mutate the set
    // mid-iteration.
    const toRemove: StateChangeCallback[] = []
    Array.from(this.stateSubscribers).forEach((callback) => {
      try {
        callback(newState)
      } catch (err) {
        console.error('[useSharedLogStream] State subscriber error:', err)
        toRemove.push(callback)
      }
    })
    toRemove.forEach((cb) => this.stateSubscribers.delete(cb))
  }

  getState(): ConnectionState {
    return this.currentState
  }

  getDroppedCount(): number {
    return this.droppedCount
  }

  /**
   * Increment the running drop counter and notify subscribers. Connect
   * back-pressure semantics (drop-oldest, server-side) push deltas
   * here so the UI sees a monotonically-growing total rather than
   * having to track per-event deltas itself.
   */
  recordDropped(delta: number): void {
    if (delta <= 0) return
    this.droppedCount += delta
    const total = this.droppedCount
    const toRemove: DroppedCallback[] = []
    Array.from(this.droppedSubscribers).forEach((callback) => {
      try {
        callback(total)
      } catch (err) {
        console.error('[useSharedLogStream] Dropped subscriber error:', err)
        toRemove.push(callback)
      }
    })
    toRemove.forEach((cb) => this.droppedSubscribers.delete(cb))
  }

  /** Reset the drop counter. Called on full disconnect (no subscribers). */
  resetDroppedCount(): void {
    if (this.droppedCount === 0) return
    this.droppedCount = 0
  }

  dispatch(entry: LogEntry): void {
    this.messageBuffer.push(entry)
    if (this.messageBuffer.length > MAX_BUFFER_SIZE) {
      this.messageBuffer.shift()
    }

    const serviceName = String(entry.service)
    const dispatchTo = (set: Set<LogCallback> | undefined) => {
      if (!set || set.size === 0) return
      const toRemove: LogCallback[] = []
      Array.from(set).forEach((callback) => {
        try {
          callback(entry)
        } catch (err) {
          console.error('[useSharedLogStream] Subscriber callback error:', err)
          toRemove.push(callback)
        }
      })
      toRemove.forEach((cb) => set.delete(cb))
    }
    dispatchTo(this.subscribers.get(serviceName))
    if (serviceName !== 'all') {
      dispatchTo(this.subscribers.get('all'))
    }
  }

  clearBuffer(): void {
    this.messageBuffer.length = 0
  }

  clearAll(): void {
    this.subscribers.clear()
    this.stateSubscribers.clear()
    this.droppedSubscribers.clear()
    this.messageBuffer.length = 0
  }
}

// =============================================================================
// Proto -> dashboard mappers
// =============================================================================

function timestampToIso(ts: Timestamp | undefined): string {
  if (!ts) return new Date().toISOString()
  const seconds = typeof ts.seconds === 'bigint' ? Number(ts.seconds) : Number(ts.seconds ?? 0)
  const nanos = typeof ts.nanos === 'number' ? ts.nanos : 0
  return new Date(seconds * 1000 + Math.floor(nanos / 1e6)).toISOString()
}

/**
 * Map proto LogLevel (enum spanning TRACE..FATAL) onto the dashboard's
 * three-bucket numeric scale (1=info, 2=warning, 3=error). The legacy
 * WS handler emits service.LogLevel as JSON which already collapses to
 * the same buckets; we faithfully reproduce that mapping so downstream
 * filters keyed on the numeric value see no behaviour change.
 *
 * UNSPECIFIED / TRACE / DEBUG collapse to INFO because the dashboard
 * has no badge for them and falling back to "info" matches what the
 * REST GetLogs path does.
 */
function protoLevelToNumeric(level: LogLevel): number {
  switch (level) {
    case LogLevel.ERROR:
    case LogLevel.FATAL:
      return 3
    case LogLevel.WARN:
      return 2
    case LogLevel.INFO:
    case LogLevel.TRACE:
    case LogLevel.DEBUG:
    case LogLevel.UNSPECIFIED:
    default:
      return 1
  }
}

/**
 * Convert a proto LogEntry to the dashboard `LogEntry` shape. The
 * legacy WS sent the raw service.LogEntry JSON (service/message/level/
 * timestamp/isStderr); preserving that shape end-to-end means none of
 * the consumer hooks or components have to change to read the new
 * wire data.
 */
export function protoToLogEntry(proto: ProtoLogEntry): LogEntry {
  return {
    service: proto.service,
    message: proto.message,
    level: protoLevelToNumeric(proto.level),
    timestamp: timestampToIso(proto.timestamp),
    // Stream is LOG_STREAM_STDERR (=2) for stderr; everything else is
    // not stderr. Comparing to the numeric enum value keeps the
    // mapping resilient to the wire enum gaining new variants.
    isStderr: Number(proto.stream) === 2,
  }
}

// =============================================================================
// Connect-backed local-log manager
// =============================================================================

/**
 * Local-log manager backed by `LogsService.StreamLocalLogs` (Connect
 * server-streaming). One upstream stream per manager subscribes to all
 * services (`service_name=""`) and the registry multiplexes incoming
 * entries to per-service subscribers client-side.
 *
 * Reconnect strategy: exponential backoff (1s → 2s → 4s ... capped at
 * 30s with light jitter) up to `maxReconnectAttempts` (10 by default).
 * Mirrors the legacy WS manager so the dashboard's "Reconnecting..."
 * messaging behaves identically across the migration.
 *
 * AbortController drives stream teardown: on full disconnect (no
 * subscribers) the controller aborts and the async-iteration loop
 * unwinds. Connect-web maps `AbortController.abort()` to a clean
 * client cancellation rather than a network error, so we don't
 * schedule a reconnect for cancellations we initiated.
 */
class ConnectLocalLogStreamManager implements LogStreamManager {
  private readonly registry = new SubscriberRegistry()
  private readonly client: PromiseClient<typeof LogsService>
  private readonly config: ManagerConfig
  private controller: AbortController | null = null
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private disconnectTimer: ReturnType<typeof setTimeout> | null = null
  private backoffDelay = MIN_BACKOFF_MS
  private reconnectAttempts = 0
  private isStreaming = false
  private isDestroyed = false

  constructor(client: PromiseClient<typeof LogsService>, config: Partial<ManagerConfig> = {}) {
    this.client = client
    this.config = { ...DEFAULT_CONFIG, ...config }
  }

  subscribe(serviceName: string, callback: LogCallback): () => void {
    if (this.isDestroyed) return () => {}

    // If a disconnect was queued because the previous subscriber list
    // emptied out, cancel it: the stream is still healthy and the new
    // subscriber should attach to it.
    if (this.disconnectTimer) {
      clearTimeout(this.disconnectTimer)
      this.disconnectTimer = null
    }

    const wasEmpty = !this.registry.hasSubscribers()
    this.registry.addLogSubscriber(serviceName, callback)

    if (wasEmpty && !this.isStreaming) {
      this.startStream()
    }

    return () => {
      const nowEmpty = this.registry.removeLogSubscriber(serviceName, callback)
      if (nowEmpty) {
        // Debounce so a rapid unmount/remount (React Strict Mode, route
        // toggling, etc.) doesn't tear down and rebuild the upstream
        // stream on every paint.
        if (this.disconnectTimer) clearTimeout(this.disconnectTimer)
        this.disconnectTimer = setTimeout(() => {
          this.disconnectTimer = null
          if (!this.registry.hasSubscribers()) {
            this.stopStream()
          }
        }, DISCONNECT_DEBOUNCE_MS)
      }
    }
  }

  subscribeToState(callback: StateChangeCallback): () => void {
    if (this.isDestroyed) return () => {}
    this.registry.addStateSubscriber(callback)
    // Echo current state asynchronously so the subscriber doesn't have
    // to special-case "did I subscribe before or after the first state
    // transition?". Defer via microtask so the caller's setState can
    // settle without a sync re-entry.
    queueMicrotask(() => {
      if (this.isDestroyed) return
      try {
        callback(this.registry.getState())
      } catch (err) {
        console.error('[useSharedLogStream] State init error:', err)
      }
    })
    return () => this.registry.removeStateSubscriber(callback)
  }

  subscribeToDroppedCount(callback: DroppedCallback): () => void {
    if (this.isDestroyed) return () => {}
    this.registry.addDroppedSubscriber(callback)
    queueMicrotask(() => {
      if (this.isDestroyed) return
      try {
        callback(this.registry.getDroppedCount())
      } catch (err) {
        console.error('[useSharedLogStream] Dropped init error:', err)
      }
    })
    return () => this.registry.removeDroppedSubscriber(callback)
  }

  getState(): ConnectionState {
    return this.registry.getState()
  }

  getDroppedCount(): number {
    return this.registry.getDroppedCount()
  }

  resetReconnectionState(): void {
    this.reconnectAttempts = 0
    this.backoffDelay = MIN_BACKOFF_MS
  }

  destroy(): void {
    this.isDestroyed = true
    this.stopStream()
    this.registry.clearAll()
  }

  private startStream(): void {
    if (this.isDestroyed || this.isStreaming) return
    if (this.reconnectAttempts >= this.config.maxReconnectAttempts) {
      this.registry.setState('error')
      return
    }

    this.isStreaming = true
    this.registry.setState('connecting')
    this.reconnectAttempts++

    const controller = new AbortController()
    this.controller = controller

    void this.consumeStream(controller)
  }

  /**
   * Stream pump. Kept as a separate async function (rather than inline
   * in startStream) so the cleanup paths stay readable. Connection
   * state flips to "connected" on the first message because Connect
   * over HTTP/1.1 only flushes response headers when the handler
   * sends its first event - until then we cannot prove the wire is
   * healthy. (Same contract as useHealthStream.)
   */
  private async consumeStream(controller: AbortController): Promise<void> {
    let firstMessageSeen = false
    try {
      const req = new StreamLocalLogsRequest({
        serviceName: '',
        // Backfill 0: each subscriber that needs initial history fetches
        // it via GetLogs (REST today, GetLogs unary RPC after the next
        // batch). Leaving backfill server-side here avoids replaying the
        // tail to every newly attached pane on a reconnect.
        backfill: 0,
      })

      for await (const resp of this.client.streamLocalLogs(req, { signal: controller.signal })) {
        if (this.isDestroyed || controller.signal.aborted) break
        if (!firstMessageSeen) {
          firstMessageSeen = true
          this.registry.setState('connected')
          this.reconnectAttempts = 0
          this.backoffDelay = MIN_BACKOFF_MS
        }
        this.handleResponse(resp)
      }

      // Stream ended cleanly. Treat as transient (server shutdown,
      // service rotation) and reschedule with backoff so the dashboard
      // recovers without a manual reload.
      if (!this.isDestroyed && !controller.signal.aborted && this.registry.hasSubscribers()) {
        this.scheduleReconnect()
      }
    } catch (err) {
      if (controller.signal.aborted) return
      if (err instanceof ConnectError && err.code === Code.Canceled) return
      if (this.isDestroyed) return
      // Only log the first failure and the eventual give-up; otherwise
      // a flapping backend would spam the console.
      if (this.reconnectAttempts === 1) {
        console.warn(
          '[useSharedLogStream] Connect stream failed:',
          err instanceof Error ? err.message : 'Unknown error',
        )
      }
      if (this.registry.hasSubscribers()) {
        this.scheduleReconnect()
      } else {
        this.registry.setState('disconnected')
      }
    } finally {
      this.isStreaming = false
      if (this.controller === controller) {
        this.controller = null
      }
    }
  }

  private handleResponse(resp: StreamLocalLogsResponse): void {
    const event = resp.event
    if (!event) return
    switch (event.case) {
      case 'entry': {
        const proto = event.value
        if (!proto || !proto.service) return
        this.registry.dispatch(protoToLogEntry(proto))
        return
      }
      case 'dropped': {
        const count = Number(event.value?.count ?? 0)
        if (count > 0) {
          this.registry.recordDropped(count)
        }
        return
      }
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer || this.isDestroyed) return
    if (this.reconnectAttempts >= this.config.maxReconnectAttempts) {
      this.registry.setState('error')
      return
    }

    this.registry.setState('error')
    const delay = this.backoffDelay
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      if (this.registry.hasSubscribers() && !this.isDestroyed) {
        this.startStream()
      }
    }, delay)
    // Exponential backoff with light jitter so a fleet of dashboards
    // recovering at the same time don't synchronise their retries.
    this.backoffDelay = Math.min(this.backoffDelay * 2 + Math.random() * 1000, MAX_BACKOFF_MS)
  }

  private stopStream(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.disconnectTimer) {
      clearTimeout(this.disconnectTimer)
      this.disconnectTimer = null
    }
    if (this.controller) {
      this.controller.abort()
      this.controller = null
    }
    this.isStreaming = false
    this.reconnectAttempts = 0
    this.backoffDelay = MIN_BACKOFF_MS
    this.registry.clearBuffer()
    this.registry.resetDroppedCount()
    if (!this.isDestroyed) {
      this.registry.setState('disconnected')
    }
  }
}

// =============================================================================
// Azure log manager (legacy WebSocket - DEFERRED migration)
// =============================================================================

/**
 * Azure realtime log manager. Still on the legacy WebSocket
 * (`/api/azure/logs/stream?realtime=true`) because `AzureService.
 * StreamAzureLogs` lives behind a different back-pressure policy
 * (block-with-backoff, see ADR-0001) that warrants its own migration
 * commit. When that migration lands this whole class collapses into a
 * sibling of `ConnectLocalLogStreamManager`.
 *
 * The class is intentionally narrower than the old singleton: no
 * heartbeat (Connect-web handles per-stream health for the local
 * path; the legacy WS path's heartbeat was tuned for the WS-specific
 * silent-disconnect failure mode and is preserved in the inline code
 * below) and no init-message machinery beyond the Azure-specific
 * `since` timestamp the realtime endpoint expects.
 */
class WebSocketAzureLogStreamManager implements LogStreamManager {
  private readonly registry = new SubscriberRegistry()
  private readonly config: ManagerConfig
  private readonly pendingInitConfigs = new Map<string, { since?: string }>()
  private ws: WebSocket | null = null
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private disconnectTimer: ReturnType<typeof setTimeout> | null = null
  private heartbeatTimer: ReturnType<typeof setTimeout> | null = null
  private heartbeatTimeoutTimer: ReturnType<typeof setTimeout> | null = null
  private backoffDelay = MIN_BACKOFF_MS
  private reconnectAttempts = 0
  private isConnecting = false
  private isDestroyed = false
  private initSent = false

  // Heartbeat tunables match the pre-migration manager.
  private static readonly HEARTBEAT_INTERVAL_MS = 30_000
  private static readonly HEARTBEAT_TIMEOUT_MS = 5_000

  constructor(config: Partial<ManagerConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config }
  }

  subscribe(
    serviceName: string,
    callback: LogCallback,
    config?: { since?: string },
  ): () => void {
    if (this.isDestroyed) return () => {}

    if (this.disconnectTimer) {
      clearTimeout(this.disconnectTimer)
      this.disconnectTimer = null
    }

    const wasEmpty = !this.registry.hasSubscribers()
    this.registry.addLogSubscriber(serviceName, callback)
    if (config?.since) {
      this.pendingInitConfigs.set(serviceName, { since: config.since })
    }

    if (wasEmpty && !this.ws && !this.isConnecting) {
      this.connect()
    }

    return () => {
      const nowEmpty = this.registry.removeLogSubscriber(serviceName, callback)
      this.pendingInitConfigs.delete(serviceName)
      if (nowEmpty) {
        if (this.disconnectTimer) clearTimeout(this.disconnectTimer)
        this.disconnectTimer = setTimeout(() => {
          this.disconnectTimer = null
          if (!this.registry.hasSubscribers()) this.disconnect()
        }, DISCONNECT_DEBOUNCE_MS)
      }
    }
  }

  subscribeToState(callback: StateChangeCallback): () => void {
    if (this.isDestroyed) return () => {}
    this.registry.addStateSubscriber(callback)
    queueMicrotask(() => {
      if (this.isDestroyed) return
      try {
        callback(this.registry.getState())
      } catch (err) {
        console.error('[useSharedLogStream:azure] State init error:', err)
      }
    })
    return () => this.registry.removeStateSubscriber(callback)
  }

  subscribeToDroppedCount(callback: DroppedCallback): () => void {
    if (this.isDestroyed) return () => {}
    this.registry.addDroppedSubscriber(callback)
    // Azure path has no drop signal; emit 0 so the React state init
    // path fires once instead of staying undefined.
    queueMicrotask(() => {
      if (this.isDestroyed) return
      try {
        callback(0)
      } catch (err) {
        console.error('[useSharedLogStream:azure] Dropped init error:', err)
      }
    })
    return () => this.registry.removeDroppedSubscriber(callback)
  }

  getState(): ConnectionState {
    return this.registry.getState()
  }

  getDroppedCount(): number {
    return this.registry.getDroppedCount()
  }

  resetReconnectionState(): void {
    this.reconnectAttempts = 0
    this.backoffDelay = MIN_BACKOFF_MS
  }

  destroy(): void {
    this.isDestroyed = true
    this.disconnect()
    this.registry.clearAll()
  }

  private getStreamUrl(): string {
    const protocol = globalThis.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${protocol}//${globalThis.location.host}/api/azure/logs/stream?realtime=true`
  }

  private connect(): void {
    if (this.isDestroyed || this.ws || this.isConnecting) return
    if (this.reconnectAttempts >= this.config.maxReconnectAttempts) {
      this.registry.setState('error')
      return
    }

    this.isConnecting = true
    this.registry.setState('connecting')
    this.reconnectAttempts++

    try {
      const ws = new WebSocket(this.getStreamUrl())
      ws.addEventListener('open', this.handleOpen)
      ws.addEventListener('message', this.handleMessage)
      ws.addEventListener('error', this.handleError)
      ws.addEventListener('close', this.handleClose)
      this.ws = ws
    } catch (err) {
      this.isConnecting = false
      this.registry.setState('error')
      if (this.reconnectAttempts === 1) {
        console.warn(
          '[useSharedLogStream:azure] Failed to create WebSocket:',
          err instanceof Error ? err.message : 'Unknown error',
        )
      }
      if (this.registry.hasSubscribers()) {
        this.scheduleReconnect()
      }
    }
  }

  private readonly handleOpen = (): void => {
    if (this.isDestroyed) return
    this.isConnecting = false
    this.backoffDelay = MIN_BACKOFF_MS
    this.reconnectAttempts = 0
    this.registry.setState('connected')
    this.sendInitMessage()
    this.startHeartbeat()
  }

  private sendInitMessage(): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN || this.initSent) return
    const firstService = Array.from(this.pendingInitConfigs.keys())[0]
    const firstConfig = firstService ? this.pendingInitConfigs.get(firstService) : undefined
    if (!firstService && !firstConfig) return
    try {
      this.ws.send(
        JSON.stringify({
          type: 'init',
          service: firstService ?? 'all',
          since: firstConfig?.since ?? '1h',
        }),
      )
      this.initSent = true
    } catch (err) {
      console.error('[useSharedLogStream:azure] Failed to send init message:', err)
    }
  }

  private readonly handleMessage = (event: MessageEvent): void => {
    if (this.heartbeatTimeoutTimer) {
      clearTimeout(this.heartbeatTimeoutTimer)
      this.heartbeatTimeoutTimer = null
    }
    try {
      const message = JSON.parse(event.data as string) as unknown
      // Status messages are heartbeats / health updates, not log lines.
      if (typeof message === 'object' && message !== null && 'type' in message) {
        const typed = message as { type?: unknown }
        if (typed.type === 'status') return
      }
      const entries = Array.isArray(message) ? (message as LogEntry[]) : [message as LogEntry]
      entries.forEach((entry) => {
        if (!entry || typeof entry !== 'object' || !entry.service) {
          console.warn('[useSharedLogStream:azure] Invalid log entry received:', entry)
          return
        }
        this.registry.dispatch(entry)
      })
    } catch (err) {
      console.error('[useSharedLogStream:azure] Failed to parse message:', err)
    }
  }

  private readonly handleError = (event: Event): void => {
    // onerror is always followed by onclose; let onclose handle state.
    void event
  }

  private readonly handleClose = (event: CloseEvent): void => {
    if (this.isDestroyed) return
    this.detachListeners()
    this.ws = null
    this.isConnecting = false
    this.stopHeartbeat()
    this.initSent = false

    // Clean close (1000) or no subscribers means we don't reconnect.
    if (event.code === 1000 || !this.registry.hasSubscribers()) {
      this.registry.setState('disconnected')
      return
    }

    this.registry.setState('error')
    this.scheduleReconnect()
  }

  private detachListeners(): void {
    if (!this.ws) return
    this.ws.removeEventListener('open', this.handleOpen)
    this.ws.removeEventListener('message', this.handleMessage)
    this.ws.removeEventListener('error', this.handleError)
    this.ws.removeEventListener('close', this.handleClose)
  }

  private startHeartbeat(): void {
    this.stopHeartbeat()
    this.heartbeatTimer = setInterval(() => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        this.stopHeartbeat()
        return
      }
      if (this.heartbeatTimeoutTimer) clearTimeout(this.heartbeatTimeoutTimer)
      this.heartbeatTimeoutTimer = setTimeout(() => {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
          this.ws.close(1000, 'Heartbeat timeout')
        }
      }, WebSocketAzureLogStreamManager.HEARTBEAT_TIMEOUT_MS)
    }, WebSocketAzureLogStreamManager.HEARTBEAT_INTERVAL_MS)
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
    if (this.reconnectAttempts >= this.config.maxReconnectAttempts) return
    const delay = this.backoffDelay
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      if (this.registry.hasSubscribers() && !this.isDestroyed) this.connect()
    }, delay)
    this.backoffDelay = Math.min(this.backoffDelay * 2 + Math.random() * 1000, MAX_BACKOFF_MS)
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
      this.detachListeners()
      const ws = this.ws
      this.ws = null
      if (ws.readyState === WebSocket.OPEN) {
        ws.close(1000, 'No more subscribers')
      }
    }
    this.isConnecting = false
    this.backoffDelay = MIN_BACKOFF_MS
    this.reconnectAttempts = 0
    this.initSent = false
    this.pendingInitConfigs.clear()
    this.registry.clearBuffer()
    if (!this.isDestroyed) {
      this.registry.setState('disconnected')
    }
  }
}

// =============================================================================
// Singletons + factory wiring
// =============================================================================

let localLogManager: ConnectLocalLogStreamManager | null = null
let azureLogManager: WebSocketAzureLogStreamManager | null = null

function getLocalLogManager(): ConnectLocalLogStreamManager {
  localLogManager ??= new ConnectLocalLogStreamManager(createLogsClient())
  return localLogManager
}

function getAzureLogManager(): WebSocketAzureLogStreamManager {
  azureLogManager ??= new WebSocketAzureLogStreamManager()
  return azureLogManager
}

/**
 * Tear down both singletons. Production code should never call this;
 * it exists for tests that need a fresh manager between runs (e.g.
 * vitest's afterEach) and for hot-reload edge cases.
 */
export function resetManagers(): void {
  localLogManager?.destroy()
  azureLogManager?.destroy()
  localLogManager = null
  azureLogManager = null
}

// =============================================================================
// Hook
// =============================================================================

interface UseSharedLogStreamOptions {
  serviceName: string
  enabled: boolean
  mode: 'local' | 'azure'
  onLogEntry: (entry: LogEntry) => void
  /** Time range for initial fetch (Azure realtime only, e.g. '1h', '30m'). */
  since?: string
  /**
   * Override the Connect transport for the local-mode manager. Tests
   * pass `createRouterTransport(...)`; production omits it. Providing a
   * transport bypasses the singleton so each test instance gets its
   * own isolated manager.
   *
   * No effect when `mode === 'azure'` (legacy WS path).
   */
  transport?: Transport
}

export interface UseSharedLogStreamReturn {
  connectionState: ConnectionState
  /**
   * Total number of log entries the server reported as dropped on the
   * current Connect stream (drop-OLDEST back-pressure). Always 0 for
   * `mode === 'azure'` until AzureService.StreamAzureLogs migration
   * lands. Resets to 0 when all subscribers disconnect.
   */
  droppedCount: number
}

/**
 * Subscribe a component to the shared local-log (Connect) or Azure-log
 * (legacy WS) stream. Returns the connection state plus a running
 * drop-counter so consumers can render a "lost N lines" banner.
 *
 * Public surface preserved from the WebSocket era: same name, same
 * positional options, same connectionState semantics. `droppedCount`
 * and `transport` are NEW additive fields - the WS-era return was
 * `{ connectionState }` and existing call sites continue to compile
 * because TypeScript widens the return type only when consumed.
 */
export function useSharedLogStream({
  serviceName,
  enabled,
  mode,
  onLogEntry,
  since,
  transport,
}: UseSharedLogStreamOptions): UseSharedLogStreamReturn {
  const callbackRef = useRef(onLogEntry)
  const enabledRef = useRef(enabled)
  const isMountedRef = useRef(true)

  // Pick (or create) the right manager for the requested mode. When a
  // transport is injected we bypass the singleton entirely so the test
  // gets a clean manager scoped to the hook instance - otherwise
  // multiple tests that override the transport would race against the
  // module-level cache.
  const manager = useMemo<LogStreamManager>(() => {
    if (mode === 'azure') {
      return getAzureLogManager()
    }
    if (transport) {
      return new ConnectLocalLogStreamManager(createLogsClient(transport))
    }
    return getLocalLogManager()
  }, [mode, transport])

  const [connectionState, setConnectionState] = useState<ConnectionState>(() => manager.getState())
  const [droppedCount, setDroppedCount] = useState<number>(() => manager.getDroppedCount())

  useEffect(() => {
    callbackRef.current = onLogEntry
  }, [onLogEntry])

  useEffect(() => {
    isMountedRef.current = true
    return () => {
      isMountedRef.current = false
    }
  }, [])

  useEffect(() => {
    enabledRef.current = enabled
  }, [enabled])

  // Connection state subscription.
  useEffect(() => {
    return manager.subscribeToState((state) => {
      if (!isMountedRef.current) return
      // Mirror the legacy hook: when disabled, the consumer wants to
      // see "disconnected" regardless of underlying transport activity.
      setConnectionState(enabledRef.current ? state : 'disconnected')
    })
  }, [manager])

  // Drop-count subscription. Lives in its own effect so unrelated
  // state churn (mode toggle) doesn't reset it.
  useEffect(() => {
    return manager.subscribeToDroppedCount((count) => {
      if (!isMountedRef.current) return
      setDroppedCount(count)
    })
  }, [manager])

  // Per-service log subscription.
  useEffect(() => {
    if (!enabled) return undefined
    const unsubscribe = manager.subscribe(
      serviceName,
      (entry) => {
        if (!isMountedRef.current) return
        callbackRef.current(entry)
      },
      since !== undefined ? { since } : undefined,
    )
    return unsubscribe
  }, [serviceName, enabled, manager, since])

  // Per-instance manager cleanup. The singleton path is intentionally
  // not destroyed on unmount because other components may share it;
  // the test path (local mode + injected transport) creates a fresh
  // manager per hook so we both can - and must - tear it down.
  useEffect(() => {
    if (mode === 'azure' || !transport) return undefined
    return () => {
      manager.destroy()
    }
  }, [manager, mode, transport])

  return { connectionState, droppedCount }
}

// =============================================================================
// Test-only helpers
// =============================================================================

/**
 * Test seam: build a Connect-backed local manager for a given
 * transport. Production code uses the singleton via `useSharedLogStream`;
 * this exists for hook-level tests that want to drive the manager
 * directly without a React render cycle.
 */
export function __createConnectLocalManagerForTesting(
  transport: Transport,
  config?: Partial<ManagerConfig>,
): LogStreamManager {
  return new ConnectLocalLogStreamManager(createLogsClient(transport), config)
}
