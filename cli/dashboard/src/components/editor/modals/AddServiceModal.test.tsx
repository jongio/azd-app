/**
 * Add Service Modal Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AddServiceModal } from './AddServiceModal'
import * as wellknownApi from '@/lib/api/wellknown'

// Mock the wellknown API
vi.mock('@/lib/api/wellknown', () => ({
  fetchWellKnownServices: vi.fn(),
  getStubWellKnownServices: vi.fn(),
}))

describe('AddServiceModal', () => {
  const mockOnClose = vi.fn()
  const mockOnAddService = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    
    // Setup default mock return value
    vi.mocked(wellknownApi.fetchWellKnownServices).mockResolvedValue([
      {
        name: 'azurite',
        displayName: 'Azurite Storage',
        description: 'Azure Storage emulator',
        category: 'storage',
        icon: '📦',
        host: 'containerapp',
        image: 'mcr.microsoft.com/azure-storage/azurite:latest',
        ports: ['10000:10000'],
        environment: {},
      },
      {
        name: 'redis',
        displayName: 'Redis Cache',
        description: 'In-memory cache',
        category: 'cache',
        icon: '🔴',
        host: 'containerapp',
        image: 'redis:7-alpine',
        ports: ['6379:6379'],
      },
    ])
  })

  it('renders when isOpen is true', () => {
    render(
      <AddServiceModal
        isOpen={true}
        onClose={mockOnClose}
        onAddService={mockOnAddService}
      />
    )

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getAllByText('Add Service')).toHaveLength(2) // Title and button
  })

  it('does not render when isOpen is false', () => {
    render(
      <AddServiceModal
        isOpen={false}
        onClose={mockOnClose}
        onAddService={mockOnAddService}
      />
    )

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('renders all three tabs', async () => {
    render(
      <AddServiceModal
        isOpen={true}
        onClose={mockOnClose}
        onAddService={mockOnAddService}
      />
    )

    expect(screen.getByRole('button', { name: 'Well-Known Services' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Application Service' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Container Service' })).toBeInTheDocument()
  })

  it('defaults to well-known services tab', async () => {
    render(
      <AddServiceModal
        isOpen={true}
        onClose={mockOnClose}
        onAddService={mockOnAddService}
      />
    )

    // Wait for services to load
    await waitFor(() => {
      expect(screen.getByText('Azurite Storage')).toBeInTheDocument()
    })
  })

  it('switches between tabs', async () => {
    const user = userEvent.setup()
    render(
      <AddServiceModal
        isOpen={true}
        onClose={mockOnClose}
        onAddService={mockOnAddService}
      />
    )

    // Switch to Application Service tab
    const appServiceTab = screen.getByRole('button', { name: 'Application Service' })
    await user.click(appServiceTab)

    expect(screen.getByLabelText(/Service Name/)).toBeInTheDocument()
    expect(screen.getByLabelText(/Project Path/)).toBeInTheDocument()

    // Switch to Container Service tab
    const containerTab = screen.getByRole('button', { name: 'Container Service' })
    await user.click(containerTab)

    expect(screen.getByLabelText(/Docker Image/)).toBeInTheDocument()
  })

  it('adds well-known service when selected and confirmed', async () => {
    const user = userEvent.setup()
    mockOnAddService.mockResolvedValue(undefined)

    render(
      <AddServiceModal
        isOpen={true}
        onClose={mockOnClose}
        onAddService={mockOnAddService}
      />
    )

    // Wait for services to load
    await waitFor(() => {
      expect(screen.getByText('Azurite Storage')).toBeInTheDocument()
    })

    // Select a service
    const azuriteCard = screen.getByText('Azurite Storage').closest('button')
    expect(azuriteCard).toBeInTheDocument()
    await user.click(azuriteCard!)

    // Click Add Service button
    const addButton = screen.getByRole('button', { name: 'Add Service' })
    await user.click(addButton)

    expect(mockOnAddService).toHaveBeenCalledWith({
      name: 'azurite',
      host: 'containerapp',
      image: 'mcr.microsoft.com/azure-storage/azurite:latest',
      ports: ['10000:10000'],
      environment: {},
      healthcheck: undefined,
    })

    await waitFor(() => {
      expect(mockOnClose).toHaveBeenCalled()
    })
  })

  it('prevents adding duplicate service names', async () => {
    const user = userEvent.setup()
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})

    render(
      <AddServiceModal
        isOpen={true}
        onClose={mockOnClose}
        onAddService={mockOnAddService}
        existingServiceNames={['azurite']}
      />
    )

    // Wait for services to load
    await waitFor(() => {
      expect(screen.getByText('Azurite Storage')).toBeInTheDocument()
    })

    // Select azurite (which already exists)
    const azuriteCard = screen.getByText('Azurite Storage').closest('button')
    await user.click(azuriteCard!)

    // Try to add
    const addButton = screen.getByRole('button', { name: 'Add Service' })
    await user.click(addButton)

    expect(alertSpy).toHaveBeenCalledWith(
      expect.stringContaining('A service named "azurite" already exists')
    )
    expect(mockOnAddService).not.toHaveBeenCalled()
    expect(mockOnClose).not.toHaveBeenCalled()

    alertSpy.mockRestore()
  })

  it('adds application service with valid data', async () => {
    const user = userEvent.setup()
    mockOnAddService.mockResolvedValue(undefined)

    render(
      <AddServiceModal
        isOpen={true}
        onClose={mockOnClose}
        onAddService={mockOnAddService}
      />
    )

    // Switch to Application Service tab
    const appServiceTab = screen.getByRole('button', { name: 'Application Service' })
    await user.click(appServiceTab)

    // Fill in form
    const nameInput = screen.getByLabelText(/Service Name/)
    await user.type(nameInput, 'my-api')

    const projectInput = screen.getByLabelText(/Project Path/)
    await user.type(projectInput, './src/api')

    // Submit
    const submitButton = screen.getByRole('button', { name: 'Add Service' })
    await user.click(submitButton)

    expect(mockOnAddService).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'my-api',
        host: 'containerapp',
        project: './src/api',
      })
    )

    await waitFor(() => {
      expect(mockOnClose).toHaveBeenCalled()
    })
  })

  it('validates application service name format', async () => {
    const user = userEvent.setup()

    render(
      <AddServiceModal
        isOpen={true}
        onClose={mockOnClose}
        onAddService={mockOnAddService}
      />
    )

    // Switch to Application Service tab
    const appServiceTab = screen.getByRole('button', { name: 'Application Service' })
    await user.click(appServiceTab)

    // Enter invalid name (uppercase)
    const nameInput = screen.getByLabelText(/Service Name/)
    await user.type(nameInput, 'MyAPI')

    const projectInput = screen.getByLabelText(/Project Path/)
    await user.type(projectInput, './src/api')

    // Submit
    const submitButton = screen.getByRole('button', { name: 'Add Service' })
    await user.click(submitButton)

    // Should show validation error
    await waitFor(() => {
      expect(screen.getByText(/must contain only lowercase/i)).toBeInTheDocument()
    })

    expect(mockOnAddService).not.toHaveBeenCalled()
  })

  it('adds container service with valid data', async () => {
    const user = userEvent.setup()
    mockOnAddService.mockResolvedValue(undefined)

    render(
      <AddServiceModal
        isOpen={true}
        onClose={mockOnClose}
        onAddService={mockOnAddService}
      />
    )

    // Switch to Container Service tab
    const containerTab = screen.getByRole('button', { name: 'Container Service' })
    await user.click(containerTab)

    // Fill in form
    const nameInput = screen.getByLabelText(/Service Name/)
    await user.type(nameInput, 'my-nginx')

    const imageInput = screen.getByLabelText(/Docker Image/)
    await user.type(imageInput, 'nginx:alpine')

    const portsInput = screen.getByLabelText(/Port Mappings/)
    await user.type(portsInput, '80:80, 443:443')

    // Submit
    const submitButton = screen.getByRole('button', { name: 'Add Service' })
    await user.click(submitButton)

    expect(mockOnAddService).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'my-nginx',
        host: 'containerapp',
        image: 'nginx:alpine',
        ports: ['80:80', '443:443'],
      })
    )

    await waitFor(() => {
      expect(mockOnClose).toHaveBeenCalled()
    })
  })

  it('closes when cancel button is clicked', async () => {
    const user = userEvent.setup()

    render(
      <AddServiceModal
        isOpen={true}
        onClose={mockOnClose}
        onAddService={mockOnAddService}
      />
    )

    // Wait for modal to load
    await waitFor(() => {
      expect(screen.getByText('Cancel')).toBeInTheDocument()
    })

    const cancelButton = screen.getByRole('button', { name: 'Cancel' })
    await user.click(cancelButton)

    expect(mockOnClose).toHaveBeenCalled()
  })

  it('resets state when modal closes and reopens', async () => {
    const user = userEvent.setup()
    const { rerender } = render(
      <AddServiceModal
        isOpen={true}
        onClose={mockOnClose}
        onAddService={mockOnAddService}
      />
    )

    // Wait for services to load
    await waitFor(() => {
      expect(screen.getByText('Azurite Storage')).toBeInTheDocument()
    })

    // Select a service
    const azuriteCard = screen.getByText('Azurite Storage').closest('button')
    await user.click(azuriteCard!)

    // Close modal
    rerender(
      <AddServiceModal
        isOpen={false}
        onClose={mockOnClose}
        onAddService={mockOnAddService}
      />
    )

    // Reopen modal
    rerender(
      <AddServiceModal
        isOpen={true}
        onClose={mockOnClose}
        onAddService={mockOnAddService}
      />
    )

    // Wait for services to load again
    await waitFor(() => {
      expect(screen.getByText('Azurite Storage')).toBeInTheDocument()
    })

    // Check that selection was reset
    const selectedText = screen.queryByText(/Selected: Azurite Storage/)
    expect(selectedText).not.toBeInTheDocument()
  })

  it('handles API error when adding service', async () => {
    const user = userEvent.setup()
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})
    const error = new Error('Network error')
    mockOnAddService.mockRejectedValue(error)

    render(
      <AddServiceModal
        isOpen={true}
        onClose={mockOnClose}
        onAddService={mockOnAddService}
      />
    )

    // Wait for services to load
    await waitFor(() => {
      expect(screen.getByText('Azurite Storage')).toBeInTheDocument()
    })

    // Select and try to add
    const azuriteCard = screen.getByText('Azurite Storage').closest('button')
    await user.click(azuriteCard!)

    const addButton = screen.getByRole('button', { name: 'Add Service' })
    await user.click(addButton)

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('Failed to add service. Please try again.')
    })

    expect(mockOnClose).not.toHaveBeenCalled()

    alertSpy.mockRestore()
  })
})
