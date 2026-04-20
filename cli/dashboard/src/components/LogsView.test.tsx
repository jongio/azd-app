/**
 * LogsView component tests.
 *
 * After the Connect-RPC migration, the component no longer talks to
 * REST (`/api/services`, `/api/logs`) or WebSockets (`/api/logs/stream`,
 * `/api/azure/logs/stream`). Tests stub the three seams the component
 * still depends on:
 *
 *   1. `useServicesContext` - source of the services dropdown. The
 *      legacy `/api/services` fetch is gone; the provider streams
 *      updates via LifecycleService.StreamBroadcast.
 *   2. `createLogsClient` / `createAzureClient` - unary historical
 *      fetches that replaced `GET /api/logs` and `GET /api/azure/logs`.
 *   3. `useSharedLogStream` - server-streaming live updates that
 *      replaced the `/api/logs/stream` WebSocket. Tests drive live
 *      entries by invoking the captured `onLogEntry` callback.
 *
 * The historical tests exercising pause, clear-debounce, the 1000-
 * entry cap, and the clearAllTrigger race are preserved verbatim -
 * they now fire entries via the stub callback instead of the removed
 * WebSocket mock.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { LogsView } from '@/components/LogsView'
import { mockLogs, mockLogsWithAnsi } from '@/test/mocks'

// Pass proto LogEntry-shaped values through unchanged. The component
// calls `protoLogEntryToView` on every entry from the Connect
// unary fetch; making the mapper an identity lets us hand it the
// dashboard `LogEntry` fixtures directly without re-building proto
// objects just to assert on dashboard fields.
vi.mock('@/lib/log-proto', () => ({
  protoLogEntryToView: <T,>(entry: T) => entry,
}))

// ServicesContext: the component reads `serviceNames` for the
// services dropdown. Return the same three mock services every test
// used to see through the legacy `/api/services` fetch so the
// dropdown assertions still pass without re-authoring them.
vi.mock('@/contexts/ServicesContext', () => ({
  useServicesContext: () => ({
    services: [],
    serviceNames: ['api', 'web', 'database'],
    loading: false,
    error: null,
    connected: true,
    refetch: vi.fn(),
    getService: vi.fn(),
  }),
  // Included so any accidental import of the provider from a test
  // doesn't break - a noop wrapper is safe here because the component
  // pulls everything through `useServicesContext`.
  ServicesProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

// useSharedLogStream: capture the options on each render so tests
// can (a) assert wiring (mode/enabled/serviceName) and (b) invoke
// the stored `onLogEntry` to simulate a live server-streamed entry,
// the same way the old WebSocket tests pushed messages into
// `wsRef.current.onmessage`.
type LogEntryArg = {
  service: string
  message: string
  level: number
  timestamp: string
  isStderr: boolean
}
type SharedStreamOpts = {
  serviceName: string
  enabled: boolean
  mode: 'local' | 'azure'
  onLogEntry: (entry: LogEntryArg) => void
  since?: string
}

let latestSharedArgs: SharedStreamOpts | null = null
vi.mock('@/hooks/useSharedLogStream', () => ({
  useSharedLogStream: (opts: SharedStreamOpts) => {
    latestSharedArgs = opts
    return { connectionState: 'connected' as const, droppedCount: 0 }
  },
}))

// Connect factories: the component calls these on every historical
// fetch. Tests mutate the mock return values per-case. We preserve
// all other exports from `@/lib/connectClient` because
// `useCodespaceEnv` (pulled in transitively by LogsView) dials
// `createLifecycleClient` and must keep working against the default
// singleton transport.
const getLogsMock = vi.fn()
const getAzureLogsMock = vi.fn()
vi.mock('@/lib/connectClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/connectClient')>()
  return {
    ...actual,
    createLogsClient: () => ({ getLogs: getLogsMock }),
    createAzureClient: () => ({ getAzureLogs: getAzureLogsMock }),
  }
})

function pushLive(entry: LogEntryArg): void {
  if (!latestSharedArgs) throw new Error('useSharedLogStream was never called')
  latestSharedArgs.onLogEntry(entry)
}

describe('LogsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    latestSharedArgs = null
    getLogsMock.mockResolvedValue({ entries: mockLogs })
    getAzureLogsMock.mockResolvedValue({ entries: [] })
  })

  it('should render logs view with controls', async () => {
    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText('All Services')).toBeInTheDocument()
    })

    expect(screen.getByPlaceholderText('Search logs...')).toBeInTheDocument()
  })

  it('should fetch and display logs on mount', async () => {
    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    expect(screen.getByText(/Application started successfully/)).toBeInTheDocument()
  })

  it('should populate service filter dropdown', async () => {
    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByRole('option', { name: 'All Services' })).toBeInTheDocument()
      expect(screen.getByRole('option', { name: 'api' })).toBeInTheDocument()
      expect(screen.getByRole('option', { name: 'web' })).toBeInTheDocument()
    })
  })

  it('should filter logs by service', async () => {
    const user = userEvent.setup()

    // When the user selects a service, LogsView re-fetches through
    // the Connect client. Return only that service's entries so the
    // assertion on the outgoing request is meaningful.
    getLogsMock.mockImplementation((req: { serviceName: string }) => {
      if (req.serviceName === 'api') {
        return Promise.resolve({ entries: [mockLogs[0], mockLogs[1]] })
      }
      return Promise.resolve({ entries: mockLogs })
    })

    render(<LogsView />)

    const select = screen.getByRole('combobox')

    await waitFor(() => {
      expect(screen.getByRole('option', { name: 'api' })).toBeInTheDocument()
    })

    await user.selectOptions(select, 'api')

    await waitFor(() => {
      expect(getLogsMock).toHaveBeenCalledWith(
        expect.objectContaining({ serviceName: 'api' }),
      )
    })
  })

  it('should filter logs by search term', async () => {
    const user = userEvent.setup()
    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    const searchInput = screen.getByPlaceholderText('Search logs...')
    await user.type(searchInput, 'Flask')

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
      expect(screen.queryByText(/Express server/)).not.toBeInTheDocument()
    })
  })

  it('should display log count', async () => {
    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Showing \d+ of \d+ log entries/)).toBeInTheDocument()
    })
  })

  it('should toggle pause/resume', async () => {
    const user = userEvent.setup()
    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    const pauseButton = screen.getByTitle('Pause')
    await user.click(pauseButton)

    await waitFor(() => {
      expect(screen.getByText(/Paused - scroll stopped/)).toBeInTheDocument()
    })

    const resumeButton = screen.getByTitle('Resume')
    await user.click(resumeButton)

    await waitFor(() => {
      expect(screen.queryByText(/Paused - scroll stopped/)).not.toBeInTheDocument()
    })
  })

  it('should export logs', async () => {
    const user = userEvent.setup()
    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    const exportButton = screen.getByTitle('Export logs')
    await user.click(exportButton)

    // eslint-disable-next-line @typescript-eslint/unbound-method
    const mockFn = globalThis.URL.createObjectURL as ReturnType<typeof vi.fn>
    expect(mockFn.mock.calls).toHaveLength(1)
  })

  it('should clear logs with confirmation', async () => {
    const user = userEvent.setup()
    const confirmSpy = vi.spyOn(globalThis, 'confirm').mockReturnValue(true)

    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    const clearButton = screen.getByTitle('Clear logs')
    await user.click(clearButton)

    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalled()
      expect(screen.getByText('No logs to display')).toBeInTheDocument()
    })

    confirmSpy.mockRestore()
  })

  it('should not clear logs when confirmation is cancelled', async () => {
    const user = userEvent.setup()
    const confirmSpy = vi.spyOn(globalThis, 'confirm').mockReturnValue(false)

    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    const clearButton = screen.getByTitle('Clear logs')
    await user.click(clearButton)

    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalled()
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    confirmSpy.mockRestore()
  })

  it('should display empty state when no logs', async () => {
    getLogsMock.mockResolvedValue({ entries: [] })

    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText('No logs to display')).toBeInTheDocument()
    })
  })

  it('should show "no matching logs" when search returns empty', async () => {
    const user = userEvent.setup()
    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    const searchInput = screen.getByPlaceholderText('Search logs...')
    await user.type(searchInput, 'nonexistenttext12345')

    await waitFor(() => {
      expect(screen.getByText('No logs match your search')).toBeInTheDocument()
    })
  })

  it('should handle live log streaming', async () => {
    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    // Shared-stream hook should be mounted and enabled for local mode.
    await waitFor(() => {
      expect(latestSharedArgs).not.toBeNull()
      expect(latestSharedArgs?.enabled).toBe(true)
      expect(latestSharedArgs?.mode).toBe('local')
    })

    // The mode-change effect in LogsView stamps `lastClearTimeRef` on
    // mount, which gates realtime entries for 100ms to avoid races
    // with clearAllTrigger. Wait past that window before pushing.
    await new Promise((resolve) => setTimeout(resolve, 150))

    act(() => {
      pushLive({
        service: 'api',
        message: 'New log message from shared stream',
        level: 0,
        timestamp: new Date().toISOString(),
        isStderr: false,
      })
    })

    await waitFor(() => {
      expect(screen.getByText('New log message from shared stream')).toBeInTheDocument()
    })
  })

  it('should format timestamps correctly', async () => {
    render(<LogsView />)

    await waitFor(() => {
      const timestamps = screen.getAllByText(/\[\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}/)
      expect(timestamps.length).toBeGreaterThan(0)
    })
  })

  it('should color-code error messages in red', async () => {
    getLogsMock.mockResolvedValue({ entries: [mockLogs[4]] }) // Error log
    const { container } = render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Error: Connection timeout/)).toBeInTheDocument()
    })

    expect(container.querySelector('.text-red-400')).toBeInTheDocument()
  })

  it('should color-code warning messages in yellow', async () => {
    getLogsMock.mockResolvedValue({ entries: [mockLogs[3]] }) // Warning log
    const { container } = render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Warning/)).toBeInTheDocument()
    })

    expect(container.querySelector('.text-yellow-400')).toBeInTheDocument()
  })

  it('should assign consistent colors to different services', async () => {
    const { container } = render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    const serviceNames = container.querySelectorAll('[class*="text-"]')
    expect(serviceNames.length).toBeGreaterThan(0)
  })

  it('should convert ANSI codes to HTML', async () => {
    getLogsMock.mockResolvedValue({ entries: mockLogsWithAnsi })
    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })
  })

  it('should show jump to bottom button when paused', async () => {
    const user = userEvent.setup()
    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    const pauseButton = screen.getByTitle('Pause')
    await user.click(pauseButton)

    await waitFor(() => {
      expect(screen.getByText('Jump to Bottom')).toBeInTheDocument()
    })
  })

  it('should jump to bottom when button is clicked', async () => {
    const user = userEvent.setup()
    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    const pauseButton = screen.getByTitle('Pause')
    await user.click(pauseButton)

    await waitFor(() => {
      expect(screen.getByText('Jump to Bottom')).toBeInTheDocument()
    })

    const jumpButton = screen.getByText('Jump to Bottom')
    await user.click(jumpButton)

    await waitFor(() => {
      expect(screen.queryByText('Jump to Bottom')).not.toBeInTheDocument()
    })
  })

  it('should limit logs to 1000 entries', async () => {
    // Pre-populate with 1000 entries via the unary fetch, then push
    // one more through the live stream and assert the cap holds.
    const manyLogs = Array.from({ length: 1000 }, (_, i) => ({
      service: 'api',
      message: `Log entry ${i}`,
      level: 0,
      timestamp: new Date().toISOString(),
      isStderr: false,
    }))
    getLogsMock.mockResolvedValue({ entries: manyLogs })

    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Showing \d+ of \d+ log entries/)).toBeInTheDocument()
    })

    await waitFor(() => {
      expect(latestSharedArgs).not.toBeNull()
    })

    act(() => {
      pushLive({
        service: 'api',
        message: 'New entry',
        level: 0,
        timestamp: new Date().toISOString(),
        isStderr: false,
      })
    })

    await waitFor(() => {
      const countText = screen.getByText(/Showing (\d+) of (\d+) log entries/)
      expect(countText.textContent).toContain('1000')
    })
  })

  it('should not re-add logs from live stream after clearing', async () => {
    const user = userEvent.setup()
    const confirmSpy = vi.spyOn(globalThis, 'confirm').mockReturnValue(true)

    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    await waitFor(() => {
      expect(latestSharedArgs).not.toBeNull()
    })

    const clearButton = screen.getByTitle('Clear logs')

    // Push one live entry immediately before clicking clear.
    act(() => {
      pushLive({
        service: 'web',
        message: 'This should not appear after clear',
        level: 0,
        timestamp: new Date().toISOString(),
        isStderr: false,
      })
    })

    await user.click(clearButton)

    // Push another right after clear - should be dropped by the 100ms
    // clear-debounce window.
    act(() => {
      pushLive({
        service: 'web',
        message: 'This should also not appear',
        level: 0,
        timestamp: new Date().toISOString(),
        isStderr: false,
      })
    })

    await waitFor(() => {
      expect(screen.getByText('No logs to display')).toBeInTheDocument()
      expect(screen.queryByText('This should not appear after clear')).not.toBeInTheDocument()
      expect(screen.queryByText('This should also not appear')).not.toBeInTheDocument()
    })

    confirmSpy.mockRestore()
  })

  describe('clearAllTrigger prop (external clear control)', () => {
    it('should clear logs when clearAllTrigger is incremented', async () => {
      const { rerender } = render(<LogsView clearAllTrigger={0} />)

      await waitFor(() => {
        expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
      })

      rerender(<LogsView clearAllTrigger={1} />)

      await waitFor(() => {
        expect(screen.getByText('No logs to display')).toBeInTheDocument()
        expect(screen.queryByText(/Starting Flask application/)).not.toBeInTheDocument()
      })
    })

    it('should clear logs without confirmation when using clearAllTrigger', async () => {
      const confirmSpy = vi.spyOn(globalThis, 'confirm')

      const { rerender } = render(<LogsView clearAllTrigger={0} hideControls={true} />)

      await waitFor(() => {
        expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
      })

      rerender(<LogsView clearAllTrigger={1} hideControls={true} />)

      await waitFor(() => {
        expect(screen.getByText('No logs to display')).toBeInTheDocument()
      })

      expect(confirmSpy).not.toHaveBeenCalled()

      confirmSpy.mockRestore()
    })

    it('should handle multiple clearAllTrigger increments', async () => {
      const { rerender } = render(<LogsView clearAllTrigger={0} />)

      await waitFor(() => {
        expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
      })

      await waitFor(() => {
        expect(latestSharedArgs).not.toBeNull()
      })

      // First clear.
      rerender(<LogsView clearAllTrigger={1} />)
      await waitFor(() => {
        expect(screen.getByText('No logs to display')).toBeInTheDocument()
      })

      // Wait past the 100ms clear-debounce, then push a live entry.
      await new Promise((resolve) => setTimeout(resolve, 150))
      act(() => {
        pushLive({
          service: 'api',
          message: 'New log after clear',
          level: 0,
          timestamp: new Date().toISOString(),
          isStderr: false,
        })
      })

      await waitFor(() => {
        expect(screen.getByText('New log after clear')).toBeInTheDocument()
      })

      // Second clear.
      rerender(<LogsView clearAllTrigger={2} />)

      await waitFor(() => {
        expect(screen.getByText('No logs to display')).toBeInTheDocument()
        expect(screen.queryByText('New log after clear')).not.toBeInTheDocument()
      })
    })

    it('should prevent live-stream entries immediately after clearAllTrigger', async () => {
      const { rerender } = render(<LogsView clearAllTrigger={0} />)

      await waitFor(() => {
        expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
      })

      await waitFor(() => {
        expect(latestSharedArgs).not.toBeNull()
      })

      rerender(<LogsView clearAllTrigger={1} />)

      // Push a live entry inside the debounce window - should be dropped.
      act(() => {
        pushLive({
          service: 'web',
          message: 'Should not appear',
          level: 0,
          timestamp: new Date().toISOString(),
          isStderr: false,
        })
      })

      await waitFor(() => {
        expect(screen.getByText('No logs to display')).toBeInTheDocument()
        expect(screen.queryByText('Should not appear')).not.toBeInTheDocument()
      })
    })
  })

  describe('controlled vs uncontrolled mode', () => {
    it('should hide controls when hideControls is true', async () => {
      render(<LogsView hideControls={true} />)

      await waitFor(() => {
        expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
      })

      expect(screen.queryByPlaceholderText('Search logs...')).not.toBeInTheDocument()
      expect(screen.queryByTitle('Clear logs')).not.toBeInTheDocument()
      expect(screen.queryByTitle('Pause')).not.toBeInTheDocument()
      expect(screen.queryByText(/Showing \d+ of \d+ log entries/)).not.toBeInTheDocument()
    })

    it('should use external globalSearchTerm when provided', async () => {
      const { rerender } = render(<LogsView globalSearchTerm="" />)

      await waitFor(() => {
        expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
        expect(screen.getByText(/Express server/)).toBeInTheDocument()
      })

      rerender(<LogsView globalSearchTerm="Flask" />)

      await waitFor(() => {
        expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
        expect(screen.queryByText(/Express server/)).not.toBeInTheDocument()
      })
    })
  })
})
