import { useMemo } from 'react'

export type AzureTimeRange = {
  preset: '15m' | '30m' | '6h' | '24h'
  end?: Date
}

export type AzureTimeRangeBounds = {
  start: Date
  end: Date
}

export const DEFAULT_AZURE_TIME_RANGE: AzureTimeRange = { preset: '15m' }

export function getAzureTimeRangeBounds(timeRange: AzureTimeRange, now: Date): AzureTimeRangeBounds | null {
  const end = timeRange.end ?? now
  const durationMs = {
    '15m': 15 * 60 * 1000,
    '30m': 30 * 60 * 1000,
    '6h': 6 * 60 * 60 * 1000,
    '24h': 24 * 60 * 60 * 1000,
  }[timeRange.preset]

  return { start: new Date(end.getTime() - durationMs), end }
}

export function formatAzureRangeTimestamp(value: Date): string {
  return value.toISOString().replace('T', ' ').replace(/\.\d{3}Z$/, 'Z')
}

export function formatAzureTimeRangePreset(preset: AzureTimeRange['preset']): string {
  switch (preset) {
    case '15m':
      return '15 minutes'
    case '30m':
      return '30 minutes'
    case '6h':
      return '6 hours'
    case '24h':
      return '24 hours'
  }
}

export function suggestAzureTimeRangePreset(preset: AzureTimeRange['preset']): string {
  if (preset === '15m' || preset === '30m' || preset === '6h') {
    return '24 hours'
  }

  return 'a wider range'
}

export function useAzureTimeRange(timeRange?: AzureTimeRange) {
  const resolvedTimeRange = useMemo(
    () => timeRange ?? DEFAULT_AZURE_TIME_RANGE,
    [timeRange]
  )

  return {
    timeRange: resolvedTimeRange,
    getAzureTimeRangeBounds,
    formatAzureRangeTimestamp,
    formatAzureTimeRangePreset,
    suggestAzureTimeRangePreset,
  }
}
