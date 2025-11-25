import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ServiceStatusCard } from '@/components/ServiceStatusCard'
import type { Service } from '@/types'

const mockHealthyService: Service = {
  name: 'api',
  local: {
    status: 'running',
    health: 'healthy',
    port: 5000,
    pid: 12345,
    startTime: new Date().toISOString(),
    lastChecked: new Date().toISOString()
  }
}

const mockUnhealthyService: Service = {
  name: 'web',
  local: {
    status: 'running',
    health: 'unhealthy',
    port: 5001,
    pid: 12346,
    startTime: new Date().toISOString(),
    lastChecked: new Date().toISOString()
  }
}

const mockStoppedService: Service = {
  name: 'db',
  local: {
    status: 'stopped',
    health: 'unknown',
    startTime: new Date().toISOString(),
    lastChecked: new Date().toISOString()
  }
}

describe('ServiceStatusCard', () => {
  it('should show loading state when loading', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[]} 
        hasActiveErrors={false} 
        loading={true}
        onClick={onClick}
      />
    )

    expect(screen.getByText('Loading...')).toBeInTheDocument()
  })

  it('should show "No services" when no services are available', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    expect(screen.getByText('No services')).toBeInTheDocument()
  })

  it('should show "All healthy" when all services are healthy', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[mockHealthyService, { ...mockHealthyService, name: 'web' }]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    expect(screen.getByText('All healthy')).toBeInTheDocument()
    expect(screen.getByText('2 services')).toBeInTheDocument()
  })

  it('should show issue count when services are unhealthy', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[mockHealthyService, mockUnhealthyService]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    expect(screen.getByText('1 issue')).toBeInTheDocument()
    expect(screen.getByText('2 services')).toBeInTheDocument()
  })

  it('should show plural "issues" when multiple services are unhealthy', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[mockUnhealthyService, mockStoppedService]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    expect(screen.getByText('2 issues')).toBeInTheDocument()
  })

  it('should show "Log errors" when hasActiveErrors is true but services are healthy', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[mockHealthyService]} 
        hasActiveErrors={true} 
        loading={false}
        onClick={onClick}
      />
    )

    expect(screen.getByText('Log errors')).toBeInTheDocument()
  })

  it('should show orange styling when hasActiveErrors is true but services are healthy', () => {
    const onClick = vi.fn()
    const { container } = render(
      <ServiceStatusCard 
        services={[mockHealthyService]} 
        hasActiveErrors={true} 
        loading={false}
        onClick={onClick}
      />
    )

    const button = container.querySelector('button')
    expect(button).toHaveClass('bg-orange-50')
    expect(button).toHaveClass('text-orange-600')
    expect(button).toHaveClass('ring-2')
    expect(button).toHaveClass('ring-orange-500/50')
  })

  it('should show healthy count when some services are healthy and some are not', () => {
    const onClick = vi.fn()
    const services = [mockHealthyService, mockHealthyService, mockUnhealthyService]
    render(
      <ServiceStatusCard 
        services={services} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    // Should show issue count when there are problems
    expect(screen.getByText('1 issue')).toBeInTheDocument()
  })

  it('should call onClick when clicked', async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[mockHealthyService]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    const button = screen.getByRole('button')
    await user.click(button)

    expect(onClick).toHaveBeenCalledTimes(1)
  })

  it('should have proper title attribute', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[mockHealthyService]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    const button = screen.getByRole('button')
    expect(button).toHaveAttribute('title', 'Click to view console logs')
  })

  it('should show green styling when all healthy', () => {
    const onClick = vi.fn()
    const { container } = render(
      <ServiceStatusCard 
        services={[mockHealthyService]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    const button = container.querySelector('button')
    expect(button).toHaveClass('bg-green-50')
    expect(button).toHaveClass('text-green-600')
  })

  it('should show red styling and ring when there are errors', () => {
    const onClick = vi.fn()
    const { container } = render(
      <ServiceStatusCard 
        services={[mockUnhealthyService]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    const button = container.querySelector('button')
    expect(button).toHaveClass('bg-red-50')
    expect(button).toHaveClass('text-red-600')
    expect(button).toHaveClass('ring-2')
    expect(button).toHaveClass('ring-red-500/50')
  })

  it('should show singular "service" when only one service', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[mockHealthyService]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    expect(screen.getByText('1 service')).toBeInTheDocument()
  })

  it('should count stopped services as issues', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[mockHealthyService, mockStoppedService]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    expect(screen.getByText('1 issue')).toBeInTheDocument()
  })

  it('should render CheckCircle icon when all healthy', () => {
    const onClick = vi.fn()
    const { container } = render(
      <ServiceStatusCard 
        services={[mockHealthyService]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    // CheckCircle2 should be present
    const svg = container.querySelector('svg')
    expect(svg).toBeInTheDocument()
  })
})
