/**
 * Format a timestamp as relative time (e.g., "5m ago")
 */
export function formatRelativeTime(timeStr?: string): string {
  if (!timeStr) return 'N/A'
  
  try {
    const date = new Date(timeStr)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const seconds = Math.floor(diff / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)

    if (seconds < 60) return `${seconds}s ago`
    if (minutes < 60) return `${minutes}m ago`
    if (hours < 24) return `${hours}h ago`
    return `${days}d ago`
  } catch {
    return timeStr
  }
}

/**
 * Format a start time for table display (HH:MM:SS)
 */
export function formatStartTime(timeStr?: string): string {
  if (!timeStr) return '-'
  try {
    const date = new Date(timeStr)
    if (Number.isNaN(date.getTime())) {
      return timeStr
    }
    return date.toLocaleTimeString('en-US', { 
      hour: '2-digit', 
      minute: '2-digit', 
      second: '2-digit' 
    })
  } catch {
    return timeStr
  }
}

/**
 * Format a timestamp for log display (MM-DD HH:MM:SS.mmm)
 * Compact format showing date and time with millisecond precision.
 * Timezone offset is omitted as it's consistent across all logs.
 */
export function formatLogTimestamp(timestamp: string): string {
  const trimmed = (timestamp ?? '').trim()
  try {
    const date = new Date(trimmed)
    if (Number.isNaN(date.getTime())) {
      return trimmed
    }

    const month = (date.getMonth() + 1).toString().padStart(2, '0')
    const day = date.getDate().toString().padStart(2, '0')
    const hours = date.getHours().toString().padStart(2, '0')
    const minutes = date.getMinutes().toString().padStart(2, '0')
    const seconds = date.getSeconds().toString().padStart(2, '0')
    const ms = date.getMilliseconds().toString().padStart(3, '0')

    return `${month}-${day} ${hours}:${minutes}:${seconds}.${ms}`
  } catch {
    return trimmed
  }
}

/**
 * Format response time from nanoseconds to human-readable string
 */
export function formatResponseTime(nanos?: number): string {
  if (!nanos || nanos <= 0) return '-'
  const ms = nanos / 1_000_000
  if (ms < 1) return '<1ms'
  if (ms < 1000) return `${Math.round(ms)}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

/**
 * Format uptime from nanoseconds to human-readable string
 */
export function formatUptime(nanos?: number): string {
  if (!nanos || nanos <= 0) return '-'
  const seconds = nanos / 1_000_000_000
  if (seconds < 60) return `${Math.round(seconds)}s`
  const minutes = seconds / 60
  if (minutes < 60) return `${Math.round(minutes)}m`
  const hours = minutes / 60
  if (hours < 24) return `${Math.floor(hours)}h ${Math.round(minutes % 60)}m`
  const days = hours / 24
  return `${Math.floor(days)}d ${Math.round(hours % 24)}h`
}
