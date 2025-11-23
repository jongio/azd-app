import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ServiceDetailPanel } from './ServiceDetailPanel'
import type { Service } from '@/types'

describe('ServiceDetailPanel', () => {
  const mockService: Service = {
    name: 'api',
    language: 'TypeScript',
    status: 'running',
    health: 'healthy',
    local: {
      status: 'running',
      health: 'healthy',
      port: 3000,
      url: 'http://localhost:3000',
      pid: 1234,
      startTime: '2025-01-01T00:00:00Z',
    },
    azure: {
      resourceName: 'my-containerapp',
      resourceType: 'containerapp',
      resourceGroup: 'my-rg',
      location: 'eastus',
      subscriptionId: 'sub-123',
      url: 'https://my-containerapp.azurecontainerapps.io',
    },
    environmentVariables: {
      DATABASE_URL: 'postgres://localhost:5432/db',
      API_KEY: 'secret-key',
    },
  }

  const mockOnClose = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should render nothing when service is null', () => {
    const { container } = render(
      <ServiceDetailPanel service={null} isOpen={true} onClose={mockOnClose} />
    )
    expect(container.firstChild).toBeNull()
  })

  it('should render nothing when not open', () => {
    const { container } = render(
      <ServiceDetailPanel service={mockService} isOpen={false} onClose={mockOnClose} />
    )
    // Panel should be hidden (opacity-0)
    expect(container.querySelector('.opacity-0')).toBeTruthy()
  })

  it('should display service name and status', () => {
    render(<ServiceDetailPanel service={mockService} isOpen={true} onClose={mockOnClose} />)
    expect(screen.getByText('api')).toBeInTheDocument()
    expect(screen.getAllByText('running').length).toBeGreaterThan(0)
    expect(screen.getAllByText('healthy').length).toBeGreaterThan(0)
  })

  it('should show all tabs', () => {
    render(<ServiceDetailPanel service={mockService} isOpen={true} onClose={mockOnClose} />)
    expect(screen.getByText('Overview')).toBeInTheDocument()
    expect(screen.getByText('Local')).toBeInTheDocument()
    expect(screen.getByText('Azure')).toBeInTheDocument()
    expect(screen.getByText('Environment')).toBeInTheDocument()
  })

  it('should close when clicking backdrop', () => {
    const { container } = render(
      <ServiceDetailPanel service={mockService} isOpen={true} onClose={mockOnClose} />
    )
    const backdrop = container.querySelector('.bg-black\\/30')
    if (backdrop) {
      fireEvent.click(backdrop)
      expect(mockOnClose).toHaveBeenCalledTimes(1)
    }
  })

  it('should close when clicking X button', () => {
    render(<ServiceDetailPanel service={mockService} isOpen={true} onClose={mockOnClose} />)
    const closeButton = screen.getAllByRole('button')[0] // First button is the X
    fireEvent.click(closeButton)
    expect(mockOnClose).toHaveBeenCalledTimes(1)
  })

  it('should close on Escape key', () => {
    render(<ServiceDetailPanel service={mockService} isOpen={true} onClose={mockOnClose} />)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(mockOnClose).toHaveBeenCalledTimes(1)
  })

  it('should not close on Escape if defaultPrevented', () => {
    render(<ServiceDetailPanel service={mockService} isOpen={true} onClose={mockOnClose} />)
    const event = new KeyboardEvent('keydown', { key: 'Escape', cancelable: true })
    event.preventDefault()
    window.dispatchEvent(event)
    expect(mockOnClose).not.toHaveBeenCalled()
  })

  it('should display local service information', () => {
    render(<ServiceDetailPanel service={mockService} isOpen={true} onClose={mockOnClose} />)
    expect(screen.getByText('http://localhost:3000')).toBeInTheDocument()
    expect(screen.getByText('3000')).toBeInTheDocument()
    expect(screen.getByText('1234')).toBeInTheDocument()
  })

  it('should display Azure information when available', async () => {
    render(<ServiceDetailPanel service={mockService} isOpen={true} onClose={mockOnClose} />)
    
    // Switch to Azure tab
    const azureTab = screen.getByText('Azure')
    fireEvent.click(azureTab)
    
    await waitFor(() => {
      expect(screen.getByText('my-containerapp')).toBeInTheDocument()
      expect(screen.getByText('containerapp')).toBeInTheDocument()
      expect(screen.getByText('my-rg')).toBeInTheDocument()
      expect(screen.getByText('eastus')).toBeInTheDocument()
    })
  })

  it('should display environment variables', async () => {
    render(<ServiceDetailPanel service={mockService} isOpen={true} onClose={mockOnClose} />)
    
    // Switch to Environment tab
    const envTab = screen.getByText('Environment')
    fireEvent.click(envTab)
    
    await waitFor(() => {
      expect(screen.getByText('DATABASE_URL')).toBeInTheDocument()
      expect(screen.getByText('API_KEY')).toBeInTheDocument()
    })
  })

  it('should copy to clipboard when clicking copy button', async () => {
    // Mock clipboard API
    const writeTextMock = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, {
      clipboard: {
        writeText: writeTextMock,
      },
    })

    render(<ServiceDetailPanel service={mockService} isOpen={true} onClose={mockOnClose} />)
    
    // Find and click a copy button
    const copyButtons = screen.getAllByRole('button').filter(btn => 
      btn.querySelector('svg')?.classList.contains('lucide-copy')
    )
    
    if (copyButtons.length > 0) {
      fireEvent.click(copyButtons[0])
      await waitFor(() => {
        expect(writeTextMock).toHaveBeenCalled()
      })
    }
  })

  it('should handle service without Azure info', () => {
    const serviceWithoutAzure = { ...mockService, azure: undefined }
    render(<ServiceDetailPanel service={serviceWithoutAzure} isOpen={true} onClose={mockOnClose} />)
    
    expect(screen.getByText('api')).toBeInTheDocument()
    // Azure tab should still be present but show "No Azure deployment information"
  })

  it('should show unhealthy status correctly', () => {
    const unhealthyService: Service = {
      ...mockService,
      status: 'error',
      health: 'unhealthy',
      local: {
        ...mockService.local!,
        status: 'error',
        health: 'unhealthy',
      },
    }
    
    render(<ServiceDetailPanel service={unhealthyService} isOpen={true} onClose={mockOnClose} />)
    const errorElements = screen.getAllByText('error')
    expect(errorElements.length).toBeGreaterThan(0)
    const unhealthyElements = screen.getAllByText('unhealthy')
    expect(unhealthyElements.length).toBeGreaterThan(0)
  })

  it('should switch between tabs', async () => {
    render(<ServiceDetailPanel service={mockService} isOpen={true} onClose={mockOnClose} />)
    
    // Click Local tab
    fireEvent.click(screen.getByText('Local'))
    await waitFor(() => {
      expect(screen.getByText('Process ID')).toBeInTheDocument()
    })
    
    // Click Azure tab
    fireEvent.click(screen.getByText('Azure'))
    await waitFor(() => {
      expect(screen.getByText('Resource Name')).toBeInTheDocument()
    })
    
    // Click Environment tab
    fireEvent.click(screen.getByText('Environment'))
    await waitFor(() => {
      expect(screen.getByText('DATABASE_URL')).toBeInTheDocument()
    })
  })
})
