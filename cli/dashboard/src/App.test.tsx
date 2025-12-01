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

// Mock the useServiceErrors hook
vi.mock('@/hooks/useServiceErrors', () => ({
  useServiceErrors: vi.fn(() => ({
    hasActiveErrors: false,
  })),
}))

// Mock the useHealthStream hook to prevent EventSource issues in jsdom
vi.mock('@/hooks/useHealthStream', () => ({
  useHealthStream: vi.fn(() => ({
    healthReport: null,
    connected: false,
    error: null,
    getServiceHealth: vi.fn(() => null),
  })),
}))

// Mock the useToast hook
vi.mock('@/components/ui/toast', () => ({
  useToast: vi.fn(() => ({
    showToast: vi.fn(),
    ToastContainer: () => null,
  })),
}))

describe('App', () => {
  beforeEach(async () => {
    vi.clearAllMocks()
    localStorage.clear()

    // Reset the useServices mock to default values
    const { useServices } = await import('@/hooks/useServices')
    vi.mocked(useServices).mockReturnValue({
      services: mockServices,
      loading: false,
      error: null,
      connected: true,
      refetch: vi.fn(),
    })

    // Reset the useServiceErrors mock to default values
    const { useServiceErrors } = await import('@/hooks/useServiceErrors')
    vi.mocked(useServiceErrors).mockReturnValue({
      hasActiveErrors: false,
    })

    const mockFetch = vi.fn((url: string) => {
      if (url === '/api/project') {
        return createMockFetchResponse(mockProjectInfo)
      }
      return createMockFetchResponse([])
    })
    globalThis.fetch = mockFetch as unknown as typeof fetch
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
      expect(localStorage.getItem('dashboard-view-preference')).toBe('cards')
    })

    // Click Table button
    const tableButton = screen.getByRole('button', { name: /table/i })
    await user.click(tableButton)

    await waitFor(() => {
      expect(localStorage.getItem('dashboard-view-preference')).toBe('table')
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

  it('should remember view preference from localStorage', async () => {
    // Set localStorage preference before rendering
    localStorage.setItem('dashboard-view-preference', 'cards')

    render(<App />)

    // Wait for the Resources view to render
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Resources' })).toBeInTheDocument()
    })

    // Since localStorage is read on mount, the grid view should be active
    // Check that the Grid button has the active indicator
    await waitFor(() => {
      const gridButton = screen.getByRole('button', { name: /grid/i })
      expect(gridButton.querySelector('.bg-primary')).toBeTruthy()
    })

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
    } as unknown as Element
    vi.spyOn(document, 'querySelector').mockReturnValue(mockMainElement)

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

  it('should switch between resources and console views', async () => {
    const user = userEvent.setup()
    render(<App />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Resources' })).toBeInTheDocument()
    })

    // Click on Console view
    const consoleButton = screen.getByRole('button', { name: /console/i })
    await user.click(consoleButton)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Console' })).toBeInTheDocument()
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
    globalThis.fetch = mockFetch as unknown as typeof fetch

    render(<App />)

    await waitFor(() => {
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        'Failed to fetch project name:',
        expect.any(Error) as Error
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

  it('should show error indicator on Console nav when errors are active', async () => {
    // To show the error indicator, we need either:
    // 1. Services with unhealthy status (from health data)
    // 2. OR empty services with hasActiveErrors=true (fallback)
    // Using the fallback approach here:
    const { useServices } = await import('@/hooks/useServices')
    vi.mocked(useServices).mockReturnValue({
      services: [], // Empty services so hasActiveErrors is used as fallback
      loading: false,
      error: null,
      connected: true,
      refetch: vi.fn(),
    })
    
    const { useServiceErrors } = await import('@/hooks/useServiceErrors')
    vi.mocked(useServiceErrors).mockReturnValue({ hasActiveErrors: true })
    
    render(<App />)

    // Wait for the component to fully render and settle
    await waitFor(() => {
      // Find Console nav button - it should have a red pulsing dot indicator
      const consoleNav = screen.getByRole('button', { name: /console/i })
      
      // Check for error indicator styles (red ring and flashing dot)
      expect(consoleNav).toBeInTheDocument()
      
      // The button should have the error indicator with status-flash animation
      const errorIndicator = consoleNav.querySelector('.animate-status-flash')
      expect(errorIndicator).toBeInTheDocument()
    })
  })

  it('should render ServiceStatusCard in the header', async () => {
    render(<App />)

    await waitFor(() => {
      // Should show the service status button
      const statusButton = screen.getByTitle('Click to view console logs')
      expect(statusButton).toBeInTheDocument()
    })
  })

  it('should navigate to console when ServiceStatusCard is clicked', async () => {
    const user = userEvent.setup()
    render(<App />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Resources' })).toBeInTheDocument()
    })

    // Find and click the service status card button
    const statusButton = screen.getByTitle('Click to view console logs')
    await user.click(statusButton)

    // Should switch to console view (the console nav should become active)
    await waitFor(() => {
      const consoleNav = screen.getByRole('button', { name: /console/i })
      expect(consoleNav).toHaveClass('bg-accent')
    })
  })
})
