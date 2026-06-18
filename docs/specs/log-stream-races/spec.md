---
issue: https://github.com/jongio/azd-app/pull/340
author: "@jongio"
status: shipped
---

# Fix Log Streaming Race Conditions

## Problem

When running `azd app run`, the dashboard sometimes fails to display logs for one
or more services. The user sees an empty log pane that never receives entries
despite the service clearly producing output. A page refresh fixes it — but only
if the service's buffer happens to exist by then. This is a regression-prone,
non-deterministic UX bug that erodes trust in the dashboard's reliability.

Root cause: several race conditions in the CLI→dashboard log streaming pipeline
mean that services starting after the stream opens, entries produced during the
backfill window, and entries lost during stream reconnection are all silently
dropped.

## Goals

- Services that start after the dashboard's log stream opens appear dynamically
  without requiring a page refresh
- No entries are lost during the backfill send window (startup burst)
- After a stream reconnect, the dashboard re-fetches initial history to close the
  gap between disconnect and new stream
- Pump goroutines start draining subscriber channels immediately upon subscribe
  (before backfill I/O)
- Live local logs stream the instant a pane mounts, without waiting on the health
  stream to connect

## Non-Goals

- Guaranteed exactly-once delivery (drop-oldest back-pressure is acceptable)
- Changing the Connect-RPC wire protocol or adding new proto messages
- Real-time service removal notifications (existing reconnect-to-discover is fine)
- Persisting logs across dashboard restarts (out of scope)

## Solution

### 1. Dynamic service subscription (backend)

Add a `OnBufferAdded() chan string` notification mechanism to `LogManager`. When
`CreateBuffer` registers a new service, all listeners receive the service name.
`StreamLocalLogs` (in all-services mode) selects on this channel in its main loop
and dynamically subscribes + starts a pump goroutine for new services mid-stream.

### 2. Reorder pump start before backfill (backend)

Start pump goroutines immediately after subscribing — before sending backfill
entries to the client. This ensures the 100-capacity subscriber channels drain
continuously and don't overflow from `broadcast()` drop-on-full during the
(potentially slow) backfill send phase.

### 3. Reconnect triggers history re-fetch (frontend)

Add a `reconnectGeneration` counter to `SubscriberRegistry` in the shared log
stream manager. It increments on each successful reconnection. The `useLogsStream`
hook incorporates this counter to detect reconnects and reset its fetch state,
triggering a fresh `GetLogs` unary call to fill the gap.

### 4. Correct defer ordering (backend)

Fix misleading comment about defer LIFO ordering and restructure the cleanup path
with explicit, correctly-documented defers.

### 5. Decouple local live logs from the health stream (frontend)

The grid view's per-service panes drove live logs through `useLogsStream`, which
gated `shouldUseSharedStream` on the health stream's `connected` signal. That
signal only flips true after the server's first health probe of ALL services
completes, and a still-starting service blocks that probe for the full timeout.
The result: live local logs were delayed for every service until the slowest
health probe finished, even though the `StreamLocalLogs` RPC was ready
immediately. Local logs now stream as soon as the pane mounts, matching the
unified view (`LogsView`), which never had the health gate. Azure realtime stays
gated on `connected` because Log Analytics genuinely needs the backend reachable.


## Alternatives Considered

- **Frontend-only reconnect with backfill > 0**: Would require the server to dedup
  entries the client already has. Adds wire complexity and doesn't solve issue #1
  (new services invisible).
- **Periodic poll of ServiceNames from the frontend**: Would work but adds
  unnecessary latency (polling interval) and network chatter for a rare event.
  Push notification is cleaner.
- **LogManager event bus (full pub/sub)**: Over-engineered. A simple channel-based
  listener list is sufficient for the single consumer (StreamLocalLogs).

## Risks & Rabbit Holes

- The `OnBufferAdded` listener channel (capacity 16) can fill if many services
  start simultaneously. The non-blocking send means the notification is dropped,
  but the service still exists in the LogManager — a worst case requires one more
  reconnect cycle to discover it. Acceptable for the expected service count (<20).
- Backfill entries and live-stream entries can overlap (duplicate delivery) for
  entries that arrive between subscribe and backfill-snapshot. The ring buffer's
  drop-oldest semantics and the dashboard's append-only log list mean duplicates
  are visually harmless. A future enhancement could add sequence-based dedup.
