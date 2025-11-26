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

const mockStartingService: Service = {
  name: 'cache',
  local: {
    status: 'starting',
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

    expect(screen.getByText('Service Status')).toBeInTheDocument()
    // Spinner should be visible (Loader2 component)
    const button = screen.getByRole('button')
    expect(button.querySelector('svg.animate-spin')).toBeInTheDocument()
  })

  it('should show "Service Status" header', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    expect(screen.getByText('Service Status')).toBeInTheDocument()
  })

  it('should show three columns with counts when services are available', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[mockHealthyService, mockUnhealthyService, mockStoppedService]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    // Should have 3 status icons with counts
    expect(screen.getByTitle('Error')).toBeInTheDocument()
    expect(screen.getByTitle('Warning')).toBeInTheDocument()
    expect(screen.getByTitle('Info')).toBeInTheDocument()
  })

  it('should count unhealthy services as errors', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[mockHealthyService, mockUnhealthyService]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    // 1 error (unhealthy), 0 warn, 1 info (healthy)
    const errorDiv = screen.getByTitle('Error')
    const infoDiv = screen.getByTitle('Info')
    expect(errorDiv.textContent).toContain('1')
    expect(infoDiv.textContent).toContain('1')
  })

  it('should count stopped services as errors', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[mockHealthyService, mockStoppedService]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    // 1 error (stopped), 0 warn, 1 info (healthy)
    const errorDiv = screen.getByTitle('Error')
    expect(errorDiv.textContent).toContain('1')
  })

  it('should count starting services as warnings', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[mockHealthyService, mockStartingService]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    // 0 error, 1 warn (starting), 1 info (healthy)
    const warnDiv = screen.getByTitle('Warning')
    const infoDiv = screen.getByTitle('Info')
    expect(warnDiv.textContent).toContain('1')
    expect(infoDiv.textContent).toContain('1')
  })

  it('should move info to warn when hasActiveErrors is true', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[mockHealthyService]} 
        hasActiveErrors={true} 
        loading={false}
        onClick={onClick}
      />
    )

    // When hasActiveErrors but no error services, info moves to warn
    const warnDiv = screen.getByTitle('Warning')
    const infoDiv = screen.getByTitle('Info')
    expect(warnDiv.textContent).toContain('1')
    expect(infoDiv.textContent).toContain('0')
  })

  it('should show orange ring when hasActiveErrors is true', () => {
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
    expect(button).toHaveClass('ring-2')
    expect(button).toHaveClass('ring-orange-500/50')
  })

  it('should show red ring when there are error services', () => {
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
    expect(button).toHaveClass('ring-2')
    expect(button).toHaveClass('ring-red-500/50')
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

  it('should show all zeros when no services', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    const errorDiv = screen.getByTitle('Error')
    const warnDiv = screen.getByTitle('Warning')
    const infoDiv = screen.getByTitle('Info')
    expect(errorDiv.textContent).toContain('0')
    expect(warnDiv.textContent).toContain('0')
    expect(infoDiv.textContent).toContain('0')
  })

  it('should count all healthy services as info', () => {
    const onClick = vi.fn()
    render(
      <ServiceStatusCard 
        services={[mockHealthyService, { ...mockHealthyService, name: 'web' }]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    const infoDiv = screen.getByTitle('Info')
    expect(infoDiv.textContent).toContain('2')
  })

  it('should render status icons', () => {
    const onClick = vi.fn()
    const { container } = render(
      <ServiceStatusCard 
        services={[mockHealthyService]} 
        hasActiveErrors={false} 
        loading={false}
        onClick={onClick}
      />
    )

    // Should have 3 svg icons (XCircle, AlertTriangle, Info)
    const svgs = container.querySelectorAll('svg')
    expect(svgs.length).toBe(3)
  })
})
