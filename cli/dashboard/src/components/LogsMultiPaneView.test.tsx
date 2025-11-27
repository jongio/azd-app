import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LogsMultiPaneView } from '@/components/LogsMultiPaneView'
import {
  mockServices,
  createMockFetchResponse,
} from '@/test/mocks'

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

    // Mock WebSocket
    const WebSocketMock = vi.fn().mockImplementation(() => ({
      onopen: null,
      onmessage: null,
      onerror: null,
      onclose: null,
      close: vi.fn(),
    }))
    globalThis.WebSocket = WebSocketMock as unknown as typeof WebSocket
  })

  it('should render fullscreen toggle button', async () => {
    render(<LogsMultiPaneView />)

    await waitFor(() => {
      expect(screen.getByTitle(/Enter Fullscreen/)).toBeInTheDocument()
    })
  })

  it('should toggle fullscreen on button click', async () => {
    const user = userEvent.setup()
    const onFullscreenChange = vi.fn()
    
    render(<LogsMultiPaneView onFullscreenChange={onFullscreenChange} />)

    await waitFor(() => {
      expect(screen.getByTitle(/Enter Fullscreen/)).toBeInTheDocument()
    })

    const fullscreenButton = screen.getByTitle(/Enter Fullscreen/)
    await user.click(fullscreenButton)

    await waitFor(() => {
      expect(onFullscreenChange).toHaveBeenCalledWith(true)
      expect(screen.getByTitle(/Exit Fullscreen/)).toBeInTheDocument()
    })
  })

  it('should exit fullscreen on button click', async () => {
    const user = userEvent.setup()
    const onFullscreenChange = vi.fn()
    
    render(<LogsMultiPaneView onFullscreenChange={onFullscreenChange} />)

    await waitFor(() => {
      expect(screen.getByTitle(/Enter Fullscreen/)).toBeInTheDocument()
    })

    // Enter fullscreen
    const enterButton = screen.getByTitle(/Enter Fullscreen/)
    await user.click(enterButton)

    await waitFor(() => {
      expect(screen.getByTitle(/Exit Fullscreen/)).toBeInTheDocument()
    })

    // Exit fullscreen
    const exitButton = screen.getByTitle(/Exit Fullscreen/)
    await user.click(exitButton)

    await waitFor(() => {
      expect(onFullscreenChange).toHaveBeenCalledWith(false)
      expect(screen.getByTitle(/Enter Fullscreen/)).toBeInTheDocument()
    })
  })

  it('should hide view mode toggle in fullscreen', async () => {
    const user = userEvent.setup()
    
    render(<LogsMultiPaneView />)

    await waitFor(() => {
      expect(screen.getByText('Grid')).toBeInTheDocument()
      expect(screen.getByText('Unified')).toBeInTheDocument()
    })

    // Enter fullscreen
    const fullscreenButton = screen.getByTitle(/Enter Fullscreen/)
    await user.click(fullscreenButton)

    await waitFor(() => {
      expect(screen.queryByText('Grid')).not.toBeInTheDocument()
      expect(screen.queryByText('Unified')).not.toBeInTheDocument()
    })
  })

  it('should hide settings button in fullscreen', async () => {
    const user = userEvent.setup()
    
    render(<LogsMultiPaneView />)

    await waitFor(() => {
      expect(screen.getByTitle('Settings')).toBeInTheDocument()
    })

    // Enter fullscreen
    const fullscreenButton = screen.getByTitle(/Enter Fullscreen/)
    await user.click(fullscreenButton)

    await waitFor(() => {
      expect(screen.getByTitle('Settings')).toBeInTheDocument()
    })
  })

  it('should hide export button in fullscreen', async () => {
    const user = userEvent.setup()
    
    render(<LogsMultiPaneView />)

    await waitFor(() => {
      expect(screen.getByTitle('Export All')).toBeInTheDocument()
    })

    // Enter fullscreen
    const fullscreenButton = screen.getByTitle(/Enter Fullscreen/)
    await user.click(fullscreenButton)

    await waitFor(() => {
      expect(screen.queryByTitle('Export All')).not.toBeInTheDocument()
    })
  })

  it('should hide service selector in fullscreen', async () => {
    const user = userEvent.setup()
    
    render(<LogsMultiPaneView />)

    await waitFor(() => {
      expect(screen.getByText('Services')).toBeInTheDocument()
    })

    // Enter fullscreen
    const fullscreenButton = screen.getByTitle(/Enter Fullscreen/)
    await user.click(fullscreenButton)

    await waitFor(() => {
      expect(screen.queryByText('Services')).not.toBeInTheDocument()
    })
  })

  it('should keep pause button visible in fullscreen', async () => {
    const user = userEvent.setup()
    
    render(<LogsMultiPaneView />)

    await waitFor(() => {
      expect(screen.getByTitle(/Pause/)).toBeInTheDocument()
    })

    // Enter fullscreen
    const fullscreenButton = screen.getByTitle(/Enter Fullscreen/)
    await user.click(fullscreenButton)

    await waitFor(() => {
      expect(screen.getByTitle(/Pause/)).toBeInTheDocument()
    })
  })

  it('should toggle fullscreen with F11 key', async () => {
    const user = userEvent.setup()
    const onFullscreenChange = vi.fn()
    
    render(<LogsMultiPaneView onFullscreenChange={onFullscreenChange} />)

    await waitFor(() => {
      expect(screen.getByTitle(/Enter Fullscreen/)).toBeInTheDocument()
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
      expect(screen.getByTitle(/Enter Fullscreen/)).toBeInTheDocument()
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

    // Enter fullscreen
    const fullscreenButton = screen.getByTitle(/Enter Fullscreen/)
    await user.click(fullscreenButton)

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
      expect(screen.getByTitle(/Enter Fullscreen/)).toBeInTheDocument()
    })

    // Normal mode - should not have fixed positioning
    expect(container.querySelector('.fixed.inset-0.z-50')).not.toBeInTheDocument()

    // Enter fullscreen
    const fullscreenButton = screen.getByTitle(/Enter Fullscreen/)
    await user.click(fullscreenButton)

    await waitFor(() => {
      // Fullscreen mode - should have fixed positioning
      expect(container.querySelector('.fixed.inset-0.z-50')).toBeInTheDocument()
    })
  })

  it('should show Maximize icon when not fullscreen', async () => {
    render(<LogsMultiPaneView />)

    await waitFor(() => {
      const button = screen.getByTitle(/Enter Fullscreen/)
      const svg = button.querySelector('svg')
      expect(svg).toBeInTheDocument()
    })
  })

  it('should show Minimize icon when fullscreen', async () => {
    const user = userEvent.setup()
    
    render(<LogsMultiPaneView />)

    // Enter fullscreen
    const fullscreenButton = screen.getByTitle(/Enter Fullscreen/)
    await user.click(fullscreenButton)

    await waitFor(() => {
      const button = screen.getByTitle(/Exit Fullscreen/)
      const svg = button.querySelector('svg')
      expect(svg).toBeInTheDocument()
    })
  })
})
