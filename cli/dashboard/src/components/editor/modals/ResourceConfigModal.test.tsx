/**
 * Resource Configuration Modal Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ResourceConfigModal } from './ResourceConfigModal'
import type { ResourceConfig } from '@/lib/editor/resource-types'

describe('ResourceConfigModal', () => {
  const mockOnClose = vi.fn()
  const mockOnSave = vi.fn()

  beforeEach(() => {
    mockOnClose.mockClear()
    mockOnSave.mockClear()
  })

  describe('rendering', () => {
    it('should render modal when open', () => {
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      expect(screen.getByRole('heading', { name: /Add Resource/i })).toBeInTheDocument()
      expect(screen.getByText(/Configure Azure resource/i)).toBeInTheDocument()
    })

    it('should not render modal when closed', () => {
      render(
        <ResourceConfigModal
          isOpen={false}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      expect(screen.queryByRole('heading', { name: /Add Resource/i })).not.toBeInTheDocument()
    })

    it('should show resource type selector initially', () => {
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      expect(screen.getByText('Resource Type')).toBeInTheDocument()
      expect(screen.getByPlaceholderText('Search resource types...')).toBeInTheDocument()
    })

    it('should show "Edit Resource" title when editing', () => {
      const initialConfig: ResourceConfig = {
        name: 'my-storage',
        type: 'Microsoft.Storage/storageAccounts',
        containers: ['uploads'],
      }

      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          initialConfig={initialConfig}
        />
      )

      expect(screen.getByRole('heading', { name: /Edit Resource/i })).toBeInTheDocument()
    })
  })

  describe('resource type selection', () => {
    it('should show resource types in grid', () => {
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      expect(screen.getByText('Storage Account')).toBeInTheDocument()
      expect(screen.getByText('Cosmos DB')).toBeInTheDocument()
      expect(screen.getByText('Event Hubs')).toBeInTheDocument()
      expect(screen.getByText('Service Bus')).toBeInTheDocument()
    })

    it('should filter resource types by category', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const storageButton = screen.getByRole('button', { name: 'Storage' })
      await user.click(storageButton)

      expect(screen.getByText('Storage Account')).toBeInTheDocument()
      expect(screen.queryByText('Cosmos DB')).not.toBeInTheDocument()
    })

    it('should filter resource types by search', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const searchInput = screen.getByPlaceholderText('Search resource types...')
      await user.type(searchInput, 'cosmos')

      expect(screen.getByText('Cosmos DB')).toBeInTheDocument()
      expect(screen.queryByText('Storage Account')).not.toBeInTheDocument()
    })

    it('should show templates after selecting type', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const storageType = screen.getByText('Storage Account').closest('button')
      await user.click(storageType!)

      await waitFor(() => {
        expect(screen.getByText('Blob Storage')).toBeInTheDocument()
      })
    })
  })

  describe('template selection', () => {
    it('should apply template configuration', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      // Select Storage Account type
      const storageType = screen.getByText('Storage Account').closest('button')
      await user.click(storageType!)

      // Select Blob Storage template
      await waitFor(() => {
        expect(screen.getByText('Blob Storage')).toBeInTheDocument()
      })

      const blobTemplate = screen.getByText('Blob Storage').closest('button')
      await user.click(blobTemplate!)

      // Should now show form with template defaults
      await waitFor(() => {
        expect(screen.getByLabelText(/Resource Name/i)).toBeInTheDocument()
      })
    })

    it('should allow skipping templates', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const storageType = screen.getByText('Storage Account').closest('button')
      await user.click(storageType!)

      await waitFor(() => {
        expect(screen.getByText(/Skip Templates/i)).toBeInTheDocument()
      })

      const skipButton = screen.getByText(/Skip Templates/i)
      await user.click(skipButton)

      await waitFor(() => {
        expect(screen.getByLabelText(/Resource Name/i)).toBeInTheDocument()
      })
    })
  })

  describe('form validation', () => {
    it('should require resource name', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      // Select type and skip template
      const storageType = screen.getByText('Storage Account').closest('button')
      await user.click(storageType!)

      await waitFor(() => {
        const skipButton = screen.getByText(/Skip Templates/i)
        return user.click(skipButton)
      })

      // Try to save without name
      await waitFor(() => {
        const saveButton = screen.getByRole('button', { name: /Add Resource/i })
        return user.click(saveButton)
      })

      await waitFor(() => {
        expect(screen.getByText(/Resource name is required/i)).toBeInTheDocument()
      })
      expect(mockOnSave).not.toHaveBeenCalled()
    })

    it('should validate resource name format', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const storageType = screen.getByText('Storage Account').closest('button')
      await user.click(storageType!)

      await waitFor(() => {
        const skipButton = screen.getByText(/Skip Templates/i)
        return user.click(skipButton)
      })

      await waitFor(() => {
        expect(screen.getByLabelText(/Resource Name/i)).toBeInTheDocument()
      })

      const nameInput = screen.getByLabelText(/Resource Name/i)
      await user.type(nameInput, 'Invalid Name!')

      const saveButton = screen.getByRole('button', { name: /Add Resource/i })
      await user.click(saveButton)

      await waitFor(() => {
        expect(screen.getByText(/lowercase letters, numbers, and hyphens/i)).toBeInTheDocument()
      })
      expect(mockOnSave).not.toHaveBeenCalled()
    })

    it('should prevent duplicate resource names', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          existingResourceNames={['existing-storage']}
        />
      )

      const storageType = screen.getByText('Storage Account').closest('button')
      await user.click(storageType!)

      await waitFor(() => {
        const skipButton = screen.getByText(/Skip Templates/i)
        return user.click(skipButton)
      })

      await waitFor(() => {
        expect(screen.getByLabelText(/Resource Name/i)).toBeInTheDocument()
      })

      const nameInput = screen.getByLabelText(/Resource Name/i)
      await user.type(nameInput, 'existing-storage')

      const saveButton = screen.getByRole('button', { name: /Add Resource/i })
      await user.click(saveButton)

      await waitFor(() => {
        expect(screen.getByText(/already exists/i)).toBeInTheDocument()
      })
      expect(mockOnSave).not.toHaveBeenCalled()
    })
  })

  describe('dependency management', () => {
    it('should allow adding dependencies', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          availableServices={['api', 'web']}
        />
      )

      const storageType = screen.getByText('Storage Account').closest('button')
      await user.click(storageType!)

      await waitFor(() => {
        const skipButton = screen.getByText(/Skip Templates/i)
        return user.click(skipButton)
      })

      await waitFor(() => {
        expect(screen.getByText('Add Dependency')).toBeInTheDocument()
      })

      const addButton = screen.getByText('Add Dependency')
      await user.click(addButton)

      await waitFor(() => {
        expect(screen.getByText('api')).toBeInTheDocument()
      })

      const apiOption = screen.getByText('api')
      await user.click(apiOption)

      await waitFor(() => {
        expect(screen.getByText(/No circular dependencies detected/i)).toBeInTheDocument()
      })
    })

    it('should detect circular dependencies', async () => {
      const user = userEvent.setup()
      
      // Create config where api uses storage, and we're trying to make storage use api
      const config = {
        services: {
          api: {
            host: 'containerapp',
            uses: ['storage'],
          },
        },
      }

      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          availableServices={['api']}
          currentConfig={config}
        />
      )

      const storageType = screen.getByText('Storage Account').closest('button')
      await user.click(storageType!)

      await waitFor(() => {
        const skipButton = screen.getByText(/Skip Templates/i)
        return user.click(skipButton)
      })

      await waitFor(() => {
        expect(screen.getByLabelText(/Resource Name/i)).toBeInTheDocument()
      })

      const nameInput = screen.getByLabelText(/Resource Name/i)
      await user.type(nameInput, 'storage')

      await waitFor(() => {
        const addButton = screen.getByText('Add Dependency')
        return user.click(addButton)
      })

      await waitFor(() => {
        const apiOption = screen.getByText('api')
        return user.click(apiOption)
      })

      await waitFor(() => {
        expect(screen.getByText(/Circular dependency detected/i)).toBeInTheDocument()
      })

      // Save button should be disabled
      const saveButton = screen.getByText('Add Resource')
      expect(saveButton).toBeDisabled()
    })

    it('should remove dependencies', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          availableServices={['api', 'web']}
        />
      )

      const storageType = screen.getByText('Storage Account').closest('button')
      await user.click(storageType!)

      await waitFor(() => {
        const skipButton = screen.getByText(/Skip Templates/i)
        return user.click(skipButton)
      })

      const addButton = screen.getByText('Add Dependency')
      await user.click(addButton)

      await waitFor(() => {
        const apiOption = screen.getByText('api')
        return user.click(apiOption)
      })

      // Find and click remove button
      await waitFor(() => {
        const removeButton = screen.getByLabelText('Remove api')
        return user.click(removeButton)
      })

      await waitFor(() => {
        expect(screen.queryByText(/No circular dependencies/i)).not.toBeInTheDocument()
      })
    })
  })

  describe('type-specific fields', () => {
    it('should show containers field for Storage Account', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const storageType = screen.getByText('Storage Account').closest('button')
      await user.click(storageType!)

      await waitFor(() => {
        const skipButton = screen.getByText(/Skip Templates/i)
        return user.click(skipButton)
      })

      await waitFor(() => {
        expect(screen.getByLabelText(/Containers/i)).toBeInTheDocument()
      })
    })

    it('should show hubs field for Event Hubs', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const eventHubType = screen.getByText('Event Hubs').closest('button')
      await user.click(eventHubType!)

      await waitFor(() => {
        const skipButton = screen.getByText(/Skip Templates/i)
        return user.click(skipButton)
      })

      await waitFor(() => {
        expect(screen.getByLabelText(/Event Hubs/i)).toBeInTheDocument()
      })
    })

    it('should show queues and topics fields for Service Bus', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const serviceBusType = screen.getByText('Service Bus').closest('button')
      await user.click(serviceBusType!)

      await waitFor(() => {
        const skipButton = screen.getByText(/Skip Templates/i)
        return user.click(skipButton)
      })

      await waitFor(() => {
        expect(screen.getByLabelText(/Queues/i)).toBeInTheDocument()
        expect(screen.getByLabelText(/Topics/i)).toBeInTheDocument()
      })
    })
  })

  describe('form submission', () => {
    it('should save new resource configuration', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const storageType = screen.getByText('Storage Account').closest('button')
      await user.click(storageType!)

      await waitFor(() => {
        const skipButton = screen.getByText(/Skip Templates/i)
        return user.click(skipButton)
      })

      await waitFor(() => {
        expect(screen.getByLabelText(/Resource Name/i)).toBeInTheDocument()
      })

      const nameInput = screen.getByLabelText(/Resource Name/i)
      await user.type(nameInput, 'my-storage')

      const containersInput = screen.getByLabelText(/Containers/i)
      await user.type(containersInput, 'uploads, static')

      const saveButton = screen.getByRole('button', { name: /Add Resource/i })
      await user.click(saveButton)

      await waitFor(() => {
        expect(mockOnSave).toHaveBeenCalledWith({
          name: 'my-storage',
          type: 'Microsoft.Storage/storageAccounts',
          containers: ['uploads', 'static'],
        })
      })
      expect(mockOnClose).toHaveBeenCalled()
    })

    it('should save resource with dependencies', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          availableServices={['api']}
        />
      )

      const storageType = screen.getByText('Storage Account').closest('button')
      await user.click(storageType!)

      await waitFor(() => {
        const skipButton = screen.getByText(/Skip Templates/i)
        return user.click(skipButton)
      })

      await waitFor(() => {
        expect(screen.getByLabelText(/Resource Name/i)).toBeInTheDocument()
      })

      const nameInput = screen.getByLabelText(/Resource Name/i)
      await user.type(nameInput, 'my-storage')

      const addButton = screen.getByText('Add Dependency')
      await user.click(addButton)

      await waitFor(() => {
        const apiOption = screen.getByText('api')
        return user.click(apiOption)
      })

      const saveButton = screen.getByRole('button', { name: /Add Resource/i })
      await user.click(saveButton)

      await waitFor(() => {
        expect(mockOnSave).toHaveBeenCalledWith({
          name: 'my-storage',
          type: 'Microsoft.Storage/storageAccounts',
          uses: ['api'],
        })
      })
    })

    it('should save existing resource flag', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const storageType = screen.getByText('Storage Account').closest('button')
      await user.click(storageType!)

      await waitFor(() => {
        const skipButton = screen.getByText(/Skip Templates/i)
        return user.click(skipButton)
      })

      await waitFor(() => {
        expect(screen.getByLabelText(/Resource Name/i)).toBeInTheDocument()
      })

      const nameInput = screen.getByLabelText(/Resource Name/i)
      await user.type(nameInput, 'my-storage')

      const existingCheckbox = screen.getByLabelText(/pre-existing resource/i)
      await user.click(existingCheckbox)

      const saveButton = screen.getByRole('button', { name: /Add Resource/i })
      await user.click(saveButton)

      await waitFor(() => {
        expect(mockOnSave).toHaveBeenCalledWith({
          name: 'my-storage',
          type: 'Microsoft.Storage/storageAccounts',
          existing: true,
        })
      })
    })
  })

  describe('editing mode', () => {
    it('should load existing configuration', () => {
      const initialConfig: ResourceConfig = {
        name: 'my-storage',
        type: 'Microsoft.Storage/storageAccounts',
        containers: ['uploads', 'static'],
        uses: ['api'],
        existing: true,
      }

      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          initialConfig={initialConfig}
          availableServices={['api', 'web']}
        />
      )

      expect(screen.getByDisplayValue('my-storage')).toBeInTheDocument()
      expect(screen.getByText('api')).toBeInTheDocument()
      expect(screen.getByLabelText(/pre-existing resource/i)).toBeChecked()
    })

    it('should disable name field when editing', () => {
      const initialConfig: ResourceConfig = {
        name: 'my-storage',
        type: 'Microsoft.Storage/storageAccounts',
      }

      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          initialConfig={initialConfig}
        />
      )

      const nameInput = screen.getByDisplayValue('my-storage')
      expect(nameInput).toBeDisabled()
    })

    it('should show Save Resource button when editing', () => {
      const initialConfig: ResourceConfig = {
        name: 'my-storage',
        type: 'Microsoft.Storage/storageAccounts',
      }

      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
          initialConfig={initialConfig}
        />
      )

      expect(screen.getByRole('button', { name: /Save Resource/i })).toBeInTheDocument()
    })
  })

  describe('user interactions', () => {
    it('should close modal on cancel', async () => {
      const user = userEvent.setup()
      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const cancelButton = screen.getByText('Cancel')
      await user.click(cancelButton)

      expect(mockOnClose).toHaveBeenCalled()
      expect(mockOnSave).not.toHaveBeenCalled()
    })

    it('should disable submit button while submitting', async () => {
      const user = userEvent.setup()
      mockOnSave.mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)))

      render(
        <ResourceConfigModal
          isOpen={true}
          onClose={mockOnClose}
          onSave={mockOnSave}
        />
      )

      const storageType = screen.getByText('Storage Account').closest('button')
      await user.click(storageType!)

      await waitFor(() => {
        const skipButton = screen.getByText(/Skip Templates/i)
        return user.click(skipButton)
      })

      await waitFor(() => {
        expect(screen.getByLabelText(/Resource Name/i)).toBeInTheDocument()
      })

      const nameInput = screen.getByLabelText(/Resource Name/i)
      await user.type(nameInput, 'my-storage')

      const saveButton = screen.getByRole('button', { name: /Add Resource/i })
      await user.click(saveButton)

      await waitFor(() => {
        expect(screen.getByText('Saving...')).toBeInTheDocument()
      })
      expect(saveButton).toBeDisabled()
    })
  })
})
