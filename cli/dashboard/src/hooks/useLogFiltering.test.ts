/**
 * Tests for useLogFiltering hook
 * Validates log filtering, classification, and level detection
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useLogFiltering, type PaneLogLevel } from './useLogFiltering'
import type { LogEntry } from '@/components/LogsPane'
import { LOG_LEVELS } from '@/lib/log-utils'

// Mock useLogClassifications
const mockGetClassificationForText = vi.fn()
vi.mock('./useLogClassifications', () => ({
  useLogClassifications: () => ({
    getClassificationForText: mockGetClassificationForText,
  }),
}))

describe('useLogFiltering', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetClassificationForText.mockReturnValue(null) // Default: no classification override
  })

  const createLogEntry = (overrides?: Partial<LogEntry>): LogEntry => ({
    service: 'test-service',
    message: 'Test log message',
    level: LOG_LEVELS.INFO,
    timestamp: '2025-12-25T12:00:00Z',
    isStderr: false,
    ...overrides,
  })

  describe('initialization', () => {
    it('should return all logs when no filters applied', () => {
      const logs = [
        createLogEntry({ message: 'Info message' }),
        createLogEntry({ message: 'Another message' }),
      ]
      const levelFilter = new Set<PaneLogLevel>(['info', 'warning', 'error'])

      const { result } = renderHook(() => useLogFiltering(logs, '', levelFilter))

      expect(result.current.filteredLogs).toEqual(logs)
    })

    it('should return pane status functions', () => {
      const { result } = renderHook(() =>
        useLogFiltering([], '', new Set(['info', 'warning', 'error']))
      )

      expect(typeof result.current.getPaneLogLevel).toBe('function')
      expect(typeof result.current.isErrorLine).toBe('function')
      expect(typeof result.current.isWarningLine).toBe('function')
      expect(result.current.paneStatus).toBeDefined()
    })
  })

  describe('text search filtering', () => {
    it('should filter logs by search term (case insensitive)', () => {
      const logs = [
        createLogEntry({ message: 'Error connecting to database' }),
        createLogEntry({ message: 'Successfully connected' }),
        createLogEntry({ message: 'Connection timeout' }),
      ]
      const levelFilter = new Set<PaneLogLevel>(['info', 'warning', 'error'])

      const { result } = renderHook(() => useLogFiltering(logs, 'error', levelFilter))

      expect(result.current.filteredLogs).toHaveLength(1)
      expect(result.current.filteredLogs[0].message).toContain('Error')
    })

    it('should be case insensitive in search', () => {
      const logs = [
        createLogEntry({ message: 'ERROR occurred' }),
        createLogEntry({ message: 'error in process' }),
        createLogEntry({ message: 'No issues' }),
      ]
      const levelFilter = new Set<PaneLogLevel>(['info', 'warning', 'error'])

      const { result } = renderHook(() => useLogFiltering(logs, 'ERROR', levelFilter))

      expect(result.current.filteredLogs).toHaveLength(2)
    })

    it('should match partial strings', () => {
      const logs = [
        createLogEntry({ message: 'Connection established' }),
        createLogEntry({ message: 'Disconnected from server' }),
        createLogEntry({ message: 'Server started' }),
      ]
      const levelFilter = new Set<PaneLogLevel>(['info', 'warning', 'error'])

      const { result } = renderHook(() => useLogFiltering(logs, 'connect', levelFilter))

      expect(result.current.filteredLogs).toHaveLength(2)
    })

    it('should filter out logs without messages', () => {
      const logs = [
        createLogEntry({ message: 'Valid message' }),
        { ...createLogEntry(), message: undefined as unknown as string },
        createLogEntry({ message: '' }),
      ]
      const levelFilter = new Set<PaneLogLevel>(['info', 'warning', 'error'])

      const { result } = renderHook(() => useLogFiltering(logs, '', levelFilter))

      // Should exclude entries without valid messages
      expect(result.current.filteredLogs.length).toBeLessThan(logs.length)
    })
  })

  describe('level filtering', () => {
    it('should filter by info level', () => {
      const logs = [
        createLogEntry({ message: 'Info message', level: LOG_LEVELS.INFO }),
        createLogEntry({ message: 'Error: failed', level: LOG_LEVELS.ERROR }),
        createLogEntry({ message: 'Warning: deprecated', level: LOG_LEVELS.WARNING }),
      ]
      const levelFilter = new Set<PaneLogLevel>(['info'])

      const { result } = renderHook(() => useLogFiltering(logs, '', levelFilter))

      expect(result.current.filteredLogs).toHaveLength(1)
      expect(result.current.filteredLogs[0].level).toBe(LOG_LEVELS.INFO)
    })

    it('should filter by error level', () => {
      const logs = [
        createLogEntry({ message: 'Info message', level: LOG_LEVELS.INFO }),
        createLogEntry({ message: 'Error occurred', level: LOG_LEVELS.ERROR }),
        createLogEntry({ message: 'Fatal error', level: LOG_LEVELS.ERROR }),
      ]
      const levelFilter = new Set<PaneLogLevel>(['error'])

      const { result } = renderHook(() => useLogFiltering(logs, '', levelFilter))

      expect(result.current.filteredLogs).toHaveLength(2)
      result.current.filteredLogs.forEach((log) => {
        expect(log.level).toBe(LOG_LEVELS.ERROR)
      })
    })

    it('should filter by warning level', () => {
      const logs = [
        createLogEntry({ message: 'Info message', level: LOG_LEVELS.INFO }),
        createLogEntry({ message: 'Warning: deprecated', level: LOG_LEVELS.WARNING }),
      ]
      const levelFilter = new Set<PaneLogLevel>(['warning'])

      const { result } = renderHook(() => useLogFiltering(logs, '', levelFilter))

      expect(result.current.filteredLogs).toHaveLength(1)
      expect(result.current.filteredLogs[0].level).toBe(LOG_LEVELS.WARNING)
    })

    it('should allow multiple level filters', () => {
      const logs = [
        createLogEntry({ message: 'Info message', level: LOG_LEVELS.INFO }),
        createLogEntry({ message: 'Error occurred', level: LOG_LEVELS.ERROR }),
        createLogEntry({ message: 'Warning: deprecated', level: LOG_LEVELS.WARNING }),
      ]
      const levelFilter = new Set<PaneLogLevel>(['info', 'error'])

      const { result } = renderHook(() => useLogFiltering(logs, '', levelFilter))

      expect(result.current.filteredLogs).toHaveLength(2)
      expect(result.current.filteredLogs.some((log) => log.level === LOG_LEVELS.INFO)).toBe(true)
      expect(result.current.filteredLogs.some((log) => log.level === LOG_LEVELS.ERROR)).toBe(true)
    })

    it('should show all logs when all levels selected', () => {
      const logs = [
        createLogEntry({ message: 'Info', level: LOG_LEVELS.INFO }),
        createLogEntry({ message: 'Error', level: LOG_LEVELS.ERROR }),
        createLogEntry({ message: 'Warning', level: LOG_LEVELS.WARNING }),
      ]
      const levelFilter = new Set<PaneLogLevel>(['info', 'warning', 'error'])

      const { result } = renderHook(() => useLogFiltering(logs, '', levelFilter))

      expect(result.current.filteredLogs).toEqual(logs)
    })

    it('should show no logs when no levels selected', () => {
      const logs = [
        createLogEntry({ message: 'Info', level: LOG_LEVELS.INFO }),
        createLogEntry({ message: 'Error', level: LOG_LEVELS.ERROR }),
      ]
      const levelFilter = new Set<PaneLogLevel>([])

      const { result } = renderHook(() => useLogFiltering(logs, '', levelFilter))

      expect(result.current.filteredLogs).toEqual([])
    })
  })

  describe('combined filtering', () => {
    it('should apply both text and level filters', () => {
      const logs = [
        createLogEntry({ message: 'Error connecting', level: LOG_LEVELS.ERROR }),
        createLogEntry({ message: 'Error parsing', level: LOG_LEVELS.ERROR }),
        createLogEntry({ message: 'Info connecting', level: LOG_LEVELS.INFO }),
        createLogEntry({ message: 'Warning parsing', level: LOG_LEVELS.WARNING }),
      ]
      const levelFilter = new Set<PaneLogLevel>(['error'])

      const { result } = renderHook(() => useLogFiltering(logs, 'connecting', levelFilter))

      expect(result.current.filteredLogs).toHaveLength(1)
      expect(result.current.filteredLogs[0].message).toBe('Error connecting')
    })
  })

  describe('getPaneLogLevel', () => {
    it('should return error for error level logs', () => {
      const log = createLogEntry({ level: LOG_LEVELS.ERROR })
      const { result } = renderHook(() =>
        useLogFiltering([log], '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.getPaneLogLevel(log)).toBe('error')
    })

    it('should return warning for warning level logs', () => {
      const log = createLogEntry({ level: LOG_LEVELS.WARNING })
      const { result } = renderHook(() =>
        useLogFiltering([log], '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.getPaneLogLevel(log)).toBe('warning')
    })

    it('should return info for info level logs', () => {
      const log = createLogEntry({ level: LOG_LEVELS.INFO })
      const { result } = renderHook(() =>
        useLogFiltering([log], '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.getPaneLogLevel(log)).toBe('info')
    })

    it('should detect error from message content', () => {
      const log = createLogEntry({ message: 'Error: connection failed', level: LOG_LEVELS.INFO })
      const { result } = renderHook(() =>
        useLogFiltering([log], '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.getPaneLogLevel(log)).toBe('error')
    })

    it('should detect warning from message content', () => {
      const log = createLogEntry({ message: 'Warning: deprecated API', level: LOG_LEVELS.INFO })
      const { result } = renderHook(() =>
        useLogFiltering([log], '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.getPaneLogLevel(log)).toBe('warning')
    })
  })

  describe('classification override', () => {
    it('should override level with classification', () => {
      mockGetClassificationForText.mockReturnValue('error')

      const log = createLogEntry({ message: 'Custom error pattern', level: LOG_LEVELS.INFO })
      const { result } = renderHook(() =>
        useLogFiltering([log], '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.getPaneLogLevel(log)).toBe('error')
    })

    it('should downgrade error to info with classification', () => {
      mockGetClassificationForText.mockReturnValue('info')

      const log = createLogEntry({ message: 'Expected error', level: LOG_LEVELS.ERROR })
      const { result } = renderHook(() =>
        useLogFiltering([log], '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.getPaneLogLevel(log)).toBe('info')
    })

    it('should use classification for isErrorLine', () => {
      mockGetClassificationForText.mockReturnValue('error')

      const log = createLogEntry({ message: 'Custom error' })
      const { result } = renderHook(() =>
        useLogFiltering([log], '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.isErrorLine('Custom error')).toBe(true)
    })

    it('should use classification for isWarningLine', () => {
      mockGetClassificationForText.mockReturnValue('warning')

      const log = createLogEntry({ message: 'Custom warning' })
      const { result } = renderHook(() =>
        useLogFiltering([log], '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.isWarningLine('Custom warning')).toBe(true)
    })

    it('should prevent false positives with info classification', () => {
      mockGetClassificationForText.mockReturnValue('info')

      const log = createLogEntry({ message: 'Error-like but classified as info' })
      const { result } = renderHook(() =>
        useLogFiltering([log], '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.isErrorLine('Error-like but classified as info')).toBe(false)
      expect(result.current.isWarningLine('Error-like but classified as info')).toBe(false)
    })
  })

  describe('paneStatus', () => {
    it('should return error when any log is error', () => {
      const logs = [
        createLogEntry({ message: 'Info' }),
        createLogEntry({ message: 'Error occurred', level: LOG_LEVELS.ERROR }),
      ]
      const { result } = renderHook(() =>
        useLogFiltering(logs, '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.paneStatus).toBe('error')
    })

    it('should return warning when any log is warning but no errors', () => {
      const logs = [
        createLogEntry({ message: 'Info' }),
        createLogEntry({ message: 'Warning: deprecated', level: LOG_LEVELS.WARNING }),
      ]
      const { result } = renderHook(() =>
        useLogFiltering(logs, '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.paneStatus).toBe('warning')
    })

    it('should return info when all logs are info', () => {
      const logs = [
        createLogEntry({ message: 'Info 1' }),
        createLogEntry({ message: 'Info 2' }),
      ]
      const { result } = renderHook(() =>
        useLogFiltering(logs, '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.paneStatus).toBe('info')
    })

    it('should prioritize error over warning in status', () => {
      const logs = [
        createLogEntry({ message: 'Warning: test', level: LOG_LEVELS.WARNING }),
        createLogEntry({ message: 'Error: test', level: LOG_LEVELS.ERROR }),
      ]
      const { result } = renderHook(() =>
        useLogFiltering(logs, '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.paneStatus).toBe('error')
    })

    it('should detect error from message content in status', () => {
      const logs = [createLogEntry({ message: 'Error: something went wrong', level: LOG_LEVELS.INFO })]
      const { result } = renderHook(() =>
        useLogFiltering(logs, '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.paneStatus).toBe('error')
    })
  })

  describe('performance and memoization', () => {
    it('should not recalculate filtered logs when inputs unchanged', () => {
      const logs = [createLogEntry({ message: 'Test' })]
      const levelFilter = new Set<PaneLogLevel>(['info', 'warning', 'error'])

      const { result, rerender } = renderHook(
        ({ logs, search, filter }) => useLogFiltering(logs, search, filter),
        { initialProps: { logs, search: '', filter: levelFilter } }
      )

      const firstResult = result.current.filteredLogs

      // Rerender with same inputs
      rerender({ logs, search: '', filter: levelFilter })

      // Should return same reference (memoized)
      expect(result.current.filteredLogs).toBe(firstResult)
    })

    it('should recalculate when search term changes', () => {
      const logs = [
        createLogEntry({ message: 'Error message' }),
        createLogEntry({ message: 'Info message' }),
      ]
      const levelFilter = new Set<PaneLogLevel>(['info', 'warning', 'error'])

      const { result, rerender } = renderHook(
        ({ logs, search, filter }) => useLogFiltering(logs, search, filter),
        { initialProps: { logs, search: '', filter: levelFilter } }
      )

      expect(result.current.filteredLogs).toHaveLength(2)

      // Change search term
      rerender({ logs, search: 'error', filter: levelFilter })

      expect(result.current.filteredLogs).toHaveLength(1)
    })

    it('should recalculate when level filter changes', () => {
      const logs = [
        createLogEntry({ message: 'Info', level: LOG_LEVELS.INFO }),
        createLogEntry({ message: 'Error', level: LOG_LEVELS.ERROR }),
      ]

      const { result, rerender } = renderHook(
        ({ logs, search, filter }) => useLogFiltering(logs, search, filter),
        {
          initialProps: {
            logs,
            search: '',
            filter: new Set<PaneLogLevel>(['info', 'warning', 'error']),
          },
        }
      )

      expect(result.current.filteredLogs).toHaveLength(2)

      // Change filter to error only
      rerender({ logs, search: '', filter: new Set<PaneLogLevel>(['error']) })

      expect(result.current.filteredLogs).toHaveLength(1)
    })
  })

  describe('edge cases', () => {
    it('should handle empty logs array', () => {
      const { result } = renderHook(() =>
        useLogFiltering([], '', new Set(['info', 'warning', 'error']))
      )

      expect(result.current.filteredLogs).toEqual([])
      expect(result.current.paneStatus).toBe('info')
    })

    it('should handle logs with special characters in search', () => {
      const logs = [
        createLogEntry({ message: 'Error: connection [failed]' }),
        createLogEntry({ message: 'Normal message' }),
      ]
      const levelFilter = new Set<PaneLogLevel>(['info', 'warning', 'error'])

      const { result } = renderHook(() => useLogFiltering(logs, '[failed]', levelFilter))

      expect(result.current.filteredLogs).toHaveLength(1)
    })

    it('should handle logs with unicode characters', () => {
      const logs = [
        createLogEntry({ message: 'Processed 日本語 data' }),
        createLogEntry({ message: 'Normal message' }),
      ]
      const levelFilter = new Set<PaneLogLevel>(['info', 'warning', 'error'])

      const { result } = renderHook(() => useLogFiltering(logs, '日本語', levelFilter))

      expect(result.current.filteredLogs).toHaveLength(1)
    })
  })
})
