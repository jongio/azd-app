import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { PerformanceMetrics } from './PerformanceMetrics'
import {
  countActiveServices,
  countActivePorts,
  calculateAverageUptime,
  calculateHealthScore,
  formatDuration,
  formatResponseTime,
  getResponseTimeVariant,
  getHealthScoreVariant,
  getServiceUptime
} from '@/lib/metrics-utils'
import type { Service, HealthReportEvent } from '@/types'

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
    port: 3000,
    startTime: new Date(Date.now() - 3600000).toISOString(), // 1 hour ago
  },
  ...overrides,
})

const mockServices: Service[] = [
  createService({
    name: 'api',
    local: { status: 'running', health: 'healthy', port: 3100, startTime: new Date(Date.now() - 7200000).toISOString() },
  }),
  createService({
    name: 'web',
    local: { status: 'ready', health: 'healthy', port: 3000, startTime: new Date(Date.now() - 7200000).toISOString() },
  }),
  createService({
    name: 'worker',
    local: { status: 'running', health: 'degraded', startTime: new Date(Date.now() - 3600000).toISOString() },
  }),
  createService({
    name: 'cache',
    local: { status: 'stopped', health: 'unknown' },
  }),
  createService({
    name: 'db',
    local: { status: 'error', health: 'unhealthy', port: 5432 },
  }),
]

const mockHealthReport: HealthReportEvent = {
  type: 'health',
  timestamp: new Date().toISOString(),
  services: [
    { serviceName: 'api', status: 'healthy', checkType: 'http', responseTime: 45000000, timestamp: new Date().toISOString() },
    { serviceName: 'web', status: 'healthy', checkType: 'http', responseTime: 12000000, timestamp: new Date().toISOString() },
    { serviceName: 'worker', status: 'degraded', checkType: 'process', responseTime: 0, timestamp: new Date().toISOString() },
    { serviceName: 'db', status: 'unhealthy', checkType: 'port', responseTime: 5000000000, timestamp: new Date().toISOString() },
  ],
  summary: {
    total: 5,
    healthy: 2,
    degraded: 1,
    unhealthy: 1,
    unknown: 1,
    overall: 'unhealthy',
  },
}

// ============================================================================
// Helper Function Tests
// ============================================================================

describe('countActiveServices', () => {
  it('counts services with status running or ready', () => {
    expect(countActiveServices(mockServices)).toBe(3) // api, web, worker
  })

  it('returns 0 for empty array', () => {
    expect(countActiveServices([])).toBe(0)
  })

  it('returns 0 when no services are running', () => {
    const stoppedServices = [
      createService({ local: { status: 'stopped', health: 'unknown' } }),
      createService({ local: { status: 'error', health: 'unhealthy' } }),
    ]
    expect(countActiveServices(stoppedServices)).toBe(0)
  })
})

describe('countActivePorts', () => {
  it('counts unique ports', () => {
    expect(countActivePorts(mockServices)).toBe(3) // 3100, 3000, 5432
  })

  it('returns 0 for empty array', () => {
    expect(countActivePorts([])).toBe(0)
  })

  it('returns 0 when no services have ports', () => {
    const services = [
      createService({ local: { status: 'running', health: 'healthy' } }),
    ]
    expect(countActivePorts(services)).toBe(0)
  })

  it('counts duplicate ports once', () => {
    const services = [
      createService({ name: 'a', local: { status: 'running', health: 'healthy', port: 3000 } }),
      createService({ name: 'b', local: { status: 'running', health: 'healthy', port: 3000 } }),
    ]
    expect(countActivePorts(services)).toBe(1)
  })
})

describe('calculateAverageUptime', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2024-01-01T12:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('calculates average uptime correctly', () => {
    const services = [
      createService({ local: { status: 'running', health: 'healthy', startTime: new Date('2024-01-01T11:00:00Z').toISOString() } }), // 1 hour
      createService({ local: { status: 'running', health: 'healthy', startTime: new Date('2024-01-01T10:00:00Z').toISOString() } }), // 2 hours
    ]
    // Average of 3600 and 7200 seconds = 5400 seconds
    expect(calculateAverageUptime(services)).toBe(5400)
  })

  it('returns 0 for empty array', () => {
    expect(calculateAverageUptime([])).toBe(0)
  })

  it('returns 0 when no services have startTime', () => {
    const services = [
      createService({ local: { status: 'running', health: 'healthy' } }),
    ]
    expect(calculateAverageUptime(services)).toBe(0)
  })
})

describe('calculateHealthScore', () => {
  it('returns 100 for empty array', () => {
    expect(calculateHealthScore([])).toBe(100)
  })

  it('returns 100 when all healthy', () => {
    const services = [
      createService({ local: { status: 'running', health: 'healthy' } }),
      createService({ local: { status: 'running', health: 'healthy' } }),
    ]
    expect(calculateHealthScore(services)).toBe(100)
  })

  it('returns 0 when none healthy', () => {
    const services = [
      createService({ local: { status: 'running', health: 'unhealthy' } }),
      createService({ local: { status: 'running', health: 'unknown' } }),
    ]
    expect(calculateHealthScore(services)).toBe(0)
  })

  it('calculates percentage correctly', () => {
    const services = [
      createService({ local: { status: 'running', health: 'healthy' } }),
      createService({ local: { status: 'running', health: 'unhealthy' } }),
      createService({ local: { status: 'running', health: 'healthy' } }),
      createService({ local: { status: 'running', health: 'unknown' } }),
    ]
    expect(calculateHealthScore(services)).toBe(50) // 2/4 = 50%
  })
})

describe('formatDuration', () => {
  it('returns - for 0 seconds', () => {
    expect(formatDuration(0)).toBe('-')
  })

  it('returns - for negative seconds', () => {
    expect(formatDuration(-1)).toBe('-')
  })

  it('formats seconds', () => {
    expect(formatDuration(30)).toBe('30s')
    expect(formatDuration(59)).toBe('59s')
  })

  it('formats minutes', () => {
    expect(formatDuration(60)).toBe('1m')
    expect(formatDuration(90)).toBe('1m')
    expect(formatDuration(3599)).toBe('59m')
  })

  it('formats hours and minutes', () => {
    expect(formatDuration(3600)).toBe('1h')
    expect(formatDuration(5400)).toBe('1h 30m')
    expect(formatDuration(7200)).toBe('2h')
  })

  it('formats days and hours', () => {
    expect(formatDuration(86400)).toBe('1d')
    expect(formatDuration(90000)).toBe('1d 1h')
    expect(formatDuration(172800)).toBe('2d')
  })
})

describe('formatResponseTime', () => {
  it('returns - for null', () => {
    expect(formatResponseTime(null)).toBe('-')
  })

  it('returns - for undefined', () => {
    expect(formatResponseTime(undefined)).toBe('-')
  })

  it('formats milliseconds', () => {
    expect(formatResponseTime(45)).toBe('45ms')
    expect(formatResponseTime(999)).toBe('999ms')
  })

  it('formats seconds', () => {
    expect(formatResponseTime(1000)).toBe('1.0s')
    expect(formatResponseTime(1500)).toBe('1.5s')
    expect(formatResponseTime(5000)).toBe('5.0s')
  })
})

describe('getResponseTimeVariant', () => {
  it('returns default for null', () => {
    expect(getResponseTimeVariant(null)).toBe('default')
  })

  it('returns success for < 100ms', () => {
    expect(getResponseTimeVariant(50)).toBe('success')
    expect(getResponseTimeVariant(99)).toBe('success')
  })

  it('returns warning for 100-499ms', () => {
    expect(getResponseTimeVariant(100)).toBe('warning')
    expect(getResponseTimeVariant(499)).toBe('warning')
  })

  it('returns error for >= 500ms', () => {
    expect(getResponseTimeVariant(500)).toBe('error')
    expect(getResponseTimeVariant(1000)).toBe('error')
  })
})

describe('getHealthScoreVariant', () => {
  it('returns success for >= 80', () => {
    expect(getHealthScoreVariant(80)).toBe('success')
    expect(getHealthScoreVariant(100)).toBe('success')
  })

  it('returns warning for 50-79', () => {
    expect(getHealthScoreVariant(50)).toBe('warning')
    expect(getHealthScoreVariant(79)).toBe('warning')
  })

  it('returns error for < 50', () => {
    expect(getHealthScoreVariant(49)).toBe('error')
    expect(getHealthScoreVariant(0)).toBe('error')
  })
})

describe('getServiceUptime', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2024-01-01T12:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('returns null when no startTime', () => {
    const service = createService({ local: { status: 'running', health: 'healthy' } })
    expect(getServiceUptime(service)).toBeNull()
  })

  it('calculates uptime in seconds', () => {
    const service = createService({
      local: { status: 'running', health: 'healthy', startTime: new Date('2024-01-01T11:00:00Z').toISOString() }
    })
    expect(getServiceUptime(service)).toBe(3600) // 1 hour
  })
})

// ============================================================================
// PerformanceMetrics Component Tests
// ============================================================================

describe('PerformanceMetrics', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2024-01-01T12:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('Rendering', () => {
    it('renders main container with correct test id', () => {
      render(<PerformanceMetrics services={mockServices} />)
      expect(screen.getByTestId('performance-metrics')).toBeInTheDocument()
    })

    it('renders with custom test id', () => {
      render(<PerformanceMetrics services={mockServices} data-testid="custom-metrics" />)
      expect(screen.getByTestId('custom-metrics')).toBeInTheDocument()
    })

    it('renders with custom className', () => {
      render(<PerformanceMetrics services={mockServices} className="custom-class" />)
      expect(screen.getByTestId('performance-metrics')).toHaveClass('custom-class')
    })

    it('renders screen reader title', () => {
      render(<PerformanceMetrics services={mockServices} />)
      expect(screen.getByText('Performance Metrics Dashboard')).toBeInTheDocument()
    })
  })

  describe('Aggregate Metrics Section', () => {
    it('renders overview heading', () => {
      render(<PerformanceMetrics services={mockServices} />)
      expect(screen.getByText('Overview')).toBeInTheDocument()
    })

    it('renders all four metric cards', () => {
      render(<PerformanceMetrics services={mockServices} />)
      expect(screen.getByTestId('metric-card-active-services')).toBeInTheDocument()
      expect(screen.getByTestId('metric-card-active-ports')).toBeInTheDocument()
      expect(screen.getByTestId('metric-card-avg-uptime')).toBeInTheDocument()
      expect(screen.getByTestId('metric-card-health-score')).toBeInTheDocument()
    })

    it('displays correct active services count', () => {
      render(<PerformanceMetrics services={mockServices} />)
      const card = screen.getByTestId('metric-card-active-services')
      expect(card).toHaveTextContent('3/5')
    })

    it('displays correct active ports count', () => {
      render(<PerformanceMetrics services={mockServices} />)
      const card = screen.getByTestId('metric-card-active-ports')
      expect(card).toHaveTextContent('3')
    })

    it('displays health score from healthReport when available', () => {
      render(<PerformanceMetrics services={mockServices} healthReport={mockHealthReport} />)
      const card = screen.getByTestId('metric-card-health-score')
      // 2 healthy / 5 total = 40%
      expect(card).toHaveTextContent('40')
      expect(card).toHaveTextContent('%')
    })

    it('calculates health score from services when no healthReport', () => {
      render(<PerformanceMetrics services={mockServices} />)
      const card = screen.getByTestId('metric-card-health-score')
      // 2 healthy out of 5 = 40%
      expect(card).toHaveTextContent('40')
    })

    it('handles empty services array', () => {
      render(<PerformanceMetrics services={[]} />)
      expect(screen.getByTestId('metric-card-active-services')).toHaveTextContent('0/0')
      expect(screen.getByTestId('metric-card-active-ports')).toHaveTextContent('0')
      expect(screen.getByTestId('metric-card-health-score')).toHaveTextContent('-')
    })
  })

  describe('Service Details Table', () => {
    it('renders service details heading', () => {
      render(<PerformanceMetrics services={mockServices} />)
      expect(screen.getByText('Service Details')).toBeInTheDocument()
    })

    it('renders table headers', () => {
      render(<PerformanceMetrics services={mockServices} />)
      expect(screen.getByText('Service')).toBeInTheDocument()
      expect(screen.getByText('Status')).toBeInTheDocument()
      expect(screen.getByText('Uptime')).toBeInTheDocument()
      expect(screen.getByText('Port')).toBeInTheDocument()
      expect(screen.getByText('Health')).toBeInTheDocument()
      expect(screen.getByText('Response')).toBeInTheDocument()
    })

    it('renders a row for each service', () => {
      render(<PerformanceMetrics services={mockServices} />)
      expect(screen.getByTestId('service-row-api')).toBeInTheDocument()
      expect(screen.getByTestId('service-row-web')).toBeInTheDocument()
      expect(screen.getByTestId('service-row-worker')).toBeInTheDocument()
      expect(screen.getByTestId('service-row-cache')).toBeInTheDocument()
      expect(screen.getByTestId('service-row-db')).toBeInTheDocument()
    })

    it('displays service names', () => {
      render(<PerformanceMetrics services={mockServices} />)
      expect(screen.getByText('api')).toBeInTheDocument()
      expect(screen.getByText('web')).toBeInTheDocument()
      expect(screen.getByText('worker')).toBeInTheDocument()
    })

    it('displays status badges', () => {
      render(<PerformanceMetrics services={mockServices} />)
      // api and worker both have 'Running' status
      expect(screen.getAllByText('Running').length).toBeGreaterThanOrEqual(1)
      expect(screen.getByText('Ready')).toBeInTheDocument()
      expect(screen.getByText('Stopped')).toBeInTheDocument()
      expect(screen.getByText('Error')).toBeInTheDocument()
    })

    it('displays health badges', () => {
      render(<PerformanceMetrics services={mockServices} />)
      expect(screen.getAllByText('Healthy').length).toBeGreaterThan(0)
      expect(screen.getByText('Degraded')).toBeInTheDocument()
      expect(screen.getByText('Unknown')).toBeInTheDocument()
      expect(screen.getByText('Unhealthy')).toBeInTheDocument()
    })

    it('displays ports or dash for missing', () => {
      render(<PerformanceMetrics services={mockServices} />)
      expect(screen.getByText('3100')).toBeInTheDocument()
      expect(screen.getByText('3000')).toBeInTheDocument()
      expect(screen.getByText('5432')).toBeInTheDocument()
    })

    it('displays empty state when no services', () => {
      render(<PerformanceMetrics services={[]} />)
      expect(screen.getByTestId('empty-state')).toBeInTheDocument()
      expect(screen.getByText('No services available')).toBeInTheDocument()
    })

    it('displays response times from health report', () => {
      render(<PerformanceMetrics services={mockServices} healthReport={mockHealthReport} />)
      // 45000000 ns = 45ms
      expect(screen.getByText('45ms')).toBeInTheDocument()
      // 12000000 ns = 12ms
      expect(screen.getByText('12ms')).toBeInTheDocument()
    })
  })

  describe('Accessibility', () => {
    it('has proper aria labels on metric cards', () => {
      render(<PerformanceMetrics services={mockServices} />)
      
      expect(screen.getByLabelText(/Active Services: 3\/5/)).toBeInTheDocument()
      expect(screen.getByLabelText(/Active Ports: 3/)).toBeInTheDocument()
    })

    it('metric cards have status role', () => {
      render(<PerformanceMetrics services={mockServices} />)
      
      const statusElements = screen.getAllByRole('status')
      expect(statusElements.length).toBe(4)
    })

    it('table has proper structure', () => {
      render(<PerformanceMetrics services={mockServices} />)
      
      const table = screen.getByRole('table')
      expect(table).toBeInTheDocument()
      
      const headers = screen.getAllByRole('columnheader')
      expect(headers.length).toBe(6)
    })
  })

  describe('Responsive Design', () => {
    it('metrics grid has correct responsive classes', () => {
      render(<PerformanceMetrics services={mockServices} />)
      const grid = screen.getByTestId('metrics-grid')
      expect(grid.className).toContain('grid-cols-1')
      expect(grid.className).toContain('sm:grid-cols-2')
      expect(grid.className).toContain('lg:grid-cols-4')
    })
  })
})
