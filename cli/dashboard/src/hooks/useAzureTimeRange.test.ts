/**
 * Tests for useAzureTimeRange hook
 * Validates time range utilities and hook behavior
 */
import { describe, it, expect } from 'vitest'
import { renderHook } from '@testing-library/react'
import {
  useAzureTimeRange,
  getAzureTimeRangeBounds,
  formatAzureRangeTimestamp,
  formatAzureTimeRangePreset,
  suggestAzureTimeRangePreset,
  DEFAULT_AZURE_TIME_RANGE,
  type AzureTimeRange,
} from './useAzureTimeRange'

describe('useAzureTimeRange', () => {
  describe('DEFAULT_AZURE_TIME_RANGE', () => {
    it('should have 15m preset as default', () => {
      expect(DEFAULT_AZURE_TIME_RANGE).toEqual({ preset: '15m' })
    })
  })

  describe('getAzureTimeRangeBounds', () => {
    const now = new Date('2025-12-25T12:00:00.000Z')

    it('should calculate 15m range correctly', () => {
      const timeRange: AzureTimeRange = { preset: '15m' }
      const bounds = getAzureTimeRangeBounds(timeRange, now)

      expect(bounds).not.toBeNull()
      expect(bounds?.end).toEqual(now)
      expect(bounds?.start).toEqual(new Date('2025-12-25T11:45:00.000Z'))
    })

    it('should calculate 30m range correctly', () => {
      const timeRange: AzureTimeRange = { preset: '30m' }
      const bounds = getAzureTimeRangeBounds(timeRange, now)

      expect(bounds).not.toBeNull()
      expect(bounds?.end).toEqual(now)
      expect(bounds?.start).toEqual(new Date('2025-12-25T11:30:00.000Z'))
    })

    it('should calculate 6h range correctly', () => {
      const timeRange: AzureTimeRange = { preset: '6h' }
      const bounds = getAzureTimeRangeBounds(timeRange, now)

      expect(bounds).not.toBeNull()
      expect(bounds?.end).toEqual(now)
      expect(bounds?.start).toEqual(new Date('2025-12-25T06:00:00.000Z'))
    })

    it('should calculate 24h range correctly', () => {
      const timeRange: AzureTimeRange = { preset: '24h' }
      const bounds = getAzureTimeRangeBounds(timeRange, now)

      expect(bounds).not.toBeNull()
      expect(bounds?.end).toEqual(now)
      expect(bounds?.start).toEqual(new Date('2025-12-24T12:00:00.000Z'))
    })

    it('should use custom end time when provided', () => {
      const customEnd = new Date('2025-12-25T10:00:00.000Z')
      const timeRange: AzureTimeRange = { preset: '15m', end: customEnd }
      const bounds = getAzureTimeRangeBounds(timeRange, now)

      expect(bounds).not.toBeNull()
      expect(bounds?.end).toEqual(customEnd)
      expect(bounds?.start).toEqual(new Date('2025-12-25T09:45:00.000Z'))
    })

    it('should handle edge case with custom end at epoch', () => {
      const customEnd = new Date(0)
      const timeRange: AzureTimeRange = { preset: '15m', end: customEnd }
      const bounds = getAzureTimeRangeBounds(timeRange, now)

      expect(bounds).not.toBeNull()
      expect(bounds?.end).toEqual(customEnd)
      // 15 minutes before epoch should be negative timestamp
      expect(bounds?.start.getTime()).toBe(-15 * 60 * 1000)
    })
  })

  describe('formatAzureRangeTimestamp', () => {
    it('should format timestamp correctly', () => {
      const date = new Date('2025-12-25T12:34:56.789Z')
      const formatted = formatAzureRangeTimestamp(date)

      expect(formatted).toBe('2025-12-25 12:34:56Z')
    })

    it('should handle midnight correctly', () => {
      const date = new Date('2025-12-25T00:00:00.000Z')
      const formatted = formatAzureRangeTimestamp(date)

      expect(formatted).toBe('2025-12-25 00:00:00Z')
    })

    it('should handle milliseconds correctly', () => {
      const date = new Date('2025-12-25T12:34:56.123Z')
      const formatted = formatAzureRangeTimestamp(date)

      // Should strip milliseconds
      expect(formatted).toBe('2025-12-25 12:34:56Z')
    })

    it('should handle dates with different millisecond values', () => {
      const date1 = new Date('2025-12-25T12:34:56.000Z')
      const date2 = new Date('2025-12-25T12:34:56.999Z')

      expect(formatAzureRangeTimestamp(date1)).toBe('2025-12-25 12:34:56Z')
      expect(formatAzureRangeTimestamp(date2)).toBe('2025-12-25 12:34:56Z')
    })

    it('should format single-digit hours and minutes with leading zeros', () => {
      const date = new Date('2025-12-25T09:05:03.000Z')
      const formatted = formatAzureRangeTimestamp(date)

      expect(formatted).toBe('2025-12-25 09:05:03Z')
    })
  })

  describe('formatAzureTimeRangePreset', () => {
    it('should format 15m preset', () => {
      expect(formatAzureTimeRangePreset('15m')).toBe('15 minutes')
    })

    it('should format 30m preset', () => {
      expect(formatAzureTimeRangePreset('30m')).toBe('30 minutes')
    })

    it('should format 6h preset', () => {
      expect(formatAzureTimeRangePreset('6h')).toBe('6 hours')
    })

    it('should format 24h preset', () => {
      expect(formatAzureTimeRangePreset('24h')).toBe('24 hours')
    })

    it('should handle all preset values exhaustively', () => {
      const presets: AzureTimeRange['preset'][] = ['15m', '30m', '6h', '24h']
      
      presets.forEach(preset => {
        expect(formatAzureTimeRangePreset(preset)).toBeTruthy()
        expect(typeof formatAzureTimeRangePreset(preset)).toBe('string')
      })
    })
  })

  describe('suggestAzureTimeRangePreset', () => {
    it('should suggest 24h for 15m preset', () => {
      expect(suggestAzureTimeRangePreset('15m')).toBe('24 hours')
    })

    it('should suggest 24h for 30m preset', () => {
      expect(suggestAzureTimeRangePreset('30m')).toBe('24 hours')
    })

    it('should suggest 24h for 6h preset', () => {
      expect(suggestAzureTimeRangePreset('6h')).toBe('24 hours')
    })

    it('should suggest wider range for 24h preset', () => {
      expect(suggestAzureTimeRangePreset('24h')).toBe('a wider range')
    })
  })

  describe('useAzureTimeRange hook', () => {
    it('should return default time range when no argument provided', () => {
      const { result } = renderHook(() => useAzureTimeRange())

      expect(result.current.timeRange).toEqual(DEFAULT_AZURE_TIME_RANGE)
    })

    it('should return custom time range when provided', () => {
      const customRange: AzureTimeRange = { preset: '6h' }
      const { result } = renderHook(() => useAzureTimeRange(customRange))

      expect(result.current.timeRange).toEqual(customRange)
    })

    it('should return custom time range with end date', () => {
      const customEnd = new Date('2025-12-25T10:00:00.000Z')
      const customRange: AzureTimeRange = { preset: '30m', end: customEnd }
      const { result } = renderHook(() => useAzureTimeRange(customRange))

      expect(result.current.timeRange).toEqual(customRange)
    })

    it('should return utility functions', () => {
      const { result } = renderHook(() => useAzureTimeRange())

      expect(result.current.getAzureTimeRangeBounds).toBe(getAzureTimeRangeBounds)
      expect(result.current.formatAzureRangeTimestamp).toBe(formatAzureRangeTimestamp)
      expect(result.current.formatAzureTimeRangePreset).toBe(formatAzureTimeRangePreset)
      expect(result.current.suggestAzureTimeRangePreset).toBe(suggestAzureTimeRangePreset)
    })

    it('should memoize time range when argument does not change', () => {
      const customRange: AzureTimeRange = { preset: '6h' }
      const { result, rerender } = renderHook(
        ({ timeRange }) => useAzureTimeRange(timeRange),
        { initialProps: { timeRange: customRange } }
      )

      const firstResult = result.current.timeRange

      // Rerender with same reference
      rerender({ timeRange: customRange })

      expect(result.current.timeRange).toBe(firstResult)
    })

    it('should update time range when argument changes', () => {
      const range1: AzureTimeRange = { preset: '15m' }
      const range2: AzureTimeRange = { preset: '30m' }

      const { result, rerender } = renderHook(
        ({ timeRange }) => useAzureTimeRange(timeRange),
        { initialProps: { timeRange: range1 } }
      )

      expect(result.current.timeRange).toEqual(range1)

      rerender({ timeRange: range2 })

      expect(result.current.timeRange).toEqual(range2)
    })

    it('should handle switching from custom to undefined', () => {
      const customRange: AzureTimeRange = { preset: '24h' }
      const { result, rerender } = renderHook(
        ({ timeRange }: { timeRange?: AzureTimeRange }) => useAzureTimeRange(timeRange),
        { initialProps: { timeRange: customRange as AzureTimeRange | undefined } }
      )

      expect(result.current.timeRange).toEqual(customRange)

      // Switch to undefined (should use default)
      rerender({ timeRange: undefined })

      expect(result.current.timeRange).toEqual(DEFAULT_AZURE_TIME_RANGE)
    })

    it('should handle switching from undefined to custom', () => {
      const { result, rerender } = renderHook(
        ({ timeRange }: { timeRange?: AzureTimeRange }) => useAzureTimeRange(timeRange),
        { initialProps: { timeRange: undefined as AzureTimeRange | undefined } }
      )

      expect(result.current.timeRange).toEqual(DEFAULT_AZURE_TIME_RANGE)

      const customRange: AzureTimeRange = { preset: '6h' }
      rerender({ timeRange: customRange as AzureTimeRange | undefined })

      expect(result.current.timeRange).toEqual(customRange)
    })
  })

  describe('integration: utility functions work with hook output', () => {
    it('should calculate bounds from hook timeRange', () => {
      const customRange: AzureTimeRange = { preset: '30m' }
      const { result } = renderHook(() => useAzureTimeRange(customRange))

      const now = new Date('2025-12-25T12:00:00.000Z')
      const bounds = result.current.getAzureTimeRangeBounds(result.current.timeRange, now)

      expect(bounds).not.toBeNull()
      expect(bounds?.start).toEqual(new Date('2025-12-25T11:30:00.000Z'))
      expect(bounds?.end).toEqual(now)
    })

    it('should format preset from hook timeRange', () => {
      const customRange: AzureTimeRange = { preset: '6h' }
      const { result } = renderHook(() => useAzureTimeRange(customRange))

      const formatted = result.current.formatAzureTimeRangePreset(result.current.timeRange.preset)

      expect(formatted).toBe('6 hours')
    })

    it('should suggest wider range from hook timeRange', () => {
      const customRange: AzureTimeRange = { preset: '15m' }
      const { result } = renderHook(() => useAzureTimeRange(customRange))

      const suggestion = result.current.suggestAzureTimeRangePreset(result.current.timeRange.preset)

      expect(suggestion).toBe('24 hours')
    })
  })
})
