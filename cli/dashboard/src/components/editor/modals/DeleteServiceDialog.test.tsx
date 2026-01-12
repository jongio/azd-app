/**
 * Delete Service Dialog Tests
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DeleteServiceDialog } from './DeleteServiceDialog'

describe('DeleteServiceDialog', () => {
  const mockOnClose = vi.fn()
  const mockOnConfirm = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders when isOpen is true', () => {
    render(
      <DeleteServiceDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        serviceName="test-service"
      />
    )

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getAllByText('Delete Service')).toHaveLength(2) // Title and button
  })

  it('does not render when isOpen is false', () => {
    render(
      <DeleteServiceDialog
        isOpen={false}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        serviceName="test-service"
      />
    )

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('displays service name in confirmation message', () => {
    render(
      <DeleteServiceDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        serviceName="my-api"
      />
    )

    expect(screen.getByText(/Delete "my-api"\?/)).toBeInTheDocument()
  })

  it('shows warning message', () => {
    render(
      <DeleteServiceDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        serviceName="test-service"
      />
    )

    expect(screen.getAllByText(/This action cannot be undone/i)).toHaveLength(2) // Description and warning
    expect(screen.getByText(/Warning:/i)).toBeInTheDocument()
  })

  it('calls onClose when cancel button is clicked', async () => {
    const user = userEvent.setup()

    render(
      <DeleteServiceDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        serviceName="test-service"
      />
    )

    const cancelButton = screen.getByRole('button', { name: 'Cancel' })
    await user.click(cancelButton)

    expect(mockOnClose).toHaveBeenCalledTimes(1)
    expect(mockOnConfirm).not.toHaveBeenCalled()
  })

  it('calls onConfirm and onClose when delete button is clicked', async () => {
    const user = userEvent.setup()
    mockOnConfirm.mockResolvedValue(undefined)

    render(
      <DeleteServiceDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        serviceName="test-service"
      />
    )

    const deleteButton = screen.getByRole('button', { name: 'Delete Service' })
    await user.click(deleteButton)

    expect(mockOnConfirm).toHaveBeenCalledTimes(1)

    await waitFor(() => {
      expect(mockOnClose).toHaveBeenCalledTimes(1)
    })
  })

  it('disables buttons when isDeleting is true', () => {
    render(
      <DeleteServiceDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        serviceName="test-service"
        isDeleting={true}
      />
    )

    const cancelButton = screen.getByRole('button', { name: 'Cancel' })
    const deleteButton = screen.getByRole('button', { name: /Deleting/i })

    expect(cancelButton).toBeDisabled()
    expect(deleteButton).toBeDisabled()
  })

  it('shows "Deleting..." text when isDeleting is true', () => {
    render(
      <DeleteServiceDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        serviceName="test-service"
        isDeleting={true}
      />
    )

    expect(screen.getByRole('button', { name: 'Deleting...' })).toBeInTheDocument()
  })

  it('handles error during delete', async () => {
    const user = userEvent.setup()
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})
    const error = new Error('Delete failed')
    mockOnConfirm.mockRejectedValue(error)

    render(
      <DeleteServiceDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        serviceName="test-service"
      />
    )

    const deleteButton = screen.getByRole('button', { name: 'Delete Service' })
    await user.click(deleteButton)

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('Failed to delete service. Please try again.')
    })

    expect(mockOnClose).not.toHaveBeenCalled()

    alertSpy.mockRestore()
  })

  it('renders warning icon', () => {
    render(
      <DeleteServiceDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        serviceName="test-service"
      />
    )

    // Check for the warning icon container
    const iconContainer = document.querySelector('.bg-red-100')
    expect(iconContainer).toBeInTheDocument()
  })

  it('closes when Escape key is pressed', async () => {
    const user = userEvent.setup()

    render(
      <DeleteServiceDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        serviceName="test-service"
      />
    )

    await user.keyboard('{Escape}')

    expect(mockOnClose).toHaveBeenCalledTimes(1)
  })

  it('has proper ARIA attributes', () => {
    render(
      <DeleteServiceDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        serviceName="test-service"
      />
    )

    const dialog = screen.getByRole('dialog')
    // ARIA attributes are set through Dialog component
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toBeInTheDocument()
  })
})
