import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QuickActions } from './QuickActions'
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
    },
  },
  {
    name: 'worker',
    framework: 'Celery',
    language: 'Python',
    local: {
      status: 'error',
      health: 'unhealthy',
    },
  },
]

describe('QuickActions', () => {
  it('should render stats cards', () => {
    render(<QuickActions services={mockServices} />)
    
    expect(screen.getByText('Running Services')).toBeInTheDocument()
    expect(screen.getByText('Healthy')).toBeInTheDocument()
    expect(screen.getByText('Errors')).toBeInTheDocument()
  })

  it('should display correct running services count', () => {
    render(<QuickActions services={mockServices} />)
    
    const runningCount = screen.getByText('Running Services').parentElement?.querySelector('.text-2xl')
    expect(runningCount).toHaveTextContent('2')
  })

  it('should display correct healthy services count', () => {
    render(<QuickActions services={mockServices} />)
    
    const healthyCount = screen.getByText('Healthy').parentElement?.querySelector('.text-2xl')
    expect(healthyCount).toHaveTextContent('2')
  })

  it('should display correct error services count', () => {
    render(<QuickActions services={mockServices} />)
    
    const errorCount = screen.getByText('Errors').parentElement?.querySelector('.text-2xl')
    expect(errorCount).toHaveTextContent('1')
  })

  it('should render quick action buttons', () => {
    render(<QuickActions services={mockServices} />)
    
    expect(screen.getByText('Refresh All')).toBeInTheDocument()
    expect(screen.getByText('Clear Logs')).toBeInTheDocument()
    expect(screen.getByText('Export Logs')).toBeInTheDocument()
    expect(screen.getByText('Open Terminal')).toBeInTheDocument()
  })

  it('should render service-specific actions section', () => {
    render(<QuickActions services={mockServices} />)
    
    expect(screen.getByText('Service Actions')).toBeInTheDocument()
  })

  it('should display all services in service actions list', () => {
    render(<QuickActions services={mockServices} />)
    
    expect(screen.getByText('api')).toBeInTheDocument()
    expect(screen.getByText('web')).toBeInTheDocument()
    expect(screen.getByText('worker')).toBeInTheDocument()
  })

  it('should render empty state with zero stats when no services', () => {
    render(<QuickActions services={[]} />)
    
    const runningCount = screen.getByText('Running Services').parentElement?.querySelector('.text-2xl')
    expect(runningCount).toHaveTextContent('0')
  })
})
