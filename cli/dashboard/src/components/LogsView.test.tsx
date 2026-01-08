import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LogsView } from '@/components/LogsView'
import {
  mockLogs,
  mockLogsWithAnsi,
  mockServices,
  createMockFetchResponse,
  createMockWebSocketMessage,
} from '@/test/mocks'

interface MockWebSocket {
  url: string
  onopen: ((event: Event) => void) | null
  onmessage: ((event: MessageEvent) => void) | null
  onerror: ((event: Event) => void) | null
  onclose: ((event: CloseEvent) => void) | null
  close: ReturnType<typeof vi.fn>
}

describe('LogsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    // Mock fetch for services and logs
    const mockFetch = vi.fn((url: string) => {
      if (url === '/api/services') {
        return createMockFetchResponse(mockServices)
      }
      if (url.includes('/api/logs')) {
        return createMockFetchResponse(mockLogs)
      }
      return createMockFetchResponse([])
    })
    globalThis.fetch = mockFetch as unknown as typeof fetch
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
    
    const mockFetch = vi.fn((url: string) => {
      if (url === '/api/services') {
        return createMockFetchResponse(mockServices)
      }
      if (url.includes('service=api')) {
        return createMockFetchResponse([mockLogs[0], mockLogs[1]])
      }
      return createMockFetchResponse(mockLogs)
    })
    globalThis.fetch = mockFetch as unknown as typeof fetch

    render(<LogsView />)

    const select = screen.getByRole('combobox')
    
    await waitFor(() => {
      expect(screen.getByRole('option', { name: 'api' })).toBeInTheDocument()
    })

    await user.selectOptions(select, 'api')

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('service=api'))
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

    // Find pause button
    const pauseButton = screen.getByTitle('Pause')
    await user.click(pauseButton)

    await waitFor(() => {
      expect(screen.getByText(/Paused - scroll stopped/)).toBeInTheDocument()
    })

    // Find resume button
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

    // Check that URL.createObjectURL was called
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
      // Logs should still be visible
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    confirmSpy.mockRestore()
  })

  it('should display empty state when no logs', async () => {
    const mockFetch = vi.fn((url: string) => {
      if (url === '/api/services') {
        return createMockFetchResponse(mockServices)
      }
      return createMockFetchResponse([])
    })
    globalThis.fetch = mockFetch as unknown as typeof fetch

    render(<LogsView />)

    // First should show loading state
    await waitFor(() => {
      expect(screen.getByText(/Fetching local logs/)).toBeInTheDocument()
    })

    // Then should show empty state after fetch completes
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

  it('should handle WebSocket log streaming', async () => {
    const wsRef: { current: MockWebSocket | null } = { current: null }
    let mockConstructorCalled = false
    class WebSocketMock {
      url: string
      onopen: ((event: Event) => void) | null = null
      onmessage: ((event: MessageEvent) => void) | null = null
      onerror: ((event: Event) => void) | null = null
      onclose: ((event: CloseEvent) => void) | null = null
      close = vi.fn()
      addEventListener = vi.fn((event: string, handler: EventListener) => {
        if (event === 'open') this.onopen = handler as (event: Event) => void
        if (event === 'message') this.onmessage = handler as (event: MessageEvent) => void
        if (event === 'error') this.onerror = handler as (event: Event) => void
        if (event === 'close') this.onclose = handler as (event: CloseEvent) => void
      })
      removeEventListener = vi.fn()
      constructor(url: string) {
        this.url = url
        wsRef.current = this
        mockConstructorCalled = true
        setTimeout(() => {
          this.onopen?.(new Event('open'))
        }, 0)
      }
    }
    globalThis.WebSocket = WebSocketMock as unknown as typeof WebSocket

    render(<LogsView />)

    // Wait for initial logs to be fetched and rendered
    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    }, { timeout: 10000 })

    // Wait for WebSocket constructor to be called
    await waitFor(() => {
      expect(mockConstructorCalled).toBe(true)
    }, { timeout: 10000 })

    // Wait for WebSocket to be fully connected (onmessage handler set)
    await waitFor(() => {
      expect(wsRef.current?.onmessage).not.toBeNull()
    }, { timeout: 10000 })

    // Give a bit more time for component to stabilize
    await new Promise(resolve => setTimeout(resolve, 100))

    // Simulate receiving a new log entry
    const newLogEntry = {
      service: 'api',
      message: 'New log message from WebSocket',
      level: 0,
      timestamp: new Date().toISOString(),
      isStderr: false,
    }

    act(() => {
      if (wsRef.current?.onmessage) {
        wsRef.current.onmessage(createMockWebSocketMessage(newLogEntry))
      }
    })

    // Wait a bit for the message to be processed
    await new Promise(resolve => setTimeout(resolve, 100))

    await waitFor(() => {
      expect(screen.getByText('New log message from WebSocket')).toBeInTheDocument()
    }, { timeout: 10000 })
  }, 15000) // Increase overall test timeout to 15s

  it('should format timestamps correctly', async () => {
    render(<LogsView />)

    await waitFor(() => {
      // Should display formatted timestamps in MM-DD HH:MM:SS.mmm format
      const timestamps = screen.getAllByText(/\[\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}/)
      expect(timestamps.length).toBeGreaterThan(0)
    })
  })

  it('should color-code error messages in red', async () => {
    const mockFetch = vi.fn((url: string) => {
      if (url === '/api/services') {
        return createMockFetchResponse(mockServices)
      }
      return createMockFetchResponse([mockLogs[4]]) // Error log
    })
    globalThis.fetch = mockFetch as unknown as typeof fetch

    const { container } = render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Error: Connection timeout/)).toBeInTheDocument()
    })

    // Check for red color class
    expect(container.querySelector('.text-red-400')).toBeInTheDocument()
  })

  it('should color-code warning messages in yellow', async () => {
    const mockFetch = vi.fn((url: string) => {
      if (url === '/api/services') {
        return createMockFetchResponse(mockServices)
      }
      return createMockFetchResponse([mockLogs[3]]) // Warning log
    })
    globalThis.fetch = mockFetch as unknown as typeof fetch

    const { container } = render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Warning/)).toBeInTheDocument()
    })

    // Check for yellow color class
    expect(container.querySelector('.text-yellow-400')).toBeInTheDocument()
  })

  it('should assign consistent colors to different services', async () => {
    const { container } = render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    // Check for service name color classes
    const serviceNames = container.querySelectorAll('[class*="text-"]')
    expect(serviceNames.length).toBeGreaterThan(0)
  })

  it('should convert ANSI codes to HTML', async () => {
    const mockFetch = vi.fn((url: string) => {
      if (url === '/api/services') {
        return createMockFetchResponse(mockServices)
      }
      return createMockFetchResponse(mockLogsWithAnsi)
    })
    globalThis.fetch = mockFetch as unknown as typeof fetch

    render(<LogsView />)

    await waitFor(() => {
      // ANSI codes should be converted (the text should still be visible)
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })
  })

  it('should show jump to bottom button when paused', async () => {
    const user = userEvent.setup()
    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    // Pause
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

    // Pause
    const pauseButton = screen.getByTitle('Pause')
    await user.click(pauseButton)

    await waitFor(() => {
      expect(screen.getByText('Jump to Bottom')).toBeInTheDocument()
    })

    // Click jump to bottom
    const jumpButton = screen.getByText('Jump to Bottom')
    await user.click(jumpButton)

    await waitFor(() => {
      expect(screen.queryByText('Jump to Bottom')).not.toBeInTheDocument()
    })
  })

  it('should limit logs to 1000 entries', async () => {
    const wsRef: { current: MockWebSocket | null } = { current: null }
    class WebSocketMock {
      url: string
      onopen: ((event: Event) => void) | null = null
      onmessage: ((event: MessageEvent) => void) | null = null
      onerror: ((event: Event) => void) | null = null
      onclose: ((event: CloseEvent) => void) | null = null
      close = vi.fn()
      constructor(url: string) {
        this.url = url
        wsRef.current = this
        setTimeout(() => {
          this.onopen?.(new Event('open'))
        }, 0)
      }
    }
    globalThis.WebSocket = WebSocketMock as unknown as typeof WebSocket

    // Create 1000 log entries
    const manyLogs = Array.from({ length: 1000 }, (_, i) => ({
      service: 'api',
      message: `Log entry ${i}`,
      level: 0,
      timestamp: new Date().toISOString(),
      isStderr: false,
    }))

    const mockFetch = vi.fn((url: string) => {
      if (url === '/api/services') {
        return createMockFetchResponse(mockServices)
      }
      return createMockFetchResponse(manyLogs)
    })
    globalThis.fetch = mockFetch as unknown as typeof fetch

    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Showing \d+ of \d+ log entries/)).toBeInTheDocument()
    })

    // Add one more via WebSocket
    if (wsRef.current?.onmessage) {
      const handler = wsRef.current.onmessage
      act(() => {
        handler(
          createMockWebSocketMessage({
            service: 'api',
            message: 'New entry',
            level: 0,
            timestamp: new Date().toISOString(),
            isStderr: false,
          })
        )
      })
    }

    // Should be limited to 1000
    await waitFor(() => {
      const countText = screen.getByText(/Showing (\d+) of (\d+) log entries/)
      expect(countText.textContent).toContain('1000')
    })
  })

  it('should not re-add logs from WebSocket after clearing', async () => {
    const user = userEvent.setup()
    const confirmSpy = vi.spyOn(globalThis, 'confirm').mockReturnValue(true)
    
    // Mock WebSocket
    const wsRef = { current: null as MockWebSocket | null }
    class WebSocketMock {
      url: string
      onopen: ((event: Event) => void) | null = null
      onmessage: ((event: MessageEvent) => void) | null = null
      onerror: ((event: Event) => void) | null = null
      onclose: ((event: CloseEvent) => void) | null = null
      close = vi.fn()
      constructor(url: string) {
        this.url = url
        wsRef.current = this
        setTimeout(() => {
          this.onopen?.(new Event('open'))
        }, 0)
      }
    }
    globalThis.WebSocket = WebSocketMock as unknown as typeof WebSocket

    render(<LogsView />)

    // Wait for initial logs to load
    await waitFor(() => {
      expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
    })

    // Simulate WebSocket message arriving just before clear
    const pendingEntry = {
      service: 'web',
      message: 'This should not appear after clear',
      level: 0,
      timestamp: new Date().toISOString(),
      isStderr: false,
    }

    // Clear logs
    const clearButton = screen.getByTitle('Clear logs')
    
    // Send WebSocket message right before clear (simulating race condition)
    if (wsRef.current?.onmessage) {
      const handler = wsRef.current.onmessage
      act(() => {
        handler(createMockWebSocketMessage(pendingEntry))
      })
    }

    // Then clear
    await user.click(clearButton)

    // Send another WebSocket message right after clear
    if (wsRef.current?.onmessage) {
      const handler = wsRef.current.onmessage
      act(() => {
        handler(
          createMockWebSocketMessage({
            service: 'web',
            message: 'This should also not appear',
            level: 0,
            timestamp: new Date().toISOString(),
            isStderr: false,
          })
        )
      })
    }

    // Should show empty state and NOT contain the WebSocket messages
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

      // Wait for initial logs to load
      await waitFor(() => {
        expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
      })

      // Increment clearAllTrigger to trigger clear
      rerender(<LogsView clearAllTrigger={1} />)

      // Logs should be cleared
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

      // Increment clearAllTrigger
      rerender(<LogsView clearAllTrigger={1} hideControls={true} />)

      await waitFor(() => {
        expect(screen.getByText('No logs to display')).toBeInTheDocument()
      })

      // Should NOT have shown confirmation dialog
      expect(confirmSpy).not.toHaveBeenCalled()
      
      confirmSpy.mockRestore()
    })

    it('should handle multiple clearAllTrigger increments', async () => {
      const wsRef = { current: null as MockWebSocket | null }
      class WebSocketMock {
        url: string
        onopen: ((event: Event) => void) | null = null
        onmessage: ((event: MessageEvent) => void) | null = null
        onerror: ((event: Event) => void) | null = null
        onclose: ((event: CloseEvent) => void) | null = null
        close = vi.fn()
        constructor(url: string) {
          this.url = url
          wsRef.current = this
          setTimeout(() => {
            this.onopen?.(new Event('open'))
          }, 0)
        }
      }
      globalThis.WebSocket = WebSocketMock as unknown as typeof WebSocket

      const { rerender } = render(<LogsView clearAllTrigger={0} />)

      await waitFor(() => {
        expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
      })

      // First clear
      rerender(<LogsView clearAllTrigger={1} />)
      await waitFor(() => {
        expect(screen.getByText('No logs to display')).toBeInTheDocument()
      })

      // Add new logs via WebSocket
      if (wsRef.current?.onmessage) {
        const handler = wsRef.current.onmessage
        // Wait 150ms to be past the 100ms race condition window
        await new Promise(resolve => setTimeout(resolve, 150))
        act(() => {
          handler(createMockWebSocketMessage({
            service: 'api',
            message: 'New log after clear',
            level: 0,
            timestamp: new Date().toISOString(),
            isStderr: false,
          }))
        })
      }
      
      // Wait for logs to appear
      await waitFor(() => {
        expect(screen.getByText('New log after clear')).toBeInTheDocument()
      })

      // Second clear
      rerender(<LogsView clearAllTrigger={2} />)
      
      await waitFor(() => {
        expect(screen.getByText('No logs to display')).toBeInTheDocument()
        expect(screen.queryByText('New log after clear')).not.toBeInTheDocument()
      })
    })

    it('should prevent WebSocket messages immediately after clearAllTrigger', async () => {
      const wsRef = { current: null as MockWebSocket | null }
      class WebSocketMock {
        url: string
        onopen: ((event: Event) => void) | null = null
        onmessage: ((event: MessageEvent) => void) | null = null
        onerror: ((event: Event) => void) | null = null
        onclose: ((event: CloseEvent) => void) | null = null
        close = vi.fn()
        constructor(url: string) {
          this.url = url
          wsRef.current = this
          setTimeout(() => {
            this.onopen?.(new Event('open'))
          }, 0)
        }
      }
      globalThis.WebSocket = WebSocketMock as unknown as typeof WebSocket

      const { rerender } = render(<LogsView clearAllTrigger={0} />)

      await waitFor(() => {
        expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
      })

      // Trigger clear
      rerender(<LogsView clearAllTrigger={1} />)

      // Try to send WebSocket message right after clear
      if (wsRef.current?.onmessage) {
        const handler = wsRef.current.onmessage
        act(() => {
          handler(createMockWebSocketMessage({
            service: 'web',
            message: 'Should not appear',
            level: 0,
            timestamp: new Date().toISOString(),
            isStderr: false,
          }))
        })
      }

      // Should show empty state
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

      // Controls should be hidden
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

      // Change search term externally
      rerender(<LogsView globalSearchTerm="Flask" />)

      // Should filter logs
      await waitFor(() => {
        expect(screen.getByText(/Starting Flask application/)).toBeInTheDocument()
        expect(screen.queryByText(/Express server/)).not.toBeInTheDocument()
      })
    })
  })
})
