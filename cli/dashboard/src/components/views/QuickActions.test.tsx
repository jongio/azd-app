import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { QuickActions } from './QuickActions'
import {
  countRunningServices,
  countHealthyServices,
  countErrorServices,
  pluralize
} from '@/lib/service-stats'
import type { Service } from '@/types'

// ============================================================================
// Test Data
// ============================================================================

const createService = (overrides: Partial<Service> = {}): Service => ({
  name: 'test-service',
  language: 'typescript',
  framework: 'node',
  project: 'test-project',
  local: {
    status: 'running',
    health: 'healthy',
  },
  ...overrides,
})

const mockServices: Service[] = [
  createService({
    name: 'api',
    local: { status: 'running', health: 'healthy' },
  }),
  createService({
    name: 'web',
    local: { status: 'ready', health: 'healthy' },
  }),
  createService({
    name: 'worker',
    local: { status: 'running', health: 'unhealthy' },
  }),
  createService({
    name: 'cache',
    local: { status: 'stopped', health: 'unknown' },
  }),
  createService({
    name: 'db',
    local: { status: 'error', health: 'unhealthy' },
  }),
]

// ============================================================================
// Helper Function Tests
// ============================================================================

describe('countRunningServices', () => {
  it('counts services with status running or ready', () => {
    expect(countRunningServices(mockServices)).toBe(3) // api, web, worker
  })

  it('returns 0 for empty array', () => {
    expect(countRunningServices([])).toBe(0)
  })

  it('returns 0 when no services are running', () => {
    const stoppedServices = [
      createService({ local: { status: 'stopped', health: 'unknown' } }),
      createService({ local: { status: 'error', health: 'unhealthy' } }),
    ]
    expect(countRunningServices(stoppedServices)).toBe(0)
  })

  it('handles services without local property', () => {
    const services = [
      createService({ local: undefined }),
      createService({ local: { status: 'running', health: 'healthy' } }),
    ]
    expect(countRunningServices(services)).toBe(1)
  })
})

describe('countHealthyServices', () => {
  it('counts services with health healthy', () => {
    expect(countHealthyServices(mockServices)).toBe(2) // api, web
  })

  it('returns 0 for empty array', () => {
    expect(countHealthyServices([])).toBe(0)
  })

  it('returns 0 when no services are healthy', () => {
    const unhealthyServices = [
      createService({ local: { status: 'running', health: 'unhealthy' } }),
      createService({ local: { status: 'running', health: 'unknown' } }),
    ]
    expect(countHealthyServices(unhealthyServices)).toBe(0)
  })

  it('handles services without local property', () => {
    const services = [
      createService({ local: undefined }),
      createService({ local: { status: 'running', health: 'healthy' } }),
    ]
    expect(countHealthyServices(services)).toBe(1)
  })
})

describe('countErrorServices', () => {
  it('counts services with error status or unhealthy health', () => {
    expect(countErrorServices(mockServices)).toBe(2) // worker (unhealthy), db (error + unhealthy)
  })

  it('returns 0 for empty array', () => {
    expect(countErrorServices([])).toBe(0)
  })

  it('returns 0 when no services have errors', () => {
    const healthyServices = [
      createService({ local: { status: 'running', health: 'healthy' } }),
      createService({ local: { status: 'ready', health: 'healthy' } }),
    ]
    expect(countErrorServices(healthyServices)).toBe(0)
  })

  it('handles services without local property', () => {
    const services = [
      createService({ local: undefined }),
      createService({ local: { status: 'error', health: 'unhealthy' } }),
    ]
    expect(countErrorServices(services)).toBe(1)
  })
})

describe('pluralize', () => {
  it('returns singular when count is 1', () => {
    expect(pluralize(1, 'service', 'services')).toBe('service')
  })

  it('returns plural when count is 0', () => {
    expect(pluralize(0, 'service', 'services')).toBe('services')
  })

  it('returns plural when count is greater than 1', () => {
    expect(pluralize(2, 'service', 'services')).toBe('services')
    expect(pluralize(100, 'service', 'services')).toBe('services')
  })
})

// ============================================================================
// QuickActions Component Tests
// ============================================================================

describe('QuickActions', () => {
  let dispatchEventSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    dispatchEventSpy = vi.spyOn(window, 'dispatchEvent')
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  describe('Rendering', () => {
    it('renders main container with correct test id', () => {
      render(<QuickActions services={mockServices} />)
      expect(screen.getByTestId('quick-actions')).toBeInTheDocument()
    })

    it('renders with custom test id', () => {
      render(<QuickActions services={mockServices} data-testid="custom-actions" />)
      expect(screen.getByTestId('custom-actions')).toBeInTheDocument()
    })

    it('renders with custom className', () => {
      render(<QuickActions services={mockServices} className="custom-class" />)
      expect(screen.getByTestId('quick-actions')).toHaveClass('custom-class')
    })

    it('renders screen reader title', () => {
      render(<QuickActions services={mockServices} />)
      expect(screen.getByText('Quick Actions Dashboard')).toBeInTheDocument()
    })
  })

  describe('Stats Section', () => {
    it('renders stats section with heading', () => {
      render(<QuickActions services={mockServices} />)
      expect(screen.getByText('Service Statistics')).toBeInTheDocument()
    })

    it('renders all three stat cards', () => {
      render(<QuickActions services={mockServices} />)
      expect(screen.getByTestId('stat-card-primary')).toBeInTheDocument()
      expect(screen.getByTestId('stat-card-success')).toBeInTheDocument()
      expect(screen.getByTestId('stat-card-error')).toBeInTheDocument()
    })

    it('displays correct running count', () => {
      render(<QuickActions services={mockServices} />)
      const runningCard = screen.getByTestId('stat-card-primary')
      expect(runningCard).toHaveTextContent('Running')
      expect(runningCard).toHaveTextContent('3')
      expect(runningCard).toHaveTextContent('services')
    })

    it('displays correct healthy count', () => {
      render(<QuickActions services={mockServices} />)
      const healthyCard = screen.getByTestId('stat-card-success')
      expect(healthyCard).toHaveTextContent('Healthy')
      expect(healthyCard).toHaveTextContent('2')
      expect(healthyCard).toHaveTextContent('services')
    })

    it('displays correct error count', () => {
      render(<QuickActions services={mockServices} />)
      const errorCard = screen.getByTestId('stat-card-error')
      expect(errorCard).toHaveTextContent('Errors')
      expect(errorCard).toHaveTextContent('2')
      expect(errorCard).toHaveTextContent('services')
    })

    it('uses singular form when count is 1', () => {
      const singleService = [
        createService({ name: 'only', local: { status: 'running', health: 'healthy' } }),
      ]
      render(<QuickActions services={singleService} />)
      const runningCard = screen.getByTestId('stat-card-primary')
      expect(runningCard).toHaveTextContent('1')
      expect(runningCard).toHaveTextContent('service')
      expect(runningCard).not.toHaveTextContent('services')
    })

    it('handles empty services array', () => {
      render(<QuickActions services={[]} />)
      expect(screen.getByTestId('stat-card-primary')).toHaveTextContent('0')
      expect(screen.getByTestId('stat-card-success')).toHaveTextContent('0')
      expect(screen.getByTestId('stat-card-error')).toHaveTextContent('0')
    })
  })

  describe('Actions Section', () => {
    it('renders actions section with heading', () => {
      render(<QuickActions services={mockServices} />)
      expect(screen.getByText('Global Actions')).toBeInTheDocument()
    })

    it('renders all action buttons', () => {
      render(<QuickActions services={mockServices} />)
      expect(screen.getByTestId('refresh-all-btn')).toBeInTheDocument()
      expect(screen.getByTestId('clear-logs-btn')).toBeInTheDocument()
      expect(screen.getByTestId('export-logs-btn')).toBeInTheDocument()
      expect(screen.getByTestId('open-terminal-btn')).toBeInTheDocument()
    })

    it('renders buttons with correct labels', () => {
      render(<QuickActions services={mockServices} />)
      expect(screen.getByText('Refresh All')).toBeInTheDocument()
      expect(screen.getByText('Clear Logs')).toBeInTheDocument()
      expect(screen.getByText('Export Logs')).toBeInTheDocument()
      expect(screen.getByText('Open Terminal')).toBeInTheDocument()
    })
  })

  describe('Refresh All Button', () => {
    it('calls onRefresh callback when clicked', () => {
      const onRefresh = vi.fn()
      render(<QuickActions services={mockServices} onRefresh={onRefresh} />)
      
      fireEvent.click(screen.getByTestId('refresh-all-btn'))
      expect(onRefresh).toHaveBeenCalledTimes(1)
    })

    it('shows loading state when clicked', () => {
      render(<QuickActions services={mockServices} />)
      
      fireEvent.click(screen.getByTestId('refresh-all-btn'))
      expect(screen.getByText('Refreshing...')).toBeInTheDocument()
    })

    it('disables button during refresh', () => {
      render(<QuickActions services={mockServices} />)
      
      fireEvent.click(screen.getByTestId('refresh-all-btn'))
      expect(screen.getByTestId('refresh-all-btn')).toBeDisabled()
    })

    it('re-enables button after timeout', () => {
      render(<QuickActions services={mockServices} />)
      
      fireEvent.click(screen.getByTestId('refresh-all-btn'))
      expect(screen.getByTestId('refresh-all-btn')).toBeDisabled()
      expect(screen.getByText('Refreshing...')).toBeInTheDocument()
      
      // Advance fake timers and wrap in act for state updates
      act(() => {
        vi.advanceTimersByTime(500)
      })
      
      expect(screen.getByTestId('refresh-all-btn')).not.toBeDisabled()
      expect(screen.getByText('Refresh All')).toBeInTheDocument()
    })

    it('prevents multiple rapid clicks', () => {
      const onRefresh = vi.fn()
      render(<QuickActions services={mockServices} onRefresh={onRefresh} />)
      
      fireEvent.click(screen.getByTestId('refresh-all-btn'))
      fireEvent.click(screen.getByTestId('refresh-all-btn'))
      fireEvent.click(screen.getByTestId('refresh-all-btn'))
      
      expect(onRefresh).toHaveBeenCalledTimes(1)
    })

    it('works without onRefresh callback', () => {
      render(<QuickActions services={mockServices} />)
      
      expect(() => {
        fireEvent.click(screen.getByTestId('refresh-all-btn'))
      }).not.toThrow()
    })
  })

  describe('Clear Logs Button', () => {
    it('dispatches clear-all-logs event when clicked', () => {
      render(<QuickActions services={mockServices} />)
      
      fireEvent.click(screen.getByTestId('clear-logs-btn'))
      
      expect(dispatchEventSpy).toHaveBeenCalledWith(
        expect.objectContaining({ type: 'clear-all-logs' })
      )
    })
  })

  describe('Export Logs Button', () => {
    it('dispatches export-all-logs event when clicked', () => {
      render(<QuickActions services={mockServices} />)
      
      fireEvent.click(screen.getByTestId('export-logs-btn'))
      
      expect(dispatchEventSpy).toHaveBeenCalledWith(
        expect.objectContaining({ type: 'export-all-logs' })
      )
    })
  })

  describe('Open Terminal Button', () => {
    it('dispatches open-terminal event when clicked', () => {
      render(<QuickActions services={mockServices} />)
      
      fireEvent.click(screen.getByTestId('open-terminal-btn'))
      
      expect(dispatchEventSpy).toHaveBeenCalledWith(
        expect.objectContaining({ type: 'open-terminal' })
      )
    })
  })

  describe('Accessibility', () => {
    it('has proper aria labels on stat cards', () => {
      render(<QuickActions services={mockServices} />)
      
      expect(screen.getByLabelText('3 running services')).toBeInTheDocument()
      expect(screen.getByLabelText('2 healthy services')).toBeInTheDocument()
      expect(screen.getByLabelText('2 errors services')).toBeInTheDocument()
    })

    it('stat cards have status role', () => {
      render(<QuickActions services={mockServices} />)
      
      const statusElements = screen.getAllByRole('status')
      expect(statusElements.length).toBe(3)
    })

    it('action buttons have aria-label', () => {
      render(<QuickActions services={mockServices} />)
      
      expect(screen.getByLabelText('Refresh all services')).toBeInTheDocument()
      expect(screen.getByLabelText('Clear all logs')).toBeInTheDocument()
      expect(screen.getByLabelText('Export all logs')).toBeInTheDocument()
      expect(screen.getByLabelText('Open terminal')).toBeInTheDocument()
    })

    it('updates refresh button aria-label when refreshing', () => {
      render(<QuickActions services={mockServices} />)
      
      fireEvent.click(screen.getByTestId('refresh-all-btn'))
      expect(screen.getByLabelText('Refreshing all services')).toBeInTheDocument()
    })

    it('action buttons are grouped', () => {
      render(<QuickActions services={mockServices} />)
      
      const group = screen.getByRole('group', { name: 'Action buttons' })
      expect(group).toBeInTheDocument()
    })

    it('buttons are keyboard accessible', () => {
      render(<QuickActions services={mockServices} />)
      
      const buttons = screen.getAllByRole('button')
      buttons.forEach(button => {
        expect(button).not.toHaveAttribute('tabindex', '-1')
      })
    })
  })

  describe('Responsive Design', () => {
    it('stats grid has correct responsive classes', () => {
      render(<QuickActions services={mockServices} />)
      const grid = screen.getByTestId('stats-grid')
      expect(grid.className).toContain('grid-cols-1')
      expect(grid.className).toContain('sm:grid-cols-2')
      expect(grid.className).toContain('lg:grid-cols-3')
    })

    it('actions group has flex wrap', () => {
      render(<QuickActions services={mockServices} />)
      const group = screen.getByTestId('actions-group')
      expect(group.className).toContain('flex-wrap')
    })
  })
})
