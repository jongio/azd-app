import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { PerformanceMetrics } from './PerformanceMetrics'
import type { Service } from '@/types'

const mockServices: Service[] = [
  {
    name: 'api',
    framework: 'Flask',
    language: 'Python',
    local: {
      status: 'ready',
      health: 'healthy',
      url: 'http://localhost:5000',
      port: 5000,
      startTime: new Date(Date.now() - 3600000).toISOString(), // 1 hour ago
    },
  },
  {
    name: 'web',
    framework: 'Express',
    language: 'Node.js',
    local: {
      status: 'ready',
      health: 'healthy',
      url: 'http://localhost:3000',
      port: 3000,
      startTime: new Date(Date.now() - 7200000).toISOString(), // 2 hours ago
    },
  },
  {
    name: 'worker',
    framework: 'Celery',
    language: 'Python',
    local: {
      status: 'error',
      health: 'unhealthy',
      startTime: new Date(Date.now() - 1800000).toISOString(), // 30 min ago
    },
  },
]

describe('PerformanceMetrics', () => {
  it('should render metric cards', () => {
    render(<PerformanceMetrics services={mockServices} />)
    
    expect(screen.getByText('Active Services')).toBeInTheDocument()
    expect(screen.getByText('Active Ports')).toBeInTheDocument()
    expect(screen.getByText('Avg Uptime')).toBeInTheDocument()
    expect(screen.getByText('Health Score')).toBeInTheDocument()
  })

  it('should display correct active services count', () => {
    render(<PerformanceMetrics services={mockServices} />)
    
    // Look for the actual value displayed in the card
    const activeServicesCard = screen.getByText('Active Services').closest('.glass')
    expect(activeServicesCard).toHaveTextContent('2')
    expect(activeServicesCard).toHaveTextContent('/ 3')
  })

  it('should display correct active ports count', () => {
    render(<PerformanceMetrics services={mockServices} />)
    
    const activePortsCard = screen.getByText('Active Ports').closest('.glass')
    expect(activePortsCard).toHaveTextContent('2')
    expect(activePortsCard).toHaveTextContent('ports')
  })

  it('should display health score as percentage', () => {
    render(<PerformanceMetrics services={mockServices} />)
    
    const healthScoreCard = screen.getByText('Health Score').closest('.glass')
    // 2 running out of 3 total = 67%
    expect(healthScoreCard).toHaveTextContent('67')
    expect(healthScoreCard).toHaveTextContent('%')
  })

  it('should render service metrics table', () => {
    render(<PerformanceMetrics services={mockServices} />)
    
    expect(screen.getByText('Service Metrics')).toBeInTheDocument()
    
    // Check table headers
    expect(screen.getByText('Service')).toBeInTheDocument()
    expect(screen.getByText('Status')).toBeInTheDocument()
    expect(screen.getByText('Uptime')).toBeInTheDocument()
    expect(screen.getByText('Port')).toBeInTheDocument()
    expect(screen.getByText('Health')).toBeInTheDocument()
  })

  it('should display all services in the table', () => {
    render(<PerformanceMetrics services={mockServices} />)
    
    expect(screen.getByText('api')).toBeInTheDocument()
    expect(screen.getByText('web')).toBeInTheDocument()
    expect(screen.getByText('worker')).toBeInTheDocument()
  })

  it('should show service frameworks in the table', () => {
    render(<PerformanceMetrics services={mockServices} />)
    
    expect(screen.getByText('Flask')).toBeInTheDocument()
    expect(screen.getByText('Express')).toBeInTheDocument()
    expect(screen.getByText('Celery')).toBeInTheDocument()
  })

  it('should render info note about performance monitoring', () => {
    render(<PerformanceMetrics services={mockServices} />)
    
    expect(screen.getByText('Performance Monitoring')).toBeInTheDocument()
  })

  it('should handle empty services array', () => {
    render(<PerformanceMetrics services={[]} />)
    
    const activeServicesCard = screen.getByText('Active Services').closest('.glass')
    expect(activeServicesCard).toHaveTextContent('0')
    expect(activeServicesCard).toHaveTextContent('/ 0')
  })

  it('should calculate 100% health score when all services are running', () => {
    const allHealthyServices = mockServices.map(s => ({
      ...s,
      local: { ...s.local, status: 'ready' as const, health: 'healthy' as const }
    }))
    
    render(<PerformanceMetrics services={allHealthyServices} />)
    
    const healthScoreCard = screen.getByText('Health Score').closest('.glass')
    expect(healthScoreCard).toHaveTextContent('100')
  })
})
