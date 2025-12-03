import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@/test/test-utils'
import userEvent from '@testing-library/user-event'
import { LogsMultiPaneView } from '@/components/LogsMultiPaneView'
import {
  mockServices,
  createMockFetchResponse,
} from '@/test/mocks'
import type { HealthReportEvent } from '@/types'

describe('LogsMultiPaneView - Fullscreen', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    // Mock fetch for services and logs
    const mockFetch = vi.fn((url: string) => {
      if (url === '/api/services') {
        return createMockFetchResponse(mockServices)
      }
      if (url.includes('/api/logs')) {
        return createMockFetchResponse([])
      }
      if (url === '/api/preferences') {
        return createMockFetchResponse({
          ui: {
            viewMode: 'grid',
            gridColumns: 2,
            theme: 'dark',
          },
          copy: {
            defaultFormat: 'plaintext',
            includeTimestamps: true,
          },
        })
      }
      if (url === '/api/patterns') {
        return createMockFetchResponse([])
      }
      return createMockFetchResponse([])
    })
    globalThis.fetch = mockFetch as unknown as typeof fetch

    // Mock WebSocket as a class for vitest 4.x compatibility
    class WebSocketMock {
      onopen: ((this: WebSocket, ev: Event) => unknown) | null = null
      onmessage: ((this: WebSocket, ev: MessageEvent) => unknown) | null = null
      onerror: ((this: WebSocket, ev: Event) => unknown) | null = null
      onclose: ((this: WebSocket, ev: CloseEvent) => unknown) | null = null
      close = vi.fn()
      send = vi.fn()
      constructor(_url: string) {
        // no-op
      }
    }
    globalThis.WebSocket = WebSocketMock as unknown as typeof WebSocket
  })

  it('should render fullscreen toggle button in toolbar', async () => {
    render(<LogsMultiPaneView />)

    await waitFor(() => {
      expect(screen.getByTitle('Enter Fullscreen (F11)')).toBeInTheDocument()
    })
  })

  it('should toggle fullscreen on button click', async () => {
    const user = userEvent.setup()
    const onFullscreenChange = vi.fn()
    
    render(<LogsMultiPaneView onFullscreenChange={onFullscreenChange} />)

    await waitFor(() => {
      expect(screen.getByTitle('Enter Fullscreen (F11)')).toBeInTheDocument()
    })

    const fullscreenButton = screen.getByTitle('Enter Fullscreen (F11)')
    await user.click(fullscreenButton)

    await waitFor(() => {
      expect(onFullscreenChange).toHaveBeenCalledWith(true)
    })
  })

  it('should exit fullscreen on button click', async () => {
    const user = userEvent.setup()
    const onFullscreenChange = vi.fn()
    
    render(<LogsMultiPaneView onFullscreenChange={onFullscreenChange} />)

    await waitFor(() => {
      expect(screen.getByTitle('Enter Fullscreen (F11)')).toBeInTheDocument()
    })

    // Enter fullscreen
    await user.click(screen.getByTitle('Enter Fullscreen (F11)'))

    await waitFor(() => {
      expect(onFullscreenChange).toHaveBeenCalledWith(true)
    })

    // Exit fullscreen
    await waitFor(() => {
      expect(screen.getByTitle('Exit Fullscreen (F11)')).toBeInTheDocument()
    })
    await user.click(screen.getByTitle('Exit Fullscreen (F11)'))

    await waitFor(() => {
      expect(onFullscreenChange).toHaveBeenCalledWith(false)
    })
  })

  it('should show view mode toggle in fullscreen', async () => {
    const user = userEvent.setup()
    
    render(<LogsMultiPaneView />)

    await waitFor(() => {
      expect(screen.getByText('Grid')).toBeInTheDocument()
      expect(screen.getByText('Unified')).toBeInTheDocument()
    })

    // Enter fullscreen via keyboard shortcut
    await user.keyboard('{F11}')

    // View mode toggle should still be visible in fullscreen
    await waitFor(() => {
      expect(screen.getByText('Grid')).toBeInTheDocument()
      expect(screen.getByText('Unified')).toBeInTheDocument()
    })
  })

  it('should keep settings accessible in fullscreen', async () => {
    const user = userEvent.setup()
    
    render(<LogsMultiPaneView />)

    await waitFor(() => {
      expect(screen.getByTitle('Settings (Ctrl+,)')).toBeInTheDocument()
    })

    // Enter fullscreen via keyboard
    await user.keyboard('{F11}')

    // Settings should still be accessible as direct button
    await waitFor(() => {
      expect(screen.getByTitle('Settings (Ctrl+,)')).toBeInTheDocument()
    })
  })

  it('should hide export button in fullscreen', async () => {
    const user = userEvent.setup()
    
    render(<LogsMultiPaneView />)

    // Export button should be present
    await waitFor(() => {
      expect(screen.getByTitle('Export All Logs')).toBeInTheDocument()
    })
    
    // Enter fullscreen
    await user.keyboard('{F11}')

    // Export should not be visible in fullscreen
    await waitFor(() => {
      expect(screen.queryByTitle('Export All Logs')).not.toBeInTheDocument()
    })
  })

  it('should keep service buttons visible in fullscreen', async () => {
    const user = userEvent.setup()
    
    render(<LogsMultiPaneView />)

    // Check service buttons are present
    await waitFor(() => {
      expect(screen.getByTitle('Start All (Ctrl+Shift+S)')).toBeInTheDocument()
    })

    // Enter fullscreen via keyboard
    await user.keyboard('{F11}')

    // Service buttons should still be visible in fullscreen
    await waitFor(() => {
      expect(screen.getByTitle('Start All (Ctrl+Shift+S)')).toBeInTheDocument()
    })
  })

  it('should keep pause button visible in fullscreen', async () => {
    const user = userEvent.setup()
    
    render(<LogsMultiPaneView />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /pause log stream/i })).toBeInTheDocument()
    })

    // Enter fullscreen via keyboard
    await user.keyboard('{F11}')

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /pause log stream/i })).toBeInTheDocument()
    })
  })

  it('should toggle fullscreen with F11 key', async () => {
    const user = userEvent.setup()
    const onFullscreenChange = vi.fn()
    
    render(<LogsMultiPaneView onFullscreenChange={onFullscreenChange} />)

    await waitFor(() => {
      expect(screen.getByRole('toolbar')).toBeInTheDocument()
    })

    // Press F11
    await user.keyboard('{F11}')

    await waitFor(() => {
      expect(onFullscreenChange).toHaveBeenCalledWith(true)
    })
  })

  it('should toggle fullscreen with Ctrl+Shift+F', async () => {
    const user = userEvent.setup()
    const onFullscreenChange = vi.fn()
    
    render(<LogsMultiPaneView onFullscreenChange={onFullscreenChange} />)

    await waitFor(() => {
      expect(screen.getByRole('toolbar')).toBeInTheDocument()
    })

    // Press Ctrl+Shift+F
    await user.keyboard('{Control>}{Shift>}f{/Shift}{/Control}')

    await waitFor(() => {
      expect(onFullscreenChange).toHaveBeenCalledWith(true)
    })
  })

  it('should exit fullscreen with Escape key', async () => {
    const user = userEvent.setup()
    const onFullscreenChange = vi.fn()
    
    render(<LogsMultiPaneView onFullscreenChange={onFullscreenChange} />)

    // Enter fullscreen via keyboard
    await user.keyboard('{F11}')

    await waitFor(() => {
      expect(onFullscreenChange).toHaveBeenCalledWith(true)
    })

    // Press Escape
    await user.keyboard('{Escape}')

    await waitFor(() => {
      expect(onFullscreenChange).toHaveBeenCalledWith(false)
    })
  })

  it('should apply fullscreen CSS classes', async () => {
    const user = userEvent.setup()
    
    const { container } = render(<LogsMultiPaneView />)

    await waitFor(() => {
      expect(screen.getByRole('toolbar')).toBeInTheDocument()
    })

    // Normal mode - should not have fixed positioning
    expect(container.querySelector('.fixed.inset-0.z-50')).not.toBeInTheDocument()

    // Enter fullscreen via keyboard
    await user.keyboard('{F11}')

    await waitFor(() => {
      // Fullscreen mode - should have fixed positioning
      expect(container.querySelector('.fixed.inset-0.z-50')).toBeInTheDocument()
    })
  })

  it('should show fullscreen button in toolbar', async () => {
    render(<LogsMultiPaneView />)

    await waitFor(() => {
      expect(screen.getByTitle('Enter Fullscreen (F11)')).toBeInTheDocument()
    })
  })

  it('should show exit fullscreen option when in fullscreen', async () => {
    const user = userEvent.setup()
    
    render(<LogsMultiPaneView />)

    // Enter fullscreen via keyboard
    await user.keyboard('{F11}')

    await waitFor(() => {
      expect(screen.getByTitle('Exit Fullscreen (F11)')).toBeInTheDocument()
    })
  })
})

describe('LogsMultiPaneView - Health Status Filter', () => {
  const mockHealthReport: HealthReportEvent = {
    type: 'health',
    timestamp: new Date().toISOString(),
    services: [
      { serviceName: 'api', status: 'healthy', checkType: 'http', responseTime: 100, timestamp: new Date().toISOString() },
      { serviceName: 'web', status: 'degraded', checkType: 'http', responseTime: 200, timestamp: new Date().toISOString() },
      { serviceName: 'worker', status: 'unhealthy', checkType: 'process', responseTime: 0, timestamp: new Date().toISOString() },
    ],
    summary: { total: 3, healthy: 1, degraded: 1, unhealthy: 1, starting: 0, stopped: 0, unknown: 0, overall: 'degraded' },
  }

  const mockServicesWithHealth = [
    { name: 'api', local: { status: 'running' as const, health: 'healthy' as const } },
    { name: 'web', local: { status: 'running' as const, health: 'degraded' as const } },
    { name: 'worker', local: { status: 'running' as const, health: 'unhealthy' as const } },
  ]

  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()

    const mockFetch = vi.fn((url: string) => {
      if (url === '/api/services') {
        return createMockFetchResponse(mockServicesWithHealth)
      }
      if (url.includes('/api/logs')) {
        return createMockFetchResponse([])
      }
      if (url === '/api/preferences') {
        return createMockFetchResponse({
          ui: { viewMode: 'grid', gridColumns: 2, theme: 'dark' },
          copy: { defaultFormat: 'plaintext', includeTimestamps: true },
        })
      }
      if (url === '/api/patterns') {
        return createMockFetchResponse([])
      }
      return createMockFetchResponse([])
    })
    globalThis.fetch = mockFetch as unknown as typeof fetch

    // Mock WebSocket as a class for vitest 4.x compatibility
    class WebSocketMock {
      onopen: ((this: WebSocket, ev: Event) => unknown) | null = null
      onmessage: ((this: WebSocket, ev: MessageEvent) => unknown) | null = null
      onerror: ((this: WebSocket, ev: Event) => unknown) | null = null
      onclose: ((this: WebSocket, ev: CloseEvent) => unknown) | null = null
      close = vi.fn()
      send = vi.fn()
      constructor(_url: string) {
        // no-op
      }
    }
    globalThis.WebSocket = WebSocketMock as unknown as typeof WebSocket
  })

  afterEach(() => {
    localStorage.clear()
  })

  it('should render Health Status filter section with all 5 checkboxes', async () => {
    render(<LogsMultiPaneView healthReport={mockHealthReport} />)

    await waitFor(() => {
      expect(screen.getByText('Health Status')).toBeInTheDocument()
    })

    expect(screen.getByText('Healthy')).toBeInTheDocument()
    expect(screen.getByText('Degraded')).toBeInTheDocument()
    expect(screen.getByText('Unhealthy')).toBeInTheDocument()
    expect(screen.getByText('Starting')).toBeInTheDocument()
    expect(screen.getByText('Unknown')).toBeInTheDocument()
  })

  it('should have all health status checkboxes checked by default', async () => {
    render(<LogsMultiPaneView healthReport={mockHealthReport} />)

    await waitFor(() => {
      expect(screen.getByText('Health Status')).toBeInTheDocument()
    })

    const healthyCheckbox = screen.getByLabelText<HTMLInputElement>('Show healthy services')
    const degradedCheckbox = screen.getByLabelText<HTMLInputElement>('Show degraded services')
    const unhealthyCheckbox = screen.getByLabelText<HTMLInputElement>('Show unhealthy services')
    const startingCheckbox = screen.getByLabelText<HTMLInputElement>('Show starting services')
    const unknownCheckbox = screen.getByLabelText<HTMLInputElement>('Show unknown services')

    expect(healthyCheckbox.checked).toBe(true)
    expect(degradedCheckbox.checked).toBe(true)
    expect(unhealthyCheckbox.checked).toBe(true)
    expect(startingCheckbox.checked).toBe(true)
    expect(unknownCheckbox.checked).toBe(true)
  })

  it('should toggle health status filter when clicking checkbox', async () => {
    const user = userEvent.setup()
    render(<LogsMultiPaneView healthReport={mockHealthReport} />)

    await waitFor(() => {
      expect(screen.getByText('Health Status')).toBeInTheDocument()
    })

    const healthyCheckbox = screen.getByLabelText<HTMLInputElement>('Show healthy services')
    expect(healthyCheckbox.checked).toBe(true)

    await user.click(healthyCheckbox)
    expect(healthyCheckbox.checked).toBe(false)

    await user.click(healthyCheckbox)
    expect(healthyCheckbox.checked).toBe(true)
  })

  it('should persist health filter state to localStorage', async () => {
    const user = userEvent.setup()
    render(<LogsMultiPaneView healthReport={mockHealthReport} />)

    await waitFor(() => {
      expect(screen.getByText('Health Status')).toBeInTheDocument()
    })

    const healthyCheckbox = screen.getByLabelText('Show healthy services')
    await user.click(healthyCheckbox)

    await waitFor(() => {
      const saved = localStorage.getItem('logs-health-status-filter')
      expect(saved).toBeDefined()
      const parsed = JSON.parse(saved!) as string[]
      expect(parsed).not.toContain('healthy')
      expect(parsed).toContain('degraded')
      expect(parsed).toContain('unhealthy')
      expect(parsed).toContain('starting')
      expect(parsed).toContain('unknown')
    })
  })

  it('should restore health filter state from localStorage', async () => {
    localStorage.setItem('logs-health-status-filter', JSON.stringify(['healthy', 'unhealthy']))

    render(<LogsMultiPaneView healthReport={mockHealthReport} />)

    await waitFor(() => {
      expect(screen.getByText('Health Status')).toBeInTheDocument()
    })

    const healthyCheckbox = screen.getByLabelText<HTMLInputElement>('Show healthy services')
    const degradedCheckbox = screen.getByLabelText<HTMLInputElement>('Show degraded services')
    const unhealthyCheckbox = screen.getByLabelText<HTMLInputElement>('Show unhealthy services')
    const startingCheckbox = screen.getByLabelText<HTMLInputElement>('Show starting services')
    const unknownCheckbox = screen.getByLabelText<HTMLInputElement>('Show unknown services')

    expect(healthyCheckbox.checked).toBe(true)
    expect(degradedCheckbox.checked).toBe(false)
    expect(unhealthyCheckbox.checked).toBe(true)
    expect(startingCheckbox.checked).toBe(false)
    expect(unknownCheckbox.checked).toBe(false)
  })

  it('should show empty state when no services match health filter', async () => {
    // Set localStorage to only show 'starting' status (none of our services have this)
    localStorage.setItem('logs-health-status-filter', JSON.stringify(['starting']))

    render(<LogsMultiPaneView healthReport={mockHealthReport} />)

    await waitFor(() => {
      expect(screen.getByText('Health Status')).toBeInTheDocument()
    })

    await waitFor(() => {
      expect(screen.getByText('No services match the current filters')).toBeInTheDocument()
    })
  })

  it('should filter panes based on health status', async () => {
    const user = userEvent.setup()
    render(<LogsMultiPaneView healthReport={mockHealthReport} />)

    await waitFor(() => {
      expect(screen.getByText('Health Status')).toBeInTheDocument()
    })

    // Initially all services should be visible (each service appears twice: in filter and pane header)
    await waitFor(() => {
      expect(screen.getAllByText('api')).toHaveLength(2)
      expect(screen.getAllByText('web')).toHaveLength(2)
      expect(screen.getAllByText('worker')).toHaveLength(2)
    })

    // Uncheck 'healthy' - api pane should be hidden (only 1 occurrence left in filter)
    const healthyCheckbox = screen.getByLabelText('Show healthy services')
    await user.click(healthyCheckbox)

    await waitFor(() => {
      expect(screen.getAllByText('api')).toHaveLength(1) // Only in filter, not in pane
      expect(screen.getAllByText('web')).toHaveLength(2) // Still in filter and pane
      expect(screen.getAllByText('worker')).toHaveLength(2) // Still in filter and pane
    })
  })

  it('should treat services without health data as unknown', async () => {
    const user = userEvent.setup()
    // Health report with only some services
    const partialHealthReport: HealthReportEvent = {
      type: 'health',
      timestamp: new Date().toISOString(),
      services: [
        { serviceName: 'api', status: 'healthy', checkType: 'http', responseTime: 100, timestamp: new Date().toISOString() },
        // web and worker are missing - should be treated as 'unknown'
      ],
      summary: { total: 1, healthy: 1, degraded: 0, unhealthy: 0, starting: 0, stopped: 0, unknown: 2, overall: 'healthy' },
    }

    render(<LogsMultiPaneView healthReport={partialHealthReport} />)

    await waitFor(() => {
      expect(screen.getByText('Health Status')).toBeInTheDocument()
    })

    // Initially all services should be visible (each service appears twice: in filter and pane header)
    await waitFor(() => {
      expect(screen.getAllByText('api')).toHaveLength(2)
      expect(screen.getAllByText('web')).toHaveLength(2)
      expect(screen.getAllByText('worker')).toHaveLength(2)
    })

    // Uncheck 'unknown' - web and worker panes should be hidden (only 1 occurrence left in filter)
    const unknownCheckbox = screen.getByLabelText('Show unknown services')
    await user.click(unknownCheckbox)

    await waitFor(() => {
      expect(screen.getAllByText('api')).toHaveLength(2) // Still in filter and pane (healthy)
      expect(screen.getAllByText('web')).toHaveLength(1) // Only in filter, not in pane
      expect(screen.getAllByText('worker')).toHaveLength(1) // Only in filter, not in pane
    })
  })

  it('should have proper ARIA attributes for accessibility', async () => {
    render(<LogsMultiPaneView healthReport={mockHealthReport} />)

    await waitFor(() => {
      expect(screen.getByText('Health Status')).toBeInTheDocument()
    })

    const filterGroup = screen.getByRole('group', { name: /health status/i })
    expect(filterGroup).toBeInTheDocument()

    const checkboxes = screen.getAllByRole('checkbox', { name: /show .* services/i })
    expect(checkboxes).toHaveLength(5)
  })

  it('should handle invalid localStorage data gracefully', async () => {
    localStorage.setItem('logs-health-status-filter', 'invalid json{')

    render(<LogsMultiPaneView healthReport={mockHealthReport} />)

    await waitFor(() => {
      expect(screen.getByText('Health Status')).toBeInTheDocument()
    })

    // Should fall back to all checked
    const healthyCheckbox = screen.getByLabelText<HTMLInputElement>('Show healthy services')
    expect(healthyCheckbox.checked).toBe(true)
  })

  it('should handle non-array localStorage data gracefully', async () => {
    localStorage.setItem('logs-health-status-filter', JSON.stringify({ invalid: 'object' }))

    render(<LogsMultiPaneView healthReport={mockHealthReport} />)

    await waitFor(() => {
      expect(screen.getByText('Health Status')).toBeInTheDocument()
    })

    // Should fall back to all checked
    const healthyCheckbox = screen.getByLabelText<HTMLInputElement>('Show healthy services')
    expect(healthyCheckbox.checked).toBe(true)
  })
})
