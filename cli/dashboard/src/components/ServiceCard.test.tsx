import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ServiceCard } from './ServiceCard'
import type { Service, HealthCheckResult } from '@/types'

// Mock hooks
const getEffectiveOperationState = vi.fn(() => 'idle')
const canPerformAction = vi.fn(() => true)
const isOperationInProgress = vi.fn(() => false)
const getOperationState = vi.fn(() => 'idle')

vi.mock('@/hooks/useServiceOperations', () => ({
  useServiceOperations: () => ({
    getEffectiveOperationState,
    canPerformAction,
    isOperationInProgress,
    getOperationState,
    startService: vi.fn(),
    stopService: vi.fn(),
    restartService: vi.fn(),
    error: null,
  }),
}))

const mockCodespaceConfig = null
vi.mock('@/hooks/useCodespaceEnv', () => ({
  useCodespaceEnv: () => ({
    config: mockCodespaceConfig,
  }),
}))

describe('ServiceCard - Alternate URL Display', () => {
  const baseService: Service = {
    name: 'web',
    host: 'local',
    language: 'typescript',
    framework: 'react',
    local: {
      status: 'ready',
      health: 'healthy',
      port: 3000,
      url: 'http://localhost:3000',
    },
  }

  const healthStatus: HealthCheckResult = {
    serviceName: 'web',
    status: 'healthy',
    checkType: 'http',
    endpoint: 'http://localhost:3000',
    statusCode: 200,
    responseTime: 50_000_000, // 50ms in nanoseconds
    uptime: 3600_000_000_000, // 1 hour in nanoseconds
    timestamp: new Date().toISOString(),
  }

  it('displays local URL when no alternate URL is configured', () => {
    render(<ServiceCard service={baseService} healthStatus={healthStatus} />)

    const urlLink = screen.getByRole('link', { name: /localhost:3000/i })
    expect(urlLink).toBeInTheDocument()
    expect(urlLink).toHaveAttribute('href', 'http://localhost:3000')
    expect(screen.queryByText('Custom URL')).not.toBeInTheDocument()
  })

  it('displays custom URL when configured', () => {
    const serviceWithAltUrl: Service = {
      ...baseService,
      azure: {
        url: 'https://default.azurewebsites.net',
        customUrl: 'https://myapp.example.com',
      },
    }

    render(<ServiceCard service={serviceWithAltUrl} healthStatus={healthStatus} />)

    expect(screen.getByText('Custom URL')).toBeInTheDocument()
    const links = screen.getAllByRole('link')
    const customUrlLink = links.find(link => link.getAttribute('href') === 'https://myapp.example.com')
    expect(customUrlLink).toBeInTheDocument()
  })

  it('navigates to custom URL when clicking the link', () => {
    const serviceWithAltUrl: Service = {
      ...baseService,
      azure: {
        url: 'https://api-default.azurewebsites.net',
        customUrl: 'https://api.myapp.example.com',
      },
    }

    render(<ServiceCard service={serviceWithAltUrl} healthStatus={healthStatus} />)

    const links = screen.getAllByRole('link')
    const customUrlLink = links.find(link => link.getAttribute('href') === 'https://api.myapp.example.com')
    expect(customUrlLink).toHaveAttribute('target', '_blank')
    expect(customUrlLink).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('shows purple visual indicator for custom URL', () => {
    const serviceWithAltUrl: Service = {
      ...baseService,
      azure: {
        url: 'https://default.azurewebsites.net',
        customUrl: 'https://myapp.example.com',
      },
    }

    render(<ServiceCard service={serviceWithAltUrl} healthStatus={healthStatus} />)

    const links = screen.getAllByRole('link')
    const customUrlLink = links.find(link => link.getAttribute('href') === 'https://myapp.example.com')
    expect(customUrlLink).toHaveClass('bg-purple-50')
  })

  it('shows tooltip explaining custom URL configuration', () => {
    const serviceWithAltUrl: Service = {
      ...baseService,
      local: {
        url: 'http://localhost:3000',
        customUrl: 'https://myapp.example.com',
        status: 'ready',
        health: 'healthy',
      },
    }

    render(<ServiceCard service={serviceWithAltUrl} healthStatus={healthStatus} />)

    const links = screen.getAllByRole('link')
    const customUrlLink = links.find(link => link.getAttribute('href') === 'https://myapp.example.com')
    expect(customUrlLink).toHaveAttribute('title', 'Custom URL configured (default: http://localhost:3000)')
  })

  it('handles service with custom URL but no local URL', () => {
    const serviceWithAltUrlOnly: Service = {
      name: 'api',
      host: 'local',
      language: 'python',
      azure: {
        url: 'https://api-default.azurewebsites.net',
        customUrl: 'https://api.example.com',
      },
    }

    render(<ServiceCard service={serviceWithAltUrlOnly} />)

    const links = screen.getAllByRole('link')
    const customUrlLink = links.find(link => link.getAttribute('href') === 'https://api.example.com')
    expect(customUrlLink).toBeInTheDocument()
    expect(screen.getByText('Custom URL')).toBeInTheDocument()
  })

  it('displays custom URL when configured in Azure', () => {
    const serviceWithCustomUrl: Service = {
      ...baseService,
      azure: {
        url: 'https://myapp-abc123.azurewebsites.net',
        customUrl: 'https://myapp.example.com',
        resourceName: 'myapp-abc123',
        resourceType: 'appservice',
      },
    }

    render(<ServiceCard service={serviceWithCustomUrl} healthStatus={healthStatus} />)

    // Custom URL should be displayed with indicator
    expect(screen.getByText('Custom URL')).toBeInTheDocument()
    const links = screen.getAllByRole('link')
    const customUrlLink = links.find(link => link.getAttribute('href') === 'https://myapp.example.com')
    expect(customUrlLink).toBeInTheDocument()
  })

  it('maintains backward compatibility for services without altUrl', () => {
    const legacyService: Service = {
      name: 'legacy-api',
      host: 'local',
      local: {
        status: 'ready',
        health: 'healthy',
        port: 8080,
        url: 'http://localhost:8080',
      },
    }

    render(<ServiceCard service={legacyService} />)

    const urlLink = screen.getByRole('link', { name: /localhost:8080/i })
    expect(urlLink).toBeInTheDocument()
    expect(urlLink).toHaveAttribute('href', 'http://localhost:8080')
    expect(screen.queryByText('Alternate URL')).not.toBeInTheDocument()
  })

  it('handles empty url gracefully', () => {
    const serviceWithEmptyAltUrl: Service = {
      ...baseService,
      azure: {
        url: '',
      },
    }

    render(<ServiceCard service={serviceWithEmptyAltUrl} healthStatus={healthStatus} />)

    // Should fall back to local URL since url is empty
    const urlLink = screen.getByRole('link', { name: /localhost:3000/i })
    expect(urlLink).toBeInTheDocument()
    expect(urlLink).toHaveAttribute('href', 'http://localhost:3000')
  })

  it('displays service card with all key elements when custom URL is configured', () => {
    const serviceWithAltUrl: Service = {
      ...baseService,
      local: {
        url: 'http://localhost:3000',
        customUrl: 'https://myapp.example.com',
        status: 'ready',
        health: 'healthy',
      },
    }

    render(<ServiceCard service={serviceWithAltUrl} healthStatus={healthStatus} />)

    // Service name
    expect(screen.getByText('web')).toBeInTheDocument()
    
    // Language/framework
    expect(screen.getByText(/typescript/i)).toBeInTheDocument()
    expect(screen.getByText(/react/i)).toBeInTheDocument()

    // Custom URL indicator
    expect(screen.getByText('Custom URL')).toBeInTheDocument()
    
    // Health metrics should still be visible
    expect(screen.getByText(/Response/i)).toBeInTheDocument()
    expect(screen.getByText(/Uptime/i)).toBeInTheDocument()
  })

  it('does not break UI when custom URL is unreachable', () => {
    // Note: UI doesn't validate URL reachability, just displays it
    const serviceWithUnreachableAltUrl: Service = {
      ...baseService,
      azure: {
        url: 'https://unreachable-service.example.com',
      },
    }

    render(<ServiceCard service={serviceWithUnreachableAltUrl} healthStatus={healthStatus} />)

    // URL should still be displayed and clickable
    const links = screen.getAllByRole('link')
    const customUrlLink = links.find(link => link.getAttribute('href') === 'https://unreachable-service.example.com')
    expect(customUrlLink).toBeInTheDocument()
  })

  it('prefers custom URL over local URL for all service types', () => {
    const processService: Service = {
      name: 'build-process',
      host: 'local',
      local: {
        url: 'http://localhost:9999',
        customUrl: 'https://build.example.com',
        status: 'ready',
        health: 'healthy',
        serviceType: 'process',
        serviceMode: 'watch',
      },
    }

    render(<ServiceCard service={processService} />)

    const links = screen.getAllByRole('link')
    const customUrlLink = links.find(link => link.getAttribute('href') === 'https://build.example.com')
    expect(customUrlLink).toBeInTheDocument()
    expect(screen.getByText('Custom URL')).toBeInTheDocument()
  })
})
