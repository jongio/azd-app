import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ServiceCard } from '@/components/ServiceCard'
import {
  mockServices,
  mockServiceWithError,
  mockServiceStopped,
  mockServiceWithAzure,
  createMockService,
} from '@/test/mocks'

describe('ServiceCard', () => {
  it('should render service with healthy status', () => {
    render(<ServiceCard service={mockServices[0]} />)

    expect(screen.getByText('api')).toBeInTheDocument()
    expect(screen.getByText('Running')).toBeInTheDocument()
    expect(screen.getByText('healthy')).toBeInTheDocument()
    expect(screen.getByText('flask')).toBeInTheDocument()
    expect(screen.getByText('python')).toBeInTheDocument()
  })

  it('should display local URL when available', () => {
    render(<ServiceCard service={mockServices[0]} />)

    const link = screen.getByRole('link', { name: /localhost:5000/i })
    expect(link).toHaveAttribute('href', 'http://localhost:5000')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('should display Azure URL when available', () => {
    render(<ServiceCard service={mockServiceWithAzure} />)

    const link = screen.getByRole('link', { name: /azurewebsites\.net/i })
    expect(link).toHaveAttribute('href', 'https://my-app.azurewebsites.net')
    expect(link).toHaveAttribute('target', '_blank')
  })

  it('should display port number when available', () => {
    render(<ServiceCard service={mockServices[0]} />)

    expect(screen.getByText('Port')).toBeInTheDocument()
    expect(screen.getByText('5000')).toBeInTheDocument()
  })

  it('should render service with error status', () => {
    render(<ServiceCard service={mockServiceWithError} />)

    expect(screen.getByText('Error')).toBeInTheDocument()
    expect(screen.getByText('unhealthy')).toBeInTheDocument()
    expect(screen.getByText('Error Detected')).toBeInTheDocument()
    expect(screen.getByText('Failed to start: Port already in use')).toBeInTheDocument()
  })

  it('should render service with stopped status', () => {
    render(<ServiceCard service={mockServiceStopped} />)

    expect(screen.getByText('Stopped')).toBeInTheDocument()
    expect(screen.getByText('unknown')).toBeInTheDocument()
  })

  it('should render service with starting status', () => {
    render(<ServiceCard service={mockServices[2]} />)

    expect(screen.getByText('database')).toBeInTheDocument()
    expect(screen.getByText('Starting')).toBeInTheDocument()
  })

  it('should display start time when available', () => {
    render(<ServiceCard service={mockServices[0]} />)

    expect(screen.getByText('Started')).toBeInTheDocument()
    // Time formatting will show relative time - use getAllByText since "ago" appears multiple times
    expect(screen.getAllByText(/ago/).length).toBeGreaterThan(0)
  })

  it('should display last checked time when available', () => {
    render(<ServiceCard service={mockServices[0]} />)

    expect(screen.getByText('Last checked')).toBeInTheDocument()
  })

  it('should handle service without local info using legacy fields', () => {
    const legacyService = createMockService({
      name: 'legacy-service',
      status: 'running',
      health: 'healthy',
      local: undefined,
      startTime: new Date().toISOString(),
      lastChecked: new Date().toISOString(),
    })

    render(<ServiceCard service={legacyService} />)

    expect(screen.getByText('legacy-service')).toBeInTheDocument()
    expect(screen.getByText('Running')).toBeInTheDocument()
    expect(screen.getByText('healthy')).toBeInTheDocument()
  })

  it('should not display URL section when URL is not available', () => {
    const serviceWithoutUrl = createMockService({
      name: 'no-url-service',
      local: {
        status: 'ready',
        health: 'healthy',
        pid: 12345,
      },
    })

    render(<ServiceCard service={serviceWithoutUrl} />)

    expect(screen.getByText('no-url-service')).toBeInTheDocument()
    // Should not have any links
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })

  it('should apply correct color classes for healthy service', () => {
    const { container } = render(<ServiceCard service={mockServices[0]} />)

    // Check for success-related classes
    expect(container.querySelector('.text-success')).toBeInTheDocument()
  })

  it('should apply correct color classes for error service', () => {
    const { container } = render(<ServiceCard service={mockServiceWithError} />)

    // Check for destructive-related classes
    expect(container.querySelector('.text-destructive')).toBeInTheDocument()
  })

  it('should show pulse animation for healthy service', () => {
    const { container } = render(<ServiceCard service={mockServices[0]} />)

    // Check for animate-pulse class
    expect(container.querySelector('.animate-pulse')).toBeInTheDocument()
  })

  it('should display framework and language information', () => {
    render(<ServiceCard service={mockServices[1]} />)

    expect(screen.getByText('Framework')).toBeInTheDocument()
    expect(screen.getByText('express')).toBeInTheDocument()
    expect(screen.getByText('Language')).toBeInTheDocument()
    expect(screen.getByText('node')).toBeInTheDocument()
  })

  it('should handle missing optional fields gracefully', () => {
    const minimalService = createMockService({
      name: 'minimal',
      local: {
        status: 'ready',
        health: 'healthy',
      },
    })

    render(<ServiceCard service={minimalService} />)

    expect(screen.getByText('minimal')).toBeInTheDocument()
    expect(screen.getByText('Running')).toBeInTheDocument()
  })

  it('should format relative time correctly for recent timestamps', () => {
    const recentTime = new Date(Date.now() - 30000).toISOString() // 30 seconds ago
    const service = createMockService({
      local: {
        ...mockServices[0].local!,
        startTime: recentTime,
      },
    })

    render(<ServiceCard service={service} />)

    expect(screen.getAllByText(/\d+s ago/).length).toBeGreaterThan(0)
  })

  it('should render service instance label', () => {
    render(<ServiceCard service={mockServices[0]} />)

    expect(screen.getByText('Service Instance')).toBeInTheDocument()
  })

  it('should display both local and azure URLs when both are present', () => {
    render(<ServiceCard service={mockServiceWithAzure} />)

    // Should have both local and Azure URLs
    const links = screen.getAllByRole('link')
    expect(links).toHaveLength(2)
  })

  it('should show Azure URL section with proper styling', () => {
    const { container } = render(<ServiceCard service={mockServiceWithAzure} />)

    expect(screen.getByText('Azure URL')).toBeInTheDocument()
    // Check for Azure-specific styling classes
    expect(container.querySelector('.text-blue-400')).toBeInTheDocument()
  })
})
