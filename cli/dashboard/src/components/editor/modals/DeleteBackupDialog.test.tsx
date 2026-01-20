/**
 * Delete Backup Dialog Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DeleteBackupDialog } from './DeleteBackupDialog'

describe('DeleteBackupDialog', () => {
  const mockOnClose = vi.fn()
  const mockOnConfirm = vi.fn()
  const mockTimestamp = '2026-01-11T14:30:00Z'

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders when isOpen is true', () => {
    render(
      <DeleteBackupDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText('Delete Backup')).toBeInTheDocument()
    expect(screen.getAllByText(/This action cannot be undone/)).toHaveLength(2) // Description and warning
  })

  it('does not render when isOpen is false', () => {
    render(
      <DeleteBackupDialog
        isOpen={false}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('displays formatted timestamp in heading', () => {
    render(
      <DeleteBackupDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    expect(screen.getByText(/Delete backup from Jan 11, 2026/)).toBeInTheDocument()
  })

  it('displays warning message', () => {
    render(
      <DeleteBackupDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    expect(screen.getAllByText(/This action cannot be undone/)).toHaveLength(2)
    expect(screen.getByText(/The backup file will be permanently removed/)).toBeInTheDocument()
  })

  it('calls onConfirm and onClose when delete button clicked', async () => {
    const user = userEvent.setup()
    render(
      <DeleteBackupDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    const deleteButton = screen.getByRole('button', { name: 'Confirm Delete' })
    await user.click(deleteButton)

    await waitFor(() => {
      expect(mockOnConfirm).toHaveBeenCalled()
      expect(mockOnClose).toHaveBeenCalled()
    })
  })

  it('calls onClose when cancel button clicked', async () => {
    const user = userEvent.setup()
    render(
      <DeleteBackupDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    const cancelButton = screen.getByRole('button', { name: 'Cancel' })
    await user.click(cancelButton)

    expect(mockOnClose).toHaveBeenCalled()
    expect(mockOnConfirm).not.toHaveBeenCalled()
  })

  it('calls onClose when X button clicked', async () => {
    const user = userEvent.setup()
    render(
      <DeleteBackupDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    const xButton = screen.getByLabelText('Close dialog')
    await user.click(xButton)

    expect(mockOnClose).toHaveBeenCalled()
    expect(mockOnConfirm).not.toHaveBeenCalled()
  })

  it('disables buttons when isDeleting is true', () => {
    render(
      <DeleteBackupDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
        isDeleting={true}
      />
    )

    const deleteButton = screen.getByRole('button', { name: 'Deleting...' })
    const cancelButton = screen.getByRole('button', { name: 'Cancel' })

    expect(deleteButton).toBeDisabled()
    expect(cancelButton).toBeDisabled()
  })

  it('shows "Deleting..." text when isDeleting is true', () => {
    render(
      <DeleteBackupDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
        isDeleting={true}
      />
    )

    expect(screen.getByText('Deleting...')).toBeInTheDocument()
  })

  it('handles async onConfirm function', async () => {
    const asyncConfirm = vi.fn().mockResolvedValue(undefined)
    const user = userEvent.setup()

    render(
      <DeleteBackupDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={asyncConfirm}
        timestamp={mockTimestamp}
      />
    )

    const deleteButton = screen.getByRole('button', { name: 'Confirm Delete' })
    await user.click(deleteButton)

    await waitFor(() => {
      expect(asyncConfirm).toHaveBeenCalled()
      expect(mockOnClose).toHaveBeenCalled()
    })
  })

  it('handles onConfirm error gracefully', async () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})
    const errorConfirm = vi.fn().mockRejectedValue(new Error('Delete failed'))
    const user = userEvent.setup()

    render(
      <DeleteBackupDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={errorConfirm}
        timestamp={mockTimestamp}
      />
    )

    const deleteButton = screen.getByRole('button', { name: 'Confirm Delete' })
    await user.click(deleteButton)

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('Failed to delete backup. Please try again.')
      expect(mockOnClose).not.toHaveBeenCalled() // Should not close on error
    })

    alertSpy.mockRestore()
  })

  it('displays trash icon with correct styling', () => {
    render(
      <DeleteBackupDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    const iconContainer = screen.getByText(/Delete backup from/).closest('div')?.previousSibling as HTMLElement | null
    const innerDiv = iconContainer?.querySelector('div')
    expect(innerDiv).toHaveClass('bg-red-100', 'dark:bg-red-900/30')
  })

  it('applies correct button styling', () => {
    render(
      <DeleteBackupDialog
        isOpen={true}
        onClose={mockOnClose}
        onConfirm={mockOnConfirm}
        timestamp={mockTimestamp}
      />
    )

    const deleteButton = screen.getByRole('button', { name: 'Confirm Delete' })
    expect(deleteButton).toHaveClass('bg-red-600')

    const cancelButton = screen.getByRole('button', { name: 'Cancel' })
    expect(cancelButton).toHaveClass('border-slate-200')
  })
})
