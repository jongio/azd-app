/**
 * useSharedLogStream — singleton multiplexer over the live local-log
 * stream so a dashboard with N panes opens one upstream connection
 * instead of N. Each pane subscribes for one service (or "all"); the
 * manager fans incoming entries out to every matching subscriber.
 *
 * Wire migration note (April 2026):
 *   - Local mode drives `LogsService.StreamLocalLogs` (Connect
 *     server-streaming) instead of the legacy `/api/logs/stream`
 *     WebSocket. The drop-OLDEST 1024-entry ring lives server-side
 *     (cli/src/internal/rpc/logs.go) so a stalled tab catches up to
 *     the latest state instead of replaying ancient history; when the
 *     ring overflows the next on-wire event is a `DroppedNotice` and
 *     we surface the running total via `droppedCount` on the hook
 *     return so the UI can show a "lost N lines" banner.
 *   - Azure mode now drives `AzureService.StreamAzureLogs` (Connect
 *     server-streaming) instead of the legacy
 *     `/api/azure/logs/stream` WebSocket. The proto requires a
 *     non-empty service per stream (one Log-Analytics resource per
 *     subscription), so the Azure manager opens one upstream stream
 *     per subscribed service name - unlike local which multiplexes
 *     all services on a single stream. The 3-event oneof (entry /
 *     status / dropped) is fanned out to: entries → registry, dropped
 *     → drop counter, status → new `streamStatus` subscriber surface
 *     that drives the polling-health UI.
 *
 * The class-based manager pattern from the WebSocket era is preserved
 * because it carries useful state independent of the wire protocol:
 * subscribe/unsubscribe ref-counting, late-subscriber message replay,
 * connection-state callbacks, and the "no subscribers? close after a
 * debounce" flap-prevention. Each manager (Connect-local for all
 * services; Connect-azure for one stream per service) owns its own
 * lifecycle but exposes the same surface so `useSharedLogStream` can
 * pick one without branching.
 */
import { useEffect, useMemo, useRef, useState } from 'react'
import { Code, ConnectError, type Client, type Transport } from '@connectrpc/connect'
import { protoInt64 } from '@bufbuild/protobuf'

import type { LogEntry } from '@/components/LogsPane'
import { createAzureClient, createLogsClient } from '@/lib/connectClient'
import type { AzureService } from '@/gen/proto/azdapp/v1/azure_pb.js'
import type { LogsService } from '@/gen/proto/azdapp/v1/logs_pb.js'
import { create } from '@bufbuild/protobuf'
import {
  StreamAzureLogsRequestSchema,
  type StreamAzureLogsResponse,
  type StreamStatus,
} from '@/gen/proto/azdapp/v1/azure_pb.js'
import {
  StreamLocalLogsRequestSchema,
  type StreamLocalLogsResponse,
} from '@/gen/proto/azdapp/v1/logs_pb.js'
import { LogLevel } from '@/gen/proto/azdapp/v1/common_pb.js'
import type { LogEntry as ProtoLogEntry } from '@/gen/proto/azdapp/v1/common_pb.js'
import type { Timestamp } from '@bufbuild/protobuf/wkt'

// =============================================================================
// Shared types
// =============================================================================

export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'error'

type StateChangeCallback = (state: ConnectionState) => void
type DroppedCallback = (count: number) => void
type LogCallback = (entry: LogEntry) => void
type StreamStatusCallback = (status: StreamStatus | null) => void

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
   * Connect (Azure) increments from realtime-mode `AzureDroppedNotice`
   * events (polling mode emits a status='degraded' StreamStatus
   * instead - never drops entries).
   */
  subscribeToDroppedCount(callback: DroppedCallback): () => void
  /**
   * Subscribe to the most-recent server-emitted stream status. Azure
   * uses this to surface polling-health transitions
   * (connected/degraded/disconnected) and realtime↔polling mode flips
   * to the LogsView UI. Local always emits `null` once because the
   * local stream has no equivalent signal.
   */
  subscribeToStreamStatus(callback: StreamStatusCallback): () => void
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
  private readonly streamStatusSubscribers = new Set<StreamStatusCallback>()
  private currentState: ConnectionState = 'disconnected'
  private droppedCount = 0
  private latestStreamStatus: StreamStatus | null = null
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

  /** True when `service` (or 'all') has at least one log subscriber. */
  hasSubscribersFor(service: string): boolean {
    const bucket = this.subscribers.get(service)
    return !!bucket && bucket.size > 0
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

  addStreamStatusSubscriber(callback: StreamStatusCallback): void {
    this.streamStatusSubscribers.add(callback)
  }

  removeStreamStatusSubscriber(callback: StreamStatusCallback): void {
    this.streamStatusSubscribers.delete(callback)
  }

  getStreamStatus(): StreamStatus | null {
    return this.latestStreamStatus
  }

  /**
   * Record the latest server-emitted stream status and notify
   * subscribers. Azure path drives this from `StreamAzureLogsResponse`
   * status events; local path never calls this so subscribers stay on
   * the initial `null` value.
   */
  recordStreamStatus(status: StreamStatus): void {
    this.latestStreamStatus = status
    const toRemove: StreamStatusCallback[] = []
    Array.from(this.streamStatusSubscribers).forEach((callback) => {
      try {
        callback(status)
      } catch (err) {
        console.error('[useSharedLogStream] Stream status subscriber error:', err)
        toRemove.push(callback)
      }
    })
    toRemove.forEach((cb) => this.streamStatusSubscribers.delete(cb))
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
    this.streamStatusSubscribers.clear()
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
  private readonly client: Client<typeof LogsService>
  private readonly config: ManagerConfig
  private controller: AbortController | null = null
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private disconnectTimer: ReturnType<typeof setTimeout> | null = null
  private backoffDelay = MIN_BACKOFF_MS
  private reconnectAttempts = 0
  private isStreaming = false
  private isDestroyed = false

  constructor(client: Client<typeof LogsService>, config: Partial<ManagerConfig> = {}) {
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

  /**
   * Local stream has no equivalent of Azure's `StreamStatus` event, so
   * subscribers receive `null` once and never again. Implementing it
   * keeps the `LogStreamManager` interface uniform - the hook calls
   * the same `manager.subscribeToStreamStatus(...)` regardless of mode.
   */
  subscribeToStreamStatus(callback: StreamStatusCallback): () => void {
    if (this.isDestroyed) return () => {}
    queueMicrotask(() => {
      if (this.isDestroyed) return
      try {
        callback(null)
      } catch (err) {
        console.error('[useSharedLogStream] Stream status init error:', err)
      }
    })
    return () => {}
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
      const req = create(StreamLocalLogsRequestSchema, {
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
// Connect-backed Azure log manager
// =============================================================================

/**
 * Parse the dashboard's "since" string (e.g. "1h", "30m", "15m", "6h",
 * "24h", "1d") into seconds for the proto `backfillSeconds` field.
 * Returns 0 on unparseable input so the server falls back to its
 * 30-minute default - matches what the legacy WS init handler did
 * when `since` was missing or malformed.
 */
function sinceToBackfillSeconds(since: string | undefined): bigint {
  if (!since) return protoInt64.zero
  const match = /^(\d+)([smhd])$/i.exec(since.trim())
  if (!match) return protoInt64.zero
  const value = parseInt(match[1], 10)
  if (!Number.isFinite(value) || value <= 0) return protoInt64.zero
  const unit = match[2].toLowerCase()
  const multiplier = unit === 's' ? 1 : unit === 'm' ? 60 : unit === 'h' ? 3600 : 86400
  return protoInt64.parse(value * multiplier)
}

/**
 * Azure realtime log manager backed by `AzureService.StreamAzureLogs`
 * (Connect server-streaming). The Azure proto requires a NON-EMPTY
 * service per stream (one Log-Analytics resource per call) so this
 * manager opens one upstream stream per subscribed service - unlike
 * the local manager which multiplexes all services on a single
 * stream.
 *
 * Wildcard "all" subscribers DO NOT open a stream of their own (the
 * proto rejects empty service); they instead piggyback on whatever
 * per-service streams happen to be open. In practice the Azure UI
 * always selects a specific service before subscribing, so this only
 * matters for tests.
 *
 * Reconnect strategy: per-service exponential backoff (1s → 2s → ...
 * capped at 30s with jitter), independent counters per service so a
 * dead resource doesn't starve a healthy one. Counter resets on first
 * successful event (matches local manager behaviour).
 */
class ConnectAzureLogStreamManager implements LogStreamManager {
  private readonly registry = new SubscriberRegistry()
  private readonly client: Client<typeof AzureService>
  private readonly config: ManagerConfig
  /** Per-service stream lifecycle. Key is service name. */
  private readonly streams = new Map<string, ServiceStreamState>()
  private readonly disconnectTimers = new Map<string, ReturnType<typeof setTimeout>>()
  /**
   * Pending backfill window per service, captured at subscribe time
   * and consumed by the next stream open for that service. Mirrors
   * the WS-era `pendingInitConfigs` map.
   */
  private readonly pendingSince = new Map<string, string>()
  private isDestroyed = false

  constructor(client: Client<typeof AzureService>, config: Partial<ManagerConfig> = {}) {
    this.client = client
    this.config = { ...DEFAULT_CONFIG, ...config }
  }

  subscribe(
    serviceName: string,
    callback: LogCallback,
    config?: { since?: string },
  ): () => void {
    if (this.isDestroyed) return () => {}

    // Cancel a queued teardown for this service if a new subscriber
    // arrives before the debounce fires.
    const pendingTeardown = this.disconnectTimers.get(serviceName)
    if (pendingTeardown) {
      clearTimeout(pendingTeardown)
      this.disconnectTimers.delete(serviceName)
    }

    this.registry.addLogSubscriber(serviceName, callback)
    if (config?.since) {
      this.pendingSince.set(serviceName, config.since)
    }

    // 'all' subscribers can't open their own stream (proto requires
    // non-empty service). They'll receive entries from any per-service
    // stream that is or becomes open via the registry's "all" fanout.
    if (serviceName !== 'all' && !this.streams.has(serviceName)) {
      this.startStream(serviceName)
    }

    return () => {
      this.registry.removeLogSubscriber(serviceName, callback)
      this.pendingSince.delete(serviceName)

      // Only the per-service bucket matters for stream lifecycle;
      // empty 'all' is meaningless because no stream was opened for it.
      if (serviceName === 'all') return

      // If this service still has subscribers, keep the stream open.
      // Otherwise debounce the teardown so a quick remount doesn't
      // tear down and re-establish the upstream stream on every paint.
      if (this.serviceHasSubscribers(serviceName)) return

      const timer = setTimeout(() => {
        this.disconnectTimers.delete(serviceName)
        if (!this.serviceHasSubscribers(serviceName)) {
          this.stopStream(serviceName)
        }
      }, DISCONNECT_DEBOUNCE_MS)
      this.disconnectTimers.set(serviceName, timer)

      // When the LAST per-service stream goes away, fully reset shared
      // state (drop counter, buffer) so the next subscribe starts
      // clean - same contract as the local manager's full disconnect.
      if (this.streams.size === 0) {
        this.registry.clearBuffer()
        this.registry.resetDroppedCount()
        this.registry.setState('disconnected')
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
    queueMicrotask(() => {
      if (this.isDestroyed) return
      try {
        callback(this.registry.getDroppedCount())
      } catch (err) {
        console.error('[useSharedLogStream:azure] Dropped init error:', err)
      }
    })
    return () => this.registry.removeDroppedSubscriber(callback)
  }

  subscribeToStreamStatus(callback: StreamStatusCallback): () => void {
    if (this.isDestroyed) return () => {}
    this.registry.addStreamStatusSubscriber(callback)
    // Echo current status (or null if none seen yet) so subscribers
    // don't have to special-case "did I subscribe before or after the
    // first status event?".
    queueMicrotask(() => {
      if (this.isDestroyed) return
      try {
        callback(this.registry.getStreamStatus())
      } catch (err) {
        console.error('[useSharedLogStream:azure] Stream status init error:', err)
      }
    })
    return () => this.registry.removeStreamStatusSubscriber(callback)
  }

  getState(): ConnectionState {
    return this.registry.getState()
  }

  getDroppedCount(): number {
    return this.registry.getDroppedCount()
  }

  resetReconnectionState(): void {
    this.streams.forEach((state) => {
      state.reconnectAttempts = 0
      state.backoffDelay = MIN_BACKOFF_MS
    })
  }

  destroy(): void {
    this.isDestroyed = true
    Array.from(this.streams.keys()).forEach((service) => this.stopStream(service))
    this.disconnectTimers.forEach((t) => clearTimeout(t))
    this.disconnectTimers.clear()
    this.pendingSince.clear()
    this.registry.clearAll()
  }

  /** Return true if `service` (or 'all') still has any registered subscriber. */
  private serviceHasSubscribers(service: string): boolean {
    // The registry's hasSubscribers() is global; we need a per-bucket
    // check. Using the public dispatch contract: dispatch a synthetic
    // probe is too heavy, so we re-implement the bucket check by
    // inspecting via the registry's add/remove return values is also
    // not enough. Fall back to a private helper on the registry.
    return this.registry.hasSubscribersFor(service)
  }

  private startStream(serviceName: string): void {
    if (this.isDestroyed) return
    const existing = this.streams.get(serviceName)
    if (existing && existing.isStreaming) return

    const reconnectAttempts = (existing?.reconnectAttempts ?? 0) + 1
    if (reconnectAttempts > this.config.maxReconnectAttempts) {
      this.registry.setState('error')
      return
    }

    const controller = new AbortController()
    const state: ServiceStreamState = {
      service: serviceName,
      controller,
      reconnectAttempts,
      backoffDelay: existing?.backoffDelay ?? MIN_BACKOFF_MS,
      reconnectTimer: null,
      isStreaming: true,
    }
    this.streams.set(serviceName, state)
    this.registry.setState('connecting')

    void this.consumeStream(state)
  }

  private async consumeStream(state: ServiceStreamState): Promise<void> {
    const { service, controller } = state
    let firstMessageSeen = false
    try {
      const since = this.pendingSince.get(service)
      const req = create(StreamAzureLogsRequestSchema, {
        service,
        // Always realtime: the legacy WS path sets ?realtime=true and
        // the dashboard never opened a polling-only stream. Server
        // falls back to polling if realtime setup fails.
        realtime: true,
        backfillSeconds: sinceToBackfillSeconds(since),
      })
      // Backfill is consumed on first stream open; clear so a
      // mid-session reconnect doesn't re-replay the original window.
      this.pendingSince.delete(service)

      for await (const resp of this.client.streamAzureLogs(req, { signal: controller.signal })) {
        if (this.isDestroyed || controller.signal.aborted) break
        if (!firstMessageSeen) {
          firstMessageSeen = true
          this.registry.setState('connected')
          state.reconnectAttempts = 0
          state.backoffDelay = MIN_BACKOFF_MS
        }
        this.handleResponse(resp)
      }

      if (!this.isDestroyed && !controller.signal.aborted && this.serviceHasSubscribers(service)) {
        this.scheduleReconnect(state)
      }
    } catch (err) {
      if (controller.signal.aborted) return
      if (err instanceof ConnectError && err.code === Code.Canceled) return
      if (this.isDestroyed) return
      if (state.reconnectAttempts === 1) {
        console.warn(
          `[useSharedLogStream:azure] Connect stream for "${service}" failed:`,
          err instanceof Error ? err.message : 'Unknown error',
        )
      }
      if (this.serviceHasSubscribers(service)) {
        this.scheduleReconnect(state)
      } else {
        this.streams.delete(service)
        if (this.streams.size === 0) {
          this.registry.setState('disconnected')
        }
      }
    } finally {
      state.isStreaming = false
    }
  }

  private handleResponse(resp: StreamAzureLogsResponse): void {
    const event = resp.event
    if (!event) return
    switch (event.case) {
      case 'entry': {
        const proto = event.value
        if (!proto || !proto.service) return
        this.registry.dispatch(protoToLogEntry(proto))
        return
      }
      case 'status': {
        if (event.value) this.registry.recordStreamStatus(event.value)
        return
      }
      case 'dropped': {
        const count = Number(event.value?.count ?? 0)
        if (count > 0) this.registry.recordDropped(count)
        return
      }
    }
  }

  private scheduleReconnect(state: ServiceStreamState): void {
    if (state.reconnectTimer || this.isDestroyed) return
    if (state.reconnectAttempts >= this.config.maxReconnectAttempts) {
      this.registry.setState('error')
      return
    }
    this.registry.setState('error')
    const delay = state.backoffDelay
    state.reconnectTimer = setTimeout(() => {
      state.reconnectTimer = null
      if (!this.isDestroyed && this.serviceHasSubscribers(state.service)) {
        this.startStream(state.service)
      }
    }, delay)
    state.backoffDelay = Math.min(state.backoffDelay * 2 + Math.random() * 1000, MAX_BACKOFF_MS)
  }

  private stopStream(serviceName: string): void {
    const state = this.streams.get(serviceName)
    if (!state) return
    if (state.reconnectTimer) {
      clearTimeout(state.reconnectTimer)
      state.reconnectTimer = null
    }
    state.controller.abort()
    state.isStreaming = false
    this.streams.delete(serviceName)
  }
}

interface ServiceStreamState {
  service: string
  controller: AbortController
  reconnectAttempts: number
  backoffDelay: number
  reconnectTimer: ReturnType<typeof setTimeout> | null
  isStreaming: boolean
}

// =============================================================================
// Singletons + factory wiring
// =============================================================================

let localLogManager: ConnectLocalLogStreamManager | null = null
let azureLogManager: ConnectAzureLogStreamManager | null = null

function getLocalLogManager(): ConnectLocalLogStreamManager {
  localLogManager ??= new ConnectLocalLogStreamManager(createLogsClient())
  return localLogManager
}

function getAzureLogManager(): ConnectAzureLogStreamManager {
  azureLogManager ??= new ConnectAzureLogStreamManager(createAzureClient())
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
   * Override the Connect transport for the active manager. Tests pass
   * `createRouterTransport(...)`; production omits it. Providing a
   * transport bypasses the singleton so each test instance gets its
   * own isolated manager - applies to BOTH local and azure modes
   * since both run on Connect.
   */
  transport?: Transport
}

export interface UseSharedLogStreamReturn {
  connectionState: ConnectionState
  /**
   * Total number of log entries the server reported as dropped on the
   * current Connect stream. Local: drop-OLDEST back-pressure.
   * Azure: realtime-mode buffer overflow (polling mode never drops
   * and emits a degraded `streamStatus` instead). Resets to 0 when
   * all subscribers disconnect.
   */
  droppedCount: number
  /**
   * Latest server-emitted stream status, or `null` if none received
   * yet (or in local mode, which has no equivalent signal). Azure
   * surfaces this so the LogsView can render polling health and
   * realtime↔polling mode flips.
   */
  streamStatus: StreamStatus | null
}

/**
 * Subscribe a component to the shared local-log or Azure-log Connect
 * stream. Returns the connection state, a running drop counter, and
 * the latest server-emitted stream status (Azure only).
 *
 * Public surface preserved from the WebSocket era: same name, same
 * positional options, same connectionState semantics. `droppedCount`,
 * `streamStatus`, and `transport` are additive fields - existing
 * call sites that destructure only `connectionState` continue to
 * compile.
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
      if (transport) {
        return new ConnectAzureLogStreamManager(createAzureClient(transport))
      }
      return getAzureLogManager()
    }
    if (transport) {
      return new ConnectLocalLogStreamManager(createLogsClient(transport))
    }
    return getLocalLogManager()
  }, [mode, transport])

  const [connectionState, setConnectionState] = useState<ConnectionState>(() => manager.getState())
  const [droppedCount, setDroppedCount] = useState<number>(() => manager.getDroppedCount())
  const [streamStatus, setStreamStatus] = useState<StreamStatus | null>(null)

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

  // Stream-status subscription (Azure only emits real values; local
  // emits null once for interface uniformity).
  useEffect(() => {
    return manager.subscribeToStreamStatus((status) => {
      if (!isMountedRef.current) return
      setStreamStatus(status)
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
  // the test path (transport injected) creates a fresh manager per
  // hook so we both can - and must - tear it down.
  useEffect(() => {
    if (!transport) return undefined
    return () => {
      manager.destroy()
    }
  }, [manager, transport])

  return { connectionState, droppedCount, streamStatus }
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
