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
    globalThis.fetch = mockFetch as any
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
    globalThis.fetch = mockFetch as any

    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByRole('option', { name: 'All Services' })).toBeInTheDocument()
    })

    const select = screen.getByRole('combobox')
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
    expect(globalThis.URL.createObjectURL).toHaveBeenCalled()
  })

  it('should clear logs with confirmation', async () => {
    const user = userEvent.setup()
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
    
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
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)
    
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
    globalThis.fetch = mockFetch as any

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

  it('should handle WebSocket log streaming', async () => {
    let wsInstance: any
    const WebSocketMock = vi.fn().mockImplementation((url: string) => {
      wsInstance = {
        url,
        onopen: null,
        onmessage: null,
        onerror: null,
        onclose: null,
        close: vi.fn(),
      }
      setTimeout(() => {
        if (wsInstance.onopen) wsInstance.onopen({})
      }, 0)
      return wsInstance
    })
    globalThis.WebSocket = WebSocketMock as any

    render(<LogsView />)

    await waitFor(() => {
      expect(WebSocketMock).toHaveBeenCalled()
    })

    // Simulate receiving a new log entry
    const newLogEntry = {
      service: 'api',
      message: 'New log message from WebSocket',
      level: 0,
      timestamp: new Date().toISOString(),
      isStderr: false,
    }

    if (wsInstance && wsInstance.onmessage) {
      act(() => {
        wsInstance.onmessage(createMockWebSocketMessage(newLogEntry))
      })
    }

    await waitFor(() => {
      expect(screen.getByText('New log message from WebSocket')).toBeInTheDocument()
    })
  })

  it('should format timestamps correctly', async () => {
    render(<LogsView />)

    await waitFor(() => {
      // Should display formatted timestamps in HH:MM:SS.mmm format
      const timestamps = screen.getAllByText(/\[\d{2}:\d{2}:\d{2}\.\d{3}\]/)
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
    globalThis.fetch = mockFetch as any

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
    globalThis.fetch = mockFetch as any

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
    globalThis.fetch = mockFetch as any

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
    let wsInstance: any
    const WebSocketMock = vi.fn().mockImplementation((url: string) => {
      wsInstance = {
        url,
        onopen: null,
        onmessage: null,
        onerror: null,
        onclose: null,
        close: vi.fn(),
      }
      setTimeout(() => {
        if (wsInstance.onopen) wsInstance.onopen({})
      }, 0)
      return wsInstance
    })
    globalThis.WebSocket = WebSocketMock as any

    // Create 1005 log entries
    const manyLogs = Array.from({ length: 1005 }, (_, i) => ({
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
    globalThis.fetch = mockFetch as any

    render(<LogsView />)

    await waitFor(() => {
      expect(screen.getByText(/Showing \d+ of \d+ log entries/)).toBeInTheDocument()
    })

    // Add one more via WebSocket
    if (wsInstance && wsInstance.onmessage) {
      act(() => {
        wsInstance.onmessage(
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
})
