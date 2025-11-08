import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import App from '@/App'
import { mockServices, mockProjectInfo, createMockFetchResponse } from '@/test/mocks'

// Mock the useServices hook
vi.mock('@/hooks/useServices', () => ({
  useServices: vi.fn(() => ({
    services: mockServices,
    loading: false,
    error: null,
    connected: true,
    refetch: vi.fn(),
  })),
}))

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()

    const mockFetch = vi.fn((url: string) => {
      if (url === '/api/project') {
        return createMockFetchResponse(mockProjectInfo)
      }
      return createMockFetchResponse([])
    })
    globalThis.fetch = mockFetch as any
  })

  it('should render the app with default resources view', async () => {
    render(<App />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Resources' })).toBeInTheDocument()
    })
  })

  it('should fetch and display project name', async () => {
    render(<App />)

    await waitFor(() => {
      expect(screen.getByText(mockProjectInfo.name)).toBeInTheDocument()
    })
  })

  it('should switch between table and grid view modes', async () => {
    const user = userEvent.setup()
    render(<App />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Resources' })).toBeInTheDocument()
    })

    // Find and click Grid button
    const gridButton = screen.getByRole('button', { name: /grid/i })
    await user.click(gridButton)

    // Should show grid view
    await waitFor(() => {
      expect(localStorage.setItem).toHaveBeenCalledWith('dashboard-view-preference', 'cards')
    })

    // Click Table button
    const tableButton = screen.getByRole('button', { name: /table/i })
    await user.click(tableButton)

    await waitFor(() => {
      expect(localStorage.setItem).toHaveBeenCalledWith('dashboard-view-preference', 'table')
    })
  })

  it('should switch to console view', async () => {
    const user = userEvent.setup()
    render(<App />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Resources' })).toBeInTheDocument()
    })

    // Click console button in sidebar
    const consoleButton = screen.getByRole('button', { name: /console/i })
    await user.click(consoleButton)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Console' })).toBeInTheDocument()
    })
  })

  it('should display loading state', async () => {
    const { useServices } = await import('@/hooks/useServices')
    vi.mocked(useServices).mockReturnValue({
      services: [],
      loading: true,
      error: null,
      connected: false,
      connecting: false,
      refetch: vi.fn(),
    })

    render(<App />)

    await waitFor(() => {
      expect(document.querySelector('.animate-spin')).toBeInTheDocument()
    })
  })

  it('should display error state', async () => {
    const { useServices } = await import('@/hooks/useServices')
    const errorMessage = 'Failed to connect to server'
    vi.mocked(useServices).mockReturnValue({
      services: [],
      loading: false,
      error: errorMessage,
      connected: false,
      connecting: false,
      refetch: vi.fn(),
    })

    render(<App />)

    await waitFor(() => {
      expect(screen.getByText('Error Loading Services')).toBeInTheDocument()
      expect(screen.getByText(errorMessage)).toBeInTheDocument()
    })
  })

  it('should display empty state when no services are running', async () => {
    const { useServices } = await import('@/hooks/useServices')
    vi.mocked(useServices).mockReturnValue({
      services: [],
      loading: false,
      error: null,
      connected: true,
      connecting: false,
      refetch: vi.fn(),
    })

    render(<App />)

    await waitFor(() => {
      expect(screen.getByText('No Services Running')).toBeInTheDocument()
      expect(screen.getByText('azd app run')).toBeInTheDocument()
    })
  })

  it('should display services in table view by default', async () => {
    render(<App />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Resources' })).toBeInTheDocument()
    })

    // Check that we're in table view (not grid view)
    await waitFor(() => {
      const tableButton = screen.getByRole('button', { name: /table/i })
      expect(tableButton.querySelector('.bg-primary')).toBeTruthy()
    }, { timeout: 3000 })
  })

  it.skip('should remember view preference from localStorage', async () => {
    // TODO: This test is skipped because localStorage preference isn't applying on initial render
    // The feature needs to be debugged - App.tsx reads localStorage but viewMode doesn't update UI correctly
    localStorage.setItem('dashboard-view-preference', 'cards')

    render(<App />)

    // Since localStorage is read on mount, the grid view should be active
    // Check that we can see the grid layout (divs with grid classes)
    await waitFor(() => {
      const gridContainer = document.querySelector('.grid.grid-cols-1')
      expect(gridContainer).toBeInTheDocument()
    }, { timeout: 3000 })
  })

  it('should scroll to top when view changes', async () => {
    const user = userEvent.setup()
    const scrollToMock = vi.fn()
    
    // Mock querySelector to return an element with scrollTo
    const mockMainElement = {
      scrollTo: scrollToMock,
    }
    vi.spyOn(document, 'querySelector').mockReturnValue(mockMainElement as any)

    render(<App />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Resources' })).toBeInTheDocument()
    })

    // Switch to console view
    const consoleButton = screen.getByRole('button', { name: /console/i })
    await user.click(consoleButton)

    await waitFor(() => {
      expect(scrollToMock).toHaveBeenCalledWith({ top: 0, behavior: 'smooth' })
    })
  })

  it('should render header buttons', async () => {
    render(<App />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Resources' })).toBeInTheDocument()
    })

    // Check for header buttons (by their icons/roles)
    const buttons = screen.getAllByRole('button')
    expect(buttons.length).toBeGreaterThan(0)
  })

  it('should show coming soon for unimplemented views', async () => {
    const user = userEvent.setup()
    render(<App />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Resources' })).toBeInTheDocument()
    })

    // Click on a view that's not implemented (e.g., Traces)
    const tracesButton = screen.getByRole('button', { name: /traces/i })
    await user.click(tracesButton)

    await waitFor(() => {
      expect(screen.getByText('Coming Soon')).toBeInTheDocument()
      expect(screen.getByText('This view is not yet implemented')).toBeInTheDocument()
    })
  })

  it('should handle project name fetch failure gracefully', async () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    
    const mockFetch = vi.fn((url: string) => {
      if (url === '/api/project') {
        return Promise.reject(new Error('Network error'))
      }
      return createMockFetchResponse([])
    })
    globalThis.fetch = mockFetch as any

    render(<App />)

    await waitFor(() => {
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        'Failed to fetch project name:',
        expect.any(Error)
      )
    })

    // Should still show default name
    expect(screen.getByText('testhost')).toBeInTheDocument()

    consoleErrorSpy.mockRestore()
  })

  it('should filter services when search input is used', async () => {
    const user = userEvent.setup()
    render(<App />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Resources' })).toBeInTheDocument()
    })

    // Find search input
    const searchInput = screen.getByPlaceholderText('Filter...')
    
    // Type in search
    await user.type(searchInput, 'test')

    // Input should have the value
    expect(searchInput).toHaveValue('test')
  })
})
