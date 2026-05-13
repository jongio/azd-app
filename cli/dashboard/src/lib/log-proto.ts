/**
 * Shared proto-LogEntry -> dashboard-LogEntry adapters.
 *
 * The Connect-RPC `LogEntry` (proto) and the dashboard's `LogEntry`
 * (plain TS interface consumed by LogsPane/LogsView) diverge in a
 * handful of fields: the proto carries enum-typed `level`/`stream`
 * and a protobuf `Timestamp`, whereas the dashboard uses a three-
 * bucket numeric level, an `isStderr` boolean, and an ISO string.
 *
 * Keeping the translation here (and not inside LogsView or
 * useSharedLogStream) avoids behavioural drift: multiple call sites
 * (unary GetLogs/GetAzureLogs, server-streaming StreamLocalLogs and
 * StreamAzureLogs) all route through the same mapper, so a paused
 * pane and a realtime pane see byte-identical entries.
 */
import {
  LogLevel,
  LogStream,
  type LogEntry as ProtoLogEntry,
} from '@/gen/proto/azdapp/v1/common_pb.js'

/**
 * Dashboard-facing log shape. Matches `LogsPane.LogEntry` but is
 * re-declared here so `lib/` has no circular import from `components/`.
 */
export interface DashboardLogEntry {
  service: string
  message: string
  /** 1=info, 2=warning, 3=error. */
  level: number
  /** ISO-8601 UTC timestamp. */
  timestamp: string
  isStderr: boolean
}

/**
 * Collapse the proto LogLevel enum onto the dashboard's three numeric
 * buckets (1=info, 2=warning, 3=error).
 */
export function protoLevelToNumeric(level: LogLevel): number {
  switch (level) {
    case LogLevel.ERROR:
    case LogLevel.FATAL:
      return 3
    case LogLevel.WARN:
      return 2
    default:
      return 1
  }
}

/**
 * Convert a protobuf Timestamp (seconds+nanos) into an ISO string.
 * Falls back to "now" if the timestamp is missing; this matches the
 * legacy REST handler which filled in the server time when the
 * underlying log source had no timestamp.
 */
export function timestampToIso(ts: ProtoLogEntry['timestamp']): string {
  if (!ts) return new Date().toISOString()
  const seconds = typeof ts.seconds === 'bigint' ? Number(ts.seconds) : Number(ts.seconds ?? 0)
  const nanos = typeof ts.nanos === 'number' ? ts.nanos : 0
  return new Date(seconds * 1000 + Math.floor(nanos / 1e6)).toISOString()
}

/**
 * Map a proto LogEntry to the dashboard's plain-JS LogEntry shape.
 * The same mapper is used for historical fetches (unary RPCs) and
 * streamed entries (server-streaming RPCs) so both paths render
 * identically.
 */
export function protoLogEntryToView(proto: ProtoLogEntry): DashboardLogEntry {
  return {
    service: proto.service,
    message: proto.message,
    level: protoLevelToNumeric(proto.level),
    timestamp: timestampToIso(proto.timestamp),
    isStderr: proto.stream === LogStream.STDERR,
  }
}
